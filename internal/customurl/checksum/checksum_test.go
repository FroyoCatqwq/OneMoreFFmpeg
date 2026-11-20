package checksum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const digestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestResolveURL(t *testing.T) {
	got, err := ResolveURL("https://example.com/ffmpeg/snapshots/20261120/ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/ffmpeg/snapshots/20261120/checksum.sha256", got)
	// root directory
	got, err = ResolveURL("http://host/a.zip")
	require.NoError(t, err)
	require.Equal(t, "http://host/checksum.sha256", got)
}

func TestParseExactAndCRLF(t *testing.T) {
	data := []byte("# comment\r\n" + digestA + "  ffmpeg-8.1.0-gpl-stable-linux-amd64-static.zip\r\n")
	got, err := Parse(data, "ffmpeg-8.1.0-gpl-stable-linux-amd64-static.zip")
	require.NoError(t, err)
	require.Equal(t, digestA, got)
}

func TestParseRejectsDuplicate(t *testing.T) {
	data := []byte(digestA + "  a.zip\n" + digestA + "  a.zip\n")
	_, err := Parse(data, "a.zip")
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestParseRejectsPathAndMissing(t *testing.T) {
	data := []byte(digestA + "  dist/a.zip\n")
	_, err := Parse(data, "dist/a.zip")
	require.Error(t, err, "path in file name must be rejected")
	_, err = Parse([]byte(digestA+"  other.zip\n"), "a.zip")
	require.Error(t, err)
}
