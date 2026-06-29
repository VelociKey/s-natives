package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	"strings"

	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func writeErrorToQUICStream(ctx context.Context, errMsg string) {
	if streamVal := ctx.Value("quic_stream"); streamVal != nil {
		if stream, ok := streamVal.(interface{ Write([]byte) (int, error) }); ok {
			if discardValLine16_0, errWrite := stream.Write([]byte("[ERROR] " + errMsg + "\n")); errWrite != nil {
				discard.Discard(discardValLine16_0)
				slog.Warn("Failed to write error message to QUIC stream", "error", errWrite)
			}
		}
	}
}

func runQueueTaskLifecycle(ctx context.Context, engine *NATVSEngine, task *TaskInput) error {
	engine.Config.Mu.Lock()
	engine.Config.CurrentWorkspace = task.Workspace
	engine.Config.Mu.Unlock()

	isCached, cacheKey, cacheErr := lifecycle.IsWorkspaceCached(engine.Config.WorkspaceRoot, task.Workspace)
	if cacheErr == nil && isCached {
		slog.Info("Workspace verification cached successfully. Skipping task lifecycle.", "workspace", task.Workspace)
		return nil
	}

	bypassNegotiateAssimilation := false
	julesQuicMu.RLock()
	hasQUIC := julesQuicConn != nil
	julesQuicMu.RUnlock()

	if hasQUIC && isWorkspaceWarm(task.Workspace) {
		slog.Info("Warm Firecracker MicroVM connection detected. Bypassing Negotiate and Assimilation.", "workspace", task.Workspace)
		bypassNegotiateAssimilation = true
	}

	if !bypassNegotiateAssimilation {
		err := engine.Negotiate(ctx, task.Workspace)
		if err != nil {
			writeErrorToQUICStream(ctx, "Negotiation failed: "+err.Error())
			return fmt.Errorf("negotiation failed: %w", err)
		}

		err = engine.Assimilation(ctx, task.Workspace)
		if err != nil {
			writeErrorToQUICStream(ctx, "Assimilation failed: "+err.Error())
			return fmt.Errorf("assimilation failed: %w", err)
		}
	}

	actionName := task.Objective
	engine.LastVerificationFailureLogs = ""

	for attempt := 1; attempt <= 3; attempt++ {
		slog.Info("Running transformation and verification lifecycle iteration", "attempt", attempt)
		err := engine.Transform(ctx, actionName, task.ContextPath)
		if err != nil {
			writeErrorToQUICStream(ctx, fmt.Sprintf("Attempt %d - Transformation failed: %s", attempt, err.Error()))
			return fmt.Errorf("transformation failed: %w", err)
		}

		importPath := task.Workspace
		if strings.HasPrefix(importPath, "00flow/") {
			importPath = strings.Replace(importPath, "00flow/", "sov.fleet/", 1)
			parts := strings.Split(filepath.ToSlash(importPath), "/")
			if len(parts) == 2 {
				importPath = importPath + "/..."
			}
		}
		importPath = filepath.ToSlash(importPath)

		err = engine.Verification(ctx, importPath)
		if err == nil {
			slog.Info("Verification passed successfully on attempt", "attempt", attempt)
			break
		}
		if strings.Contains(err.Error(), "CRITICAL ENVIRONMENTAL FAULT") {
			writeErrorToQUICStream(ctx, "Critical environmental fault: "+err.Error())
			return err
		}

		writeErrorToQUICStream(ctx, fmt.Sprintf("Attempt %d - Verification failed: %s", attempt, err.Error()))
		slog.Warn("Verification failed on attempt", "attempt", attempt, "error", err)
		if attempt == 3 {
			return fmt.Errorf("verification failed after 3 attempts: %w", err)
		}
	}

	err := engine.Synthesis(ctx, filepath.Base(task.Workspace)+"-synthesized")
	if err != nil {
		writeErrorToQUICStream(ctx, "Synthesis failed: "+err.Error())
		return fmt.Errorf("synthesis failed: %w", err)
	}

	if cacheErr == nil {
		if errUpdate := lifecycle.UpdateWorkspaceCache(engine.Config.WorkspaceRoot, task.Workspace, cacheKey); errUpdate != nil {
			slog.Warn("Failed to update workspace cache", "workspace", task.Workspace, "error", errUpdate)
		}
	}

	if hasQUIC {
		markWorkspaceWarm(task.Workspace)
	}

	return nil
}
