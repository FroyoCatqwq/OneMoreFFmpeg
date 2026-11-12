//go:build windows

package ffexec

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// configureProcess terminates the whole process tree. taskkill is preferred
// over Process.Kill because ffmpeg builds are sometimes launched through
// wrapper scripts that spawn a grandchild holding the output pipes.
func configureProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err == nil {
			return nil
		}
		return cmd.Process.Kill()
	}
}

// processSignal has no meaning on Windows: killed processes report a
// non-zero exit code instead of a WaitStatus signal.
func processSignal(*exec.ExitError) string {
	return ""
}
