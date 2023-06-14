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
	"sync"
	"time"
)

// Cached is the two-tiered cache provider, adding a caching layer on top of a `Provider“.
type Cached struct {
	// inner is the tier-two cache (remote, network).
	inner Provider

	// outer is the tier-one cache (local, in-memory).
	outer Provider

	// name is the layered cache name.
	name string

	// ttl is the default TTL.
	ttl time.Duration

	mu sync.Mutex
}

// NewCached adds a caching layer on top of a cache `Provider` (typically a remote cache) and
// wraps it with a local in-memory cache. Items will always be stored in both caches. Fetches are
// only satified by the underlying remote cache, if the item does not exist in the local cache.
// The local cache will remove items, depending on the capacity constraints of the cache or the
// lifetime constraints of the cached item, respectively.
func NewCached(cache Provider, name string, ttl time.Duration, config InMemoryCacheConfig) (*Cached, error) {
	config.Sanitize()
	if config.MaxItemSize > config.MaxSize {
		return nil, fmt.Errorf("max item size (%v) must not exceed overall cache size (%v)",
			config.MaxItemSize, config.MaxSize)
	}

	l, err := NewInMemoryCache(InMemoryCacheConfig{
		MaxSize:     config.MaxSize,
		MaxItemSize: config.MaxItemSize,
	})
	if err != nil {
		return nil, err
	}

	cached := &Cached{
		inner: cache,
		outer: l,
		ttl:   ttl,
		name:  "layered-" + name,
	}

	return cached, nil
}

// Get retrieves an element based on a key, returning nil if the element
// does not exist.
func (c *Cached) Get(ctx context.Context, key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	val := c.outer.Get(ctx, key)
	if val != nil {
		// TODO: handle expiration.
		return val
	}

	val = c.inner.Get(ctx, key)
	if val != nil {
		c.outer.Set(key, val, c.ttl)
	}

	return val
}

// Set adds an element to the cache.
func (c *Cached) Set(key string, value []byte, ttl time.Duration) {
	c.inner.Set(key, value, ttl)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outer.Set(key, value, ttl)
}

// Delete deletes an element in the cache.
func (c *Cached) Delete(ctx context.Context, key string) bool {
	c.mu.Lock()
	c.outer.Delete(ctx, key)
	c.mu.Unlock()
	return c.inner.Delete(ctx, key)
}

// Keys returns a slice of cache keys.
func (c *Cached) Keys(ctx context.Context, prefix string) []string {
	return c.inner.Keys(ctx, prefix) // always satisfied by inner cache.
}

// Size returns the number of entries currently stored in the Cache.
func (c *Cached) Size() int {
	return len(c.inner.Keys(context.Background(), ""))
}
