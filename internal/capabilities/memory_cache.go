package capabilities

import (
	"os"
	"sync"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cache"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// memoryCacheTTL bounds the in-memory tier. The on-disk tier is validated by
// fingerprint (and file identity), so a short memory TTL only avoids repeated
// JSON decoding within one process — it is not the source of truth.
const memoryCacheTTL = 5 * time.Minute

type fullCacheKey struct {
	path           string
	fingerprint    string
	installationID string
	topic          string
}

type listCacheKey struct {
	path           string
	fingerprint    string
	installationID string
	kind           string
}

// memoryCaches holds the process-local otter-backed L1 caches. It is created
// lazily so a --no-cache run never allocates a cache and never resolves a data
// root.
type memoryCaches struct {
	once sync.Once
	full *cache.Store[fullCacheKey, model.CapabilitySet]
	list *cache.Store[listCacheKey, model.CapabilityListData]
}

func (m *memoryCaches) stores() (*cache.Store[fullCacheKey, model.CapabilitySet],
	*cache.Store[listCacheKey, model.CapabilityListData]) {
	m.once.Do(func() {
		// otter-backed, memory-only; disk remains the authoritative L2.
		full, err := cache.New[fullCacheKey, model.CapabilitySet](cache.Options{
			TTL: memoryCacheTTL, MaxSize: 128,
		})
		if err == nil {
			m.full = full
		}
		list, err := cache.New[listCacheKey, model.CapabilityListData](cache.Options{
			TTL: memoryCacheTTL, MaxSize: 128,
		})
		if err == nil {
			m.list = list
		}
	})
	return m.full, m.list
}

// fileIdentity makes an L1 entry miss when the on-disk cache is replaced or
// tampered with outside this process, preserving the corruption semantics.
func fileIdentity(path string) (int64, int64) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0
	}
	return info.Size(), info.ModTime().UnixNano()
}

func cacheIdentity(path string) string {
	size, mod := fileIdentity(path)
	return path + "@" + itoa(size) + ":" + itoa(mod)
}

func fullKey(cachePath, fingerprint, installationID, topic string) fullCacheKey {
	return fullCacheKey{
		path: cacheIdentity(cachePath), fingerprint: fingerprint,
		installationID: installationID, topic: topic,
	}
}

func listKey(cachePath, fingerprint, installationID, kind string) listCacheKey {
	return listCacheKey{
		path: cacheIdentity(cachePath), fingerprint: fingerprint,
		installationID: installationID, kind: kind,
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [24]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
