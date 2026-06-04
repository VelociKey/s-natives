package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
	"sov.fleet/s-sacp/81000-active-source/pkg/broker"
)

func runDaemonMode(ctx context.Context, engine *NATVSEngine) {
	slog.Info("Running as background coordination daemon...")
	slog.Info("Initializing work and establishing SACP telemetry stream.")
	slog.Info("Updating transport layer to QUIC...", "net", "udp", "addr", "127.0.0.1:0", "protocol", "QUIC")
	sockPath := filepath.Join(engine.Config.WorkspaceRoot, "00flow/s-fab-aides/c0990-ephemeral-scratch/sacp.sock")
	cfg := broker.Config{
		Name:          "natvs-engine",
		WorkspacePath: engine.Config.WorkspaceRoot,
		SocketPath:    sockPath,
		IdleTimeout:   engine.Config.IdleTimeout,
	}
	coordinator := broker.NewCoordinator(cfg, &EchoBackend{})
	if err := coordinator.Start(); err != nil {
		logFatal("Failed to start coordinator daemon: %v", err)
	}
	<-coordinator.ShutdownContext().Done()
}

func runSelfConformance(ctx context.Context, engine *lifecycle.NATVSEngine) {
	engine.Config.Mu.Lock()
	engine.Config.CurrentWorkspace = "00flow/s-natives"
	engine.Config.Mu.Unlock()

	err := engine.Negotiate(ctx, "00flow/s-natives")
	if err != nil {
		logFatal("Negotiation check failed: %v", err)
	}

	err = engine.Assimilation(ctx, "00flow/s-natives")
	if err != nil {
		logFatal("Assimilation check failed: %v", err)
	}

	err = engine.Transform(ctx, "verify-self", "")
	if err != nil {
		logFatal("Transformation check failed: %v", err)
	}

	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err != nil {
		logFatal("Verification check failed: %v", err)
	}

	err = engine.Synthesis(ctx, "natives_wasm_gc.wasm")
	if err != nil {
		logFatal("Synthesis check failed: %v", err)
	}

	slog.Info("Self-conformance successfully verified!")
}

