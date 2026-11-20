// Package cache's file helpers provide the shared, cross-process-safe JSON
// file primitives used by every cache implementation, including capabilities.
// Business modules keep their own validity rules (fingerprints, schema checks)
// but no longer duplicate atomic-write / lock / JSON details.
package cache

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rogpeppe/go-internal/lockedfile"
)

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

// WriteJSON writes JSON to path under the SAME sidecar lock (<path>.lock) used
// by ReadJSON, so readers and writers across processes exclude each other.
// lockedfile.Write locks the file for the whole overwrite, eliminating the
// prior bug where reader and writer used different lock files.
func WriteJSON(path string, value any, perm os.FileMode) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return lockedfile.Write(path, bytes.NewReader(raw), perm)
}
