package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

type EchoBackend struct{}

func (b *EchoBackend) Initialize(ctx context.Context) error { return nil }
func (b *EchoBackend) Validate(ctx context.Context) error   { return nil }
func (b *EchoBackend) Dial(ctx context.Context) (net.Conn, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		defer l.Close()
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 1024)
		for {
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			if _, err := c.Write([]byte("ACK: " + string(buf[:n]))); err != nil {
				slog.Error("Failed to write to TCP connection", "error", err)
				return
			}
		}
	}()
	return net.Dial("tcp", l.Addr().String())
}
func (b *EchoBackend) Shutdown(ctx context.Context) error { return nil }

// Facade Types and Constants for Backward Compatibility with Tests and External Packages

type NATVSPhase = lifecycle.NATVSPhase

const (
	PhaseNegotiate    = lifecycle.PhaseNegotiate
	PhaseAssimilation = lifecycle.PhaseAssimilation
	PhaseTransform    = lifecycle.PhaseTransform
	PhaseVerification = lifecycle.PhaseVerification
	PhaseSynthesis    = lifecycle.PhaseSynthesis
)

type NATVSEngine = lifecycle.NATVSEngine
type OrchestrationConfig = lifecycle.OrchestrationConfig

func NewNATVSEngine(workspace string) *NATVSEngine {
	return lifecycle.NewNATVSEngine(workspace)
}

type TaskState = lifecycle.TaskState

const (
	StateCreated          = lifecycle.StateCreated
	StateInProgress       = lifecycle.StateInProgress
	StateCompletedSuccess = lifecycle.StateCompletedSuccess
	StateCompletedFailure = lifecycle.StateCompletedFailure
)

type TaskInput = lifecycle.TaskInput
type TaskOutput = lifecycle.TaskOutput

var CheckExternalServerRunning = lifecycle.CheckExternalServerRunning
var AcquireServerLock = lifecycle.AcquireServerLock
var ReleaseServerLock = lifecycle.ReleaseServerLock
var GenerateTaskID = lifecycle.GenerateTaskID
var SpawnDetachedServer = lifecycle.SpawnDetachedServer
var ParseQueueFileAtomic = lifecycle.ParseQueueFileAtomic
var WriteQueueFileAtomic = lifecycle.WriteQueueFileAtomic
var ParseCompletedFileAtomic = lifecycle.ParseCompletedFileAtomic
var AppendCompletedFileAtomic = lifecycle.AppendCompletedFileAtomic

var logFatal = func(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	slog.Error("Fatal error encountered", "details", msg)
	os.Exit(1)
}

