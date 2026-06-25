package lifecycle

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type TaskState string

const (
	StateCreated          TaskState = "created"
	StateInProgress       TaskState = "in_progress"
	StateCompletedSuccess TaskState = "completed_success"
	StateCompletedFailure TaskState = "completed_failure"
)

type TaskInput struct {
	ID          string
	Timestamp   time.Time
	State       TaskState
	Objective   string
	ContextPath string
	Workspace   string
}

type TaskOutput struct {
	ID         string
	Timestamp  time.Time
	Status     TaskState
	DeltaPaths []string
	Error      string
}

// QueueTransaction manages a locked queue file transaction
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

func (t *QueueTransaction) Read() ([]TaskInput, error) {
	return ParseQueueFile(t.path)
}

func (t *QueueTransaction) Write(tasks []TaskInput) error {
	return WriteQueueFile(t.path, tasks)
}

func (t *QueueTransaction) Commit() {
	if t.lockFile != nil {
		t.lockFile.Close()
		t.lockFile = nil
	}
}

// ExclusiveLock holds the locked file pointer
var ExclusiveLock *os.File

func CheckExternalServerRunning(lockPath string) bool {
	f, err := LockFileExclusive(lockPath)
	if err != nil {
		// Sharing violation or access denied means it is locked (running)
		return true
	}
	f.Close()
	return false
}

func AcquireServerLock(lockPath string) error {
	f, err := LockFileExclusive(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire server lock: %w", err)
	}
	ExclusiveLock = f
	return nil
}

func ReleaseServerLock() {
	if ExclusiveLock != nil {
		ExclusiveLock.Close()
		ExclusiveLock = nil
	}
}

func LockFileExclusive(path string) (*os.File, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, // Exclusive access
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

func GenerateTaskID() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		slog.Error("Failed to generate secure random number", "error", err)
		nBig = big.NewInt(0)
	}
	return fmt.Sprintf("task-%d-%06d", time.Now().UnixNano(), nBig.Int64())
}

func SpawnDetachedServer(binaryPath string, queuePath string, outPath string) (io.ReadCloser, error) {
	cmd := exec.Command(binaryPath, "--queue="+queuePath, "--out="+outPath)

	var cleanEnv []string
	whitelistedKeys := map[string]bool{
		"USERPROFILE":         true,
		"APPDATA":             true,
		"LOCALAPPDATA":        true,
		"SYSTEMROOT":          true,
		"PATH":                true,
		"TEMP":                true,
		"TMP":                 true,
		"HOMEDRIVE":           true,
		"HOMEPATH":            true,
		"COMPUTERNAME":        true,
		"COMSPEC":             true,
		"PATHEXT":             true,
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

	// Spawn detached process in new group on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS
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

func ParseQueueFileAtomic(path string) ([]TaskInput, error) {
	lockPath := path + ".lock"
	var f *os.File
	var err error
	for i := 0; i < 100; i++ {
		f, err = LockFileExclusive(lockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock queue file for reading: %w", err)
	}
	defer f.Close()

	return ParseQueueFile(path)
}

func WriteQueueFileAtomic(path string, tasks []TaskInput) error {
	lockPath := path + ".lock"
	var f *os.File
	var err error
	for i := 0; i < 100; i++ {
		f, err = LockFileExclusive(lockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to lock queue file for writing: %w", err)
	}
	defer f.Close()

	return WriteQueueFile(path, tasks)
}

func AppendCompletedFileAtomic(path string, out TaskOutput) error {
	lockPath := path + ".lock"
	var f *os.File
	var err error
	for i := 0; i < 100; i++ {
		f, err = LockFileExclusive(lockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to lock completed file for appending: %w", err)
	}
	defer f.Close()

	return AppendCompletedFile(path, out)
}

func ParseCompletedFileAtomic(path string) ([]TaskOutput, error) {
	lockPath := path + ".lock"
	var f *os.File
	var err error
	for i := 0; i < 100; i++ {
		f, err = LockFileExclusive(lockPath)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock completed file for reading: %w", err)
	}
	defer f.Close()

	return ParseCompletedFile(path)
}

func ParseQueueFile(path string) ([]TaskInput, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var tasks []TaskInput
	var currentTask *TaskInput

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // Support tokens up to 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Task ") && strings.HasSuffix(line, "{") {
			id := strings.TrimSuffix(line, "{")
			id = strings.TrimSpace(id)
			id = strings.TrimPrefix(id, "Task ")
			id = strings.Trim(id, "\" \t")
			currentTask = &TaskInput{ID: id}
			continue
		}

		if line == "}" {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
				currentTask = nil
			}
			continue
		}

		if currentTask != nil && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)

			switch key {
			case "timestamp":
				if t, err := time.Parse(time.RFC3339, val); err == nil {
					currentTask.Timestamp = t
				}
			case "state":
				currentTask.State = TaskState(val)
			case "objective":
				currentTask.Objective = val
			case "context_path":
				currentTask.ContextPath = val
			case "workspace":
				currentTask.Workspace = val
			}
		}
	}
	return tasks, scanner.Err()
}

