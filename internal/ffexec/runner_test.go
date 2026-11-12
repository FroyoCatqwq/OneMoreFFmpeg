package ffexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func TestRunnerSuccess(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{Env: fake.Env("hello\n", "warn\n", 0)}
	result, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)
	require.Contains(t, result.Stdout, "hello")
	require.Contains(t, result.Stderr, "warn")
	require.Len(t, result.StdoutSHA256, 64)
	require.Len(t, result.StderrSHA256, 64)
	require.Contains(t, result.Summary(), "exit=0")
}

func TestRunnerNonZeroExitIsResult(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{Env: fake.Env("", "bad\n", 3)}
	result, err := runner.Run(context.Background(), fake.FFprobePath, "-version")
	require.NoError(t, err)
	require.Equal(t, 3, result.ExitCode)
	require.Contains(t, result.Stderr, "bad")
}

func TestRunnerTimeoutReturnsPartialOutput(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		Timeout:   50 * time.Millisecond,
		WaitDelay: 100 * time.Millisecond,
		Env: append(fake.Env("before-timeout\n", "", 0),
			"OMF_FAKE_SLEEP_MS=500"),
	}
	result, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E1006, typed.Code)
	require.Contains(t, result.Stdout, "before-timeout")
	require.Equal(t, ffexec.ExitCodeNotStarted, result.ExitCode)
	require.Equal(t, -1, result.ExitCode)
}

func TestRunnerGrandchildPipeDoesNotHang(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		Timeout:   50 * time.Millisecond,
		WaitDelay: 100 * time.Millisecond,
		Env: append(fake.Env("", "", 0),
			"OMF_FAKE_GRANDCHILD_MS=500"),
	}
	start := time.Now()
	_, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	require.Error(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E1006, typed.Code)
}

func TestRunnerContextCanceled(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	runner := ffexec.Runner{
		Timeout:   5 * time.Second,
		WaitDelay: 100 * time.Millisecond,
		Env: append(fake.Env("", "", 0),
			"OMF_FAKE_SLEEP_MS=500"),
	}
	_, err := runner.Run(ctx, fake.FFmpegPath, "-version")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E9004, typed.Code)
}

func TestRunnerMissingExecutable(t *testing.T) {
	runner := ffexec.Runner{}
	result, err := runner.Run(context.Background(), "does-not-exist-omf")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E1012, typed.Code)
	require.Equal(t, ffexec.ExitCodeNotStarted, result.ExitCode)
}

func TestRunnerEmptyExecutable(t *testing.T) {
	runner := ffexec.Runner{}
	_, err := runner.Run(context.Background(), "")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E1002, typed.Code)
}

func TestRunnerOutputLimit(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		MaxOutputBytes: 16,
		Env:            append(fake.Env("", "", 0), "OMF_FAKE_BYTES=100"),
	}
	result, err := runner.Run(context.Background(), fake.FFplayPath, "-version")
	require.NoError(t, err)
	require.True(t, result.Truncated)
	require.Len(t, result.Stdout, 16)
	require.Equal(t, int64(100), result.StdoutTotalBytes)
}

func TestRunnerEnvOverride(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		Env: []string{
			"OMF_FAKE_ENV=OMF_TEST_VALUE",
			"OMF_TEST_VALUE=from-env",
		},
	}
	result, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	require.NoError(t, err)
	require.Equal(t, "from-env", result.Stdout)
}

func TestRunnerCleanEnv(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		CleanEnv: true,
		Env: []string{
			"OMF_FAKE_ENV=OMF_TEST_VALUE",
			"OMF_TEST_VALUE=clean-env",
		},
	}
	result, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	require.NoError(t, err)
	require.Equal(t, "clean-env", result.Stdout)
}

func TestRunnerUnsetEnv(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	runner := ffexec.Runner{
		Env: []string{
			"OMF_FAKE_ENV=OMF_TEST_VALUE",
			"OMF_TEST_VALUE=should-be-unset",
		},
		UnsetEnv: []string{"OMF_TEST_VALUE"},
	}
	result, err := runner.Run(context.Background(), fake.FFmpegPath, "-version")
	require.NoError(t, err)
	require.Empty(t, result.Stdout)
}
