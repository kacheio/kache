package provider

import (
	"container/list"
	"sync"
	"time"
)

var (
	// DefaultCreateTime is the create time used by all entries in the cache.
	DefaultCreateTime = time.Time{}
)

// simpleCache provides a simple in-memory cache implementation.
// Example cache backend that is non bounded and never evicts.
// Not suitable for production use!
type simpleCache struct {
	mu          sync.RWMutex
	entryMap    map[interface{}]*list.Element
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
		entryMap:    make(map[interface{}]*list.Element, opts.InitialCapacity),
	}
	return cache, nil
}

// Get retrieves the value with specified key.
func (c *simpleCache) Get(key interface{}) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry := c.entryMap[key]
	if entry == nil {
		return nil
	}
	return entry.Value.([]byte)
}

// Set sets a new value associated with the given key, returning the existing value (if present).
func (c *simpleCache) Set(key interface{}, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entryMap[key] = c.iterateList.PushFront(val)
}

// Delete deletes the key/value associated with th given key.
func (c *simpleCache) Delete(key interface{}) bool {
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
func (c *simpleCache) Keys() []any {
	keys := make([]interface{}, len(c.entryMap))
	i := 0
	for k := range c.entryMap {
		keys[i] = k
		i++
	}
	return keys
}
