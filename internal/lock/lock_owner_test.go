package lock

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockFileIsNeverWritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, defaultWait())
	require.NoError(t, err)
	unlock()

	// The lock file's identity is the OS lock; its contents must stay empty.
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Empty(t, data)
}

func TestLockPathSymlinkIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("keep-me"), 0o644))
	link := filepath.Join(dir, fileName)
	require.NoError(t, os.Symlink(victim, link))

	_, err := acquire(link, defaultWait())
	require.Error(t, err)
	require.Contains(t, err.Error(), "symbolic link")
	// The victim file must be untouched.
	data, readErr := os.ReadFile(victim)
	require.NoError(t, readErr)
	require.Equal(t, "keep-me", string(data))
}

func TestReleaseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, defaultWait())
	require.NoError(t, err)
	unlock()
	unlock() // second call must be a no-op, not panic or affect another holder

	reacquire, err := acquire(path, defaultWait())
	require.NoError(t, err, "lock must be re-acquirable after a double release")
	reacquire()
}
