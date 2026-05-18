package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNATVSComprehensive(t *testing.T) {
	// Create hermetic temp workspace
	tempDir := t.TempDir()
	
	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "000all/s-cognition"),
		filepath.Join(tempDir, "00flow/s-aether"),
		filepath.Join(tempDir, "00flow/s-forge/90000-authority"),
		filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed"),
		filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed"),
		filepath.Join(tempDir, "00flow/s-natives"),
		filepath.Join(tempDir, "00flow/s-seed"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}
	
	// Write dummy files for casing refactoring tests
	dummyGoFile := filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed/test.go")
	err := os.WriteFile(dummyGoFile, []byte("package main\nimport \"s" + "Seed\"\n// s" + "Forge and s" + "Natives"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy Go file: %v", err)
	}
	
	dummyIgnoredFile := filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed/ignored.txt")
	err = os.WriteFile(dummyIgnoredFile, []byte("s" + "Seed and s" + "Forge"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy ignored file: %v", err)
	}
	
	dummyMDFile := filepath.Join(tempDir, "000all/s-cognition/test.md")
	err = os.WriteFile(dummyMDFile, []byte("# s" + "Natives and s" + "Aether"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy MD file: %v", err)
	}
	
	engine := NewNATVSEngine(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 1. Negotiation Phase
	err = engine.Negotiate(ctx, "00flow/s-aether")
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
	
	// 3. Transformation Phase (Refactoring)
	err = engine.Transform(ctx, "refactor-taxonomy-casing")
	if err != nil {
		t.Fatalf("Transformation refactoring failed: %v", err)
	}
	
	// Verify that conformed Go file was refactored
	conformedContent, err := os.ReadFile(dummyGoFile)
	if err != nil {
		t.Fatalf("Failed to read conformed file: %v", err)
	}
	strContent := string(conformedContent)
	if strings.Contains(strContent, "s" + "Seed") || strings.Contains(strContent, "s" + "Forge") || strings.Contains(strContent, "s" + "Natives") {
		t.Errorf("Conformed Go file was not refactored correctly: %q", strContent)
	}
	
	// Verify that conformed MD file was refactored
	conformedMDContent, err := os.ReadFile(dummyMDFile)
	if err != nil {
		t.Fatalf("Failed to read conformed MD file: %v", err)
	}
	strMD := string(conformedMDContent)
	if strings.Contains(strMD, "s" + "Natives") || strings.Contains(strMD, "s" + "Aether") {
		t.Errorf("Conformed MD file was not refactored correctly: %q", strMD)
	}
	
	// Verify that ignored file in 9xxxx directory was NOT modified
	ignoredContent, err := os.ReadFile(dummyIgnoredFile)
	if err != nil {
		t.Fatalf("Failed to read ignored file: %v", err)
	}
	strIgnored := string(ignoredContent)
	if !strings.Contains(strIgnored, "s" + "Seed") {
		t.Errorf("Ignored 9xxxx file was modified: %q", strIgnored)
	}
	
	// Check coverage of Transformation's unrecognized action fallback
	err = engine.Transform(ctx, "unknown-action")
	if err != nil {
		t.Fatalf("Transformation fallback failed: %v", err)
	}
	
	// 4. Verification Phase
	// (a) Single package fallback
	err = engine.Verification(ctx, "sov.fleet/s-aether")
	if err != nil {
		t.Fatalf("Verification phase single package failed: %v", err)
	}
	// (b) All workspaces package walk (no test files case)
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

func TestNATVSListenDaemonAndSACP(t *testing.T) {
	engine := NewNATVSEngine("C:\\aCogSpaceSeed")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	port := 51989
	
	// Start UDP server daemon
	go func() {
		_ = engine.RunDaemon(ctx, port)
	}()
	
	// Allow UDP socket start
	time.Sleep(100 * time.Millisecond)
	
	// Dial UDP client port
	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:51989")
	if err != nil {
		t.Fatalf("Failed to resolve RAddr: %v", err)
	}
	
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatalf("Failed to dial UDP server: %v", err)
	}
	defer conn.Close()
	
	// Dispatch SACP command frame
	cmdMsg := "CALL:orchestrate-workspace-migration;TARGET=s-aether"
	_, err = conn.Write([]byte(cmdMsg))
	if err != nil {
		t.Fatalf("Failed to write to UDP socket: %v", err)
	}
	
	// Read Response SACP frame
	buf := make([]byte, 1024)
	err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from UDP socket: %v", err)
	}
	
	response := string(buf[:n])
	if response != "ACK:orchestrate-workspace-migration;STATUS=PASS" {
		t.Errorf("Expected SACP ACK response frame, got: %q", response)
	}
	
	// Test sending invalid SACP frame to server to cover fallback paths
	_, _ = conn.Write([]byte("INVALID_FRAME"))
	time.Sleep(50 * time.Millisecond)
	
	// Cancel context to shut down RunDaemon cleanly
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestMainFunc(t *testing.T) {
	t.Setenv("SKIP_RECURSIVE_TESTS", "true")
	main()
}

func TestNATVSVerificationFailure(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "00flow/s-aether"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}
	
	// Create a dummy test file in s-aether to make it look like it has tests
	dummyTestGoFile := filepath.Join(tempDir, "00flow/s-aether/dummy_test.go")
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

func TestNATVSDaemonListenFailure(t *testing.T) {
	engine := NewNATVSEngine(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Bind to the port first
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:51999")
	if err != nil {
		t.Fatalf("Failed to resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()
	
	// Try to run RunDaemon on the same port, which must fail!
	err = engine.RunDaemon(ctx, 51999)
	if err == nil {
		t.Errorf("Expected RunDaemon to fail when port is already in use, but it passed")
	}
}

func TestMainFuncFailure(t *testing.T) {
	// Mock logFatal
	oldFatal := logFatal
	defer func() { logFatal = oldFatal }()
	
	fatalCalled := 0
	logFatal = func(format string, v ...interface{}) {
		fatalCalled++
	}
	
	// Force failure by running with invalid workspace root
	t.Setenv("TEST_WORKSPACE_ROOT", "/nonexistent_root_dir_abc")
	t.Setenv("SKIP_RECURSIVE_TESTS", "true")
	
	main()
	
	if fatalCalled == 0 {
		t.Errorf("Expected logFatal to be called, but it was not")
	}
}
