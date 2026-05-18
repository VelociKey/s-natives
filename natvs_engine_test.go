package main

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNATVSLifecyclePhases(t *testing.T) {
	engine := NewNATVSEngine("C:\\aCogSpaceSeed")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Negotiation Phase
	err := engine.Negotiate(ctx, "00flow/s-aether")
	if err != nil {
		t.Fatalf("Negotiation phase failed: %v", err)
	}
	if engine.State != PhaseNegotiate {
		t.Errorf("Expected state %s, got %s", PhaseNegotiate, engine.State)
	}

	// 2. Assimilation Phase
	err = engine.Assimilation(ctx, "00flow/s-aether")
	if err != nil {
		t.Fatalf("Assimilation phase failed: %v", err)
	}
	if engine.State != PhaseAssimilation {
		t.Errorf("Expected state %s, got %s", PhaseAssimilation, engine.State)
	}

	// 3. Transformation Phase
	err = engine.Transform(ctx, "test-refactoring-task")
	if err != nil {
		t.Fatalf("Transformation phase failed: %v", err)
	}
	if engine.State != PhaseTransform {
		t.Errorf("Expected state %s, got %s", PhaseTransform, engine.State)
	}

	// 4. Verification Phase
	err = engine.Verification(ctx, "sov.fleet/s-aether")
	if err != nil {
		t.Fatalf("Verification phase failed: %v", err)
	}
	if engine.State != PhaseVerification {
		t.Errorf("Expected state %s, got %s", PhaseVerification, engine.State)
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	port := 50989
	
	// Start UDP server daemon
	go func() {
		err := engine.RunDaemon(ctx, port)
		if err != nil {
			t.Errorf("Daemon failed: %v", err)
		}
	}()

	// Allow UDP socket start
	time.Sleep(100 * time.Millisecond)

	// Dial UDP client port
	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:50989")
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
}
