# SACP Coordination Daemon Implementation

This document specifies the Go-level interfaces and configuration controls for daemon connection management.

## 1. Stale Socket Cleaning

When dialing a socket file, if dialing fails with `syscall.ECONNREFUSED` or `WSAECONNREFUSED` (on Windows), the CLI automatically removes the file:

```go
func DialOrResetSocket(socketPath string) (net.Conn, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		// Clean up stale socket file if it exists but is dead
		if _, statErr := os.Stat(socketPath); statErr == nil {
			_ = os.Remove(socketPath)
		}
		return nil, err
	}
	return conn, nil
}
```

## 2. Inactivity Watchdog

A `time.Timer` maintains the active state. It defaults to **15 minutes** unless overridden via the `--idle-timeout` parameter. Every incoming packet or stream initiation calls `.Reset(timeout)` on the timer. If it expires, it closes the socket listener and exits.

## 3. Delta File Synchronization (Host-Guest)

To support the `--repo` file transfer efficiently, SACP syncs only modified files by hashing and comparing local workspace timestamps against the remote index:
1. **Delta Calculation**: Walks the local workspace using `filepath.Walk` and collects files modified since the last sync.
2. **Chunk Streaming**: Opens a dedicated SACP data channel, streams the delta files inside a lightweight archive payload, and updates the guest file index.
