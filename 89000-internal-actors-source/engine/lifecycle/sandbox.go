package lifecycle

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	"strings"
)

func GetExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func ReadGoalFromFile(path string) (string, error) {
	if discardValLine21_0, err := os.Stat(path); err == nil {
		discard.Discard(discardValLine21_0)
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
	sandboxHome := filepath.Join(workspaceRoot, "00flow/s-natives/c1000-credentials")
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

	// Metabolic prune Chrome User Data bloat before spawning tasks
	go PruneChromeProfile(sandboxHome)

	return env
}

// PruneChromeProfile scans the remapped Chrome profile directory and recursively deletes all files
// that do not contain token or session cookies (such as code caches, dictionaries, extensions, model stores).
func PruneChromeProfile(sandboxHome string) {
	localApp := filepath.Join(sandboxHome, "localappdata", "Google", "Chrome", "User Data")
	if discardValLine116_0, err := os.Stat(localApp); os.IsNotExist(err) {
		discard.Discard(discardValLine116_0)
		return
	}

	// Whitelisted files and folders essential for cookie-based session token storage
	isWhitelisted := func(path string) bool {
		pathSlash := strings.ToLower(filepath.ToSlash(path))

		// Always whitelisting parent structure paths leading to network cookies and local storage
		if strings.Contains(pathSlash, "/user data/default") {
			if strings.Contains(pathSlash, "/network") || strings.Contains(pathSlash, "/local storage") {
				return true
			}
			// Let directory walk proceed down into Network or Local Storage
			if pathSlash == strings.ToLower(filepath.ToSlash(filepath.Join(localApp, "Default"))) {
				return true
			}
		}

		parts := strings.Split(pathSlash, "/")
		for _, part := range parts {
			if part == "whoami.txt" || part == "tier.txt" {
				return true
			}
		}
		return false
	}

	filepath.WalkDir(localApp, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// If it's a file, delete if not whitelisted
		if !isWhitelisted(path) {
			discard.Discard(os.Remove(path))
		}
		return nil
	})
}
