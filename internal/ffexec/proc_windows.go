//go:build windows

package ffexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
		// Resolve taskkill from System32, never from a caller-controlled PATH:
		// a hijacked PATH could otherwise execute an arbitrary program during
		// the cancellation phase. If SystemRoot is unavailable, skip taskkill
		// entirely and fall back to Process.Kill.
		root := os.Getenv("SystemRoot")
		if root != "" {
			taskkill := filepath.Join(root, "System32", "taskkill.exe")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			kill := exec.CommandContext(ctx, taskkill, "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
			kill.Env = []string{"SystemRoot=" + root, "PATH=" + filepath.Join(root, "System32")}
			if err := kill.Run(); err == nil {
				return nil
			}
		}
		return cmd.Process.Kill()
	}
}

// processSignal has no meaning on Windows: killed processes report a
// non-zero exit code instead of a WaitStatus signal.
func processSignal(*exec.ExitError) string {
	return ""
}
