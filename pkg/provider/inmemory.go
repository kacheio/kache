package provider

import (
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
)

const (
	maxInt          = int(^uint(0) >> 1)
	sliceHeaderSize = 24
)

// inMemoryCache is the in-memory cache.
type inMemoryCache struct {
	mu sync.RWMutex

	inner *lru.Cache[interface{}, []byte]

	maxSizeBytes     uint64
	maxItemSizeBytes uint64

	curSize uint64
}

// DefaultInMemoryCacheConfig provides default config values for the cache.
var DefaultInMemoryCacheConfig = InMemoryCacheConfig{
	MaxSize:     250 * 1024 * 1024,
	MaxItemSize: 125 * 1024 * 1024,
}

// InMemoryCacheConfig holds the in-memory cache config.
type InMemoryCacheConfig struct {
	// MaxSize is the overall maximum number of bytes the cache can hold.
	MaxSize uint64 `yaml:"max_size"`
	// MaxItemSize is the maximum size of a single item.
	MaxItemSize uint64 `yaml:"max_item_size"`
}

// NewInMemoryCache creates a new thread-safe LRU in memory cache.
// It ensures the total cache size approximately does not exceed maxBytes.
func NewInMemoryCache(config InMemoryCacheConfig) (Provider, error) {
	if config.MaxItemSize > config.MaxSize {
		return nil, fmt.Errorf("max item size (%v) must not exceed overall cache size (%v)",
			config.MaxItemSize, config.MaxSize)
	}

	c := &inMemoryCache{
		maxSizeBytes:     config.MaxSize,
		maxItemSizeBytes: config.MaxItemSize,
	}

	// Initialize LRU cache with a high size limit, since
	// evictions are managed internally based on item size.
	l, err := lru.NewWithEvict[interface{}, []byte](maxInt, c.onEvict)
	if err != nil {
		return nil, err
	}
	c.inner = l

	return c, nil
}

// onEvict is the eviction callback.
func (c *inMemoryCache) onEvict(key interface{}, val []byte) {
	c.curSize -= itemSize(val)
}

// Get retrieves an element based on the provided key.
func (c *inMemoryCache) Get(key interface{}) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.inner.Get(key)
	if !ok {
		return nil
	}
	return v
}

// Set adds an item to the cache. If the item is too large,
// the cache evicts older items unitl it fits.
func (c *inMemoryCache) Set(key interface{}, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	size := itemSize(value)

	if ent, ok := c.inner.Get(key); ok {
		entSize := itemSize(ent)
		if size <= entSize {
			c.inner.Add(key, value)
			c.curSize -= (entSize - size)
			return
		}
		c.inner.Remove(key)
	}

	if !c.ensureCapacity(size) {
		return
	}

	c.inner.Add(key, value)
	c.curSize += size
}

// ensureCapacity ensures there is enough capacity for the new item.
func (c *inMemoryCache) ensureCapacity(size uint64) bool {
	if size > c.maxSizeBytes {
		log.Debug().Msg("Item is bigger than maxItemSize")
		return false
	}

	for c.curSize+size > c.maxSizeBytes {
		if _, _, ok := c.inner.RemoveOldest(); !ok {
			log.Debug().Msg("Failed to allocate space for new item.")
			c.reset()
		}
	}

	return true
}

// itemSize calculates the actual size of the provided slice.
func itemSize(b []byte) uint64 {
	return sliceHeaderSize + uint64(len(b))
}

// reset resets the cache.
func (c *inMemoryCache) reset() {
	c.inner.Purge()
	c.curSize = 0
}

// Delete deletes an element in the cache.
func (c *inMemoryCache) Delete(key interface{}) bool {
	return c.inner.Remove(key)
}

// Size returns the number of entries currently stored in the Cache.
func (c *inMemoryCache) Size() int {
	return c.inner.Len()
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *inMemoryCache) Keys() []interface{} {
	return c.inner.Keys()
}