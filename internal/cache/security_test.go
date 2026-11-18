package cache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cache"
)

func TestKeyCollisionAndStableFilename(t *testing.T) {
	dir := t.TempDir()
	s, err := cache.New[string, string](cache.Options{Dir: dir})
	require.NoError(t, err)
	require.NoError(t, s.Set("a/b", "first"))
	require.NoError(t, s.Set("a_b", "second"))
	v1, ok, err := s.GetErr("a/b")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "first", v1)
	v2, ok, err := s.GetErr("a_b")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "second", v2)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2, "distinct keys must not share a filename")
}

func TestCorruptJSONIsErrorNotPanic(t *testing.T) {
	dir := t.TempDir()
	s, err := cache.New[string, string](cache.Options{Dir: dir})
	require.NoError(t, err)
	require.NoError(t, s.Set("k", "value"))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, entries[0].Name()), []byte("{not json"), 0o600))
	s2, err := cache.New[string, string](cache.Options{Dir: dir})
	require.NoError(t, err)
	_, ok, err := s2.GetErr("k")
	require.Error(t, err)
	require.False(t, ok)
}

func TestExpiredCacheIsMissAndPruned(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	s, err := cache.New[string, string](cache.Options{Dir: dir, TTL: time.Second, Now: func() time.Time { return now }})
	require.NoError(t, err)
	require.NoError(t, s.Set("k", "v"))
	// Advance past TTL with a fresh store sharing the same clock.
	later := now.Add(2 * time.Second)
	s2, err := cache.New[string, string](cache.Options{Dir: dir, TTL: time.Second, Now: func() time.Time { return later }})
	require.NoError(t, err)
	_, ok, err := s2.GetErr("k")
	require.NoError(t, err)
	require.False(t, ok, "expired entry must be a miss")
	removed, err := s2.Prune()
	require.NoError(t, err)
	require.GreaterOrEqual(t, removed, 1)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestInvalidateMissingIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := cache.New[string, string](cache.Options{Dir: dir})
	require.NoError(t, err)
	require.NoError(t, s.InvalidateErr("never-existed"))
}

func TestPruneIgnoresSymlinksAndDirs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dir := t.TempDir()
	s, err := cache.New[string, string](cache.Options{Dir: dir})
	require.NoError(t, err)
	require.NoError(t, s.Set("k", "v"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))
	target := filepath.Join(t.TempDir(), "outside.json")
	require.NoError(t, os.WriteFile(target, []byte("keep"), 0o600))
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink unavailable")
	}
	removed, err := s.Prune()
	require.NoError(t, err)
	require.Equal(t, 0, removed, "valid entry, dir and symlink must not be pruned")
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data), "symlink target outside cache must be untouched")
}
