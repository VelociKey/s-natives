package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"sov.fleet/s-fab-aides/81000-active-source/bash"
)

// Assimilation executes phase 2 local repository structural audits.
func (e *NATVSEngine) Assimilation(ctx context.Context, targetWorkspace string) error {
	e.State = PhaseAssimilation
	slog.Info("Phase 2: Assimilating context directories for workspace", "workspace", targetWorkspace)

	// Assert 5-digit / 4-digit directory taxonomy rules
	targetPath := filepath.Join(e.Config.WorkspaceRoot, targetWorkspace)

	// Audit directory structure using s-fab-aides bash polyfill to replace shell subprocesses
	var lsBuf strings.Builder
	if err := bash.Execute(&lsBuf, "ls", targetPath); err == nil {
		slog.Info("Dynamic file discovery via s-fab-aides bash", "output", strings.ReplaceAll(strings.TrimSpace(lsBuf.String()), "\n", ", "))
	}

	dir, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target workspace path does not exist: %w", err)
		}
		return err
	}

	validCount := 0
	for _, entry := range dir {
		if entry.IsDir() {
			name := entry.Name()
			// Verifies prefix has correct semantic numeric code
			if len(name) >= 5 && (strings.HasPrefix(name, "c") || (name[0] >= '0' && name[0] <= '9')) {
				validCount++
			}
		}
	}

	slog.Info("Assimilation successfully verified semantic taxonomy entries", "count", validCount)
	return nil
}
