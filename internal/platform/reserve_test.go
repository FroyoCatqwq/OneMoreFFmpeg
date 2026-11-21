package platform_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

func TestReserveLinkRejectsInvalidKind(t *testing.T) {
	_, _, err := platform.ReserveLink(t.TempDir(), "x-*", "target", 0)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}

func TestReserveLinkFileTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))

	path, release, err := platform.ReserveLink(dir, "link-*", target, platform.LinkTargetFile)
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "x", string(body))
	require.NoError(t, release())
	_, err = os.Lstat(path)
	require.True(t, os.IsNotExist(err))
}

func TestReserveLinkDanglingFileOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file symlinks to missing targets are allowed but this test checks Unix semantics")
	}
	dir := t.TempDir()
	path, release, err := platform.ReserveLink(dir, "dangling-*", filepath.Join(dir, "missing"), platform.LinkTargetFile)
	require.NoError(t, err)
	info, err := os.Lstat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink)
	require.NoError(t, release())
}

func TestReserveLinkOwnershipChangeIsRejected(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	path, release, err := platform.ReserveLink(dir, "own-*", target, platform.LinkTargetFile)
	require.NoError(t, err)

	// Replace the reserved link with a plain file.
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, []byte("foreign"), 0o600))
	err = release()
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E5009, typed.Code)
	require.Equal(t, "ownership_changed", typed.Details["reason"])
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "foreign", string(body))
}

func TestReserveLinkCommittedPathIsNotRemoved(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	path, release, err := platform.ReserveLink(dir, "commit-*", target, platform.LinkTargetFile)
	require.NoError(t, err)
	committed := filepath.Join(dir, "current")
	require.NoError(t, os.Rename(path, committed))
	require.NoError(t, release())
	_, err = os.Lstat(committed)
	require.NoError(t, err)
}

func TestReserveLinkDirTargetOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows directory-link semantics")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target-dir")
	require.NoError(t, os.Mkdir(target, 0o755))

	path, release, err := platform.ReserveLink(dir, "dirlink-*", target, platform.LinkTargetDir)
	require.NoError(t, err)
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	require.Empty(t, entries)
	// Must use os.Remove, not RemoveAll: the target directory still exists.
	require.NoError(t, release())
	_, err = os.Stat(target)
	require.NoError(t, err, "removing the link must not delete its target")

	// Missing directory target must be rejected (no dangling dir links).
	_, _, err = platform.ReserveLink(dir, "missing-*", filepath.Join(dir, "nope"), platform.LinkTargetDir)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E5009, typed.Code)
}

func TestReserveLinkWindowsFileTargetDirRejected(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows link-type semantics")
	}
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "target-dir")
	require.NoError(t, os.Mkdir(targetDir, 0o755))
	_, _, err := platform.ReserveLink(dir, "mismatch-*", targetDir, platform.LinkTargetFile)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, "link_type_mismatch", typed.Details["reason"])
}
