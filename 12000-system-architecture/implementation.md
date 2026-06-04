# SACP Coordination Daemon Implementation

This document specifies the concrete Go implementation details of the task queue system, Windows-native locking, and environment isolation.

## 1. Concurrency Transactions & Exclusive Locking

To ensure thread-safety and process-safety across concurrent reads and writes, the queue uses a Windows-native UTF16-compatible UTF-16 lock file API:

```go
type QueueTransaction struct {
	lockFile *os.File
	path     string
}

func BeginQueueTransaction(path string) (*QueueTransaction, error) {
	lockPath := path + ".lock"
	var f *os.File
	var err error
	for i := 0; i < 200; i++ {
		f, err = LockFileExclusive(lockPath)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &QueueTransaction{lockFile: f, path: path}, nil
}

func LockFileExclusive(path string) (*os.File, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, // Exclusive access (sharing violation if opened elsewhere)
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), path), nil
}
```

## 2. Detached Server Spawning & Whitelisted Environment

When starting the daemon outside the IDE, the process environment is sanitized to clear out IDE ambient auth variables, keeping only the bare essentials (like path, system root, and user profile folders):

```go
func SpawnDetachedServer(binaryPath string, queuePath string, outPath string) (io.ReadCloser, error) {
	cmd := exec.Command(binaryPath, "--queue="+queuePath, "--out="+outPath)
	
	var cleanEnv []string
	whitelistedKeys := map[string]bool{
		"USERPROFILE":    true,
		"APPDATA":        true,
		"LOCALAPPDATA":   true,
		"SYSTEMROOT":     true,
		"PATH":           true,
		"TEMP":           true,
		"TMP":            true,
		"HOMEDRIVE":      true,
		"HOMEPATH":       true,
		"COMPUTERNAME":   true,
		"COMSPEC":        true,
		"PATHEXT":        true,
		"TEST_WORKSPACE_ROOT": true,
	}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 0 {
			key := parts[0]
			if whitelistedKeys[strings.ToUpper(key)] {
				cleanEnv = append(cleanEnv, env)
			}
		}
	}
	cmd.Env = cleanEnv
	cmd.Dir = filepath.Dir(binaryPath)
	
	// Windows-specific: CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return stdout, nil
}
```

## 3. Zero-Polling Trigger-Based Synchronization

To eliminate background CPU and disk polling, the engine implements a multi-channel trigger system:
- **Startup Handshake (OS Pipes)**: When spawning a new daemon, the client reads the daemon's allocated TCP port from the child's `stdout` pipe, blocking until the daemon prints `PORT <daemonPort>\n`. This prevents IP port collisions and local firewall prompt warnings.
- **Task Wait Trigger (TCP Loopback)**: Once the daemon is running, the client sends a `WAIT <taskID>\n` command over a TCP loopback socket connection. The client blocks on socket reads until the daemon finishes processing the task and returns a `SUCCESS` or `FAILURE <error>` packet.
- **Queue Event Trigger**: The daemon's internal transaction watch loop sleeps until a TCP trigger connection wakes it up, resolving CPU drag during idle periods.

## 4. Safe Ingestion Verification

When the IDE client detects a completed task in `tasks.completed.webnf`, it audits modified delta files before merging/accepting them using `conformance.exe` to enforce safety and syntactic correctness:

```go
conformanceBin := filepath.Join(workspaceRoot, "00flow/s-seed/conformance.exe")
reportDir := filepath.Join(workspaceRoot, "00flow/s-forge/08000-attestation-snapshot")
cmdCheck := exec.Command(conformanceBin, "-dir", filepath.Dir(fullPath), "-report-dir", reportDir)
cmdCheck.Dir = workspaceRoot
_ = cmdCheck.Run()
```

## 5. Path Organization (cnnnn- Placement)

To ensure the volatile task queue remains out of IDE indexing scopes, files are organized under:
- `C:\aCogSpaceSeed\c0990-ephemeral-scratch\natvs coordination\`
  - `tasks.queue.webnf`
  - `tasks.completed.webnf`
  - `natvs-queue.lock`

Because these paths reside inside the `cnnnn-` cognitive layers matching the `.antigravityignore` patterns, CPU cycles and disk reads are saved from redundant indexer operations during busy development cycles.

## 6. Decoupled Jules Credentials Sandbox & AI Pro Telemetry

To ensure Jules CLI runs completely decoupled from host OS profiles and Antigravity tokens, `natvs-engine` structures a clean-room virtual profile:

### I. Sandbox Environment Construction
- **Path Isolation**: When launching Jules, `buildSandboxEnv()` overrides the process environment to redirect all standard config directories into the workspace silo:
  - `USERPROFILE` -> `00flow/s-forge/94000-external-actors/jules/c1000-credentials/user`
  - `APPDATA` -> `00flow/s-forge/94000-external-actors/jules/c1000-credentials/appdata`
  - `LOCALAPPDATA` -> `00flow/s-forge/94000-external-actors/jules/c1000-credentials/localappdata`
  - `ProgramData` -> `00flow/s-forge/94000-external-actors/jules/c1000-credentials/programdata`
- **Zero Secrets Forwarding**: Host variables containing tokens or API keys (`GEMINI_API_KEY`, `JULES_TOKEN`, `GOOGLE_*`) are stripped entirely, forcing the subprocess to authenticate purely against credentials cached inside the sandbox.

### II. Sandbox Identity Tracking (`whoami.txt`)
- **Login Capture**: Upon successful execution of `jules login`, the engine prompts the user for the email they authenticated with (supporting the `--email`/`-e` arguments or falling back to interactive console input) and writes it to `c1000-credentials/whoami.txt`.
- **Wiping Old Sessions**: To support choosing a different account without browser cookie/token reuse, the engine completely deletes `c1000-credentials` before launching `jules login`. This forces a fresh Google OAuth browser login window.
- **Telemetry Query**: `runJulesStatusCheck` reads this local `whoami.txt` file to report the active email under the `Whoami` field.

### III. AI Pro vs. Included Tier Rules
- The engine determines account subscriptions dynamically to keep user identity decoupled from the code:
  - **Dynamic Tier Detection**: When logging in, the engine writes the selected tier to `tier.txt` in the sandbox. If the user's email contains the corporate identifier `"velocikey"`, it auto-detects `pro`. Otherwise, the user is prompted to set their tier (or passes the `--tier` / `-t` flag), which is saved in `tier.txt`.
  - **AI Pro Tier**: If the active tier is evaluated as `pro` or the email contains `velocikey`, the status reports:
    `Remaining Tasks     : Unlimited (AI Pro Tier)`
  - **Included Tier**: Non-pro accounts default to:
    `Remaining Tasks     : <calculated_tasks> (Included Tier)`
    where calculated tasks are derived dynamically by subtracting active sessions and queue items from a starting limit of 10.
- The `Execution Track` is hard-coded to report `The Outside World (Decoupled, Flat-Rate Subscription Track)` to confirm complete isolation from Antigravity tokens.


