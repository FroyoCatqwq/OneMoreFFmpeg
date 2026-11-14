package archive

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafePathRejectsUnsafeNames(t *testing.T) {
	dest := t.TempDir()
	unsafe := []string{
		"",
		"../escape.txt",
		"foo/../../escape.txt",
		"/absolute.txt",
		"//unc/share.txt",
		"C:/drive.txt",
		"C:\\drive.txt",
		"file.txt:ads",          // NTFS alternate data stream
		"file.txt::$DATA",       // NTFS ADS
		"dir:name/child.txt",    // colon in a component
		"nul",                   // reserved device
		"NUL.txt",               // reserved with extension
		"con", "prn", "aux",
		"COM1", "lpt9.log",
		"trailing.",             // Windows strips trailing dot
		"trailing ",             // Windows strips trailing space
		"foo/./bar.txt",         // dot component
		"foo//bar.txt",          // empty component
		"bad\x00name.txt",       // NUL
	}
	for _, name := range unsafe {
		t.Run(name, func(t *testing.T) {
			_, err := safePath(dest, name)
			require.Error(t, err, "name %q must be rejected", name)
		})
	}
}

func TestSafePathAcceptsNormalNames(t *testing.T) {
	dest := t.TempDir()
	for _, name := range []string{"bin/ffmpeg.exe", "LICENSE.txt", "doc/ffplay.html", "presets/", "README"} {
		target, err := safePath(dest, name)
		require.NoError(t, err, "name %q must be accepted", name)
		require.Contains(t, target, filepath.Base(dest))
	}
}

func TestIsWindowsReservedName(t *testing.T) {
	for _, name := range []string{"NUL", "nul.txt", "CON", "aux.log", "COM1", "LPT9", "com3.tar"} {
		require.True(t, isWindowsReservedName(name), name)
	}
	for _, name := range []string{"ffmpeg", "README.txt", "COM0", "LPT0", "COM10", "console", "null"} {
		require.False(t, isWindowsReservedName(name), name)
	}
}

func TestRatioExceeded(t *testing.T) {
	require.True(t, ratioExceeded(2000, 1))
	require.True(t, ratioExceeded(1000001, 1000))
	require.False(t, ratioExceeded(1000, 1))
	require.False(t, ratioExceeded(500, 1))
	require.False(t, ratioExceeded(1<<40, 0), "unknown compressed size must not trip the guard")
}