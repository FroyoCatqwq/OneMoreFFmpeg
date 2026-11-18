package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgressNonTTYUsesStageLines(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	p := NewProgress(&buf, "ffmpeg.zip", 100)
	p.Update(25)
	p.Update(50)
	p.Complete()
	out := buf.String()
	require.Contains(t, out, "ffmpeg.zip: 25%")
	require.Contains(t, out, "ffmpeg.zip: 50%")
	require.Contains(t, out, "ffmpeg.zip: completed")
	require.Contains(t, out, "100%")
	require.False(t, strings.ContainsAny(out, "\r\x1b"), "stage logs must not contain animation: %q", out)
}

func TestProgressFailNeverShowsComplete(t *testing.T) {
	t.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, "ffmpeg.zip", 100)
	p.Update(50) // real progress: half way
	p.Fail(errors.New("network read failed"))
	out := buf.String()
	require.Contains(t, out, "failed")
	require.Contains(t, out, "50%")
	require.NotContains(t, out, "100%")
	require.NotContains(t, out, "completed")
}

func TestProgressAbortNeverShowsComplete(t *testing.T) {
	t.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, "ffmpeg.zip", 100)
	p.Update(30)
	p.Abort()
	out := buf.String()
	require.Contains(t, out, "cancelled")
	require.NotContains(t, out, "100%")
	require.NotContains(t, out, "completed")
}

func TestProgressUnknownTotalHasNoFakePercent(t *testing.T) {
	t.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, "ffmpeg.zip", -1)
	p.Update(12345)
	p.Fail(errors.New("boom"))
	out := buf.String()
	require.NotContains(t, out, "%", "unknown total must never fabricate a percentage")
	require.Contains(t, out, "failed")
	require.Contains(t, out, "12")
}

func TestProgressTerminalIdempotent(t *testing.T) {
	t.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, "x", 100)
	p.Update(100)
	p.Complete()
	p.Complete()
	p.Fail(errors.New("late")) // ignored after terminal
	p.Abort()                  // ignored after terminal
	p.Close()
	require.Equal(t, 1, strings.Count(buf.String(), "completed"),
		"terminal states must be idempotent: %q", buf.String())
}

func TestProgressNilWriterIsNoop(t *testing.T) {
	p := NewProgress(nil, "x", 100)
	require.NotNil(t, p)
	p.Update(50)
	p.Complete()
	p.Fail(errors.New("ignored"))
	p.Abort()
	p.Close()
}

func TestInteractiveDetectsCI(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "xterm")
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	require.False(t, interactive(&buf), "CI must disable animation")
}

func TestInteractiveDetectsDumbTerm(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	require.False(t, interactive(&buf))
}