func WriteQueueFile(path string, tasks []TaskInput) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString("# tasks.queue.webnf\n\n"); err != nil {
		return err
	}
	for _, task := range tasks {
		if _, err := writer.WriteString(fmt.Sprintf("Task %q {\n", task.ID)); err != nil {
			return err
		}
		if _, err := writer.WriteString(fmt.Sprintf("    timestamp = %q\n", task.Timestamp.Format(time.RFC3339))); err != nil {
			return err
		}
		if _, err := writer.WriteString(fmt.Sprintf("    state = %q\n", task.State)); err != nil {
			return err
		}
		if _, err := writer.WriteString(fmt.Sprintf("    objective = %q\n", task.Objective)); err != nil {
			return err
		}
		if _, err := writer.WriteString(fmt.Sprintf("    context_path = %q\n", task.ContextPath)); err != nil {
			return err
		}
		if _, err := writer.WriteString(fmt.Sprintf("    workspace = %q\n", task.Workspace)); err != nil {
			return err
		}
		if _, err := writer.WriteString("}\n\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func AppendCompletedFile(path string, out TaskOutput) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var deltaLines []string
	for _, p := range out.DeltaPaths {
		deltaLines = append(deltaLines, fmt.Sprintf("        %q", p))
	}
	deltaStr := "[]"
	if len(deltaLines) > 0 {
		deltaStr = "[\n" + strings.Join(deltaLines, ",\n") + "\n    ]"
	}

	_, err = file.WriteString(fmt.Sprintf("TaskResult %q {\n"+
		"    timestamp = %q\n"+
		"    status = %q\n"+
		"    delta_paths = %s\n"+
		"    error = %q\n"+
		"}\n\n",
		out.ID,
		out.Timestamp.Format(time.RFC3339),
		out.Status,
		deltaStr,
		out.Error,
	))
	return err
}

func ParseCompletedFile(path string) ([]TaskOutput, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var results []TaskOutput
	var currentResult *TaskOutput
	var inDeltaPaths bool

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // Support tokens up to 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "TaskResult ") && strings.HasSuffix(line, "{") {
			id := strings.TrimSuffix(line, "{")
			id = strings.TrimSpace(id)
			id = strings.TrimPrefix(id, "TaskResult ")
			id = strings.Trim(id, "\" \t")
			currentResult = &TaskOutput{ID: id}
			continue
		}

		if line == "}" {
			if currentResult != nil {
				results = append(results, *currentResult)
				currentResult = nil
			}
			inDeltaPaths = false
			continue
		}

		if currentResult != nil {
			if inDeltaPaths {
				if strings.HasSuffix(line, "]") {
					inDeltaPaths = false
					val := strings.TrimSuffix(line, "]")
					val = strings.TrimSpace(val)
					if val != "" {
						currentResult.DeltaPaths = append(currentResult.DeltaPaths, strings.Trim(val, `",`))
					}
				} else {
					val := strings.TrimSuffix(line, ",")
					val = strings.TrimSpace(val)
					if val != "" {
						currentResult.DeltaPaths = append(currentResult.DeltaPaths, strings.Trim(val, `"`))
					}
				}
				continue
			}

			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])

				if key == "delta_paths" {
					if val == "[]" {
						continue
					}
					inDeltaPaths = true
					if strings.HasPrefix(val, "[") {
						val = strings.TrimPrefix(val, "[")
						val = strings.TrimSpace(val)
						if val != "" && !strings.HasSuffix(val, "]") {
							currentResult.DeltaPaths = append(currentResult.DeltaPaths, strings.Trim(val, `",`))
						} else if strings.HasSuffix(val, "]") {
							val = strings.TrimSuffix(val, "]")
							val = strings.TrimSpace(val)
							if val != "" {
								currentResult.DeltaPaths = append(currentResult.DeltaPaths, strings.Trim(val, `",`))
							}
							inDeltaPaths = false
						}
					}
					continue
				}

				val = strings.Trim(val, `"'`)
				switch key {
				case "timestamp":
					if t, err := time.Parse(time.RFC3339, val); err == nil {
						currentResult.Timestamp = t
					}
				case "status":
					currentResult.Status = TaskState(val)
				case "error":
					currentResult.Error = val
				}
			}
		}
	}
	return results, scanner.Err()
}

