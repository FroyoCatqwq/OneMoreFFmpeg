package cache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cache"
)

func TestStoreMemoryOnly(t *testing.T) {
	s, err := cache.New[string, int](cache.Options{MaxSize: 8})
	require.NoError(t, err)
	_, ok := s.Get("k")
	require.False(t, ok)
	require.NoError(t, s.Set("k", 42))
	v, ok := s.Get("k")
	require.True(t, ok)
	require.Equal(t, 42, v)
	s.Invalidate("k")
	_, ok = s.Get("k")
	require.False(t, ok)
}

func TestStorePersistenceAndTTL(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	s, err := cache.New[string, string](cache.Options{
		Dir: dir, TTL: time.Hour, Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.NoError(t, s.Set("asset", "sha256-abc"))

	// A fresh store with no memory sees the persisted value.
	s2, err := cache.New[string, string](cache.Options{
		Dir: dir, TTL: time.Hour, Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	v, ok := s2.Get("asset")
	require.True(t, ok)
	require.Equal(t, "sha256-abc", v)

	// Advance past TTL: entry is ignored.
	later := now.Add(2 * time.Hour)
	s3, err := cache.New[string, string](cache.Options{
		Dir: dir, TTL: time.Hour, Now: func() time.Time { return later },
	})
	require.NoError(t, err)
	_, ok = s3.Get("asset")
	require.False(t, ok)
}

func TestStoreKeySafety(t *testing.T) {
	dir := t.TempDir()
	s, err := cache.New[string, int](cache.Options{Dir: dir})
	require.NoError(t, err)
	require.NoError(t, s.Set("../../escape", 1))
	v, ok := s.Get("../../escape")
	require.True(t, ok)
	require.Equal(t, 1, v)
}
