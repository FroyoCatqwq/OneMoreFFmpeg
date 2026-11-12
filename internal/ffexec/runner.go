// Package ffexec runs FFmpeg-like executables with timeout and output limits.
package ffexec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const DefaultMaxOutputBytes = 8 << 20
const DefaultTimeout = 15 * time.Second
const DefaultWaitDelay = 2 * time.Second

type Runner struct {
	Timeout        time.Duration
	WaitDelay      time.Duration
	MaxOutputBytes int
	Env            []string
	UnsetEnv       []string
	CleanEnv       bool
}

type Result struct {
	Executable       string   `json:"executable"`
	Args             []string `json:"args"`
	Stdout           string   `json:"stdout"`
	Stderr           string   `json:"stderr"`
	StdoutSHA256     string   `json:"stdout_sha256"`
	StderrSHA256     string   `json:"stderr_sha256"`
	StdoutTotalBytes int64    `json:"stdout_total_bytes"`
	StderrTotalBytes int64    `json:"stderr_total_bytes"`
	ExitCode         int      `json:"exit_code"`
	Signal           string   `json:"signal,omitempty"`
	DurationMs       int64    `json:"duration_ms"`
	Truncated        bool     `json:"truncated"`
	StdoutTruncated  bool     `json:"stdout_truncated"`
	StderrTruncated  bool     `json:"stderr_truncated"`
	MaxOutputBytes   int      `json:"max_output_bytes"`
}

// ExitCodeNotStarted marks a process that never produced an exit status
// (timeout, cancellation, I/O failure or start failure).
const ExitCodeNotStarted = -1

func (r Result) Summary() string {
	return fmt.Sprintf(
		"exit=%d stdout=%s stderr=%s duration=%dms truncated=%t",
		r.ExitCode,
		humanize.Bytes(uint64(r.StdoutTotalBytes)),
		humanize.Bytes(uint64(r.StderrTotalBytes)),
		r.DurationMs,
		r.Truncated,
	)
}

type limitedBuffer struct {
	buf       bytes.Buffer
	max       int
	total     int64
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.total += int64(len(p))
	if b.max <= 0 {
		b.max = DefaultMaxOutputBytes
	}
	remaining := b.max - b.buf.Len()
	if remaining <= 0 {
		if len(p) > 0 {
			b.truncated = true
		}
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func envKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func buildEnv(base, overrides, unset []string, clean bool) []string {
	entries := map[string]string{}
	if !clean {
		for _, pair := range base {
			key, value, ok := strings.Cut(pair, "=")
			// Windows drive-relative entries look like "=C:=C:\dir" and have
			// an empty key. They cannot be rebuilt unambiguously, so drop them
			// instead of collapsing several drives into one entry.
			if !ok || key == "" {
				continue
			}
			entries[envKey(key)] = key + "=" + value
		}
	}
	for _, pair := range overrides {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			continue
		}
		entries[envKey(key)] = key + "=" + value
	}
	for _, key := range unset {
		delete(entries, envKey(strings.TrimSpace(key)))
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, entries[key])
	}
	return out
}

func (r Runner) environment() []string {
	if !r.CleanEnv && len(r.Env) == 0 && len(r.UnsetEnv) == 0 {
		return nil
	}
	return buildEnv(os.Environ(), r.Env, r.UnsetEnv, r.CleanEnv)
}

func (r Runner) Run(ctx context.Context, executable string, args ...string) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(executable) == "" {
		return Result{}, apperrors.New(apperrors.E1002, "executable path is required")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	waitDelay := r.WaitDelay
	if waitDelay <= 0 {
		waitDelay = DefaultWaitDelay
	}
	maxOutput := r.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = DefaultMaxOutputBytes
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, executable, args...)
	configureProcess(cmd)
	cmd.WaitDelay = waitDelay
	cmd.Env = r.environment()

	stdout := &limitedBuffer{max: maxOutput}
	stderr := &limitedBuffer{max: maxOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := r.result(executable, args, stdout, stderr, duration, maxOutput)
	if stderrors.Is(ctx.Err(), context.Canceled) {
		result.ExitCode = ExitCodeNotStarted
		return result, apperrors.New(apperrors.E9004, "context canceled")
	}
	if stderrors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.ExitCode = ExitCodeNotStarted
		return result, apperrors.New(apperrors.E1006, "process timeout").WithCause(runCtx.Err().Error())
	}
	if stderrors.Is(err, exec.ErrWaitDelay) {
		result.ExitCode = ExitCodeNotStarted
		return result, apperrors.New(apperrors.E1006, "process I/O did not complete").WithCause(err.Error())
	}
	if err != nil {
		var exitErr *exec.ExitError
		if stderrors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			if result.ExitCode < 0 {
				result.ExitCode = ExitCodeNotStarted
				result.Signal = processSignal(exitErr)
			}
			return result, nil
		}
		result.ExitCode = ExitCodeNotStarted
		if stderrors.Is(err, os.ErrPermission) {
			return result, apperrors.New(apperrors.E1005, "process is not executable").WithCause(err.Error())
		}
		return result, apperrors.New(apperrors.E1012, "process could not be started").WithCause(err.Error())
	}
	return result, nil
}

func (r Runner) result(executable string, args []string, stdout, stderr *limitedBuffer, duration time.Duration, maxOutput int) Result {
	stdoutBytes := stdout.Bytes()
	stderrBytes := stderr.Bytes()
	return Result{
		Executable:       executable,
		Args:             append([]string(nil), args...),
		Stdout:           stdout.String(),
		Stderr:           stderr.String(),
		StdoutSHA256:     sha256Hex(stdoutBytes),
		StderrSHA256:     sha256Hex(stderrBytes),
		StdoutTotalBytes: stdout.total,
		StderrTotalBytes: stderr.total,
		DurationMs:       duration.Milliseconds(),
		Truncated:        stdout.truncated || stderr.truncated,
		StdoutTruncated:  stdout.truncated,
		StderrTruncated:  stderr.truncated,
		MaxOutputBytes:   maxOutput,
	}
}
