//go:build unix

package releaseops

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureGitCancellation(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return nil
}
