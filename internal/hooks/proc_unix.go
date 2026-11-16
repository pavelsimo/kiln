//go:build unix

package hooks

import (
	"os/exec"
	"syscall"
)

// ownGroup gives the hook its own process group, so a timeout kills the shell
// and everything it started, not only the shell.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
