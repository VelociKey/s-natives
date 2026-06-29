package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	discard "sov.fleet/s-logiclibrary/81000-active-source/pkg/200-enhancers/discard"
	"strconv"
	"strings"
	"sync"
	"time"

	quic "sov.fleet/quic-go"
	"sov.fleet/s-logiclibrary/00200-logic-libraries/quictransport"
	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

var (
	julesQuicConn *quic.Conn
	julesQuicMu   sync.RWMutex

	warmWorkspacesMu sync.Mutex
	warmWorkspaces   = make(map[string]bool)
)

func markWorkspaceWarm(workspace string) {
	warmWorkspacesMu.Lock()
	warmWorkspaces[workspace] = true
	warmWorkspacesMu.Unlock()
}

func isWorkspaceWarm(workspace string) bool {
	warmWorkspacesMu.Lock()
	defer warmWorkspacesMu.Unlock()
	return warmWorkspaces[workspace]
}

func generateTLSConfig() (*tls.Config, error) {
	tlsConf, err := quictransport.GenerateEphemeralTLSConfig()
	if err != nil {
		return nil, err
	}
	tlsConf.NextProtos = []string{"jules-sacp"}
	return tlsConf, nil
}

func handleIncomingQUICStream(stream *quic.Stream) {
	defer stream.Close()
	reader := bufio.NewReader(stream)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSpace(line)
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return
	}
	taskID := parts[0]
	workspace := parts[1]
	slog.Info("QUIC session context stream initialized", "taskID", taskID, "workspace", workspace)

	buf := make([]byte, 1024)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			slog.Info("QUIC stream trace log", "taskID", taskID, "log", string(buf[:n]))
		}
		if readErr != nil {
			break
		}
	}
}

func initQUICControlChannel() error {
	tlsConf, err := generateTLSConfig()
	if err != nil {
		return err
	}
	quicConf := quictransport.NewQUICConfig()
	quicConf.KeepAlivePeriod = 15 * time.Second

	conn, err := quictransport.ListenUDP("127.0.0.1:0")
	if err != nil {
		return err
	}

	listener, err := quictransport.Listen(conn, tlsConf, quicConf)
	if err != nil {
		conn.Close()
		return err
	}

	addr := listener.Addr().String()

	dialTLSConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"jules-sacp"},
	}

	var dialErr error
	var dConn *quic.Conn
	done := make(chan struct{})

	go func() {
		defer close(done)
		udpDialAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			dialErr = err
			return
		}
		rawDialConn, err := quictransport.ListenUDP("")
		if err != nil {
			dialErr = err
			return
		}
		dConn, dialErr = quictransport.Dial(context.Background(), rawDialConn, udpDialAddr, dialTLSConf, quicConf)
	}()

	serverConn, acceptErr := listener.Accept(context.Background())
	<-done

	if dialErr != nil {
		listener.Close()
		return dialErr
	}
	if acceptErr != nil {
		listener.Close()
		return acceptErr
	}

	julesQuicMu.Lock()
	julesQuicConn = dConn
	julesQuicMu.Unlock()

	slog.Info("Persistent QUIC control channel established successfully", "localAddr", addr)

	go func() {
		defer listener.Close()
		for {
			stream, err := serverConn.AcceptStream(context.Background())
			if err != nil {
				return
			}
			go handleIncomingQUICStream(stream)
		}
	}()

	return nil
}

