package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	Mu                sync.Mutex
}

// NATVSEngine represents the core fleet-aware execution manager.
type NATVSEngine struct {
	Config *OrchestrationConfig
	State  NATVSPhase
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
		},
		State: PhaseNegotiate,
	}
}
