// Package cache provides the single caching abstraction used by
// OneMoreFFmpeg. Callers (capabilities, download, self-update) never import a
// caching library directly: the in-memory LRU is backed by otter and disk
// persistence uses atomic writes through go-internal/lockedfile.
package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/rogpeppe/go-internal/lockedfile"
)

// Store is a bounded, in-memory cache with optional on-disk persistence.
// It is safe for concurrent use.
type Store[K comparable, V any] struct {
	mem *otter.Cache[K, V]
	dir string
	now func() time.Time
	ttl time.Duration
}

type Options struct {
	MaxSize int           // max in-memory entries (0 => 512)
	Dir     string        // optional persistence directory ("" => memory only)
	TTL     time.Duration // disk entry freshness (0 => no expiry)
	Now     func() time.Time
}

func New[K comparable, V any](opts Options) (*Store[K, V], error) {
	if opts.MaxSize <= 0 {
		opts.MaxSize = 512
	}
	mem, err := otter.New[K, V](&otter.Options[K, V]{MaximumSize: opts.MaxSize})
	if err != nil {
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	if opts.Dir != "" {
		if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
			return nil, fmt.Errorf("cache directory: %w", err)
		}
	}
	return &Store[K, V]{mem: mem, dir: opts.Dir, now: now, ttl: opts.TTL}, nil
}

// Get is the convenience form; errors (corruption/I-O) are treated as a miss.
func (s *Store[K, V]) Get(key K) (V, bool) {
	v, ok, _ := s.GetErr(key)
	return v, ok
}

// GetErr returns the value and whether it was found, plus any real I/O or
// decode error. Corrupt entries are reported as errors, never panics.
func (s *Store[K, V]) GetErr(key K) (V, bool, error) {
	var zero V
	if v, ok := s.mem.GetIfPresent(key); ok {
		return v, true, nil
	}
	if s.dir == "" {
		return zero, false, nil
	}
	path, wantDigest, err := s.path(key)
	if err != nil {
		return zero, false, err
	}
	raw, err := lockedfile.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("read cache: %w", err)
	}
	var entry diskEntry[V]
	if err := json.Unmarshal(raw, &entry); err != nil {
		return zero, false, fmt.Errorf("decode cache (corrupt): %w", err)
	}
	if entry.Schema != diskSchema {
		return zero, false, fmt.Errorf("unknown cache schema %q", entry.Schema)
	}
	if entry.KeyDigest != wantDigest {
		return zero, false, fmt.Errorf("cache key digest mismatch (filename collision or tampering)")
	}
	if s.ttl > 0 && s.now().After(time.Unix(0, entry.SavedAt).Add(s.ttl)) {
		return zero, false, nil // expired: a miss; Prune removes the file
	}
	s.mem.Set(key, entry.Value)
	return entry.Value, true, nil
}

// Set writes to memory and, if configured, disk atomically under a sidecar lock.
func (s *Store[K, V]) Set(key K, value V) error {
	s.mem.Set(key, value)
	if s.dir == "" {
		return nil
	}
	path, digest, err := s.path(key)
	if err != nil {
		return err
	}
	entry := diskEntry[V]{
		Schema: diskSchema, KeyDigest: digest,
		SavedAt: s.now().UnixNano(), Value: value,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return lockedfile.Write(path, bytes.NewReader(raw), 0o600)
}

// Invalidate is the best-effort convenience form (used where errors cannot be
// surfaced); prefer InvalidateErr when the caller can act on failures.
func (s *Store[K, V]) Invalidate(key K) { _ = s.InvalidateErr(key) }

// InvalidateErr removes a value from both layers. A missing file is success;
// every other error (permissions, I/O) is returned, never swallowed.
func (s *Store[K, V]) InvalidateErr(key K) error {
	s.mem.Invalidate(key)
	if s.dir == "" {
		return nil
	}
	path, _, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("invalidate cache: %w", err)
	}
	return nil
}

// Prune removes expired or schema-invalid cache files inside the cache root.
// It never follows symlinks and never escapes the root. Returns removed count.
func (s *Store[K, V]) Prune() (int, error) {
	if s.dir == "" {
		return 0, nil
	}
	root, err := filepath.Abs(s.dir)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, fmt.Errorf("list cache: %w", err)
	}
	removed := 0
	var errs []error
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		full := filepath.Join(root, e.Name())
		if info, err := os.Lstat(full); err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue // never follow symlinks
		}
		rel, err := filepath.Rel(root, full)
		if err != nil || rel == ".." || filepath.Dir(rel) != "." {
			continue
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", e.Name(), err))
			continue
		}
		var probe struct {
			Schema  string `json:"schema"`
			SavedAt int64  `json:"saved_at"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			if err := os.Remove(full); err == nil {
				removed++
			} else {
				errs = append(errs, err)
			}
			continue
		}
		expired := probe.Schema != diskSchema ||
			(s.ttl > 0 && s.now().After(time.Unix(0, probe.SavedAt).Add(s.ttl)))
		if !expired {
			continue
		}
		if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("prune %s: %w", e.Name(), err))
			continue
		}
		removed++
	}
	return removed, errors.Join(errs...)
}

const diskSchema = "onemoreffmpeg.cache-entry/v1"

type diskEntry[V any] struct {
	Schema    string `json:"schema"`
	KeyDigest string `json:"key_digest"`
	SavedAt   int64  `json:"saved_at"`
	Value     V      `json:"value"`
}

// path maps a key to a stable filename derived from the SHA-256 of its JSON
// encoding. Two distinct keys can never collide through sanitization, and the
// original digest is stored in the entry so readers can detect tampering.
func (s *Store[K, V]) path(key K) (string, string, error) {
	raw, err := json.Marshal(key)
	if err != nil {
		return "", "", fmt.Errorf("encode cache key: %w", err)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	return filepath.Join(s.dir, digest+".json"), digest, nil
}
