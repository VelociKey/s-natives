//go:build !windows

package lifecycle

import (
	"os/exec"
)

// SetNoWindow is a no-op on non-Windows platforms.
func SetNoWindow(cmd *exec.Cmd) {
	// No-op
}
