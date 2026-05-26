package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Coordinator struct {
	engine      *NATVSEngine
	listener    net.Listener
	shutdownCtx context.Context
	cancelFn    context.CancelFunc
	mu          sync.Mutex
	watchdog    *time.Timer
}

func NewCoordinator(engine *NATVSEngine) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		engine:      engine,
		shutdownCtx: ctx,
		cancelFn:    cancel,
	}
}

// Start launches the Unix domain socket or TCP fallback listener.
func (c *Coordinator) Start() error {
	netType, addr := c.engine.GetCoordinationEndpoint()
	
	// Create parent directory for UDS if needed
	if netType == "unix" {
		if len(addr) >= 104 {
			slog.Warn("UDS socket path too long for platform limits, falling back to TCP", "path", addr)
			netType = "tcp"
			addr = fmt.Sprintf("127.0.0.1:%d", c.engine.GetDeterministicTCPPort())
		} else {
			if err := os.MkdirAll(filepath.Dir(addr), 0755); err != nil {
				return fmt.Errorf("failed to create socket dir: %w", err)
			}
			// Clean up stale socket file before listening
			_ = os.Remove(addr)
		}
	}

	l, err := net.Listen(netType, addr)
	if err != nil {
		if netType == "unix" {
			slog.Warn("UDS listen failed, falling back to TCP", "error", err)
			netType = "tcp"
			addr = fmt.Sprintf("127.0.0.1:%d", c.engine.GetDeterministicTCPPort())
			l, err = net.Listen(netType, addr)
		}
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", netType, err)
		}
	}
	c.listener = l
	slog.Info("SACP Coordinator Daemon active", "net", netType, "addr", addr)

	// Setup inactivity watchdog
	timeout := c.engine.Config.IdleTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	c.watchdog = time.AfterFunc(timeout, func() {
		slog.Warn("Coordinator watchdog idle timeout reached. Shutting down daemon...", "timeout", timeout)
		c.Close()
	})

	go c.acceptLoop()
	return nil
}

func (c *Coordinator) acceptLoop() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			slog.Error("Coordinator failed to accept connection", "error", err)
			continue
		}

		c.watchdog.Reset(c.engine.Config.IdleTimeout)
		go c.handleClient(conn)
	}
}

func (c *Coordinator) handleClient(conn net.Conn) {
	defer conn.Close()
	slog.Info("Coordinator accepted new client stream")

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				slog.Error("Coordinator stream read failed", "error", err)
			}
			return
		}

		c.watchdog.Reset(c.engine.Config.IdleTimeout)

		// Echo or command routing logic (mocked for reconnect validation)
		slog.Info("Coordinator received command payload", "bytes", n, "content", string(buf[:n]))
		_, err = conn.Write([]byte("ACK: " + string(buf[:n])))
		if err != nil {
			slog.Error("Coordinator write response failed", "error", err)
			return
		}
	}
}

func (c *Coordinator) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cancelFn()
	if c.watchdog != nil {
		c.watchdog.Stop()
	}
	if c.listener != nil {
		c.listener.Close()
	}
	
	netType, addr := c.engine.GetCoordinationEndpoint()
	if netType == "unix" {
		_ = os.Remove(addr)
	}
}

// DialOrSpawnCoordinator attempts to connect to the coordinator. 
// If connection fails, it cleans stale files, spawns a new daemon process, and retries.
func DialOrSpawnCoordinator(engine *NATVSEngine) (net.Conn, error) {
	netType, addr := engine.GetCoordinationEndpoint()
	if netType == "unix" && len(addr) >= 104 {
		netType = "tcp"
		addr = fmt.Sprintf("127.0.0.1:%d", engine.GetDeterministicTCPPort())
	}
	
	conn, err := net.DialTimeout(netType, addr, 200*time.Millisecond)
	if err == nil {
		return conn, nil
	}

	// Dial failed. Cleanup stale socket if present.
	if netType == "unix" {
		if _, statErr := os.Stat(addr); statErr == nil {
			slog.Warn("Stale UDS socket detected on disk. Removing...", "path", addr)
			_ = os.Remove(addr)
		}
	}

	slog.Info("Spawning SACP Coordination Daemon in background...")
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(exe, "daemon")
	cmd.Dir = engine.Config.WorkspaceRoot
	cmd.Env = append(os.Environ(), "TEST_WORKSPACE_ROOT="+engine.Config.WorkspaceRoot)
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to spawn coordination daemon: %w", err)
	}

	// Retry dial with backoff (up to 1 second)
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.DialTimeout(netType, addr, 200*time.Millisecond)
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("failed to connect to coordination daemon after spawn: %w", err)
}

// ShutdownContext returns the coordination daemon's shutdown context.
func (c *Coordinator) ShutdownContext() context.Context {
	return c.shutdownCtx
}
