package natives_attestations

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func TestJulesQueueInvocation(t *testing.T) {
	tempWS := tempDirInWorkspace(t)

	// 1. Create directory structure
	julesDir := filepath.Join(tempWS, "00flow/s-forge/94000-external-actors/jules")
	coordDir := filepath.Join(tempWS, "c0990-ephemeral-scratch/natvs coordination")
	nativesDir := filepath.Join(tempWS, "00flow/s-natives")

	for _, dir := range []string{julesDir, coordDir, nativesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create s-seed directory and copy conformance.exe into it
	seedDir := filepath.Join(tempWS, "00flow/s-seed")
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		t.Fatalf("Failed to create s-seed dir: %v", err)
	}
	srcConf := filepath.Join(getWorkspaceRoot(), "00flow", "s-seed", "conformance.exe")
	destConf := filepath.Join(seedDir, "conformance.exe")
	if err := copyFile(srcConf, destConf); err != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", err)
	}

	// Create dummy go.mod in nativesDir to keep it valid
	err := os.WriteFile(filepath.Join(nativesDir, "go.mod"), []byte("module check\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy go.mod: %v", err)
	}

	// Create dummy natvs_engine.go in nativesDir to keep targetMarker search valid
	dummyEngineGo := filepath.Join(nativesDir, "natvs_engine.go")
	err = os.WriteFile(dummyEngineGo, []byte(`package main
import "log/slog"
func main() {
	slog.Info("=========================================================")
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy natvs_engine.go: %v", err)
	}

	// 2. Write a mock jules source file and compile it
	mockGoSrc := filepath.Join(tempWS, "mock_jules.go")
	mockSrcCode := `package main
import "os"
func main() {
	_, _ = os.Stdout.Write([]byte("Mock Jules executed\n"))
}
`
	if err := os.WriteFile(mockGoSrc, []byte(mockSrcCode), 0644); err != nil {
		t.Fatalf("Failed to write mock jules source: %v", err)
	}

	goBin := filepath.Join(getWorkspaceRoot(), "00flow", "s-forge", "92000-external-toolchains", "go", "bin", "go"+getExeSuffix())
	mockJulesExe := filepath.Join(julesDir, "jules.exe")

	cmdBuild := exec.Command(goBin, "build", "-o", mockJulesExe, mockGoSrc)
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to compile mock jules: %v", err)
	}

	// 3. Write empty queue file
	queuePath := filepath.Join(coordDir, "tasks.queue.webnf")
	outPath := filepath.Join(coordDir, "tasks.completed.webnf")
	if err := WriteQueueFileAtomic(queuePath, []TaskInput{}); err != nil {
		t.Fatalf("Failed to initialize queue file: %v", err)
	}

	// 4. Create a WebNF markdown goal file
	goalFile := filepath.Join(tempWS, "goal.webnf")
	goalContent := `# Goal
Review and improve s-natives workspace code.
`
	if err := os.WriteFile(goalFile, []byte(goalContent), 0644); err != nil {
		t.Fatalf("Failed to write goal file: %v", err)
	}

	// 5. Build/Locate compiled natvs-engine
	engineExe := filepath.Join(getWorkspaceRoot(), "00flow/s-natives", "natvs-engine.exe")

	// 6. Spawn the daemon manually or let delegation start it.
	cmdRun := exec.Command(engineExe, "jules-run", goalFile, "00flow/s-natives")
	cmdRun.Env = append(os.Environ(),
		"TEST_WORKSPACE_ROOT="+tempWS,
		"ANTIGRAVITY_AGENT=1",
	)

	output, err := cmdRun.CombinedOutput()
	if err != nil {
		t.Fatalf("natvs-engine delegation command failed (output: %s): %v", string(output), err)
	}

	// 7. Wait and verify task result in tasks.completed.webnf
	var completed []TaskOutput
	for i := 0; i < 30; i++ {
		completed, err = ParseCompletedFileAtomic(outPath)
		if err == nil && len(completed) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if len(completed) == 0 {
		t.Fatalf("Timed out waiting for task completion")
	}

	if completed[0].Status != StateCompletedSuccess {
		t.Fatalf("Expected task status to be success, got: %v (error: %s)", completed[0].Status, completed[0].Error)
	}
}

func TestDirectExecutionWithoutDaemon(t *testing.T) {
	tempWS := tempDirInWorkspace(t)

	// 1. Create directory structure
	julesDir := filepath.Join(tempWS, "00flow/s-forge/94000-external-actors/jules")
	nativesDir := filepath.Join(tempWS, "00flow/s-natives")

	for _, dir := range []string{julesDir, nativesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create s-seed directory and copy conformance.exe into it
	seedDir := filepath.Join(tempWS, "00flow/s-seed")
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		t.Fatalf("Failed to create s-seed dir: %v", err)
	}
	srcConf := filepath.Join(getWorkspaceRoot(), "00flow", "s-seed", "conformance.exe")
	destConf := filepath.Join(seedDir, "conformance.exe")
	if err := copyFile(srcConf, destConf); err != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", err)
	}

	// Create dummy go.mod in nativesDir to keep it valid
	err := os.WriteFile(filepath.Join(nativesDir, "go.mod"), []byte("module check\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy go.mod: %v", err)
	}

	// Create dummy natvs_engine.go in nativesDir to keep targetMarker search valid
	dummyEngineGo := filepath.Join(nativesDir, "natvs_engine.go")
	err = os.WriteFile(dummyEngineGo, []byte(`package main
import "log/slog"
func main() {
	slog.Info("=========================================================")
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy natvs_engine.go: %v", err)
	}

	// 2. Write mock jules
	mockGoSrc := filepath.Join(tempWS, "mock_jules.go")
	mockSrcCode := `package main
import "os"
func main() {
	_, _ = os.Stdout.Write([]byte("Mock Jules direct\n"))
}
`
	if err := os.WriteFile(mockGoSrc, []byte(mockSrcCode), 0644); err != nil {
		t.Fatalf("Failed to write mock jules source: %v", err)
	}

	goBin := filepath.Join(getWorkspaceRoot(), "00flow", "s-forge", "92000-external-toolchains", "go", "bin", "go"+getExeSuffix())
	mockJulesExe := filepath.Join(julesDir, "jules.exe")

	cmdBuild := exec.Command(goBin, "build", "-o", mockJulesExe, mockGoSrc)
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to compile mock jules: %v", err)
	}

	// 3. Create a WebNF markdown goal file
	goalFile := filepath.Join(tempWS, "goal.webnf")
	goalContent := `# Goal
Review and improve s-natives workspace code directly.
`
	if err := os.WriteFile(goalFile, []byte(goalContent), 0644); err != nil {
		t.Fatalf("Failed to write goal file: %v", err)
	}

	// 4. Build/Locate compiled natvs-engine
	engineExe := filepath.Join(getWorkspaceRoot(), "00flow/s-natives", "natvs-engine.exe")

	// 5. Invoke directly
	cmdRun := exec.Command(engineExe, "jules-run", goalFile, "00flow/s-natives", "--direct")
	cmdRun.Env = append(os.Environ(),
		"TEST_WORKSPACE_ROOT="+tempWS,
		"ANTIGRAVITY_AGENT=1",
	)

	output, err := cmdRun.CombinedOutput()
	if err != nil {
		t.Fatalf("natvs-engine direct execution failed (output: %s): %v", string(output), err)
	}

	// 6. Verify output confirms direct execution bypassing daemon queue
	outStr := string(output)
	if !strings.Contains(outStr, "Executing task directly in-process (bypassing daemon queue)...") {
		t.Errorf("Expected logs to indicate direct execution, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Direct task execution succeeded!") {
		t.Errorf("Expected logs to confirm direct task success, got: %s", outStr)
	}
}
