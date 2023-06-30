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
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

var _ Provider = (*simpleCache)(nil)

var (
	// DefaultCreateTime is the create time used by all entries in the cache.
	DefaultCreateTime = time.Time{}
)

// simpleCache provides a simple in-memory cache implementation.
// Example cache backend that is non bounded and never evicts.
// Not suitable for production use!
type simpleCache struct {
	mu          sync.RWMutex
	entryMap    map[string]*list.Element
	iterateList *list.List
}

// NewSimpleCache creates a new simple cache with given options.
// Simple cache will never evict entries and it will never reorder the elements.
func NewSimpleCache(opts *SimpleOptions) (Provider, error) {
	if opts == nil {
		opts = &SimpleOptions{}
	}
	cache := &simpleCache{
		iterateList: list.New(),
		entryMap:    make(map[string]*list.Element, opts.InitialCapacity),
	}
	return cache, nil
}

// Get retrieves the value with specified key.
func (c *simpleCache) Get(_ context.Context, key string) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry := c.entryMap[key]
	if entry == nil {
		return nil
	}
	return entry.Value.([]byte)
}

// Set sets a new value associated with the given key, returning the existing value (if present).
func (c *simpleCache) Set(key string, val []byte, _ time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entryMap[key] = c.iterateList.PushFront(val)
}

// Delete deletes the key/value associated with th given key.
func (c *simpleCache) Delete(_ context.Context, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent := c.entryMap[key]
	if ent == nil {
		return false
	}
	_ = c.iterateList.Remove(ent).([]byte)
	delete(c.entryMap, key)
	return true
}

// Size returns the number of entries currently in the cache.
func (c *simpleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entryMap)
}

// Keys returns a slice of the keys in the cache.
func (c *simpleCache) Keys(_ context.Context, _ string) []string {
	keys := make([]string, len(c.entryMap))
	i := 0
	for k := range c.entryMap {
		keys[i] = k
		i++
	}
	return keys
}

func (c *simpleCache) Purge(_ context.Context, _ string) error {
	return errors.New("not yet implemented")
}
