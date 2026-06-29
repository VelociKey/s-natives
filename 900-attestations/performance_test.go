package natives_attestations

import (
	"io/ioutil"
	"os"
	"path/filepath"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
	"testing"
)

func TestGitCacheOperations(t *testing.T) {
	workspaceRoot := "C:\\aCogSpaceSeed"
	if envRoot := os.Getenv("TEST_WORKSPACE_ROOT"); envRoot != "" {
		workspaceRoot = envRoot
	}

	targetWS := "00flow/s-natives"

	// Ensure caching works
	isCached, key, err := IsWorkspaceCached(workspaceRoot, targetWS)
	if err != nil {
		t.Fatalf("Failed to check workspace cache status: %v", err)
	}

	t.Logf("Computed Git Cache Key: %s, IsCached: %v", key, isCached)

	// Update the cache
	err = UpdateWorkspaceCache(workspaceRoot, targetWS, key)
	if err != nil {
		t.Fatalf("Failed to update workspace cache: %v", err)
	}

	// Verify it's cached now
	isCachedNow, keyNow, err := IsWorkspaceCached(workspaceRoot, targetWS)
	if err != nil {
		t.Fatalf("Failed to verify workspace cache: %v", err)
	}
	if !isCachedNow {
		t.Fatal("Expected workspace to be cached after cache update")
	}
	if keyNow != key {
		t.Fatalf("Expected cache key to be %s, got %s", key, keyNow)
	}

	// Clean up / Invalidate cache
	err = InvalidateWorkspaceCache(workspaceRoot, targetWS)
	if err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	isCachedPost, discardValLine51_1, err := IsWorkspaceCached(workspaceRoot, targetWS)
	discard.Discard(discardValLine51_1)
	if err != nil {
		t.Fatalf("Failed to verify cache status post invalidation: %v", err)
	}
	if isCachedPost {
		t.Fatal("Expected workspace cache to be invalid/empty after explicit invalidation")
	}
}

func TestWorkspaceDirtyStateHandling(t *testing.T) {
	workspaceRoot := "C:\\aCogSpaceSeed"
	if envRoot := os.Getenv("TEST_WORKSPACE_ROOT"); envRoot != "" {
		workspaceRoot = envRoot
	}

	// Create a temporary dummy file in the s-natives workspace to simulate a dirty status
	dummyFile := filepath.Join(workspaceRoot, "00flow/s-natives/dummy_dirty_state_test_xyz.go")
	if err := os.Remove(dummyFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to clean up dummyFile pre-test: %v", err)
	}

	defer func() {
		if err := os.Remove(dummyFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up dummyFile post-test: %v", err)
		}
	}()

	keyClean, err := ComputeGitStateKey(workspaceRoot, "00flow/s-natives")
	if err != nil {
		t.Fatalf("Failed to compute clean key: %v", err)
	}

	// Write dirty file
	err = ioutil.WriteFile(dummyFile, []byte("package main\n\n// dirty content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy dirty file: %v", err)
	}

	keyDirty, err := ComputeGitStateKey(workspaceRoot, "00flow/s-natives")
	if err != nil {
		t.Fatalf("Failed to compute dirty key: %v", err)
	}

	if keyClean == keyDirty {
		t.Fatal("Expected dirty workspace state key to differ from clean state key")
	}

	t.Logf("Clean key: %s", keyClean)
	t.Logf("Dirty key: %s", keyDirty)
}
