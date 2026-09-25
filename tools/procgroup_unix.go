//go:build unix

package tools

import (
	"os/exec"
	"syscall"
)

// a check that wraps another process would leave it running after a timeout, so the whole group is killed
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
