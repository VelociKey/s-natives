package natives_attestations

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func TestNATVSComprehensive(t *testing.T) {
	// Create hermetic temp workspace
	tempDir := tempDirInWorkspace(t)

	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "000all/s-cognition"),
		filepath.Join(tempDir, "00flow/s-forge/90000-authority"),
		filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed"),
		filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}

	workspaces := []string{
		"s-actors",
		"s-forge",
		"s-latentlingua",
		"s-natives",
		"s-seed",
		"s-adk",
		"s-a2a",
	}
	for _, ws := range workspaces {
		wsDir := filepath.Join(tempDir, "00flow", ws)
		err := os.MkdirAll(wsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create workspace directory %s: %v", ws, err)
		}
		// Write dummy go.mod
		err = os.WriteFile(filepath.Join(wsDir, "go.mod"), []byte("module sov.fleet/"+ws+"\ngo 1.26.3\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to write dummy go.mod for %s: %v", ws, err)
		}
	}

	// Copy conformance.exe to tempDir/00flow/s-seed/conformance.exe
	srcConf := filepath.Join(getWorkspaceRoot(), "00flow", "s-seed", "conformance.exe")
	destConf := filepath.Join(tempDir, "00flow/s-seed/conformance.exe")
	errCopy := copyFile(srcConf, destConf)
	if errCopy != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", errCopy)
	}

	// Write dummy files for casing refactoring tests
	dummyGoFile := filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed/test.go")
	err := os.WriteFile(dummyGoFile, []byte("package main\nimport \"s"+"Seed\"\n// s"+"Forge and s"+"Natives"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy Go file: %v", err)
	}

	dummyIgnoredFile := filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed/ignored.txt")
	err = os.WriteFile(dummyIgnoredFile, []byte("s"+"Seed and s"+"Forge"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy ignored file: %v", err)
	}

	dummyMDFile := filepath.Join(tempDir, "000all/s-cognition/test.md")
	err = os.WriteFile(dummyMDFile, []byte("# s"+"Natives and s"+"Aether"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy MD file: %v", err)
	}

	engine := NewNATVSEngine(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Negotiation Phase
	err = engine.Negotiate(ctx, "00flow/s-natives")
	if err != nil {
		t.Fatalf("Negotiation phase failed: %v", err)
	}
	if engine.State != PhaseNegotiate {
		t.Errorf("Expected state %s, got %s", PhaseNegotiate, engine.State)
	}

	// 2. Assimilation Phase
	// (a) Valid workspace
	err = engine.Assimilation(ctx, "00flow/s-forge")
	if err != nil {
		t.Fatalf("Assimilation phase failed: %v", err)
	}
	if engine.State != PhaseAssimilation {
		t.Errorf("Expected state %s, got %s", PhaseAssimilation, engine.State)
	}
	// (b) Non-existent workspace (error check)
	err = engine.Assimilation(ctx, "00flow/nonexistent")
	if err == nil {
		t.Errorf("Expected assimilation failure for non-existent workspace")
	}

	// 3. Transformation Phase (Fallback Check)
	err = engine.Transform(ctx, "verify-self", "")
	if err != nil {
		t.Fatalf("Transformation failed: %v", err)
	}

	// 4. Verification Phase
	// All workspaces package walk (no test files case)
	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err != nil {
		t.Fatalf("Verification phase all workspaces failed: %v", err)
	}

	// 5. Synthesis Phase
	err = engine.Synthesis(ctx, "aether_wasm_gc.wasm")
	if err != nil {
		t.Fatalf("Synthesis phase failed: %v", err)
	}
	if engine.State != PhaseSynthesis {
		t.Errorf("Expected state %s, got %s", PhaseSynthesis, engine.State)
	}
}

func TestMainFunc(t *testing.T) {
	t.Setenv("SKIP_RECURSIVE_TESTS", "true")

	tempDir := tempDirInWorkspace(t)
	dirs := []string{
		filepath.Join(tempDir, "000all/s-cognition"),
		filepath.Join(tempDir, "00flow/s-natives"),
		filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory for main check: %v", err)
		}
	}

	goBin := filepath.Clean(filepath.Join(getWorkspaceRoot(), "00flow", "s-forge", "92000-external-toolchains", "go", "bin", "go"+getExeSuffix()))
	mainGoDir := filepath.Clean(filepath.Join(getWorkspaceRoot(), "00flow", "s-natives", "89000-internal-actors-source"))
	cmd := exec.Command(goBin, "run", mainGoDir)
	cmd.Env = append(os.Environ(), "TEST_WORKSPACE_ROOT="+tempDir, "SKIP_RECURSIVE_TESTS=true")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to run natvs_engine package: %v", err)
	}
}

func TestNATVSVerificationFailure(t *testing.T) {
	tempDir := tempDirInWorkspace(t)

	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "00flow/s-actors"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}

	// Create a dummy test file in s-actors to make it look like it has tests
	dummyTestGoFile := filepath.Join(tempDir, "00flow/s-actors/dummy_test.go")
	err := os.WriteFile(dummyTestGoFile, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy test file: %v", err)
	}

	engine := NewNATVSEngine(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Ensure SKIP_RECURSIVE_TESTS is NOT set
	t.Setenv("SKIP_RECURSIVE_TESTS", "")

	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err == nil {
		t.Errorf("Expected Verification to fail because of missing/invalid goBin, but it passed")
	}
}

func TestMainFuncFailure(t *testing.T) {
	goBin := filepath.Clean(filepath.Join(getWorkspaceRoot(), "00flow", "s-forge", "92000-external-toolchains", "go", "bin", "go"+getExeSuffix()))
	mainGoDir := filepath.Clean(filepath.Join(getWorkspaceRoot(), "00flow", "s-natives", "89000-internal-actors-source"))
	cmd := exec.Command(goBin, "run", mainGoDir)
	cmd.Env = append(os.Environ(), "TEST_WORKSPACE_ROOT=/nonexistent_root_dir_abc", "SKIP_RECURSIVE_TESTS=true")
	err := cmd.Run()
	// Since we expect it to fail (fatal error exit 1), cmd.Run() should return an error.
	if err == nil {
		t.Error("Expected execution to fail with non-zero exit code, but it succeeded")
	}
}
