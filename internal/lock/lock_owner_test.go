package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockFileCarriesDiagnosticOwner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	unlock, err := acquire(path, defaultWait())
	require.NoError(t, err)
	unlock()

	// Read after release: on Windows an exclusive LockFileEx also blocks reads
	// by other handles, so inspecting the diagnostic nonce is only safe once
	// the descriptor is closed.
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.NotEmpty(t, strings.TrimSpace(string(data)), "owner nonce is written for diagnostics")
	require.Contains(t, string(data), "-")
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
