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

/// SimpleCache provides a simple in-memory cache implementation.
/// Example cache backend that is non bounded and never evicts.
/// Not suitable for production use!

type (
	simpleCache struct {
		mu          sync.RWMutex
		entryMap    map[interface{}]*list.Element
		iterateList *list.List
	}

	simpleCacheIter struct {
		cache    *simpleCache
		nextItem *list.Element
	}

	simpleCacheEntry struct {
		key interface{}
		val interface{}
	}
)

/// Entry interface implementation

// Key returns the cache entry key.
func (e *simpleCacheEntry) Key() interface{} {
	return e.key
}

// Value return the cache entry value.
func (e *simpleCacheEntry) Value() interface{} {
	return e.val
}

// CreateTime is not implemented for simple cache entries.
func (e *simpleCacheEntry) CreateTime() time.Time {
	return DefaultCreateTime
}

/// Provider interface implementation

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
func (c *simpleCache) Get(key interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry := c.entryMap[key]
	if entry == nil {
		return nil
	}
	return entry.Value.(*simpleCacheEntry).Value()
}

// Put sets a new value associated with the given key, returning the existing value (if present).
func (c *simpleCache) Put(key interface{}, val interface{}) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c._put(key, val, true)
}

func (c *simpleCache) _put(key interface{}, val interface{}, update bool) interface{} {
	ent := c.entryMap[key]
	if ent != nil {
		entry := ent.Value.(*simpleCacheEntry)
		current := entry.val
		if update {
			entry.val = val
		}
		return current
	}
	entry := &simpleCacheEntry{
		key: key,
		val: val,
	}
	c.entryMap[key] = c.iterateList.PushFront(entry)
	return nil
}

// Delete deletes the key/value associated with th given key.
func (c *simpleCache) Delete(key interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent := c.entryMap[key]
	if ent == nil {
		return false
	}
	entry := c.iterateList.Remove(ent).(*simpleCacheEntry)
	delete(c.entryMap, entry.key)
	return true
}

// Size returns the number of entries currently in the cache.
func (c *simpleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entryMap)
}

// Iterator creates a new cache iterator.
func (c *simpleCache) Iterator() Iterator {
	c.mu.RLock()
	iter := &simpleCacheIter{
		cache:    c,
		nextItem: c.iterateList.Front(),
	}
	return iter
}

/// Iterator interface implementation

// Next returns the next item in the cache.
func (it *simpleCacheIter) Next() Entry {
	if it.nextItem == nil {
		return nil
	}

	entry := it.nextItem.Value.(*simpleCacheEntry)
	it.nextItem = it.nextItem.Next()

	entry = &simpleCacheEntry{
		key: entry.key,
		val: entry.val,
	}
	return entry
}

// HasNext returns true if there are more items to be returned.
func (it *simpleCacheIter) HasNext() bool {
	return it.nextItem != nil
}

// Close closes the iterator.
func (it *simpleCacheIter) Close() {
	it.cache.mu.RUnlock()
}
