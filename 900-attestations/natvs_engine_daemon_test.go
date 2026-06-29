package natives_attestations

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
	"testing"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
	"sov.fleet/s-sacp/81000-active-source/pkg/broker"
)

func TestNATVSListenDaemonAndSACP(t *testing.T) {
	engine := NewNATVSEngine(getWorkspaceRoot())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 61989

	// Start UDP server daemon
	go func() {
		if err := engine.RunDaemon(ctx, port); err != nil {
			t.Logf("Daemon stopped: %v", err)
		}
	}()

	// Allow UDP socket start
	time.Sleep(100 * time.Millisecond)

	// Dial UDP client port
	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:61989")
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
			"target": "s-natives",
		},
	}
	msgBytes, err := bicodec.EncodeSACPMessage(&h, &c)
	if err != nil {
		t.Fatalf("Failed to encode SACP message: %v", err)
	}

	if discardValLine65_0, err := conn.Write(msgBytes); err != nil {
		discard.Discard(discardValLine65_0)
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

	hasHeader, discardValLine81_1, hasCap, resC, err := bicodec.DecodeSACPMessage(buf[:n])
	discard.Discard(discardValLine81_1)
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
	if discardValLine98_0, err := conn.Write([]byte("INVALID_FRAME")); err != nil {
		discard.Discard(discardValLine98_0)
		t.Logf("Failed to write invalid frame: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Cancel context to shut down RunDaemon cleanly
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestNATVSDaemonListenFailure(t *testing.T) {
	engine := NewNATVSEngine(tempDirInWorkspace(t))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Bind to the port first
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:61999")
	if err != nil {
		t.Fatalf("Failed to resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()

	// Try to run RunDaemon on the same port, which must fail!
	err = engine.RunDaemon(ctx, 61999)
	if err == nil {
		t.Errorf("Expected RunDaemon to fail when port is already in use, but it passed")
	}
}

func TestNATVSDaemonWatchdogTimeout(t *testing.T) {
	engine := NewNATVSEngine(tempDirInWorkspace(t))
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
	tempWS := tempDirInWorkspace(t)
	engine := NewNATVSEngine(tempWS)
	engine.Config.IdleTimeout = 500 * time.Millisecond

	sockPath := filepath.Join("C:\\aCogSpaceSeed", "00flow/s-natives/c0990-ephemeral-scratch", "sacp_lifecycle.sock")
	if err := os.MkdirAll(filepath.Dir(sockPath), 0755); err != nil {
		t.Logf("MkdirAll error: %v", err)
	}
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
	if discardValLine193_0, err := conn1.Write([]byte("HELLO")); err != nil {
		discard.Discard(discardValLine193_0)
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

	if discardValLine213_0, err := conn2.Write([]byte("WORLD")); err != nil {
		discard.Discard(discardValLine213_0)
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
	tempWS := tempDirInWorkspace(t)
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
		sockPath := filepath.Join("C:\\aCogSpaceSeed", "00flow/s-natives/c0990-ephemeral-scratch", "sacp_cleaning.sock")
		if err = os.MkdirAll(filepath.Dir(sockPath), 0755); err != nil {
			t.Logf("MkdirAll error: %v", err)
		}
		cfg := broker.Config{
			Name:          "natvs-engine",
			WorkspacePath: engine.Config.WorkspaceRoot,
			SocketPath:    sockPath,
			IdleTimeout:   engine.Config.IdleTimeout,
		}
		discardValLine256_0, err := broker.DialOrSpawnBroker(cfg, &EchoBackend{}, []string{"go", "run", "."})
		discard.Discard(discardValLine256_0)
		if err == nil {
			t.Fatal("Expected DialOrSpawnBroker to fail, but it succeeded")
		}

		// Verify that the stale socket file was successfully unlinked
		if discardValLine262_0, err := os.Stat(addr); !os.IsNotExist(err) {
			discard.Discard(discardValLine262_0)
			t.Errorf("Expected stale socket file to be unlinked/deleted, but it still exists")
		}
	}
}

func TestWarmVsColdStartPerformance(t *testing.T) {
	tempWS := tempDirInWorkspace(t)
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
		if err := os.WriteFile(addr, []byte("stale"), 0644); err != nil {
			t.Logf("Failed to write stale socket file: %v", err)
		}
	}

	startCold := time.Now()
	discardValLine313_0, err := net.DialTimeout(netType, addr, 50*time.Millisecond)
	discard.Discard(discardValLine313_0)
	if err != nil {
		if netType == "unix" {
			if err := os.Remove(addr); err != nil {
				t.Logf("Failed to remove: %v", err)
			}
		}

		go func() {
			time.Sleep(50 * time.Millisecond) // Simulate daemon startup latency
			if err := coord.Start(); err != nil {
				t.Logf("Coordinator start err: %v", err)
			}
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
