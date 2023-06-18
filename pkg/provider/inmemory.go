// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
)

var _ Provider = (*inMemoryCache)(nil)

const (
	maxInt          = int(^uint(0) >> 1)
	sliceHeaderSize = 24
)

// inMemoryCache is the in-memory cache.
type inMemoryCache struct {
	mu sync.RWMutex

	// inner is the actual LRU cache.
	inner *lru.Cache[string, []byte]

	// maxSizeBytes is the max bytes the cache can hold.
	maxSizeBytes uint64

	// maxItemSizeBytes is the max size of a single item.
	maxItemSizeBytes uint64

	// curSize is the current cache size in bytes.
	curSize uint64

	// defaultTTL is the item default ttl.
	defaultTTL time.Duration

	// ttl holds the ttl to an item.
	ttl map[string]time.Time

	// currentTime is the time source.
	currentTime func() time.Time
}

// DefaultInMemoryCacheConfig provides default config values for the cache.
var DefaultInMemoryCacheConfig = InMemoryCacheConfig{
	MaxSize:     1 << 28, // 256 MiB
	MaxItemSize: 1 << 27, // 128 Mib
	DefaultTTL:  "120s",
}

// InMemoryCacheConfig holds the in-memory cache config.
type InMemoryCacheConfig struct {
	// MaxSize is the overall maximum number of bytes the cache can hold.
	MaxSize uint64 `yaml:"max_size"`
	// MaxItemSize is the maximum size of a single item.
	MaxItemSize uint64 `yaml:"max_item_size"`
	// DefaultTTL is the defautl ttl of a single item.
	DefaultTTL string `yaml:"default_ttl"`
}

// Sanitize checks the config and adds defaults to missing values.
func (c *InMemoryCacheConfig) Sanitize() {
	if c.MaxSize == 0 {
		c.MaxSize = DefaultInMemoryCacheConfig.MaxSize
	}
	if c.MaxItemSize == 0 {
		c.MaxItemSize = DefaultInMemoryCacheConfig.MaxItemSize
	}
	if len(c.DefaultTTL) == 0 {
		c.DefaultTTL = DefaultInMemoryCacheConfig.DefaultTTL
	}
}

// NewInMemoryCache creates a new thread-safe LRU in memory cache.
// It ensures the total cache size approximately does not exceed maxBytes.
func NewInMemoryCache(config InMemoryCacheConfig) (Provider, error) {
	config.Sanitize()
	if config.MaxItemSize > config.MaxSize {
		return nil, fmt.Errorf("max item size (%v) must not exceed overall cache size (%v)",
			config.MaxItemSize, config.MaxSize)
	}

	ttl, err := time.ParseDuration(config.DefaultTTL)
	if err != nil {
		ttl = time.Duration(120 * time.Second)
	}

	c := &inMemoryCache{
		maxSizeBytes:     config.MaxSize,
		maxItemSizeBytes: config.MaxItemSize,
		defaultTTL:       ttl,
		ttl:              make(map[string]time.Time),
		currentTime:      time.Now,
	}

	// Initialize LRU cache with a high size limit, since
	// evictions are managed internally based on item size.
	l, err := lru.NewWithEvict[string, []byte](maxInt, c.onEvict)
	if err != nil {
		return nil, err
	}
	c.inner = l

	// TODO: create and start backgound job to evict expired items.

	return c, nil
}

// onEvict is the eviction callback.
func (c *inMemoryCache) onEvict(key string, val []byte) {
	c.curSize -= itemSize(val)
}

// Get retrieves an element based on the provided key.
func (c *inMemoryCache) Get(ctx context.Context, key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if expires, ok := c.ttl[key]; ok && expires.Before(c.currentTime()) {
		c._delete(ctx, key)
		return nil
	}

	v, ok := c.inner.Get(key)
	if !ok {
		return nil
	}
	return v
}

// Set adds an item to the cache. If the item is too large,
// the cache evicts older items unitl it fits.
func (c *inMemoryCache) Set(key string, value []byte, ttl time.Duration) {
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
	c.ttl[key] = c.currentTime().Add(ttl)
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
	c.ttl = make(map[string]time.Time)
}

// Delete deletes an element in the cache.
func (c *inMemoryCache) Delete(ctx context.Context, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c._delete(ctx, key)
}

// _delete deletes and item in the cache. Guarded by caller.
func (c *inMemoryCache) _delete(_ context.Context, key string) bool {
	delete(c.ttl, key)
	return c.inner.Remove(key)
}

// Size returns the number of entries currently stored in the Cache.
func (c *inMemoryCache) Size() int {
	return c.inner.Len()
}

// Keys returns a slice of the keys in the cache, from oldest to newest. It doesn't check TTL
// of the returned keys (TODO).
func (c *inMemoryCache) Keys(_ context.Context, prefix string) []string {
	if prefix == "" {
		return c.inner.Keys()
	}
	var keys []string
	for _, k := range c.inner.Keys() {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}
