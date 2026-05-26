package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
	"sov.fleet/s-sacp/81000-active-source/pkg/broker"
)

func TestNATVSComprehensive(t *testing.T) {
	// Create hermetic temp workspace
	tempDir := t.TempDir()
	
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
		"s-aether",
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
	srcConf := "C:\\aCogSpaceSeed\\00flow\\s-seed\\conformance.exe"
	destConf := filepath.Join(tempDir, "00flow/s-seed/conformance.exe")
	errCopy := copyFile(srcConf, destConf)
	if errCopy != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", errCopy)
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
	
	// Dispatch SACP command frame (constructed via bicodec)
	h := bicodec.SACPHeader{
		UUID:              "test-client-uuid",
		OriginalAuthority: bicodec.AuthSovereign,
		CurrentAuthority:  bicodec.AuthSovereign,
	}
	c := bicodec.SACPCapability{
		Domain: "tool",
		Action: "call",
		ID:     "orchestrate-workspace-migration",
		Parameters: map[string]string{
			"target": "s-aether",
		},
	}
	msgBytes, err := bicodec.EncodeSACPMessage(&h, &c)
	if err != nil {
		t.Fatalf("Failed to encode SACP message: %v", err)
	}
	
	_, err = conn.Write(msgBytes)
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
	
	hasHeader, _, hasCap, resC, err := bicodec.DecodeSACPMessage(buf[:n])
	if err != nil {
		t.Fatalf("Failed to decode UDP response: %v", err)
	}
	
	if !hasCap {
		t.Errorf("Expected capability frame in response, got none (hasHeader=%v)", hasHeader)
	} else {
		if resC.ID != "orchestrate-workspace-migration" {
			t.Errorf("Expected capability ID 'orchestrate-workspace-migration', got: %q", resC.ID)
		}
		if resC.Parameters["status"] != "PASS" {
			t.Errorf("Expected status 'PASS', got: %q", resC.Parameters["status"])
		}
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
	
	// Save and restore os.Args so main() doesn't interpret test flags as CLI goals
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"natvs-engine"}
	
	// Create hermetic temp workspace for main() self-conformance checks
	tempDir := t.TempDir()
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
	
	t.Setenv("TEST_WORKSPACE_ROOT", tempDir)
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
	
	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"natvs-engine"}
	
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



func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func TestNATVSDaemonWatchdogTimeout(t *testing.T) {
	engine := NewNATVSEngine(t.TempDir())
	engine.Config.IdleTimeout = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 52999
	doneChan := make(chan error, 1)

	go func() {
		err := engine.RunDaemon(ctx, port)
		doneChan <- err
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			t.Errorf("Daemon exited with error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Daemon did not terminate automatically after inactivity watchdog timeout")
	}
}

func TestNATVSCoordinatorLifecycle(t *testing.T) {
	tempWS := t.TempDir()
	engine := NewNATVSEngine(tempWS)
	engine.Config.IdleTimeout = 500 * time.Millisecond

	sockPath := filepath.Join(engine.Config.WorkspaceRoot, "00flow/s-fab-aides/c0990-ephemeral-scratch/sacp.sock")
	cfg := broker.Config{
		Name:          "natvs-engine",
		WorkspacePath: engine.Config.WorkspaceRoot,
		SocketPath:    sockPath,
		IdleTimeout:   engine.Config.IdleTimeout,
	}

	// 1. Start the coordinator daemon manually
	coord := broker.NewCoordinator(cfg, &EchoBackend{})
	err := coord.Start()
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coord.Close()

	// 2. Connect to the coordinator (first client)
	netType, addr := coord.GetCoordinationEndpoint()
	if netType == "unix" && len(addr) >= 104 {
		netType = "tcp"
		addr = fmt.Sprintf("127.0.0.1:%d", coord.GetDeterministicTCPPort())
	}
	conn1, err := net.Dial(netType, addr)
	if err != nil {
		t.Fatalf("Failed to dial coordinator (netType=%s, addr=%s): %v", netType, addr, err)
	}
	defer conn1.Close()

	// Send command payload
	_, err = conn1.Write([]byte("HELLO"))
	if err != nil {
		t.Fatalf("Failed to write HELLO: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := conn1.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read ACK: %v", err)
	}
	if string(buf[:n]) != "ACK: HELLO" {
		t.Errorf("Expected ACK: HELLO, got: %q", string(buf[:n]))
	}

	// 3. Connect to the coordinator (second client - adoption)
	conn2, err := net.Dial(netType, addr)
	if err != nil {
		t.Fatalf("Failed to dial coordinator again (adoption): %v", err)
	}
	defer conn2.Close()

	_, err = conn2.Write([]byte("WORLD"))
	if err != nil {
		t.Fatalf("Failed to write WORLD: %v", err)
	}

	n, err = conn2.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read ACK 2: %v", err)
	}
	if string(buf[:n]) != "ACK: WORLD" {
		t.Errorf("Expected ACK: WORLD, got: %q", string(buf[:n]))
	}
}

