package doctor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasVersionBanner(t *testing.T) {
	require.True(t, hasVersionBanner("ffmpeg version 9.9.9 Copyright (c) 2026\nconfiguration: ...", "ffmpeg"))
	require.True(t, hasVersionBanner("\n  ffprobe version N-123\n", "ffprobe"))
	require.True(t, hasVersionBanner("FFPLAY version 8.1\n", "ffplay"), "case-insensitive")
	// Exit-0 stubs without a version banner must not count as healthy.
	require.False(t, hasVersionBanner("", "ffmpeg"))
	require.False(t, hasVersionBanner("OK\n", "ffmpeg"))
	require.False(t, hasVersionBanner("some other tool 1.0\n", "ffmpeg"))
}
