package lock

import (
	"os"
	"path/filepath"
	"sync/atomic"
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
	// The lock file is intentionally kept (its identity is the lock's identity);
	// a fresh acquire must still succeed after release.
	again, err := DataRoot(root)
	require.NoError(t, err, "lock must be re-acquirable after release")
	again()
}

func TestAcquireContentionFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, defaultWait())
	require.NoError(t, err)

	_, err = acquire(path, 120*time.Millisecond)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E6009, typed.Code)
	unlock()
}

func TestAcquireWaitsForRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, defaultWait())
	require.NoError(t, err)

	var acquired atomic.Bool
	go func() {
		time.Sleep(80 * time.Millisecond)
		acquired.Store(true)
		unlock()
	}()

	got, err := acquire(path, 2*time.Second)
	require.NoError(t, err, "acquire must succeed once the holder releases")
	require.True(t, acquired.Load())
	got()
}
