// Package lock serializes data-root write transactions across processes.
package lock

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	fileName     = ".transaction.lock"
	staleAfter   = 30 * time.Second
	waitTimeout  = 5 * time.Second
	pollInterval = 50 * time.Millisecond

	// heartbeatEvery must stay well below staleAfter: the mtime is the only
	// liveness signal, so a holder that stops refreshing is reclaimable.
	heartbeatEvery = 5 * time.Second
)

// DataRoot acquires the global data-root transaction lock so install, env and
// uninstall writers cannot interleave. The returned function releases it. A
// lock whose mtime has not moved for staleAfter is treated as abandoned (a
// crashed process stops refreshing) and reclaimed; a live lock held past
// waitTimeout yields OMF-E6009.
func DataRoot(dataRoot string) (func(), error) {
	return acquire(filepath.Join(dataRoot, fileName), waitTimeout, staleAfter)
}

func acquire(path string, wait, stale time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock directory").WithCause(err.Error())
	}
	nonce := ownerNonce()
	deadline := time.Now().Add(wait)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			// Record who holds it before anybody can see an empty file: the
			// name is what release() and the heartbeat check.
			_, _ = file.WriteString(nonce)
			_ = file.Close()
			stop := make(chan struct{})
			go keepFresh(path, nonce, stop)
			var once sync.Once
			return func() {
				once.Do(func() {
					close(stop)
					release(path, nonce)
				})
			}, nil
		}
		if !os.IsExist(err) {
			return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock").WithCause(err.Error())
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > stale {
			// The heartbeat is the liveness signal: a holder that is still
			// working refreshes the mtime every heartbeatEvery, so an old
			// mtime means a crashed process -- except when the file is ours
			// from an earlier call, which cannot happen while we hold once.
			if owner(path) != nonce {
				_ = os.Remove(path)
			}
			continue
		}
		if time.Now().After(deadline) {
			return nil, apperrors.New(apperrors.E6009, "data root is locked by another transaction").WithDetails(map[string]any{"lock": path})
		}
		time.Sleep(pollInterval)
	}
}

// keepFresh refreshes the lock's mtime until stop is closed. Without it a
// legitimate multi-minute install (extract 512 MiB, copy, probe, journal) used
// to look abandoned after 30 s and a second process would walk in.
func keepFresh(path, nonce string, stop <-chan struct{}) {
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if owner(path) != nonce {
				// Somebody reclaimed and re-created it: never touch a lock
				// that is not ours any more.
				return
			}
			now := time.Now()
			if err := os.Chtimes(path, now, now); err != nil {
				// v0.6.x: ignoring this silently broke the mutex 30 s later -- the
				// mtime stops moving, another process reclaims the lock, and two
				// writers share the critical section. Chtimes can fail when an AV
				// scanner or the indexer holds the file, so fall back to rewriting
				// the same content, which refreshes the mtime through a path that
				// usually still works.
				if writeErr := os.WriteFile(path, []byte(nonce), 0o600); writeErr != nil {
					// Neither path works: the file is gone or unwritable, so there
					// is nothing to keep fresh.
					return
				}
			}
		}
	}
}

// release removes the lock only while it still carries our nonce. The old
// unconditional Remove deleted a lock a second process had legitimately
// acquired after reclaiming ours, handing two writers the same critical section.
func release(path, nonce string) {
	if owner(path) == nonce {
		_ = os.Remove(path)
	}
}

func owner(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ownerNonce() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", os.Getpid(), hex.EncodeToString(raw[:]))
}