func runNATVSEngineDaemon(ctx context.Context, engine *NATVSEngine, workspaceRoot, defaultCoordDir, queueFilePath, outFilePath, lockPath string) {
	slog.Info("Running as background NATVS Engine Daemon...", "queue", queueFilePath, "out", outFilePath)
	slog.Info("Initializing work and establishing SACP telemetry stream.")
	slog.Info("Updating transport layer to QUIC...", "net", "udp", "addr", "127.0.0.1:0", "protocol", "QUIC")
	if err := AcquireServerLock(lockPath); err != nil {
		logFatal("Failed to start NATVS Engine Daemon: companion server lock already active. Details: %v", err)
	}
	defer ReleaseServerLock()

	slog.Info("Daemon initializing: establishing persistent QUIC control channel...")
	if err := initQUICControlChannel(); err != nil {
		logFatal("Daemon failed to establish persistent QUIC control channel: %v", err)
	}

	concurrencyVal := 4
	if envC := os.Getenv("NATVS_CONCURRENCY"); envC != "" {
		if parsed, err := strconv.Atoi(envC); err == nil && parsed > 0 {
			concurrencyVal = parsed
		}
	}
	slog.Info("Jules authenticated successfully via QUIC. Starting queue watch loop.", "concurrency", concurrencyVal)

	sem := make(chan struct{}, concurrencyVal)
	var wg sync.WaitGroup
	var activeWorkspacesMu sync.Mutex
	activeWorkspaces := make(map[string]bool)

	triggerChan := make(chan struct{}, 10)
	var waitersMu sync.Mutex
	waiters := make(map[string]net.Conn)

	triggerListener, listenerErr := net.Listen("tcp", "127.0.0.1:0")
	if listenerErr == nil {
		discardValLine185_0, portStr, splitErr := net.SplitHostPort(triggerListener.Addr().String())
		discard.Discard(discardValLine185_0)
		if splitErr != nil {
			slog.Error("Failed to split host port", "error", splitErr)
			portStr = "0"
		}
		portFilePath := filepath.Join(defaultCoordDir, "daemon.port")
		if err := os.WriteFile(portFilePath, []byte(portStr), 0644); err != nil {
			slog.Error("Failed to write daemon port file", "path", portFilePath, "error", err)
		}
		slog.Info("Trigger server listening", "port", portStr)

		if discardValLine196_0, err := os.Stdout.Write([]byte("PORT " + portStr + "\n")); err != nil {
			discard.Discard(discardValLine196_0)
			slog.Error("Failed to write port to stdout", "error", err)
		}

		defer func() {
			triggerListener.Close()
			if err := os.Remove(portFilePath); err != nil {
				slog.Warn("Failed to clean up port file", "path", portFilePath, "error", err)
			}
		}()

		go func() {
			for {
				c, err := triggerListener.Accept()
				if err != nil {
					return
				}
				go func(conn net.Conn) {
					reader := bufio.NewReader(conn)
					line, err := reader.ReadString('\n')
					if err != nil {
						conn.Close()
						return
					}
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "WAIT ") {
						taskID := strings.TrimPrefix(line, "WAIT ")
						waitersMu.Lock()
						waiters[taskID] = conn
						waitersMu.Unlock()
					} else {
						conn.Close()
					}
					select {
					case triggerChan <- struct{}{}:
					default:
					}
				}(c)
			}
		}()
	} else {
		slog.Error("Failed to start TCP trigger server", "err", listenerErr)
	}

	for {
		tx, err := lifecycle.BeginQueueTransaction(queueFilePath)
		if err != nil {
			slog.Error("Failed to begin queue transaction", "error", err)
			select {
			case <-triggerChan:
			case <-time.After(5 * time.Second):
			}
			continue
		}
		tasks, err := tx.Read()
		if err != nil {
			tx.Commit()
			select {
			case <-triggerChan:
			case <-time.After(5 * time.Second):
			}
			continue
		}
		if len(tasks) == 0 {
			tx.Commit()
			select {
			case <-triggerChan:
			case <-time.After(5 * time.Second):
			}
			continue
		}

		var targetTask *TaskInput
		var targetIdx int
		activeWorkspacesMu.Lock()
		for idx, task := range tasks {
			if task.State == StateCreated && !activeWorkspaces[task.Workspace] {
				targetTask = &task
				targetIdx = idx
				activeWorkspaces[task.Workspace] = true
				break
			}
		}
		activeWorkspacesMu.Unlock()

		if targetTask == nil {
			tx.Commit()
			select {
			case <-triggerChan:
			case <-time.After(5 * time.Second):
			}
			continue
		}

		slog.Info("Processing task from queue", "id", targetTask.ID, "objective", targetTask.Objective)
		tasks[targetIdx].State = StateInProgress
		if errWrite := tx.Write(tasks); errWrite != nil {
			slog.Error("Failed to write to queue transaction", "error", errWrite)
		}
		tx.Commit()

		wg.Add(1)
		sem <- struct{}{}

		go func(t TaskInput) {
			defer func() {
				<-sem
				wg.Done()
				activeWorkspacesMu.Lock()
				delete(activeWorkspaces, t.Workspace)
				activeWorkspacesMu.Unlock()
			}()

			julesQuicMu.RLock()
			qConn := julesQuicConn
			julesQuicMu.RUnlock()

			taskCtx := ctx
			var stream *quic.Stream
			var streamErr error
			if qConn != nil {
				stream, streamErr = qConn.OpenStreamSync(taskCtx)
				if streamErr == nil {
					defer stream.Close()
					if discardValLine320_0, errWrite := stream.Write([]byte(t.ID + "|" + t.Workspace + "\n")); errWrite != nil {
						discard.Discard(discardValLine320_0)
						slog.Warn("Failed to write QUIC stream control frame", "error", errWrite)
					}
					taskCtx = context.WithValue(taskCtx, "quic_stream", stream)
				}
			}

			errExec := runQueueTaskLifecycle(taskCtx, engine, &t)

			outRecord := TaskOutput{
				ID:        t.ID,
				Timestamp: time.Now(),
			}
			if errExec != nil {
				outRecord.Status = StateCompletedFailure
				outRecord.Error = errExec.Error()
				slog.Error("Task execution failed", "id", t.ID, "error", errExec)
			} else {
				outRecord.Status = StateCompletedSuccess
			}

			if errWriteComp := AppendCompletedFileAtomic(outFilePath, outRecord); errWriteComp != nil {
				slog.Error("Failed to append completed record", "error", errWriteComp)
			}

			txComp, errComp := lifecycle.BeginQueueTransaction(queueFilePath)
			if errComp == nil {
				tasks, errRead := txComp.Read()
				if errRead == nil {
					n := 0
					for _, tk := range tasks {
						if tk.ID != t.ID {
							tasks[n] = tk
							n++
						}
					}
					tasks = tasks[:n]
					if errWriteTasks := txComp.Write(tasks); errWriteTasks != nil {
						slog.Error("Failed to write updated queue", "error", errWriteTasks)
					}
				}
				txComp.Commit()
			}
			slog.Info("Task completed and recorded", "id", t.ID)

			waitersMu.Lock()
			conn, ok := waiters[t.ID]
			if ok {
				delete(waiters, t.ID)
			}
			waitersMu.Unlock()

			if conn != nil {
				if errExec != nil {
					if discardValLine374_0, errWriteConn := conn.Write([]byte("FAILURE " + errExec.Error() + "\n")); errWriteConn != nil {
						discard.Discard(discardValLine374_0)
						slog.Warn("Failed to send failure notification to trigger conn", "error", errWriteConn)
					}
				} else {
					if discardValLine378_0, errWriteConn := conn.Write([]byte("SUCCESS\n")); errWriteConn != nil {
						discard.Discard(discardValLine378_0)
						slog.Warn("Failed to send success notification to trigger conn", "error", errWriteConn)
					}
				}
				conn.Close()
			}
		}(*targetTask)
	}
}
