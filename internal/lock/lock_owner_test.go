package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLockFileCarriesItsOwner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, time.Second, 30*time.Second)
	require.NoError(t, err)
	defer unlock()

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	// The nonce is <pid>-<hex>; release() and the heartbeat compare against it,
	// so an empty file would make both of them blind.
	require.NotEmpty(t, strings.TrimSpace(string(data)), "owner nonce must be written")
	require.Contains(t, string(data), "-")
}

func TestUnlockLeavesAForeignLockAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, time.Second, 30*time.Second)
	require.NoError(t, err)

	// Simulate another process reclaiming the lock after it looked stale: the
	// file now carries a different owner.
	require.NoError(t, os.WriteFile(path, []byte("999999-cafebabe"), 0o600))
	unlock()

	_, statErr := os.Stat(path)
	require.NoError(t, statErr,
		"unlock must not delete a lock that belongs to another transaction")
}

func TestReleaseOfAnAlreadyRemovedLockIsQuiet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, time.Second, 30*time.Second)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	unlock() // must not panic or recreate the file
	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr))
}
