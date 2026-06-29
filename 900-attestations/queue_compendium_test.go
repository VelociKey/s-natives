package natives_attestations

import (
	"os"
	"os/exec"
	"path/filepath"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	"strings"
	"testing"
	"time"

	. "sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func TestCreateCompendiumsThroughQueue(t *testing.T) {
	// Skip recursive tests if requested, but we want this to run locally
	if os.Getenv("SKIP_RECURSIVE_TESTS") == "true" {
		t.Skip("Skipping create-compendium test under dry-run")
	}

	workspaceRoot := getWorkspaceRoot()
	coordDir := filepath.Clean(filepath.Join(workspaceRoot, "00flow/s-natives/c0990-ephemeral-scratch", "natvs coordination"))
	if err := os.MkdirAll(coordDir, 0755); err != nil {
		t.Fatalf("Failed to create coordDir: %v", err)
	}

	queuePath := filepath.Join(coordDir, "tasks.queue.webnf")
	outPath := filepath.Join(coordDir, "tasks.completed.webnf")

	// 1. Discover all workspaces under 00flow
	siloDir := filepath.Join(workspaceRoot, "00flow")
	entries, err := os.ReadDir(siloDir)
	if err != nil {
		t.Fatalf("Failed to read 00flow directory: %v", err)
	}

	var targetWS []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.Contains(name, "c0411-compendium") {
			continue
		}
		targetWS = append(targetWS, "00flow/"+name)
	}

	// 2. Clear old completion records for our test tasks
	if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to remove old completion file: %v", err)
	}

	// 3. Write all tasks to the queue at once
	tx, err := BeginQueueTransaction(queuePath)
	if err != nil {
		t.Fatalf("Failed to begin queue transaction: %v", err)
	}
	tasks, err := tx.Read()
	if err != nil {
		tx.Commit()
		t.Fatalf("Failed to read queue: %v", err)
	}

	// Filter out any stale created tasks of this type to avoid duplicates
	var cleanTasks []TaskInput
	for _, tk := range tasks {
		if !strings.HasPrefix(tk.ID, "task-compendium-") {
			cleanTasks = append(cleanTasks, tk)
		}
	}
	tasks = cleanTasks

	var testTasks []TaskInput
	for _, ws := range targetWS {
		taskName := filepath.Base(ws)
		task := TaskInput{
			ID:        "task-compendium-" + taskName,
			Timestamp: time.Now().Truncate(time.Second),
			State:     StateCreated,
			Objective: "create-compendium-" + ws,
			Workspace: ws,
		}
		tasks = append(tasks, task)
		testTasks = append(testTasks, task)
	}

	if err := tx.Write(tasks); err != nil {
		tx.Commit()
		t.Fatalf("Failed to write tasks to queue: %v", err)
	}
	tx.Commit()

	t.Logf("Enqueued %d workspaces for compendium generation.", len(testTasks))

	// 4. Start the natvs-engine daemon process
	engineExe := filepath.Join(workspaceRoot, "00flow/s-natives/natvs-engine.exe")
	cmdDaemon := exec.Command(engineExe, "--queue="+queuePath, "--out="+outPath)
	cmdDaemon.Env = append(os.Environ(),
		"TEST_WORKSPACE_ROOT="+workspaceRoot,
		"ANTIGRAVITY_AGENT=0",
		"ANTIGRAVITY_PROJECT_ID=",
	)

	if err := cmdDaemon.Start(); err != nil {
		t.Fatalf("Failed to start natvs-engine daemon: %v", err)
	}
	defer func() {
		if cmdDaemon.Process != nil {
			if err := cmdDaemon.Process.Kill(); err != nil {
				t.Logf("Failed to kill daemon process: %v", err)
			}
		}
	}()

	// 5. Wait and monitor completions
	t.Logf("Waiting for compendium tasks to complete...")
	completedMap := make(map[string]TaskOutput)
	startTimes := make(map[string]time.Time)
	for _, tk := range testTasks {
		startTimes[tk.ID] = time.Now()
	}

	deadline := time.Now().Add(10 * time.Minute)
	for len(completedMap) < len(testTasks) && time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		completedList, err := ParseCompletedFileAtomic(outPath)
		if err != nil {
			continue
		}
		for _, c := range completedList {
			if strings.HasPrefix(c.ID, "task-compendium-") {
				if discardValLine132_0, ok := completedMap[c.ID]; !ok {
					discard.Discard(discardValLine132_0)
					completedMap[c.ID] = c
					t.Logf("[+] Completed: %s [status: %s, duration: %v]", c.ID, c.Status, time.Since(startTimes[c.ID]))
				}
			}
		}
	}

	// 6. Print Execution Metrics via t.Logf
	t.Logf("==================================================================================================")
	t.Logf("                       COMPENDIUM ORCHESTRATION EXECUTION METRICS                                 ")
	t.Logf("==================================================================================================")
	t.Logf("%-20s | %-12s | %-12s | %-12s | %-12s", "Workspace", "Start Time", "Completed", "Net Exec", "Total Turnaround")
	t.Logf("--------------------------------------------------------------------------------------------------")

	// Pre-sort or map completedList timestamps to measure net execution times
	var lastCompletion = testTasks[0].Timestamp
	for i, tk := range testTasks {
		wsName := filepath.Base(tk.Workspace)
		cRecord, finished := completedMap[tk.ID]
		if finished {
			totalDuration := cRecord.Timestamp.Sub(tk.Timestamp)
			netDuration := cRecord.Timestamp.Sub(lastCompletion)
			if i == 0 {
				netDuration = cRecord.Timestamp.Sub(tk.Timestamp)
			}
			t.Logf("%-20s | %-12s | %-12s | %-12v | %-12v",
				wsName,
				tk.Timestamp.Format("15:04:05"),
				cRecord.Timestamp.Format("15:04:05"),
				netDuration.Round(time.Millisecond),
				totalDuration.Round(time.Millisecond),
			)
			lastCompletion = cRecord.Timestamp
		} else {
			t.Logf("%-20s | %-12s | %-12s | %-12s | %-12s",
				wsName,
				tk.Timestamp.Format("15:04:05"),
				"TIMEOUT",
				"N/A",
				"N/A",
			)
		}
	}
	t.Logf("==================================================================================================")

	if len(completedMap) < len(testTasks) {
		t.Fatalf("Timed out waiting for compendium generation. Processed %d/%d tasks.", len(completedMap), len(testTasks))
	}
}