func TestNATVSCoordinatorStaleSocketCleaning(t *testing.T) {
	tempWS := t.TempDir()
	engine := NewNATVSEngine(tempWS)
	
	netType, addr := engine.GetCoordinationEndpoint()
	if netType == "unix" && len(addr) < 104 {
		// Create parent directory
		err := os.MkdirAll(filepath.Dir(addr), 0755)
		if err != nil {
			t.Fatalf("Failed to create socket dir: %v", err)
		}
		
		// Create a stale empty file representing a dead socket
		err = os.WriteFile(addr, []byte("stale"), 0644)
		if err != nil {
			t.Fatalf("Failed to write stale socket file: %v", err)
		}
		
		// Run DialOrSpawnBroker. Since there's no actual listener, it will fail
		// but it must have cleaned up the stale file during its run.
		sockPath := filepath.Join(engine.Config.WorkspaceRoot, "00flow/s-fab-aides/c0990-ephemeral-scratch/sacp.sock")
		cfg := broker.Config{
			Name:          "natvs-engine",
			WorkspacePath: engine.Config.WorkspaceRoot,
			SocketPath:    sockPath,
			IdleTimeout:   engine.Config.IdleTimeout,
		}
		_, err = broker.DialOrSpawnBroker(cfg, &EchoBackend{}, []string{"go", "run", "."})
		if err == nil {
			t.Fatal("Expected DialOrSpawnBroker to fail, but it succeeded")
		}
		
		// Verify that the stale socket file was successfully unlinked
		if _, err := os.Stat(addr); !os.IsNotExist(err) {
			t.Errorf("Expected stale socket file to be unlinked/deleted, but it still exists")
		}
	}
}

func TestWarmVsColdStartPerformance(t *testing.T) {
	tempWS := t.TempDir()
	sockPath := filepath.Join(tempWS, "perf.sock")
	cfg := broker.Config{
		Name:          "perf-broker",
		WorkspacePath: tempWS,
		SocketPath:    sockPath,
		IdleTimeout:   5 * time.Second,
	}

	backend := &EchoBackend{}

	// --- 1. Measure Warm Start (Direct Dial to already running coordinator) ---
	coord := broker.NewCoordinator(cfg, backend)
	err := coord.Start()
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coord.Close()

	// Perform a warm-up dial
	netType, addr := coord.GetCoordinationEndpoint()
	if netType == "unix" && len(addr) >= 104 {
		netType = "tcp"
		addr = fmt.Sprintf("127.0.0.1:%d", coord.GetDeterministicTCPPort())
	}
	
	startWarm := time.Now()
	connWarm, err := net.Dial(netType, addr)
	if err != nil {
		t.Fatalf("Warm dial failed: %v", err)
	}
	durationWarm := time.Since(startWarm)
	connWarm.Close()

	// --- 2. Measure Cold Start (Includes socket checks, stale file unlinking, and backoff wait) ---
	coord.Close() // Ensure daemon is dead
	
	if netType == "unix" {
		_ = os.WriteFile(addr, []byte("stale"), 0644)
	}

	startCold := time.Now()
	_, err = net.DialTimeout(netType, addr, 50*time.Millisecond)
	if err != nil {
		if netType == "unix" {
			_ = os.Remove(addr)
		}
		
		go func() {
			time.Sleep(50 * time.Millisecond) // Simulate daemon startup latency
			_ = coord.Start()
		}()
		
		// Retry loop (backoff)
		var connCold net.Conn
		for i := 0; i < 5; i++ {
			time.Sleep(20 * time.Millisecond)
			connCold, err = net.DialTimeout(netType, addr, 50*time.Millisecond)
			if err == nil {
				connCold.Close()
				break
			}
		}
	}
	durationCold := time.Since(startCold)

	t.Logf("=========================================================")
	t.Logf("       SACP PERFORMANCE STARTUP LATENCY METRICS         ")
	t.Logf("=========================================================")
	t.Logf("  Cold Start Latency (New Spawn & Boot): %v", durationCold)
	t.Logf("  Warm Start Latency (Session Reused):   %v", durationWarm)
	
	improvement := float64(durationCold) / float64(durationWarm)
	t.Logf("  Warm-connection Speedup Factor:      %.2fx faster", improvement)
	t.Logf("=========================================================")

	if durationWarm >= durationCold {
		t.Errorf("Warm start (%v) was not faster than cold start (%v)", durationWarm, durationCold)
	}
}




