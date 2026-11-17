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

	"github.com/gofrs/flock"

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
// The mutex is an OS advisory lock implemented by github.com/gofrs/flock
// (flock on Unix, LockFileEx on Windows) and is bound to a file descriptor
// owned by the flock handle. The kernel drops it when the descriptor closes,
// including when the process crashes -- there is no mtime "stale" heuristic and
// therefore no reclaim-vs-acquire TOCTOU. The lock file itself is intentionally
// never deleted: deleting and recreating it changes its identity and would let
// two waiters hold locks on different inodes at the same time.
func DataRoot(dataRoot string) (func(), error) {
	return acquire(filepath.Join(dataRoot, fileName), waitTimeout)
}

// FilePath takes the same OS-level advisory lock on an arbitrary lock file,
// for configuration files (e.g. providers.toml) that need cross-process
// serialization. The returned function releases it.
func FilePath(lockFilePath string) (func(), error) {
	return acquire(lockFilePath, waitTimeout)
}

// acquire opens (or creates) the lock file and takes the OS lock via the
// gofrs/flock library. wait bounds how long to poll when another process holds
// it. The nonce written into the file is informational only (diagnostics);
// correctness comes from the kernel lock.
func acquire(path string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock directory").WithCause(err.Error())
	}
	fl := flock.New(path, flock.SetPermissions(0o600))
	// Write the diagnostic nonce before locking: Windows LockFileEx holds an
	// exclusive handle that blocks other writers to the same file, so writing
	// while locked would silently fail. The nonce is informational only;
	// correctness comes from the OS lock, not the file contents.
	_ = os.WriteFile(path, []byte(ownerNonce()), 0o600)
	deadline := time.Now().Add(wait)
	for {
		locked, err := fl.TryLock()
		if err != nil {
			return nil, apperrors.New(apperrors.E6009, "cannot acquire transaction lock").WithCause(err.Error())
		}
		if locked {
			var once sync.Once
			return func() {
				once.Do(func() {
					_ = fl.Unlock()
					_ = fl.Close()
				})
			}, nil
		}
		if time.Now().After(deadline) {
			_ = fl.Close()
			return nil, apperrors.New(apperrors.E6009, "data root is locked by another transaction").WithDetails(map[string]any{"lock": path})
		}
		time.Sleep(pollInterval)
	}
}

// defaultWait is exposed for tests that need the default bound.
func defaultWait() time.Duration { return waitTimeout }

func ownerNonce() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", os.Getpid(), hex.EncodeToString(raw[:]))
}
