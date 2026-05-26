package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Verification executes phase 4 conformance audits and test compilations.
func (e *NATVSEngine) Verification(ctx context.Context, testPackage string) error {
	e.State = PhaseVerification
	slog.Info("Phase 4: Triggering verification harness suite", "package", testPackage)
	
	goBin := e.discoverGoBinary()
	
	if testPackage == "all-00flow-workspaces" {
		if os.Getenv("SKIP_RECURSIVE_TESTS") == "true" {
			slog.Info("Skipping recursive workspace verification walk under test environment")
			return nil
		}
		
		workspaces, err := e.discoverWorkspaces()
		if err != nil {
			slog.Warn("Failed to discover workspaces from go.work; falling back to default list", "err", err)
			workspaces = []string{
				"s-actors",
				"s-forge",
				"s-latentlingua",
				"s-natives",
				"s-seed",
				"s-mcp",
				"s-adk",
				"s-a2a",
				"s-aether",
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
			
			// Run Conformance Scan
			if err := e.runConformanceCheck(ctx, wsPath); err != nil {
				slog.Error("Conformance failed for workspace", "workspace", ws)
				failedWorkspaces = append(failedWorkspaces, ws+" (conformance)")
				continue
			}
			
			dirsToTest := make([]string, 0, 10)
			rootHasTests := false
			files, err := os.ReadDir(wsPath)
			if err == nil {
				for _, f := range files {
					if !f.IsDir() {
						if strings.HasSuffix(f.Name(), "_test.go") {
							rootHasTests = true
						}
					} else {
						name := f.Name()
						if !(len(name) >= 5 && name[0] == '9') {
							subHasTests := false
							if walkErr := filepath.Walk(filepath.Join(wsPath, name), func(path string, info os.FileInfo, err error) error {
								if err != nil {
									return nil
								}
								if info.IsDir() {
									subName := info.Name()
									if len(subName) >= 5 && subName[0] == '9' {
										return filepath.SkipDir
									}
									return nil
								}
								if strings.HasSuffix(info.Name(), "_test.go") {
									subHasTests = true
									return errors.New("stop walking")
								}
								return nil
							}); walkErr != nil && walkErr.Error() != "stop walking" {
								slog.Warn("filepath.Walk encountered error", "err", walkErr)
							}
							if subHasTests {
								dirsToTest = append(dirsToTest, "./"+name+"/...")
							}
						}
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
			cmd.Dir = wsPath
			var logBuf bytes.Buffer
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf
			
			runErr := cmd.Run()
			if runErr != nil {
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
	
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if err := e.runConformanceCheck(ctx, targetDir); err != nil {
			return err
		}
	} else {
		slog.Warn("Conformance scan skipped; target directory not found or not a directory", "dir", targetDir)
	}
	
	cmd := exec.CommandContext(ctx, goBin, "test", "-v", testPackage)
	cmd.Dir = e.Config.WorkspaceRoot
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Run(); err != nil {
		slog.Error("Verification failed for package", "package", testPackage, "output", logBuf.String())
		return fmt.Errorf("verification failed for package %s: %w", testPackage, err)
	}
	
	slog.Info("Conformance verified! Tests successfully executed under Go runtime.")
	return nil
}

func (e *NATVSEngine) runConformanceCheck(ctx context.Context, targetDir string) error {
	conformanceBin := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-seed/conformance.exe")
	reportDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-forge/08000-attestation-snapshot")
	
	slog.Info("Running static conformance scanner", "dir", targetDir)
	
	cmd := exec.CommandContext(ctx, conformanceBin, "-dir", targetDir, "-report-dir", reportDir)
	cmd.Dir = e.Config.WorkspaceRoot
	
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	
	if err := cmd.Run(); err != nil {
		slog.Error("Conformance check failed for directory", "directory", targetDir, "output", logBuf.String())
		return fmt.Errorf("conformance check failed: %w", err)
	}
	slog.Info("Conformance checks passed for directory", "dir", targetDir)
	return nil
}

// discoverWorkspaces dynamically reads go.work to extract registered workspace paths.
func (e *NATVSEngine) discoverWorkspaces() ([]string, error) {
	goWorkPath := filepath.Join(e.Config.WorkspaceRoot, "go.work")
	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(data), "\n")
	discovered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "./00flow/") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "./00flow/") {
					ws := strings.TrimPrefix(p, "./00flow/")
					ws = strings.Trim(ws, `/"'()`)
					ws = strings.TrimSpace(ws)
					if ws != "" {
						discovered = append(discovered, ws)
					}
				}
			}
		}
	}
	return discovered, nil
}

// discoverGoBinary attempts to find the Go binary from parent variables or path lookup.
func (e *NATVSEngine) discoverGoBinary() string {
	if goBin := os.Getenv("ANTIGRAVITY_GO_BIN"); goBin != "" {
		return goBin
	}
	if pathBin, err := exec.LookPath("go"); err == nil {
		return pathBin
	}
	return filepath.Join(e.Config.WorkspaceRoot, "00flow/s-forge/92000-external-toolchains/go/bin/go.exe")
}
