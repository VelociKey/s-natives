//go:build windows

package lifecycle

import (
	"os/exec"
	"syscall"
)

// SetNoWindow configures the process attributes to hide the console window.
func SetNoWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
