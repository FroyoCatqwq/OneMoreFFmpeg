// Package lock serializes data-root write transactions across processes.
package lock

import (
	"os"
	"path/filepath"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	fileName     = ".transaction.lock"
	staleAfter   = 30 * time.Second
	waitTimeout  = 5 * time.Second
	pollInterval = 50 * time.Millisecond
)

// DataRoot acquires the global data-root transaction lock so install, env and
// uninstall writers cannot interleave. The returned function releases it. A
// lock older than staleAfter is treated as abandoned (e.g. a crashed process)
// and reclaimed; a live lock held past waitTimeout yields OMF-E6009.
func DataRoot(dataRoot string) (func(), error) {
	return acquire(filepath.Join(dataRoot, fileName), waitTimeout, staleAfter)
}

func acquire(path string, wait, stale time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock directory").WithCause(err.Error())
	}
	deadline := time.Now().Add(wait)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, apperrors.New(apperrors.E6009, "cannot create transaction lock").WithCause(err.Error())
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > stale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, apperrors.New(apperrors.E6009, "data root is locked by another transaction").WithDetails(map[string]any{"lock": path})
		}
		time.Sleep(pollInterval)
	}
}