package output

import (
	"bytes"
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
	p.Finish()
	out := buf.String()
	require.Contains(t, out, "ffmpeg.zip: 25%")
	require.Contains(t, out, "ffmpeg.zip: 50%")
	require.Contains(t, out, "ffmpeg.zip: 100%")
	// No ANSI escape / carriage-return animation in non-interactive mode.
	require.False(t, strings.ContainsAny(out, "\r\x1b"), "stage logs must not contain animation: %q", out)
}

func TestProgressNilWriterIsNoop(t *testing.T) {
	p := NewProgress(nil, "x", 100)
	require.NotNil(t, p)
	p.Update(50)
	p.Finish() // must not panic
}

func TestInteractiveDetectsCI(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "xterm")
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	require.False(t, interactive(&buf), "CI must disable animation even for a TTY-like writer")
}

func TestInteractiveDetectsDumbTerm(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	require.False(t, interactive(&buf))
}
