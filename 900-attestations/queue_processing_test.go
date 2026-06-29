package natives_attestations

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func TestParallelAtomicAccess(t *testing.T) {
	tempDir := tempDirInWorkspace(t)
	queuePath := filepath.Join(tempDir, "tasks.queue.webnf")

	// Pre-create empty queue
	err := WriteQueueFileAtomic(queuePath, []TaskInput{})
	if err != nil {
		t.Fatalf("Failed to initialize queue: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				// Atomic read-modify-write via transaction
				tx, err := BeginQueueTransaction(queuePath)
				if err != nil {
					t.Errorf("Worker %d failed to begin transaction: %v", id, err)
					return
				}
				tasks, err := tx.Read()
				if err != nil {
					tx.Commit()
					t.Errorf("Worker %d failed parsing: %v", id, err)
					return
				}
				newTask := TaskInput{
					ID:        GenerateTaskID(),
					Timestamp: time.Now(),
					State:     StateCreated,
					Objective: "verify-netbench",
				}
				tasks = append(tasks, newTask)
				err = tx.Write(tasks)
				if err != nil {
					tx.Commit()
					t.Errorf("Worker %d failed writing: %v", id, err)
					return
				}
				tx.Commit()
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	finalTasks, err := ParseQueueFileAtomic(queuePath)
	if err != nil {
		t.Fatalf("Failed to parse final queue: %v", err)
	}

	expectedLen := workers * 5
	if len(finalTasks) != expectedLen {
		t.Errorf("Expected %d tasks in queue, got %d", expectedLen, len(finalTasks))
	}
}

func TestQueueWatcherPersistentSession(t *testing.T) {
	tempWS := tempDirInWorkspace(t)

	// 1. Create directory structures
	julesDir := filepath.Join(tempWS, "00flow/s-forge/94000-external-actors/jules")
	coordDir := filepath.Join(tempWS, "c0990-ephemeral-scratch/natvs coordination")
	nativesDir := filepath.Join(tempWS, "00flow/s-natives")

	for _, dir := range []string{julesDir, coordDir, nativesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
	}

	// Create s-seed directory and copy conformance.exe into it
	seedDir := filepath.Join(tempWS, "00flow/s-seed")
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		t.Fatalf("Failed to create s-seed dir: %v", err)
	}
	srcConf := filepath.Join(getWorkspaceRoot(), "00flow", "s-seed", "conformance.exe")
	destConf := filepath.Join(seedDir, "conformance.exe")
	if err := copyFile(srcConf, destConf); err != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", err)
	}

	// Create dummy go.mod in nativesDir to keep it valid
	err := os.WriteFile(filepath.Join(nativesDir, "go.mod"), []byte("module check\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy go.mod: %v", err)
	}

	// Create dummy natvs_engine.go in nativesDir to keep targetMarker search valid
	dummyEngineGo := filepath.Join(nativesDir, "natvs_engine.go")
	err = os.WriteFile(dummyEngineGo, []byte(`package main
import "log/slog"
func main() {
	slog.Info("=========================================================")
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy natvs_engine.go: %v", err)
	}

	// 2. Write a mock jules source file and compile it
	mockGoSrc := filepath.Join(tempWS, "mock_jules.go")
	mockSrcCode := `package main
import "os"
func main() {
	_, _ = os.Stdout.Write([]byte("Mock Jules execution\n"))
}
`
	if err := os.WriteFile(mockGoSrc, []byte(mockSrcCode), 0644); err != nil {
		t.Fatalf("Failed to write mock jules source: %v", err)
	}

	goBin := filepath.Join(getWorkspaceRoot(), "00flow", "s-forge", "92000-external-toolchains", "go", "bin", "go"+getExeSuffix())
	mockJulesExe := filepath.Join(julesDir, "jules.exe")

	cmdBuild := exec.Command(goBin, "build", "-o", mockJulesExe, mockGoSrc)
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to compile mock jules: %v", err)
	}

	// 3. Write empty queue file
	queuePath := filepath.Join(coordDir, "tasks.queue.webnf")
	outPath := filepath.Join(coordDir, "tasks.completed.webnf")
	if err := WriteQueueFileAtomic(queuePath, []TaskInput{}); err != nil {
		t.Fatalf("Failed to write empty queue file: %v", err)
	}

	// 4. Start the natvs-engine daemon process
	engineExe := filepath.Join(getWorkspaceRoot(), "00flow/s-natives/natvs-engine.exe")
	cmdDaemon := exec.Command(engineExe, "--queue="+queuePath, "--out="+outPath)
	cmdDaemon.Env = append(os.Environ(),
		"TEST_WORKSPACE_ROOT="+tempWS,
		"ANTIGRAVITY_AGENT=0",
		"ANTIGRAVITY_PROJECT_ID=",
		"SKIP_RECURSIVE_TESTS=true",
	)

	var daemonBuf bytes.Buffer
	cmdDaemon.Stdout = &daemonBuf
	cmdDaemon.Stderr = &daemonBuf

	if err := cmdDaemon.Start(); err != nil {
		t.Fatalf("Failed to start natvs-engine daemon: %v", err)
	}
	defer func() {
		if cmdDaemon.Process != nil {
			if err := cmdDaemon.Process.Kill(); err != nil {
				t.Logf("Failed to kill daemon process: %v", err)
			}
			t.Logf("Daemon log output:\n%s", daemonBuf.String())
		}
	}()

	// Wait for daemon to initialize and start watching
	time.Sleep(500 * time.Millisecond)

	// 5. Append first task
	task1 := TaskInput{
		ID:        "task-persistent-1",
		Timestamp: time.Now(),
		State:     StateCreated,
		Objective: "verify-s-natives",
		Workspace: "00flow/s-natives",
	}

	tx, err := BeginQueueTransaction(queuePath)
	if err != nil {
		t.Fatalf("Failed to begin transaction for task 1: %v", err)
	}
	tasks, err := tx.Read()
	if err != nil {
		tx.Commit()
		t.Fatalf("Failed to read queue: %v", err)
	}
	tasks = append(tasks, task1)
	if err := tx.Write(tasks); err != nil {
		tx.Commit()
		t.Fatalf("Failed to write task 1: %v", err)
	}
	tx.Commit()

	// 6. Poll for completion of task 1
	var completed []TaskOutput
	var completedTask1 *TaskOutput
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		completed, err = ParseCompletedFileAtomic(outPath)
		if err == nil {
			for _, c := range completed {
				if c.ID == task1.ID {
					completedTask1 = &c
					break
				}
			}
		}
		if completedTask1 != nil {
			break
		}
	}

	if completedTask1 == nil {
		t.Fatalf("Daemon failed to process task 1 or time out")
	}

	if completedTask1.Status != StateCompletedSuccess {
		t.Fatalf("Expected task 1 to succeed, got status: %v, error: %s", completedTask1.Status, completedTask1.Error)
	}

	// 7. Verify that the daemon process is still running (kept Jules open/active)
	if cmdDaemon.ProcessState != nil && cmdDaemon.ProcessState.Exited() {
		t.Fatal("Daemon process exited after task 1. It should keep running persistently.")
	}

	// 8. Append second task to verify quick reuse
	task2 := TaskInput{
		ID:        "task-persistent-2",
		Timestamp: time.Now(),
		State:     StateCreated,
		Objective: "verify-s-natives",
		Workspace: "00flow/s-natives",
	}

	tx2, err := BeginQueueTransaction(queuePath)
	if err != nil {
		t.Fatalf("Failed to begin transaction for task 2: %v", err)
	}
	tasks, err = tx2.Read()
	if err != nil {
		tx2.Commit()
		t.Fatalf("Failed to read queue for task 2: %v", err)
	}
	tasks = append(tasks, task2)
	if err := tx2.Write(tasks); err != nil {
		tx2.Commit()
		t.Fatalf("Failed to write task 2: %v", err)
	}
	tx2.Commit()

	// 9. Poll for completion of task 2
	var completedTask2 *TaskOutput
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		completed, err = ParseCompletedFileAtomic(outPath)
		if err == nil {
			for _, c := range completed {
				if c.ID == task2.ID {
					completedTask2 = &c
					break
				}
			}
		}
		if completedTask2 != nil {
			break
		}
	}

	if completedTask2 == nil {
		t.Fatalf("Daemon failed to process task 2 or time out")
	}

	if completedTask2.Status != StateCompletedSuccess {
		t.Fatalf("Expected task 2 to succeed, got status: %v, error: %s", completedTask2.Status, completedTask2.Error)
	}
}
