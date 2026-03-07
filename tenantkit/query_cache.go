package tenantkit

import (
	"hash/fnv"
	"strings"
	"sync"
	"sync/atomic"
)

// queryCacheEntry represents a cached query transformation
type queryCacheEntry struct {
	originalQuery    string
	transformedQuery string
	argCount         int
	isTenantQuery    bool
	hash             uint64
}

// QueryCache is a thread-safe LRU cache for query transformations
type QueryCache struct {
	mu      sync.RWMutex
	entries map[uint64]*queryCacheEntry
	maxSize int
	hits    uint64
	misses  uint64
}

// NewQueryCache creates a new query cache with specified max size
func NewQueryCache(maxSize int) *QueryCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}
	return &QueryCache{
		entries: make(map[uint64]*queryCacheEntry, maxSize),
		maxSize: maxSize,
	}
}

// hashQuery creates a fast hash for a query string
func hashQuery(query string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(query)) //nolint:errcheck,gosec // fnv.Hash.Write never returns an error #nosec G104
	return h.Sum64()
}

// Get retrieves a cached query transformation
func (qc *QueryCache) Get(originalQuery string) (transformedQuery string, argCount int, isTenantQuery bool, found bool) {
	hash := hashQuery(originalQuery)

	qc.mu.RLock()
	entry, exists := qc.entries[hash]
	qc.mu.RUnlock()

	if exists {
		atomic.AddUint64(&qc.hits, 1)
		return entry.transformedQuery, entry.argCount, entry.isTenantQuery, true
	}

	atomic.AddUint64(&qc.misses, 1)
	return "", 0, false, false
}

// Put stores a query transformation in the cache
func (qc *QueryCache) Put(originalQuery, transformedQuery string, argCount int, isTenantQuery bool) {
	hash := hashQuery(originalQuery)

	qc.mu.Lock()
	defer qc.mu.Unlock()

	// Simple eviction: if cache is full, clear half of it
	if len(qc.entries) >= qc.maxSize {
		qc.evictHalf()
	}

	qc.entries[hash] = &queryCacheEntry{
		originalQuery:    originalQuery,
		transformedQuery: transformedQuery,
		argCount:         argCount,
		isTenantQuery:    isTenantQuery,
		hash:             hash,
	}
}

// evictHalf removes approximately half the cache entries
// This is a simple eviction strategy that avoids tracking access times
func (qc *QueryCache) evictHalf() {
	targetSize := qc.maxSize / 2
	toRemove := len(qc.entries) - targetSize

	if toRemove <= 0 {
		return
	}

	// Remove arbitrary entries (map iteration is random in Go)
	removed := 0
	for hash := range qc.entries {
		delete(qc.entries, hash)
		removed++
		if removed >= toRemove {
			break
		}
	}
}

// Clear removes all entries from the cache
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.entries = make(map[uint64]*queryCacheEntry, qc.maxSize)
	atomic.StoreUint64(&qc.hits, 0)
	atomic.StoreUint64(&qc.misses, 0)
}

// Stats returns cache statistics
func (qc *QueryCache) Stats() (hits, misses uint64, size int, hitRate float64) {
	hits = atomic.LoadUint64(&qc.hits)
	misses = atomic.LoadUint64(&qc.misses)

	qc.mu.RLock()
	size = len(qc.entries)
	qc.mu.RUnlock()

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return hits, misses, size, hitRate
}

// stringBuilderPool pools string builders for query construction
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// getStringBuilder gets a string builder from the pool
func getStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder) //nolint:errcheck // sync.Pool.Get returns interface{}
	sb.Reset()
	return sb
}

// putStringBuilder returns a string builder to the pool
func putStringBuilder(sb *strings.Builder) {
	// Don't pool builders that are too large (>4KB)
	if sb.Cap() > 4096 {
		return
	}
	stringBuilderPool.Put(sb)
}

// slicePool pools small slices for arguments
var slicePool = sync.Pool{
	New: func() interface{} {
		s := make([]interface{}, 0, 8)
		return &s
	},
}

// getArgsSlice gets a slice from the pool
func getArgsSlice() *[]interface{} {
	return slicePool.Get().(*[]interface{}) //nolint:errcheck // sync.Pool.Get returns interface{}
}

// putArgsSlice returns a slice to the pool
func putArgsSlice(s *[]interface{}) {
	// Clear the slice but keep capacity
	*s = (*s)[:0]
	// Don't pool large slices
	if cap(*s) > 64 {
		return
	}
	slicePool.Put(s)
}
