package archive_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/archive"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// Malicious entry names must be rejected end-to-end for every archive format,
// not just by the unit-level path guard.
func TestExtractRejectsUnsafeEntryNames(t *testing.T) {
	names := []string{
		"ffmpeg.exe:evil",   // NTFS alternate data stream
		"ffmpeg.exe::$DATA", // NTFS ADS
		"NUL",               // reserved device name
		"bin/ffmpeg.",       // trailing dot alias
	}
	for _, name := range names {
		t.Run("zip/"+name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "a.zip")
			writeZip(t, path, map[string]string{name: "payload"})
			assertRejected(t, path, dir)
		})
		t.Run("tarxz/"+name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "a.tar.xz")
			writeTarXZ(t, path, map[string]string{name: "payload"})
			assertRejected(t, path, dir)
		})
	}
}

func assertRejected(t *testing.T, archivePath, dir string) {
	t.Helper()
	dest := filepath.Join(dir, "out")
	require.NoError(t, os.Mkdir(dest, 0o755))
	err := archive.Extract(context.Background(), archivePath, dest)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E5002, typed.Code)
	entries, readErr := os.ReadDir(dest)
	require.NoError(t, readErr)
	require.Empty(t, entries, "nothing may be written when an entry is rejected")
}