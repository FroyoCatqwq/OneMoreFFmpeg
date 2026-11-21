package fsatomic_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/fsatomic"
)

func codeOf(t *testing.T, err error) apperrors.Code {
	t.Helper()
	typed, ok := apperrors.As(err)
	require.True(t, ok, "expected typed error, got %v", err)
	return typed.Code
}

func TestWriteFileReplacesExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	require.NoError(t, fsatomic.WriteFile(path, []byte("new"), 0o644))
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new", string(body))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.False(t, filepath.Ext(e.Name()) == ".tmp", "temp file must be cleaned: %s", e.Name())
	}
}

func TestWriteFileModeZeroKeepsTempMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.lock")
	require.NoError(t, fsatomic.WriteFile(path, []byte("data"), 0))
	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
			"mode=0 must keep os.CreateTemp's 0600 default")
	}
}

func TestWriteLockedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "journal.json")
	require.NoError(t, fsatomic.WriteLockedFile(path, []byte(`{"ok":true}`), 0o600))
	require.NoError(t, fsatomic.WriteLockedFile(path, []byte(`{"ok":false}`), 0o600))
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `{"ok":false}`, string(body))
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestReserveFileLifecycle(t *testing.T) {
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "reserve-*.tmp")
	require.NoError(t, err)
	require.NotNil(t, f)
	require.Equal(t, filepath.Dir(path), dir)

	_, err = f.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, f.Sync())
	require.NoError(t, f.Close())
	require.NoError(t, release())

	_, err = os.Lstat(path)
	require.True(t, os.IsNotExist(err), "release must remove an unreferenced reservation")
}

func TestReserveFileReleaseClosesOpenHandle(t *testing.T) {
	// On Windows an open handle prevents os.Remove; release must own the
	// handle and close it before removing even if the caller forgot.
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "open-*.tmp")
	require.NoError(t, err)
	_, err = f.Write([]byte("data"))
	require.NoError(t, err)
	// intentionally do NOT close f
	require.NoError(t, release())
	_, err = os.Lstat(path)
	require.True(t, os.IsNotExist(err))
	_, err = f.Write([]byte("more"))
	require.Error(t, err, "release must have closed the handle")
}

func TestReserveFileCommittedPathIsNotRemoved(t *testing.T) {
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "commit-*.tmp")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Simulate a successful commit: caller renames the reservation away.
	committed := filepath.Join(dir, "committed.bin")
	require.NoError(t, os.Rename(path, committed))
	require.NoError(t, release(), "path is gone -> ownership check returns nil")
	_, err = os.Stat(committed)
	require.NoError(t, err)
}

func TestReserveFileOwnershipChangeIsRejected(t *testing.T) {
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "replace-*.tmp")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, []byte("foreign"), 0o600))

	err = release()
	require.Error(t, err)
	require.Equal(t, apperrors.E5009, codeOf(t, err))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, "ownership_changed", typed.Details["reason"])
	body, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, "foreign", string(body), "foreign resource must not be deleted")
}

func TestReserveReleaseCachesFirstResult(t *testing.T) {
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "cache-*.tmp")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, release())

	// Recreate a foreign file at the same path: a second release must still
	// return the cached nil and must not touch the filesystem.
	require.NoError(t, os.WriteFile(path, []byte("foreign"), 0o600))
	require.NoError(t, release())
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "foreign", string(body))
}

func TestReserveReleaseCachesFirstFailure(t *testing.T) {
	dir := t.TempDir()
	path, f, release, err := fsatomic.ReserveFile(dir, "fail-*.tmp")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, []byte("foreign"), 0o600))

	first := release()
	require.Error(t, first)
	second := release()
	require.Error(t, second)
	require.True(t, errors.Is(second, first) || second.Error() == first.Error(),
		"second release must return the cached first error, not silently succeed")
}
