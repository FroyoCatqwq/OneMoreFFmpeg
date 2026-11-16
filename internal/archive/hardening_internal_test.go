package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
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
		"file.txt:ads",       // NTFS alternate data stream
		"file.txt::$DATA",    // NTFS ADS
		"dir:name/child.txt", // colon in a component
		"nul",                // reserved device
		"NUL.txt",            // reserved with extension
		"con", "prn", "aux",
		"COM1", "lpt9.log",
		"trailing.",       // Windows strips trailing dot
		"trailing ",       // Windows strips trailing space
		"foo/./bar.txt",   // dot component
		"foo//bar.txt",    // empty component
		"bad\x00name.txt", // NUL
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

func TestWriteFileStopsAtExpansionCap(t *testing.T) {
	target := filepath.Join(t.TempDir(), "entry.bin")
	// A small compressed size yields a small streaming budget; the writer must
	// fail well before the old 2 GiB ceiling and leave no partial file.
	_, err := writeFile(target, bytes.NewReader(bytes.Repeat([]byte("A"), 4096)), 0o644, 1024)
	require.Error(t, err)
	_, statErr := os.Stat(target)
	require.True(t, os.IsNotExist(statErr), "over-budget entry must be removed")
}

func TestExtractionCapSafety(t *testing.T) {
	require.Equal(t, int64(1000), expansionCap(1))
	require.Equal(t, int64(maxFileSize), expansionCap(maxFileSize))
	require.Equal(t, int64(maxFileSize), expansionCap(0), "unknown size is conservative")
}

func TestGlobalExpansionCap(t *testing.T) {
	require.Equal(t, int64(1000), globalExpansionCap(1))
	require.Equal(t, int64(maxTotal), globalExpansionCap(maxTotal), "unknown/huge size saturates at maxTotal")
	require.Equal(t, int64(maxTotal), globalExpansionCap(0))
	require.Equal(t, int64(maxTotal), globalExpansionCap(1<<40), "no overflow wrap")
}

func TestZipHighCompressionStillExtracts(t *testing.T) {
	// Deflate tops out near 1032:1, so a real archive cannot easily exceed the
	// 1000x ceiling. Verify the guard does not reject legitimate highly
	// compressible content: declared total must stay under globalExpansionCap.
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "data.zip")
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	entry := bytes.Repeat([]byte("A"), 64*1024)
	for i := 0; i < 8; i++ {
		w, err := zw.Create(fmt.Sprintf("entry%d.bin", i))
		require.NoError(t, err)
		_, err = w.Write(entry)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	dest := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, Extract(context.Background(), archivePath, dest))
	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	require.Len(t, entries, 8)
}

func TestDeclaredTotalUsesGlobalCap(t *testing.T) {
	// The zip pre-check must compare declared expansion against the ratio cap,
	// not only the absolute maxTotal. A 1 KiB archive may declare at most
	// 1 KiB * maxRatio, never 8 GiB.
	compressed := int64(1024)
	require.Equal(t, int64(1024*maxRatio), globalExpansionCap(compressed))
	require.Less(t, globalExpansionCap(compressed), int64(maxTotal))
	require.True(t, ratioExceeded(globalExpansionCap(compressed)+1, compressed))
}

func TestRatioExceeded(t *testing.T) {
	require.True(t, ratioExceeded(2000, 1))
	require.True(t, ratioExceeded(1000001, 1000))
	require.False(t, ratioExceeded(1000, 1))
	require.False(t, ratioExceeded(500, 1))
	require.False(t, ratioExceeded(1<<40, 0), "unknown compressed size must not trip the guard")
}
