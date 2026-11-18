// Package cache's file helpers provide the shared, cross-process-safe JSON
// file primitives used by every cache implementation, including capabilities.
// Business modules keep their own validity rules (fingerprints, schema checks)
// but no longer duplicate atomic-write / lock / JSON details.
package cache

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/lock"
	"github.com/rogpeppe/go-internal/lockedfile"
)

func lockedFilePath(path string) (func(), error) { return lock.FilePath(path) }

// ReadJSON reads and decodes a JSON file produced by WriteJSON. A missing
// file is reported with the os.ErrNotExist error so callers can treat it as a
// cache miss; corrupt JSON returns a typed error instead of panicking.
func ReadJSON(path string, dst any) error {
	raw, err := lockedfile.Read(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode cache %s: %w", filepath.Base(path), err)
	}
	return nil
}

// WriteJSON atomically writes JSON to path under a sidecar lock in a sibling
// .locks directory (never inside the cache directory itself).
func WriteJSON(path string, value any, perm os.FileMode) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Sidecar locks live outside the cache directory so enumerators only see
	// data files, and are deterministic across processes.
	lockDir := filepath.Join(filepath.Dir(dir), "."+filepath.Base(dir)+".locks")
	unlock, err := lockedFilePath(filepath.Join(lockDir, filepath.Base(path)+".lock"))
	if err != nil {
		return err
	}
	defer unlock()

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	discard := func() { _ = tmp.Close(); _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(raw); err != nil {
		discard()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
