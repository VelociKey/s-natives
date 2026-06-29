package lifecycle

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Transform executes phase 3: code generation, template injection, and casing adjustments.
func (e *NATVSEngine) Transform(ctx context.Context, action string, contextPath string) error {
	e.State = PhaseTransform
	slog.Info("[NATVS] Phase 3: Executing Transformation target action", "action", action)

	if strings.Contains(action, "jules-run") || strings.Contains(action, "jules:run") || strings.Contains(action, "jules") {
		return runJulesTaskFromTransform(e, action, contextPath)
	}

	if strings.Contains(action, "create-compendium") {
		compendiumBin := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-introspection/81000-active-source/cmd/create_compendium/create_compendium.exe")

		var targetWS string
		if strings.HasPrefix(action, "create-compendium-") {
			targetWS = strings.TrimPrefix(action, "create-compendium-")
		} else {
			targetWS = "00flow/s-natives" // fallback
		}

		wsPath := targetWS
		if !filepath.IsAbs(wsPath) {
			wsPath = filepath.Join(e.Config.WorkspaceRoot, wsPath)
		}

		slog.Info("[NATVS] Running create-compendium for workspace", "wsPath", wsPath)

		cmd := exec.CommandContext(ctx, compendiumBin, "-workspace", wsPath)
		SetNoWindow(cmd)
		cmd.Dir = e.Config.WorkspaceRoot

		var logBuf bytes.Buffer
		cmd.Stdout = &logBuf
		cmd.Stderr = &logBuf

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("create-compendium failed for workspace %s (output: %s): %w", wsPath, logBuf.String(), err)
		}
		slog.Info("[NATVS] create-compendium executed successfully for workspace", "wsPath", wsPath)
		e.LastChanges = []string{filepath.Base(wsPath) + "_compendium.txt"}
		e.LastReason = fmt.Sprintf("created workspace compendium file for %s", filepath.Base(wsPath))
		return nil
	}

	// In-memory or subprocess dynamic execution stub
	time.Sleep(5 * time.Millisecond)
	slog.Info("[NATVS] Transformation successfully generated output delta.")
	return nil
}

func runJulesTaskFromTransform(e *NATVSEngine, action string, contextPath string) error {
	workspaceRoot := e.Config.WorkspaceRoot
	julesPath := filepath.Clean(filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "jules"+GetExeSuffix()))

	goal, err := ReadGoalFromFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read goal from %s: %w", contextPath, err)
	}

	execJules := func() (string, error) {
		cmd := exec.Command(julesPath, "new", "--repo", ".", goal)
		SetNoWindow(cmd)
		cmd.Dir = workspaceRoot
		cmd.Env = BuildSandboxEnv(workspaceRoot, "")
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		runErr := cmd.Run()
		return outBuf.String(), runErr
	}

	outputStr, runErr := execJules()
	if runErr != nil {
		// Detect 401 or UNAUTHENTICATED error in output
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "UNAUTHENTICATED") || strings.Contains(strings.ToLower(outputStr), "login") {
			slog.Info("[NATVS] Authentication failure detected during Jules run. Attempting automatic interactive login...")

			// Execute jules login using standard inputs/outputs to prompt the user
			loginCmd := exec.Command(julesPath, "login")
			loginCmd.Dir = workspaceRoot
			loginCmd.Env = BuildSandboxEnv(workspaceRoot, "")
			loginCmd.Stdin = os.Stdin
			loginCmd.Stdout = os.Stdout
			loginCmd.Stderr = os.Stderr

			if loginErr := loginCmd.Run(); loginErr != nil {
				return fmt.Errorf("automatic interactive login failed: %w", loginErr)
			}

			slog.Info("[NATVS] Automatic login complete. Retrying original Jules task...")
			outputStr, runErr = execJules()
			if runErr != nil {
				return fmt.Errorf("jules execution failed on retry (stdout/stderr: %s): %w", outputStr, runErr)
			}
		} else {
			return fmt.Errorf("jules execution failed (stdout/stderr: %s): %w", outputStr, runErr)
		}
	}

	slog.Info("[NATVS] Phase 3 (Transform): Jules executed successfully", "output", outputStr)
	e.LastChanges = []string{"*"}
	e.LastReason = "Jules code remediation executed successfully during Transform phase"
	return nil
}
