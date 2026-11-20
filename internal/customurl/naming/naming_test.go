package naming

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBuildRoundTrip(t *testing.T) {
	for _, name := range []string{
		"ffmpeg-8.1.0-lgpl-stable-linux-amd64-static.tar.xz",
		"ffmpeg-20261120-gpl-snapshot-windows-amd64-shared.zip",
		"ffmpeg-20261120T153000Z-nonfree-snapshot-linux-amd64-static.tar.zst",
	} {
		a, err := Parse(name)
		require.NoError(t, err, name)
		built, err := Build(a)
		require.NoError(t, err)
		require.Equal(t, name, built)
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, name := range []string{
		"ffmpeg-8.1-lgpl-stable-linux-amd64-static.tar.xz",  // stable needs >=3 parts
		"ffmpeg-v8.1.0-gpl-stable-linux-amd64-static.zip",   // v prefix
		"ffmpeg-8.1.0-bsd-stable-linux-amd64-static.zip",    // bsd not allowed
		"ffmpeg-8.1.0-gpl-beta-linux-amd64-static.zip",      // bad release type
		"ffmpeg-8.1.0-gpl-stable-windows-x86_64-static.zip", // arch alias
		"ffmpeg-8.1.0-gpl-stable-linux-amd64-static.7z",     // no 7z
		"FFmpeg-8.1.0-gpl-stable-linux-amd64-static.zip",    // uppercase
		"ffmpeg-8.1.0-gpl-stable-linux-amd64.zip",           // missing field
	} {
		_, err := Parse(name)
		require.Error(t, err, name)
	}
}

func TestSnapshotRules(t *testing.T) {
	_, err := Parse("ffmpeg-20261120T153000Z-gpl-snapshot-windows-amd64-shared.zip")
	require.NoError(t, err)
	_, err = Parse("ffmpeg-2026-gpl-snapshot-linux-amd64-static.zip")
	require.Error(t, err)
}

func TestFormatOf(t *testing.T) {
	require.Equal(t, "tar.gz", FormatOf("a.tar.gz"))
	require.Equal(t, "tar.zst", FormatOf("a.tar.zst"))
	require.Equal(t, "zip", FormatOf("a.zip"))
	require.Empty(t, FormatOf("a.7z"))
}
