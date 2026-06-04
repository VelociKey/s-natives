package lifecycle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func GetExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func ReadGoalFromFile(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(content)), nil
	}
	// If the file does not exist, treat the path parameter itself as the raw goal string!
	return strings.TrimSpace(path), nil
}

func BuildSandboxEnv(workspaceRoot string, trackingScope string) []string {
	sandboxHome := filepath.Join(workspaceRoot, "00flow/s-forge/94000-external-actors/jules/c1000-credentials")
	if trackingScope != "" {
		sandboxHome = filepath.Join(sandboxHome, trackingScope)
	}
	if err := os.MkdirAll(sandboxHome, 0755); err != nil {
		slog.Error("Failed to create credentials directory", "error", err, "path", sandboxHome)
	}

	var pathVal string
	hermeticGoBin := filepath.Join(workspaceRoot, "00flow/s-forge/92000-external-toolchains/go/bin")
	if runtime.GOOS == "windows" {
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = `C:\Windows`
		}
		pathVal = fmt.Sprintf("%s;%s;%s", hermeticGoBin, filepath.Join(sysRoot, "System32"), sysRoot)
	} else {
		pathVal = fmt.Sprintf("%s:/bin:/usr/bin:/usr/local/bin", hermeticGoBin)
	}

	env := []string{
		"PATH=" + pathVal,
	}

	if runtime.GOOS == "windows" {
		userDir := filepath.Join(sandboxHome, "user")
		appDataDir := filepath.Join(sandboxHome, "appdata")
		localAppDataDir := filepath.Join(sandboxHome, "localappdata")
		programDataDir := filepath.Join(sandboxHome, "programdata")

		if err := os.MkdirAll(userDir, 0755); err != nil {
			slog.Error("Failed to create user directory", "error", err, "path", userDir)
		}
		if err := os.MkdirAll(appDataDir, 0755); err != nil {
			slog.Error("Failed to create appdata directory", "error", err, "path", appDataDir)
		}
		if err := os.MkdirAll(localAppDataDir, 0755); err != nil {
			slog.Error("Failed to create localappdata directory", "error", err, "path", localAppDataDir)
		}
		if err := os.MkdirAll(programDataDir, 0755); err != nil {
			slog.Error("Failed to create programdata directory", "error", err, "path", programDataDir)
		}

		env = append(env,
			"USERPROFILE="+userDir,
			"APPDATA="+appDataDir,
			"LOCALAPPDATA="+localAppDataDir,
			"ProgramData="+programDataDir,
			"SystemDrive="+os.Getenv("SystemDrive"),
			"SystemRoot="+os.Getenv("SystemRoot"),
		)
	} else {
		homeDir := filepath.Join(sandboxHome, "home")
		cacheDir := filepath.Join(sandboxHome, "cache")
		configDir := filepath.Join(sandboxHome, "config")

		if err := os.MkdirAll(homeDir, 0755); err != nil {
			slog.Error("Failed to create home directory", "error", err, "path", homeDir)
		}
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			slog.Error("Failed to create cache directory", "error", err, "path", cacheDir)
		}
		if err := os.MkdirAll(configDir, 0755); err != nil {
			slog.Error("Failed to create config directory", "error", err, "path", configDir)
		}

		env = append(env,
			"HOME="+homeDir,
			"XDG_CACHE_HOME="+cacheDir,
			"XDG_CONFIG_HOME="+configDir,
		)
	}
	return env
}

