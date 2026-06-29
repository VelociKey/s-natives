package natives_attestations

import (
	"path/filepath"
	"testing"
	"time"

	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func TestQueueSerialization(t *testing.T) {
	tempDir := tempDirInWorkspace(t)
	queuePath := filepath.Join(tempDir, "tasks.queue.webnf")

	t1 := TaskInput{
		ID:          "task-1",
		Timestamp:   time.Now().Truncate(time.Second),
		State:       StateCreated,
		Objective:   "verify-netbench",
		ContextPath: "some/context",
		Workspace:   "00flow/s-logiclibrary",
	}
	t2 := TaskInput{
		ID:          "task-2",
		Timestamp:   time.Now().Truncate(time.Second),
		State:       StateInProgress,
		Objective:   "verify-s-latentlingua",
		ContextPath: "other/context",
		Workspace:   "00flow/s-latentlingua",
	}

	err := WriteQueueFileAtomic(queuePath, []TaskInput{t1, t2})
	if err != nil {
		t.Fatalf("Failed to write queue: %v", err)
	}

	parsed, err := ParseQueueFileAtomic(queuePath)
	if err != nil {
		t.Fatalf("Failed to parse queue: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(parsed))
	}

	if parsed[0].ID != t1.ID || parsed[0].Objective != t1.Objective || parsed[0].State != t1.State || parsed[0].Workspace != t1.Workspace {
		t.Errorf("Task 1 mismatch: got %+v, expected %+v", parsed[0], t1)
	}

	if parsed[1].ID != t2.ID || parsed[1].Objective != t2.Objective || parsed[1].State != t2.State || parsed[1].Workspace != t2.Workspace {
		t.Errorf("Task 2 mismatch: got %+v, expected %+v", parsed[1], t2)
	}
}

func TestCompletedSerialization(t *testing.T) {
	tempDir := tempDirInWorkspace(t)
	completedPath := filepath.Join(tempDir, "tasks.completed.webnf")

	out := TaskOutput{
		ID:         "task-1",
		Timestamp:  time.Now().Truncate(time.Second),
		Status:     StateCompletedSuccess,
		DeltaPaths: []string{"file1.go", "file2.go"},
		Error:      "",
	}

	err := AppendCompletedFileAtomic(completedPath, out)
	if err != nil {
		t.Fatalf("Failed to append completed task: %v", err)
	}

	parsed, err := ParseCompletedFileAtomic(completedPath)
	if err != nil {
		t.Fatalf("Failed to parse completed task: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("Expected 1 completion record, got %d", len(parsed))
	}

	res := parsed[0]
	if res.ID != out.ID || res.Status != out.Status || res.Error != out.Error {
		t.Errorf("Completion mismatch: got %+v, expected %+v", res, out)
	}

	if len(res.DeltaPaths) != 2 || res.DeltaPaths[0] != "file1.go" || res.DeltaPaths[1] != "file2.go" {
		t.Errorf("DeltaPaths mismatch: got %v, expected %v", res.DeltaPaths, out.DeltaPaths)
	}
}

func TestExclusiveLocking(t *testing.T) {
	tempDir := tempDirInWorkspace(t)
	lockPath := filepath.Join(tempDir, "test.lock")

	f1, err := LockFileExclusive(lockPath)
	if err != nil {
		t.Fatalf("Expected to lock successfully first time: %v", err)
	}
	defer func() {
		if err := f1.Close(); err != nil {
			t.Logf("Warning: f1 close error: %v", err)
		}
	}()

	f2, err2 := LockFileExclusive(lockPath)
	if err2 == nil {
		if errClose := f2.Close(); errClose != nil {
			t.Logf("Warning: f2 close error: %v", errClose)
		}
		t.Fatal("Expected second lock attempt to fail, but it succeeded")
	}

	if err := f1.Close(); err != nil {
		t.Fatalf("Failed to close first lock: %v", err)
	}

	f3, err3 := LockFileExclusive(lockPath)
	if err3 != nil {
		t.Fatalf("Expected lock to succeed after releasing first lock: %v", err3)
	}
	if err := f3.Close(); err != nil {
		t.Fatalf("Failed to close f3 lock: %v", err)
	}
}
