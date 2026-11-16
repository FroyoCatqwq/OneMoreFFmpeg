// Package lock serializes data-root write transactions across processes.
package lock

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	fileName     = ".transaction.lock"
	waitTimeout  = 5 * time.Second
	pollInterval = 50 * time.Millisecond
)

// DataRoot acquires the global data-root transaction lock so install, env and
// uninstall writers cannot interleave. The returned function releases it.
//
// The mutex is an OS advisory lock (flock on Unix, LockFileEx on Windows) bound
// to an open file descriptor. The kernel drops it when the descriptor closes,
// including when the process crashes -- there is no mtime "stale" heuristic and
// therefore no reclaim-vs-acquire TOCTOU. The lock file itself is intentionally
// never deleted: deleting and recreating a lock file changes its identity and
// would let two waiters hold locks on different inodes at the same time.
func DataRoot(dataRoot string) (func(), error) {
	return acquire(filepath.Join(dataRoot, fileName), waitTimeout)
}

// FilePath takes the same OS-level advisory lock on an arbitrary lock file,
// for configuration files (e.g. providers.toml) that need cross-process
// serialization. The returned function releases it.
func FilePath(lockFilePath string) (func(), error) {
	return acquire(lockFilePath, waitTimeout)
}

// acquire opens (or creates) the lock file and takes the OS lock. wait bounds
// how long to poll when another process holds it. The stale parameter is kept
// out of the signature on purpose; callers previously passed a TTL which no
// longer applies.
func acquire(path string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock directory").WithCause(err.Error())
	}
	deadline := time.Now().Add(wait)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, apperrors.New(apperrors.E6009, "cannot open transaction lock").WithCause(err.Error())
		}
		locked, lockErr := tryOSLock(file)
		if lockErr != nil {
			_ = file.Close()
			return nil, apperrors.New(apperrors.E6009, "cannot acquire transaction lock").WithCause(lockErr.Error())
		}
		if locked {
			// Nonce is informational only (diagnostics); correctness comes from
			// the kernel lock, so a stale/foreign nonce can never drive release.
			nonce := ownerNonce()
			_ = file.Truncate(0)
			_, _ = file.WriteString(nonce)
			_ = file.Sync()
			var once sync.Once
			return func() {
				once.Do(func() {
					releaseOSLock(file)
					_ = file.Close()
				})
			}, nil
		}
		_ = file.Close()
		if time.Now().After(deadline) {
			return nil, apperrors.New(apperrors.E6009, "data root is locked by another transaction").WithDetails(map[string]any{"lock": path})
		}
		time.Sleep(pollInterval)
	}
}

// waitTimeout is exposed for tests that need the default bound.
func defaultWait() time.Duration { return waitTimeout }

func ownerNonce() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", os.Getpid(), hex.EncodeToString(raw[:]))
}
