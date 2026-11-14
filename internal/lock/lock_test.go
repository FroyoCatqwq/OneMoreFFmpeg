package lock

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func TestDataRootAcquireRelease(t *testing.T) {
	root := t.TempDir()
	unlock, err := DataRoot(root)
	require.NoError(t, err)
	path := filepath.Join(root, fileName)
	_, err = os.Stat(path)
	require.NoError(t, err, "lock file must exist while held")
	unlock()
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err), "lock file must be removed on release")
}

func TestAcquireContentionFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(path, []byte("held"), 0o600))

	_, err := acquire(path, 120*time.Millisecond, 30*time.Second)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E6009, typed.Code)
}

func TestAcquireReclaimsStaleLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(path, []byte("abandoned"), 0o600))
	old := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(path, old, old))

	unlock, err := acquire(path, time.Second, 30*time.Second)
	require.NoError(t, err, "an abandoned lock must be reclaimed")
	unlock()
}

func TestAcquireWaitsForRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(path, []byte("held"), 0o600))
	go func() {
		time.Sleep(80 * time.Millisecond)
		_ = os.Remove(path)
	}()

	unlock, err := acquire(path, 2*time.Second, 30*time.Second)
	require.NoError(t, err, "acquire must succeed once the holder releases")
	unlock()
}