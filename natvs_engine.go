package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sov.fleet/s-natives/engine/lifecycle"
	"sov.fleet/s-sacp/81000-active-source/pkg/broker"
)

type EchoBackend struct{}

func (b *EchoBackend) Initialize(ctx context.Context) error { return nil }
func (b *EchoBackend) Validate(ctx context.Context) error   { return nil }
func (b *EchoBackend) Dial(ctx context.Context) (net.Conn, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		defer l.Close()
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 1024)
		for {
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			_, _ = c.Write([]byte("ACK: " + string(buf[:n])))
		}
	}()
	return net.Dial("tcp", l.Addr().String())
}
func (b *EchoBackend) Shutdown(ctx context.Context) error { return nil }

// Facade Types and Constants for Backward Compatibility with Tests and External Packages

type NATVSPhase = lifecycle.NATVSPhase

const (
	PhaseNegotiate    = lifecycle.PhaseNegotiate
	PhaseAssimilation = lifecycle.PhaseAssimilation
	PhaseTransform    = lifecycle.PhaseTransform
	PhaseVerification = lifecycle.PhaseVerification
	PhaseSynthesis    = lifecycle.PhaseSynthesis
)

type NATVSEngine = lifecycle.NATVSEngine
type OrchestrationConfig = lifecycle.OrchestrationConfig

func NewNATVSEngine(workspace string) *NATVSEngine {
	return lifecycle.NewNATVSEngine(workspace)
}

var logFatal = func(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	slog.Error("Fatal error encountered", "details", msg)
	os.Exit(1)
}

func main() {
	slog.Info("=========================================================")
	slog.Info("         SACP WARM-START DAEMON ENGAGED          ")
	slog.Info("         SOVEREIGN NATIVES ORCHESTRATOR          ")
	slog.Info("=========================================================")
	
	allowedWorkspaces := make([]string, 0, len(os.Args))
	var idleTimeoutOverride time.Duration
	argsFiltered := make([]string, 0, len(os.Args))
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--workspaces=") {
			wsList := strings.TrimPrefix(arg, "--workspaces=")
			parts := strings.Split(wsList, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					allowedWorkspaces = append(allowedWorkspaces, part)
				}
			}
		} else if strings.HasPrefix(arg, "--idle-timeout=") {
			timeoutStr := strings.TrimPrefix(arg, "--idle-timeout=")
			if parsed, err := time.ParseDuration(timeoutStr); err == nil {
				idleTimeoutOverride = parsed
			} else {
				slog.Error("Invalid idle-timeout duration", "value", timeoutStr, "error", err)
			}
		} else {
			argsFiltered = append(argsFiltered, arg)
		}
	}
	os.Args = argsFiltered

	workspaceRoot := "C:\\aCogSpaceSeed"
	if envRoot := os.Getenv("TEST_WORKSPACE_ROOT"); envRoot != "" {
		workspaceRoot = envRoot
	}
	engine := NewNATVSEngine(workspaceRoot)
	if len(allowedWorkspaces) > 0 {
		engine.Config.AllowedWorkspaces = allowedWorkspaces
	}
	if idleTimeoutOverride > 0 {
		engine.Config.IdleTimeout = idleTimeoutOverride
	}
	ctx := context.Background()
	
	if len(os.Args) > 1 {
		objective := os.Args[1]
		contextPath := ""
		if len(os.Args) > 2 {
			contextPath = os.Args[2]
		}
		
		slog.Info("Received Goal", "objective", objective)
		slog.Info("Context Path", "path", contextPath)

		if objective == "daemon" {
			slog.Info("Running as background coordination daemon...")
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
			return
		}

		if strings.Contains(strings.ToLower(objective), "remediate") || strings.Contains(strings.ToLower(objective), "remediation") {
			targetWS := contextPath
			if targetWS == "" {
				targetWS = "00flow/s-logiclibrary/00200-logic-libraries/netbench"
			}
			
			slog.Info("Starting general remediation lifecycle for workspace", "workspace", targetWS)

			err := engine.Negotiate(ctx, targetWS)
			if err != nil {
				logFatal("Negotiation failed: %v", err)
			}

			err = engine.Assimilation(ctx, targetWS)
			if err != nil {
				logFatal("Assimilation failed: %v", err)
			}

			actionName := "remediate-" + filepath.Base(filepath.FromSlash(targetWS))
			err = engine.Transform(ctx, actionName)
			if err != nil {
				logFatal("Transformation failed: %v", err)
			}

			importPath := targetWS
			if strings.HasPrefix(targetWS, "00flow/s-logiclibrary") {
				importPath = strings.Replace(targetWS, "00flow/s-logiclibrary", "sov.fleet/s-logiclibrary", 1)
			} else if strings.HasPrefix(targetWS, "00flow/s-natives") {
				importPath = strings.Replace(targetWS, "00flow/s-natives", "sov.fleet/s-natives", 1)
			} else if strings.HasPrefix(targetWS, "00flow/s-latentlingua") {
				importPath = strings.Replace(targetWS, "00flow/s-latentlingua", "sov.fleet/s-latentlingua", 1)
				if importPath == "sov.fleet/s-latentlingua" {
					importPath = "sov.fleet/s-latentlingua/..."
				}
			}
			importPath = filepath.ToSlash(importPath)

			err = engine.Verification(ctx, importPath)
			if err != nil {
				logFatal("Verification failed: %v", err)
			}

			err = engine.Synthesis(ctx, filepath.Base(targetWS)+"-remediated")
			if err != nil {
				logFatal("Synthesis failed: %v", err)
			}

			slog.Info("Objective successfully completed!", "objective", objective)
			return
		}
		
		logFatal("Unsupported objective: %s", objective)
	}

	// Verify self-conformance fallback
	err := engine.Negotiate(ctx, "00flow/s-natives")
	if err != nil {
		logFatal("Negotiation check failed: %v", err)
	}
	
	err = engine.Assimilation(ctx, "00flow/s-natives")
	if err != nil {
		logFatal("Assimilation check failed: %v", err)
	}
	
	err = engine.Transform(ctx, "refactor-taxonomy-casing")
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
