//go:build unix

package shell

import (
	"os/exec"
	"syscall"
	"time"
)

// stopWithChildren runs the command in its own process group, and stops the entire group when the command is
// cancelled. exec.CommandContext only kills the shell, anything the shell started would keep running (and keep
// the output pipes open, blocking the command from ever returning).
func stopWithChildren(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// negative pid signals the process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	// anything that escaped the group and still holds the output open can't block the command forever.
	cmd.WaitDelay = 5 * time.Second
}
