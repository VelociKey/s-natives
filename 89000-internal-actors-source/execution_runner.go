package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func runExecutionRequest(ctx context.Context, engine *NATVSEngine, defaultCoordDir string, queueFilePath string, outFilePath string, lockPath string, isDirect bool) {
	objective := os.Args[1]
	contextPath := ""
	if len(os.Args) > 2 {
		contextPath = os.Args[2]
	}

	workspaceDir := "00flow/s-natives"
	if len(os.Args) > 3 {
		workspaceDir = os.Args[3]
	} else if contextPath != "" && !strings.Contains(objective, "jules") {
		workspaceDir = contextPath
	}

	taskID := GenerateTaskID()
	task := TaskInput{
		ID:          taskID,
		Timestamp:   time.Now(),
		State:       StateCreated,
		Objective:   objective,
		ContextPath: contextPath,
		Workspace:   workspaceDir,
	}

	if isDirect {
		slog.Info("Executing task directly in-process (bypassing daemon queue)...")
		task.State = StateInProgress
		errExec := runQueueTaskLifecycle(ctx, engine, &task)
		if errExec != nil {
			logFatal("Direct task execution failed: %v", errExec)
		}
		slog.Info("Direct task execution succeeded!")
		os.Exit(0)
	}

	spawnLockPath := filepath.Join(defaultCoordDir, "natvs-spawn.lock")
	var spawnLock *os.File
	var errLock error
	for i := 0; i < 100; i++ {
		spawnLock, errLock = lifecycle.LockFileExclusive(spawnLockPath)
		if errLock == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if errLock != nil {
		logFatal("Failed to acquire spawn serialization lock: %v", errLock)
	}

	tx, err := lifecycle.BeginQueueTransaction(queueFilePath)
	if err != nil {
		spawnLock.Close()
		logFatal("Failed to begin queue transaction for delegation: %v", err)
	}
	tasks, err := tx.Read()
	if err != nil {
		tx.Commit()
		spawnLock.Close()
		logFatal("Failed to read queue for delegation: %v", err)
	}
	tasks = append(tasks, task)
	err = tx.Write(tasks)
	tx.Commit()
	if err != nil {
		spawnLock.Close()
		logFatal("Failed to delegate task: %v", err)
	}
	slog.Info("Task successfully delegated to external server queue", "taskID", taskID)

	portFilePath := filepath.Join(defaultCoordDir, "daemon.port")
	var triggerConn net.Conn
	var triggerErr error

	portData, readErr := os.ReadFile(portFilePath)
	if readErr == nil {
		portStr := strings.TrimSpace(string(portData))
		if port, parseErr := strconv.Atoi(portStr); parseErr == nil {
			triggerConn, triggerErr = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		}
	}

	if triggerConn == nil {
		execPath, err := os.Executable()
		if err != nil {
			execPath = filepath.Clean(filepath.Join(engine.Config.WorkspaceRoot, "00flow", "s-natives", "natvs-engine"+lifecycle.GetExeSuffix()))
		}
		slog.Info("External companion server not running. Spawning detached queue watcher...")
		stdout, err := SpawnDetachedServer(execPath, queueFilePath, outFilePath)
		if err != nil {
			spawnLock.Close()
			logFatal("Failed to spawn detached server: %v", err)
		}

		var daemonPort int
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "PORT ") {
				daemonPortStr := strings.TrimPrefix(line, "PORT ")
				if dp, parseErr := strconv.Atoi(daemonPortStr); parseErr == nil {
					daemonPort = dp
					slog.Info("Daemon signaled it is ready via stdout pipe", "port", daemonPort)
					triggerConn, triggerErr = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", daemonPort), 1*time.Second)
				}
			}
		}
		stdout.Close()

		if triggerConn == nil {
			slog.Info("Fallback to polling for daemon startup...")
			probeStart := time.Now()
			for {
				if CheckExternalServerRunning(lockPath) {
					break
				}
				if time.Since(probeStart) > 10*time.Second {
					spawnLock.Close()
					logFatal("Timed out waiting for spawned detached daemon server to acquire lock")
				}
				time.Sleep(100 * time.Millisecond)
			}

			for attempt := 0; attempt < 5; attempt++ {
				portData, readErr := os.ReadFile(portFilePath)
				if readErr == nil {
					portStr := strings.TrimSpace(string(portData))
					if port, parseErr := strconv.Atoi(portStr); parseErr == nil {
						triggerConn, triggerErr = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
						if triggerErr == nil {
							break
						}
					}
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
	spawnLock.Close()

	if triggerErr == nil && triggerConn != nil {
		defer triggerConn.Close()
		_, writeErr := triggerConn.Write([]byte("WAIT " + task.ID + "\n"))
		if writeErr == nil {
			slog.Info("Blocking on trigger connection for task completion...")
			reader := bufio.NewReader(triggerConn)
			line, readErr := reader.ReadString('\n')
			if readErr == nil {
				line = strings.TrimSpace(line)
				slog.Info("Trigger connection received event", "event", line)
				if strings.HasPrefix(line, "SUCCESS") {
					slog.Info("Task succeeded via trigger!")
					os.Exit(0)
				} else {
					errStr := strings.TrimPrefix(line, "FAILURE ")
					logFatal("Task execution failed via trigger with error: %s", errStr)
				}
			} else {
				slog.Warn("Failed to read from trigger connection, falling back to polling", "err", readErr)
			}
		} else {
			slog.Warn("Failed to send WAIT to trigger connection, falling back to polling", "err", writeErr)
		}
	} else {
		slog.Warn("Could not establish trigger connection, falling back to polling", "err", triggerErr)
	}

	slog.Info("Waiting for task execution outcome (polling fallback)...")
	for {
		time.Sleep(1 * time.Second)
		completed, err := ParseCompletedFileAtomic(outFilePath)
		if err != nil {
			continue
		}
		for _, comp := range completed {
			if comp.ID == task.ID {
				slog.Info("Task completion record detected!", "status", comp.Status)
				if comp.Status == StateCompletedSuccess {
					slog.Info("Task succeeded. Ingestion complete.")
					os.Exit(0)
				} else {
					logFatal("Task execution failed with error: %s", comp.Error)
				}
			}
		}
	}
}

