package provider

import (
	"time"
)

// Provider is a generalized interface to a cache.
// See provider.Simple for a specific implementation.
type Provider interface {
	// Get retrieves an element based on a key, returning nil if the element
	// does not exist.
	Get(key interface{}) interface{}

	// Put adds an element to the cache, returning the previous element.
	Put(key interface{}, value interface{}) interface{}

	// PutIfNotExist puts a value associated with a given key if it does not exist
	// PutIfNotExist(key interface{}, value interface{}) (interface{}, error)

	// Delete deletes an element in the cache.
	Delete(key interface{}) bool

	// Iterator returns the iterator into cache.
	Iterator() Iterator

	// Size returns the number of entries currently stored in the Cache.
	Size() int
}

// Options control the behavior of the cache.
type Options struct {
	// TTL controls the time-to-live for a given cache entry.
	// Cache entries that are older than the TTL will not be returned.
	TTL time.Duration

	// InitialCapacity controls the initial capacity of the cache.
	InitialCapacity int
}

// SimpleOptions provides options that can be used to configure SimpleCache.
type SimpleOptions struct {
	// InitialCapacity controls the initial capacity of the cache.
	InitialCapacity int
}

// Iterator represents the interface for cache iterators.
type Iterator interface {
	// HasNext return true if there is more items to be returned.
	HasNext() bool
	// Next return the next item.
	Next() Entry
	// Close closes the iterator
	// and releases any allocated resources.
	Close()
}

// Entry represents a key-value entry within the map.
type Entry interface {
	// Key represents the key.
	Key() interface{}
	// Value represents the value.
	Value() interface{}
	// CreateTime represents the time when the entry is created.
	CreateTime() time.Time
}
