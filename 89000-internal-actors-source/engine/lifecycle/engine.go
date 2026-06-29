package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sov.fleet/s-sacp/81000-active-source/pkg/broker"
)

// NATVSPhase represents a single state in the sovereign orchestration lifecycle.
type NATVSPhase string

const (
	PhaseNegotiate    NATVSPhase = "NEGOTIATE"
	PhaseAssimilation NATVSPhase = "ASSIMILATE"
	PhaseTransform    NATVSPhase = "TRANSFORM"
	PhaseVerification NATVSPhase = "VERIFY"
	PhaseSynthesis    NATVSPhase = "SYNTHESIS"
)

// OrchestrationConfig houses system paths and validation rails.
type OrchestrationConfig struct {
	WorkspaceRoot     string
	RegistryPath      string
	OutputChannel     chan string
	AllowedWorkspaces []string
	IdleTimeout       time.Duration
	CurrentWorkspace  string
	Mu                sync.Mutex
}

// NATVSEngine represents the core fleet-aware execution manager.
type NATVSEngine struct {
	Config                      *OrchestrationConfig
	State                       NATVSPhase
	LastChanges                 []string
	LastReason                  string
	LastVerificationFailureLogs string
}

// NewNATVSEngine instantiates a new sovereign engine container.
func NewNATVSEngine(workspace string) *NATVSEngine {
	allowed := make([]string, 0, 10)
	if envWS := os.Getenv("ALLOWED_WORKSPACES"); envWS != "" {
		parts := strings.Split(envWS, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				allowed = append(allowed, part)
			}
		}
	}
	return &NATVSEngine{
		Config: &OrchestrationConfig{
			WorkspaceRoot:     workspace,
			RegistryPath:      filepath.Join(workspace, "00flow/s-forge/90100-rehydration-seed"),
			OutputChannel:     make(chan string, 100),
			AllowedWorkspaces: allowed,
			IdleTimeout:       15 * time.Minute,
		},
		State: PhaseNegotiate,
	}
}

// GetCoordinationEndpoint returns the network type and address (UDS path) for coordination.
func (e *NATVSEngine) GetCoordinationEndpoint() (network, address string) {
	sockPath := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-fab-aides/c0990-ephemeral-scratch/sacp.sock")
	cfg := broker.Config{
		WorkspacePath: e.Config.WorkspaceRoot,
		SocketPath:    sockPath,
	}
	coord := broker.NewCoordinator(cfg, nil)
	return coord.GetCoordinationEndpoint()
}

// GetDeterministicTCPPort returns a deterministic private TCP port based on workspace root hashing.
func (e *NATVSEngine) GetDeterministicTCPPort() int {
	cfg := broker.Config{
		WorkspacePath: e.Config.WorkspaceRoot,
	}
	coord := broker.NewCoordinator(cfg, nil)
	return coord.GetDeterministicTCPPort()
}