func main() {
	// Check for PoC Hello flag
	pocHello := false
	for _, arg := range os.Args {
		if arg == "--poc-hello" {
			pocHello = true
			break
		}
	}

	if pocHello {
		workspaceRoot := findWorkspaceRoot()
		withinAntigravity := os.Getenv("ANTIGRAVITY_AGENT") == "1" || os.Getenv("ANTIGRAVITY_PROJECT_ID") != ""

		julesPath := filepath.Clean(filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "jules"+lifecycle.GetExeSuffix()))

		// Run a quick check command to verify auth state
		cmd := exec.Command(julesPath, "remote", "list", "--session")
		cmd.Env = lifecycle.BuildSandboxEnv(workspaceRoot, "")

		err := cmd.Run()
		authState := "ACTIVE"
		if err != nil {
			authState = fmt.Sprintf("INACTIVE (error: %v)", err)
		}

		location := "the outside world"
		if withinAntigravity {
			location = "within Antigravity"
		}

		if _, err := os.Stdout.Write([]byte("Jules says Hello from " + location + "\n")); err != nil {
			slog.Error("Failed to write to stdout", "error", err)
		}
		slog.Info("PoC Auth Check Status", "state", authState)
		os.Exit(0)
	}

	// Check for jules-status / jules:status / jules status / jules auth status
	isStatusCheck := false
	for i, arg := range os.Args {
		if arg == "jules-status" || arg == "jules:status" {
			isStatusCheck = true
			break
		}
		if arg == "jules" && i+1 < len(os.Args) && os.Args[i+1] == "status" {
			isStatusCheck = true
			break
		}
		if arg == "jules" && i+2 < len(os.Args) && os.Args[i+1] == "auth" && os.Args[i+2] == "status" {
			isStatusCheck = true
			break
		}
	}

	if isStatusCheck {
		workspaceRoot := findWorkspaceRoot()
		engine := NewNATVSEngine(workspaceRoot)
		err := runJulesStatusCheck(engine)
		if err != nil {
			logFatal("Status check failed: %v", err)
		}
		os.Exit(0)
	}

	// Check for jules-login / jules:login / jules login / jules auth login
	isLogin := false
	for i, arg := range os.Args {
		if arg == "jules-login" || arg == "jules:login" {
			isLogin = true
			break
		}
		if arg == "jules" && i+1 < len(os.Args) && os.Args[i+1] == "login" {
			isLogin = true
			break
		}
		if arg == "jules" && i+2 < len(os.Args) && os.Args[i+1] == "auth" && os.Args[i+2] == "login" {
			isLogin = true
			break
		}
	}

	if isLogin {
		workspaceRoot := findWorkspaceRoot()
		engine := NewNATVSEngine(workspaceRoot)
		err := runJulesLogin(engine)
		if err != nil {
			logFatal("Login failed: %v", err)
		}
		os.Exit(0)
	}

	slog.Info("=========================================================")
	slog.Info("         SACP WARM-START DAEMON ENGAGED          ")
	slog.Info("         SOVEREIGN NATIVES ORCHESTRATOR          ")
	slog.Info("=========================================================")

	allowedWorkspaces := make([]string, 0, len(os.Args))
	var idleTimeoutOverride time.Duration
	var queueFilePath string
	var outFilePath string
	var checkCompleted bool
	var hasQueueArg bool

	var isDirect bool
	argsFiltered := make([]string, 0, len(os.Args))
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--workspaces=") {
			wsList := strings.TrimPrefix(arg, "--workspaces=")
			parts := strings.Split(wsList, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					allowedWorkspaces = append(allowedWorkspaces, part)
				}
			}
		} else if strings.HasPrefix(arg, "--idle-timeout=") {
			timeoutStr := strings.TrimPrefix(arg, "--idle-timeout=")
			if parsed, err := time.ParseDuration(timeoutStr); err == nil {
				idleTimeoutOverride = parsed
			} else {
				slog.Error("Invalid idle-timeout duration", "value", timeoutStr, "error", err)
			}
		} else if strings.HasPrefix(arg, "--queue=") {
			queueFilePath = strings.TrimPrefix(arg, "--queue=")
			hasQueueArg = true
		} else if strings.HasPrefix(arg, "--out=") {
			outFilePath = strings.TrimPrefix(arg, "--out=")
		} else if arg == "--check-completed" {
			checkCompleted = true
		} else if arg == "--direct" {
			isDirect = true
		} else {
			argsFiltered = append(argsFiltered, arg)
		}
	}

	isExecutionRequest := len(argsFiltered) > 1 && argsFiltered[1] != "daemon" && argsFiltered[1] != "--check-completed" && argsFiltered[1] != "--poc-hello" && !hasQueueArg

	os.Args = argsFiltered

	workspaceRoot := findWorkspaceRoot()

	defaultCoordDir := filepath.Join(workspaceRoot, "c0990-ephemeral-scratch/natvs coordination")
	if err := os.MkdirAll(defaultCoordDir, 0755); err != nil {
		slog.Error("Failed to create default coordination directory", "path", defaultCoordDir, "error", err)
	}

	if queueFilePath == "" {
		queueFilePath = filepath.Join(defaultCoordDir, "tasks.queue.webnf")
	}
	if outFilePath == "" {
		outFilePath = filepath.Join(defaultCoordDir, "tasks.completed.webnf")
	}
	lockPath := filepath.Join(defaultCoordDir, "natvs-queue.lock")

	if checkCompleted {
		slog.Info("Checking completed external tasks...")
		completed, err := ParseCompletedFileAtomic(outFilePath)
		if err != nil {
			logFatal("Failed to read completed tasks: %v", err)
		}
		for _, comp := range completed {
			if _, err := os.Stdout.Write([]byte("TaskResult \"" + comp.ID + "\" [status: " + string(comp.Status) + ", timestamp: " + comp.Timestamp.Format(time.RFC3339) + "]\n")); err != nil {
				slog.Error("Failed to write task result to stdout", "error", err)
			}
			if len(comp.DeltaPaths) > 0 {
				if _, err := os.Stdout.Write([]byte("  Delta paths:\n")); err != nil {
					slog.Error("Failed to write delta paths header", "error", err)
				}
				for _, p := range comp.DeltaPaths {
					if _, err := os.Stdout.Write([]byte("    - " + p + "\n")); err != nil {
						slog.Error("Failed to write delta path", "error", err)
					}
				}
			}
			if comp.Error != "" {
				if _, err := os.Stdout.Write([]byte("  Error: " + comp.Error + "\n")); err != nil {
					slog.Error("Failed to write error details", "error", err)
				}
			}
		}
		os.Exit(0)
	}

	engine := NewNATVSEngine(workspaceRoot)
	if len(allowedWorkspaces) > 0 {
		engine.Config.AllowedWorkspaces = allowedWorkspaces
	}
	if idleTimeoutOverride > 0 {
		engine.Config.IdleTimeout = idleTimeoutOverride
	}
	ctx := context.Background()

	if isExecutionRequest {
		runExecutionRequest(ctx, engine, defaultCoordDir, queueFilePath, outFilePath, lockPath, isDirect)
	}


	runAsDaemon := queueFilePath != "" && hasQueueArg
	if runAsDaemon {
		runNATVSEngineDaemon(ctx, engine, workspaceRoot, defaultCoordDir, queueFilePath, outFilePath, lockPath)
	}

	if len(os.Args) > 1 {
		objective := os.Args[1]
		contextPath := ""
		if len(os.Args) > 2 {
			contextPath = os.Args[2]
		}

		slog.Info("Received Goal", "objective", objective)
		slog.Info("Context Path", "path", contextPath)

		if objective == "daemon" {
			runDaemonMode(ctx, engine)
			return
		}

		logFatal("Unsupported objective: %s", objective)
	}

	runSelfConformance(ctx, engine)
}

func findWorkspaceRoot() string {
	if envRoot := os.Getenv("TEST_WORKSPACE_ROOT"); envRoot != "" {
		return filepath.Clean(envRoot)
	}
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, ".gitroot")); err == nil {
				return filepath.Clean(dir)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		fallback := filepath.Join(home, "aCogSpaceSeed")
		if _, err := os.Stat(fallback); err == nil {
			return filepath.Clean(fallback)
		}
	}
	return "."
}

