package ffexec

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildEnvDropsDriveRelativeAndInvalidEntries(t *testing.T) {
	env := buildEnv(
		[]string{`=C:=C:\work`, "PATH=C:\\bin", "NOEQUALS"},
		[]string{"OMF_TEST=1"},
		nil,
		false,
	)
	require.Contains(t, env, "PATH=C:\\bin")
	require.Contains(t, env, "OMF_TEST=1")
	for _, entry := range env {
		require.NotContains(t, entry, `=C:=`)
		require.NotEqual(t, "NOEQUALS", entry)
	}
}

func TestResultTracksPerStreamTruncation(t *testing.T) {
	stdout := &limitedBuffer{max: 4}
	stderr := &limitedBuffer{max: 4}
	_, err := stdout.Write([]byte("123456"))
	require.NoError(t, err)
	_, err = stderr.Write([]byte("ok"))
	require.NoError(t, err)

	result := Runner{}.result("ffmpeg", nil, stdout, stderr, 10*time.Millisecond, 4)
	require.True(t, result.Truncated)
	require.True(t, result.StdoutTruncated)
	require.False(t, result.StderrTruncated)
	require.Equal(t, int64(6), result.StdoutTotalBytes)
}

func TestResultWithoutTruncation(t *testing.T) {
	stdout := &limitedBuffer{max: 64}
	stderr := &limitedBuffer{max: 64}
	_, _ = stdout.Write([]byte("ok"))
	_, _ = stderr.Write([]byte("warn"))

	result := Runner{}.result("ffmpeg", nil, stdout, stderr, 0, 64)
	require.False(t, result.Truncated)
	require.False(t, result.StdoutTruncated)
	require.False(t, result.StderrTruncated)
}
