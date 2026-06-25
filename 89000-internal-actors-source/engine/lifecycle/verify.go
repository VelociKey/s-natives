package lifecycle

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Verification executes phase 4 conformance audits and test compilations.
func (e *NATVSEngine) Verification(ctx context.Context, testPackage string) error {
	if os.Getenv("SKIP_RECURSIVE_TESTS") == "true" {
		slog.Info("Skipping verification harness under test environment")
		return nil
	}

	testPackage = filepath.ToSlash(testPackage)
	testPackage = strings.TrimPrefix(testPackage, "./")

	goBin := discoverGoBinary(e.Config.WorkspaceRoot)

	if testPackage == "all-00flow-workspaces" {

		workspaces, err := e.discoverWorkspaces()
		if err != nil {
			slog.Warn("Failed to discover workspaces from go.work; falling back to default list", "err", err)
			workspaces = []string{
				"s-actors",
				"s-forge",
				"s-latentlingua",
				"s-natives",
				"s-seed",
				"s-adk",
				"s-a2a",
			}
		}

		if len(e.Config.AllowedWorkspaces) > 0 {
			filtered := make([]string, 0, len(workspaces))
			for _, ws := range workspaces {
				for _, allowed := range e.Config.AllowedWorkspaces {
					if ws == allowed || strings.TrimPrefix(ws, "s-") == strings.TrimPrefix(allowed, "s-") {
						filtered = append(filtered, ws)
						break
					}
				}
			}
			workspaces = filtered
			slog.Info("Restricting verification scope", "workspaces", workspaces)
		}

		failedWorkspaces := make([]string, 0, len(workspaces))
		for _, ws := range workspaces {
			wsPath := filepath.Join(e.Config.WorkspaceRoot, "00flow", ws)
			if _, err := os.Stat(wsPath); os.IsNotExist(err) {
				slog.Debug("Skipping non-existent workspace directory", "path", wsPath)
				continue
			}
			slog.Info("Checking workspace", "workspace", ws)

			// Pre-compile using int-rehydrator in local-only mode
			rehydratorBin := filepath.Clean(filepath.Join(e.Config.WorkspaceRoot, "00flow", "s-hydration", "int-rehydrator"+getExeSuffix()))
			harnessPath := filepath.Clean(filepath.Join(wsPath, "71000-build-harness", "workspace.harness"))
			if _, err := os.Stat(harnessPath); os.IsNotExist(err) {
				harnessPath = filepath.Join(wsPath, "workspace.harness")
			}
			if _, err := os.Stat(harnessPath); err == nil {
				slog.Info("Executing pre-verification local rehydration", "workspace", ws)
				cmdRehydrate := exec.CommandContext(ctx, rehydratorBin, "-harness", harnessPath, "-local-only")
				SetNoWindow(cmdRehydrate)
				cmdRehydrate.Dir = e.Config.WorkspaceRoot
				var rehydrateBuf bytes.Buffer
				rehydrateBuf.Grow(8192)
				cmdRehydrate.Stdout = &rehydrateBuf
				cmdRehydrate.Stderr = &rehydrateBuf
				if err := cmdRehydrate.Run(); err != nil {
					slog.Error("Pre-verification rehydration failed", "workspace", ws, "output", rehydrateBuf.String())
					if isSystemCrash(rehydrateBuf.String()) {
						return fmt.Errorf("CRITICAL ENVIRONMENTAL FAULT DETECTED: %s", rehydrateBuf.String())
					}
					e.LastVerificationFailureLogs = rehydrateBuf.String()
					failedWorkspaces = append(failedWorkspaces, ws+" (rehydration)")
					continue
				}
			}

			// Run Conformance Scan
			if err := e.runConformanceCheck(ctx, wsPath); err != nil {
				slog.Error("Conformance failed for workspace", "workspace", ws)
				if isSystemCrash(err.Error()) {
					return err
				}
				failedWorkspaces = append(failedWorkspaces, ws+" (conformance)")
				continue
			}

			dirsToTest := make([]string, 0, 10)
			rootHasTests := false
			files, err := os.ReadDir(wsPath)
			if err == nil {
				type subResult struct {
					name     string
					hasTests bool
				}
				var scanWg sync.WaitGroup
				resultsChan := make(chan subResult, len(files))

				for _, f := range files {
					if !f.IsDir() {
						if strings.HasSuffix(f.Name(), "_test.go") {
							rootHasTests = true
						}
					} else {
						name := f.Name()
						if !(len(name) >= 5 && name[0] == '9') {
							scanWg.Add(1)
							go func(subName string) {
								defer scanWg.Done()
								subHasTests := false
								walkErr := filepath.WalkDir(filepath.Join(wsPath, subName), func(path string, d os.DirEntry, err error) error {
									if err != nil {
										return nil
									}
									if d.IsDir() {
										nameOnly := d.Name()
										if len(nameOnly) >= 5 && nameOnly[0] == '9' {
											return filepath.SkipDir
										}
										return nil
									}
									if strings.HasSuffix(d.Name(), "_test.go") {
										subHasTests = true
										return errors.New("stop walking")
									}
									return nil
								})
								if walkErr != nil && walkErr.Error() != "stop walking" {
									slog.Warn("filepath.WalkDir encountered error", "err", walkErr)
								}
								resultsChan <- subResult{name: subName, hasTests: subHasTests}
							}(name)
						}
					}
				}

				scanWg.Wait()
				close(resultsChan)

				for res := range resultsChan {
					if res.hasTests {
						dirsToTest = append(dirsToTest, "./"+res.name+"/...")
					}
				}
			}

			if rootHasTests {
				dirsToTest = append(dirsToTest, ".")
			}

			if len(dirsToTest) == 0 {
				slog.Info("No non-9xxxx test packages found in workspace. Skipping.", "workspace", ws)
				continue
			}

			slog.Info("Running tests in packages", "packages", dirsToTest, "workspace", ws)
			args := append([]string{"test", "-v"}, dirsToTest...)
			cmd := exec.CommandContext(ctx, goBin, args...)
			SetNoWindow(cmd)
			cmd.Dir = wsPath
			var logBuf bytes.Buffer
			logBuf.Grow(8192)
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf

			runErr := cmd.Run()
			if runErr != nil {
				if isSystemCrash(logBuf.String()) {
					return fmt.Errorf("CRITICAL ENVIRONMENTAL FAULT DETECTED: %s", logBuf.String())
				}
				e.LastVerificationFailureLogs = logBuf.String()
				slog.Error("Tests failed in workspace", "workspace", ws, "output", logBuf.String())
				slog.Error("Tests failed in workspace details", "workspace", ws, "err", runErr)
				failedWorkspaces = append(failedWorkspaces, ws+" (tests)")
			} else {
				slog.Info("Workspace conformed successfully!", "workspace", ws)
			}
		}

		if len(failedWorkspaces) > 0 {
			return fmt.Errorf("verification failed for workspaces: %s", strings.Join(failedWorkspaces, ", "))
		}

		slog.Info("Conformance verified! All tests successfully executed across all 00flow workspaces.")
		return nil
	}

	// Specific package verification path
	slog.Info("Running verification for specific target package", "package", testPackage)

	// Run Conformance check for specific package
	var targetDir string
	if strings.HasPrefix(testPackage, "sov.fleet/") {
		relDir := strings.Replace(testPackage, "sov.fleet/", "00flow/", 1)
		relDir = strings.TrimSuffix(relDir, "/...")
		targetDir = filepath.Join(e.Config.WorkspaceRoot, relDir)
	} else {
		targetDir = filepath.Join(e.Config.WorkspaceRoot, testPackage)
	}

	// Pre-compile specific package using int-rehydrator
	rehydratorBin := filepath.Clean(filepath.Join(e.Config.WorkspaceRoot, "00flow", "s-hydration", "int-rehydrator"+getExeSuffix()))
	harnessPath := filepath.Clean(filepath.Join(targetDir, "71000-build-harness", "workspace.harness"))
	if _, err := os.Stat(harnessPath); os.IsNotExist(err) {
		harnessPath = filepath.Join(targetDir, "workspace.harness")
	}
	if _, err := os.Stat(harnessPath); err == nil {
		slog.Info("Executing pre-verification local rehydration for target", "package", testPackage)
		cmdRehydrate := exec.CommandContext(ctx, rehydratorBin, "-harness", harnessPath, "-local-only")
		SetNoWindow(cmdRehydrate)
		cmdRehydrate.Dir = e.Config.WorkspaceRoot
		var rehydrateBuf bytes.Buffer
		rehydrateBuf.Grow(8192)
		cmdRehydrate.Stdout = &rehydrateBuf
		cmdRehydrate.Stderr = &rehydrateBuf
		if err := cmdRehydrate.Run(); err != nil {
			slog.Error("Pre-verification rehydration failed", "package", testPackage, "output", rehydrateBuf.String())
			if isSystemCrash(rehydrateBuf.String()) {
				return fmt.Errorf("CRITICAL ENVIRONMENTAL FAULT DETECTED: %s", rehydrateBuf.String())
			}
			e.LastVerificationFailureLogs = rehydrateBuf.String()
			return fmt.Errorf("pre-verification rehydration failed: %w", err)
		}
	}

	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if err := e.runConformanceCheck(ctx, targetDir); err != nil {
			return err
		}
	} else {
		slog.Warn("Conformance scan skipped; target directory not found or not a directory", "dir", targetDir)
	}

	// Determine package targets to test (avoiding 9xxxx and handling workspaces with no tests)
	var targets []string
	isWorkspacePackage := false
	var wsPath string
	if strings.HasPrefix(testPackage, "sov.fleet/") {
		trimmed := strings.TrimPrefix(testPackage, "sov.fleet/")
		trimmed = strings.TrimSuffix(trimmed, "/...")
		if !strings.Contains(trimmed, "/") {
			isWorkspacePackage = true
			wsPath = filepath.Join(e.Config.WorkspaceRoot, "00flow", trimmed)
		}
	} else if strings.HasPrefix(testPackage, "00flow/") {
		trimmed := strings.TrimPrefix(testPackage, "00flow/")
		trimmed = strings.TrimSuffix(trimmed, "/...")
		if !strings.Contains(trimmed, "/") {
			isWorkspacePackage = true
			wsPath = filepath.Join(e.Config.WorkspaceRoot, "00flow", trimmed)
		}
	} else if strings.HasPrefix(testPackage, "00xper/") {
		trimmed := strings.TrimPrefix(testPackage, "00xper/")
		trimmed = strings.TrimSuffix(trimmed, "/...")
		if !strings.Contains(trimmed, "/") {
			isWorkspacePackage = true
			wsPath = filepath.Join(e.Config.WorkspaceRoot, "00xper", trimmed)
		}
	}

	if isWorkspacePackage {
		dirsToTest := make([]string, 0, 10)
		rootHasTests := false
		files, err := os.ReadDir(wsPath)
		if err == nil {
			type subResult struct {
				name     string
				hasTests bool
			}
			var scanWg sync.WaitGroup
			resultsChan := make(chan subResult, len(files))

			for _, f := range files {
				if !f.IsDir() {
					if strings.HasSuffix(f.Name(), "_test.go") {
						rootHasTests = true
					}
				} else {
					name := f.Name()
					if !(len(name) >= 5 && name[0] == '9') {
						scanWg.Add(1)
						go func(subName string) {
							defer scanWg.Done()
							subHasTests := false
							walkErr := filepath.WalkDir(filepath.Join(wsPath, subName), func(path string, d os.DirEntry, err error) error {
								if err != nil {
									return nil
								}
								if d.IsDir() {
									nameOnly := d.Name()
									if len(nameOnly) >= 5 && nameOnly[0] == '9' {
										return filepath.SkipDir
									}
									return nil
								}
								if strings.HasSuffix(d.Name(), "_test.go") {
									subHasTests = true
									return errors.New("stop walking")
								}
								return nil
							})
							if walkErr != nil && walkErr.Error() != "stop walking" {
								slog.Warn("filepath.WalkDir encountered error", "err", walkErr)
							}
							resultsChan <- subResult{name: subName, hasTests: subHasTests}
						}(name)
					}
				}
			}

			scanWg.Wait()
			close(resultsChan)

			for res := range resultsChan {
				if res.hasTests {
					dirsToTest = append(dirsToTest, "sov.fleet/"+filepath.Base(wsPath)+"/"+res.name+"/...")
				}
			}
		}
		if rootHasTests {
			dirsToTest = append(dirsToTest, "sov.fleet/"+filepath.Base(wsPath))
		}
		if len(dirsToTest) == 0 {
			slog.Info("No non-9xxxx test packages found in workspace. Skipping test execution.", "workspace", filepath.Base(wsPath))
			return nil
		}
		targets = dirsToTest
	} else {
		targets = []string{testPackage}
	}

	if len(targets) > 0 {
		slog.Info("Executing batched verification test targets", "targets", targets)
		args := append([]string{"test", "-v"}, targets...)
		cmd := exec.CommandContext(ctx, goBin, args...)
		SetNoWindow(cmd)
		cmd.Dir = e.Config.WorkspaceRoot
		var logBuf bytes.Buffer
		logBuf.Grow(8192)
		cmd.Stdout = &logBuf
		cmd.Stderr = &logBuf
		if err := cmd.Run(); err != nil {
			if isSystemCrash(logBuf.String()) {
				return fmt.Errorf("CRITICAL ENVIRONMENTAL FAULT DETECTED: %s", logBuf.String())
			}
			e.LastVerificationFailureLogs = logBuf.String()
			slog.Error("Verification failed for batched packages", "targets", targets, "output", logBuf.String())
			return fmt.Errorf("verification failed for packages %v: %w", targets, err)
		}
	}

	slog.Info("Conformance verified! Tests successfully executed under Go runtime.")
	return nil
}

