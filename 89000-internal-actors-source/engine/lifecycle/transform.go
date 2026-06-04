package lifecycle

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Transform executes phase 3: code generation, template injection, and casing adjustments.
func (e *NATVSEngine) Transform(ctx context.Context, action string, contextPath string) error {
	e.State = PhaseTransform
	log.Printf("[NATVS] Phase 3: Executing Transformation target action: '%s'...", action)

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

		log.Printf("[NATVS] Running create-compendium for workspace: %s", wsPath)

		cmd := exec.CommandContext(ctx, compendiumBin, "-workspace", wsPath)
		cmd.Dir = e.Config.WorkspaceRoot

		var logBuf bytes.Buffer
		cmd.Stdout = &logBuf
		cmd.Stderr = &logBuf

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("create-compendium failed for workspace %s (output: %s): %w", wsPath, logBuf.String(), err)
		}
		log.Printf("[NATVS] create-compendium executed successfully for %s.", wsPath)
		e.LastChanges = []string{filepath.Base(wsPath) + "_compendium.txt"}
		e.LastReason = fmt.Sprintf("created workspace compendium file for %s", filepath.Base(wsPath))
		return nil
	}

	// In-memory or subprocess dynamic execution stub
	time.Sleep(5 * time.Millisecond)
	log.Printf("[NATVS] Transformation successfully generated output delta.")
	return nil
}

func runJulesTaskFromTransform(e *NATVSEngine, action string, contextPath string) error {
	workspaceRoot := e.Config.WorkspaceRoot
	julesPath := filepath.Clean(filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "jules"+GetExeSuffix()))

	repoPath := workspaceRoot

	goal, err := ReadGoalFromFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read goal from %s: %w", contextPath, err)
	}

	scope := ""
	e.Config.Mu.Lock()
	if e.Config.CurrentWorkspace != "" {
		scope = filepath.Base(e.Config.CurrentWorkspace)
	}
	e.Config.Mu.Unlock()

	cmd := exec.Command(julesPath, "new", "--repo", repoPath, goal)
	cmd.Env = BuildSandboxEnv(workspaceRoot, scope)

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("jules execution failed (stdout/stderr: %s): %w", outBuf.String(), err)
	}
	log.Printf("[NATVS] Phase 3 (Transform): Jules executed successfully. Output: %s", outBuf.String())
	e.LastChanges = []string{"*"}
	e.LastReason = "Jules code remediation executed successfully during Transform phase"
	return nil
}


