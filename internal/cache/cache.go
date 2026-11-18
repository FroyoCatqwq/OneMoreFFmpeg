// Package cache provides the single caching abstraction used by
// OneMoreFFmpeg. Callers (capabilities, download, self-update) never import a
// caching library directly: the in-memory LRU is backed by otter and disk
// persistence uses atomic writes through go-internal/lockedfile.
package cache

import (
	"bytes"
	"encoding/json/v2"
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

// New creates a Store. The otter cache handles in-memory eviction; this
// wrapper adds the persistent layer and TTL semantics.
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
	return &Store[K, V]{mem: mem, dir: opts.Dir, now: now, ttl: opts.TTL}, nil
}

// Get returns a live (non-expired) value from memory or disk.
func (s *Store[K, V]) Get(key K) (V, bool) {
	if v, ok := s.mem.GetIfPresent(key); ok {
		return v, true
	}
	if s.dir == "" {
		var zero V
		return zero, false
	}
	path := s.path(key)
	raw, err := lockedfile.Read(path)
	if err != nil {
		var zero V
		return zero, false
	}
	var entry diskEntry[V]
	if err := json.Unmarshal(raw, &entry); err != nil {
		var zero V
		return zero, false
	}
	if s.ttl > 0 && s.now().After(time.Unix(0, entry.SavedAt).Add(s.ttl)) {
		var zero V
		return zero, false
	}
	s.mem.Set(key, entry.Value)
	return entry.Value, true
}

// Set writes a value to memory and, if configured, disk (atomically).
func (s *Store[K, V]) Set(key K, value V) error {
	s.mem.Set(key, value)
	if s.dir == "" {
		return nil
	}
	entry := diskEntry[V]{SavedAt: s.now().UnixNano(), Value: value}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return lockedfile.Write(s.path(key), bytes.NewReader(raw), 0o600)
}

// Invalidate removes a value from both layers.
func (s *Store[K, V]) Invalidate(key K) {
	s.mem.Invalidate(key)
	if s.dir != "" {
		_ = os.Remove(s.path(key))
	}
}

type diskEntry[V any] struct {
	SavedAt int64 `json:"saved_at"`
	Value   V     `json:"value"`
}

func (s *Store[K, V]) path(key K) string {
	// Keys are string-like; fmt via JSON for a stable, safe filename.
	raw, err := json.Marshal(key)
	if err != nil {
		return filepath.Join(s.dir, "invalid")
	}
	name := string(raw)
	if len(name) > 120 {
		name = name[:120]
	}
	return filepath.Join(s.dir, sanitize(name)+".json")
}

func sanitize(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}
