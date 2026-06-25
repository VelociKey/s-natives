package lifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorkspaceCache represents the cached state of all workspaces.
type WorkspaceCache struct {
	Keys map[string]string `json:"keys"`
}

// GetCacheFilePath returns the path to the cache JSON file.
func GetCacheFilePath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, "c0990-ephemeral-scratch", "natvs-cache", "cache.json")
}

// ComputeGitStateKey returns a string representing the Git state of a workspace relative folder.
func ComputeGitStateKey(workspaceRoot, workspaceRelPath string) (string, error) {
	targetDir := filepath.Join(workspaceRoot, workspaceRelPath)

	// 1. Get latest commit hash affecting the relative path
	cmdLog := exec.Command("git", "log", "-n", "1", "--pretty=format:%H", ".")
	SetNoWindow(cmdLog)
	cmdLog.Dir = targetDir
	var outLog bytes.Buffer
	cmdLog.Stdout = &outLog
	if err := cmdLog.Run(); err != nil {
		return "", fmt.Errorf("failed to get git log: %w", err)
	}
	commitHash := strings.TrimSpace(outLog.String())

	// 2. Get git status --porcelain for the relative path
	cmdStatus := exec.Command("git", "status", "--porcelain", ".")
	SetNoWindow(cmdStatus)
	cmdStatus.Dir = targetDir
	var outStatus bytes.Buffer
	cmdStatus.Stdout = &outStatus
	if err := cmdStatus.Run(); err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}
	statusStr := strings.TrimSpace(outStatus.String())

	// If clean, key is just commitHash
	if statusStr == "" {
		return commitHash, nil
	}

	// If dirty, compile metadata for the modified files to build a fast cache key
	lines := strings.Split(statusStr, "\n")
	var metadata strings.Builder
	metadata.WriteString(commitHash)
	metadata.WriteString("|")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		// Format: XY path or XY "path"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		relFile := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		absFile := filepath.Join(targetDir, relFile)
		info, err := os.Stat(absFile)
		if err == nil {
			metadata.WriteString(fmt.Sprintf("%s:%d:%d|", relFile, info.Size(), info.ModTime().UnixNano()))
		} else {
			// File might have been deleted, append its path and deletion signifier
			metadata.WriteString(fmt.Sprintf("%s:DELETED|", relFile))
		}
	}

	hasher := sha256.New()
	hasher.Write([]byte(metadata.String()))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// IsWorkspaceCached checks if the workspace has a matching successful cached state.
func IsWorkspaceCached(workspaceRoot, workspaceRelPath string) (bool, string, error) {
	key, err := ComputeGitStateKey(workspaceRoot, workspaceRelPath)
	if err != nil {
		return false, "", err
	}

	cacheFile := GetCacheFilePath(workspaceRoot)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, key, nil
		}
		return false, key, err
	}

	var cache WorkspaceCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return false, key, nil
	}

	cachedKey, ok := cache.Keys[workspaceRelPath]
	if !ok || cachedKey != key {
		return false, key, nil
	}

	return true, key, nil
}

// UpdateWorkspaceCache updates the cached key for a given workspace.
func UpdateWorkspaceCache(workspaceRoot, workspaceRelPath, key string) error {
	cacheFile := GetCacheFilePath(workspaceRoot)
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		slog.Error("Failed to create cache directory", "error", err, "path", filepath.Dir(cacheFile))
	}

	var cache WorkspaceCache
	cache.Keys = make(map[string]string)

	data, err := os.ReadFile(cacheFile)
	if err == nil {
		if errUnmarshal := json.Unmarshal(data, &cache); errUnmarshal != nil {
			slog.Error("Failed to unmarshal cache data", "error", errUnmarshal)
		}
	}

	if cache.Keys == nil {
		cache.Keys = make(map[string]string)
	}
	cache.Keys[workspaceRelPath] = key

	newData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, newData, 0644)
}

// InvalidateWorkspaceCache removes a workspace from the cache.
func InvalidateWorkspaceCache(workspaceRoot, workspaceRelPath string) error {
	cacheFile := GetCacheFilePath(workspaceRoot)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var cache WorkspaceCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}

	if cache.Keys != nil {
		delete(cache.Keys, workspaceRelPath)
		newData, err := json.MarshalIndent(cache, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(cacheFile, newData, 0644)
	}
	return nil
}

