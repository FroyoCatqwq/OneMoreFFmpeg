package env

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollbackTargetsMustStayInsideTheDataRoot(t *testing.T) {
	root := t.TempDir()
	require.True(t, safeRollbackPath(filepath.Join(root, "bin"), root))
	require.True(t, safeRollbackPath(filepath.Join(root, "bin", "ffmpeg"), root))

	// A journal is an ordinary file: a crafted one must not be able to point a
	// rollback at an arbitrary directory.
	require.False(t, safeRollbackPath(filepath.Join(root, "..", "escape"), root))
	require.False(t, safeRollbackPath("/etc", root))
	require.False(t, safeRollbackPath("relative/path", root))
	require.False(t, safeRollbackPath("", root))
	require.False(t, safeRollbackPath("  ", root))
}

func TestOnlyTheKnownShellProfilesAreAccepted(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	require.True(t, safeShellProfile(filepath.Join(home, ".profile")))
	// The system profile only exists on Linux; on Windows the path is not even
	// absolute, which is exactly why it must not be accepted there.
	if runtime.GOOS != "windows" {
		require.True(t, safeShellProfile("/etc/profile.d/onemoreffmpeg.sh"))
	}
	require.False(t, safeShellProfile("/etc/profile.d/onemoreffmpeg.sh#evil"))
	require.False(t, safeShellProfile(filepath.Join(home, ".bashrc")))
	require.False(t, safeShellProfile("/etc/passwd"))
	require.False(t, safeShellProfile(""))

	root := t.TempDir()
	require.True(t, safeRollbackPath(filepath.Join(home, ".profile"), root))
}

func TestOnlyTheKnownShimDirsAreAccepted(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	require.True(t, safeShimDir(filepath.Join(home, ".local", "bin")))
	if runtime.GOOS != "windows" {
		require.True(t, safeShimDir("/usr/local/bin"))
	}
	// Shim dirs live outside the data root; arbitrary directories must still be
	// rejected so a crafted journal cannot redirect a rollback.
	require.False(t, safeShimDir(filepath.Join(home, ".local", "bin", "evil")))
	require.False(t, safeShimDir("/etc"))
	require.False(t, safeShimDir("/usr/local/bin/evil"))
	require.False(t, safeShimDir("relative/bin"))
	require.False(t, safeShimDir(""))
}