func (e *NATVSEngine) runConformanceCheck(ctx context.Context, targetDir string) error {
	conformanceBin := filepath.Clean(filepath.Join(e.Config.WorkspaceRoot, "00flow", "s-seed", "conformance"+getExeSuffix()))
	reportDir := filepath.Clean(filepath.Join(e.Config.WorkspaceRoot, "00flow", "s-forge", "08000-attestation-snapshot"))

	slog.Info("Running static conformance scanner", "dir", targetDir)

	cmd := exec.CommandContext(ctx, conformanceBin, "-dir", targetDir, "-report-dir", reportDir)
	SetNoWindow(cmd)
	cmd.Dir = e.Config.WorkspaceRoot

	var logBuf bytes.Buffer
	logBuf.Grow(8192)
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Run(); err != nil {
		slog.Error("Conformance check failed for directory", "directory", targetDir, "output", logBuf.String())
		if isSystemCrash(logBuf.String()) {
			return fmt.Errorf("CRITICAL ENVIRONMENTAL FAULT DETECTED: %s", logBuf.String())
		}
		return fmt.Errorf("conformance check failed: %w", err)
	}
	slog.Info("Conformance checks passed for directory", "dir", targetDir)
	return nil
}

// discoverWorkspaces dynamically reads go.work to extract registered workspace paths.
func (e *NATVSEngine) discoverWorkspaces() ([]string, error) {
	goWorkPath := filepath.Join(e.Config.WorkspaceRoot, "go.work")
	file, err := os.Open(goWorkPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var discovered []string
	scanner := bufio.NewScanner(file)
	inUseBlock := false

	for scanner.Scan() {
		line := scanner.Bytes()
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte("use (")) {
			inUseBlock = true
			continue
		}
		if inUseBlock && bytes.HasPrefix(line, []byte(")")) {
			inUseBlock = false
			continue
		}

		if inUseBlock {
			idx := bytes.Index(line, []byte("./00flow/"))
			if idx >= 0 {
				path := line[idx+len("./00flow/"):]
				path = bytes.TrimFunc(path, func(r rune) bool {
					return r == '/' || r == '\\' || r == '"' || r == '\'' || r == '(' || r == ')'
				})
				if len(path) > 0 {
					discovered = append(discovered, string(path))
				}
			}
		} else if bytes.HasPrefix(line, []byte("use ")) {
			idx := bytes.Index(line, []byte("./00flow/"))
			if idx >= 0 {
				path := line[idx+len("./00flow/"):]
				path = bytes.TrimFunc(path, func(r rune) bool {
					return r == '/' || r == '\\' || r == '"' || r == '\'' || r == '(' || r == ')'
				})
				if len(path) > 0 {
					discovered = append(discovered, string(path))
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return discovered, nil
}

// discoverGoBinary attempts to find the Go binary from parent variables or path lookup.
func discoverGoBinary(workspaceRoot string) string {
	forgeGo := filepath.Join(workspaceRoot, "00flow/s-forge/92000-external-toolchains/go/bin/go"+getExeSuffix())
	if _, err := os.Stat(forgeGo); err == nil {
		return forgeGo
	}
	if goBin := os.Getenv("ANTIGRAVITY_GO_BIN"); goBin != "" {
		return goBin
	}
	if pathBin, err := exec.LookPath("go"); err == nil {
		return pathBin
	}
	return forgeGo
}

func getExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func isSystemCrash(output string) bool {
	lower := strings.ToLower(output)
	crashes := []string{
		"segfault",
		"segmentation fault",
		"access_violation",
		"status_access_violation",
		"kernel panic",
		"disk access restricted",
		"corrupted header",
		"fatal error: cgo",
	}
	for _, term := range crashes {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

