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
	"errors"
	"time"
)

// Provider is a generalized interface to a cache.
// See provider.Simple for a specific implementation.
type Provider interface {
	// Get retrieves an element based on a key, returning nil if the element
	// does not exist.
	Get(ctx context.Context, key string) []byte

	// Set adds an element to the cache.
	Set(key string, value []byte, ttl time.Duration)

	// Delete deletes an element in the cache.
	Delete(ctx context.Context, key string) bool

	// // Iterator returns the iterator into cache.
	// Iterator() Iterator

	// Keys returns a slice of cache keys.
	Keys(ctx context.Context, prefix string) []string

	// Size returns the number of entries currently stored in the Cache.
	Size() int
}

// RemoteCacheClient is a generalized interface to interact with a remote cache.
type RemoteCacheClient interface {
	// Fetch fetches a key from the remote cache.
	// Returns nil if an error occurs.
	Fetch(ctx context.Context, keys string) []byte

	// Store stores a key and value into the the remote cache.
	// Returns an error in case the operation fails.
	Store(key string, value []byte, ttl time.Duration) error

	// Delete deletes a key from the remote cache.
	Delete(ctx context.Context, key string) error

	// Keys returns a slice of cache keys.
	Keys(ctx context.Context, prefix string) []string

	// Stop closes the client connection.
	Stop()

	// TODO: add iterator and to replace Keys(..).
	// Iterator returns the iterator into cache.
	// Iterator() Iterator
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
	Key() string
	// Value represents the value.
	Value() []byte
	// CreateTime represents the time when the entry is created.
	CreateTime() time.Time
}

const (
	BackendInMemory = "inmemory"
	BackendRedis    = "redis"
)

var errUnsupportedCacheBackend = errors.New("unsupported cache backend")

// ProviderBackendConfig holds the configuration for the caching provider backend.
type ProviderBackendConfig struct {
	Backend    string              `yaml:"backend"`
	Layered    bool                `yaml:"layered"`
	LayeredTTL string              `yaml:"layered_ttl"`
	InMemory   InMemoryCacheConfig `yaml:"inmemory"`
	Redis      RedisClientConfig   `yaml:"redis"`
}

// CreateCacheProvider creates a cache backend based on the provided configuration.
func CreateCacheProvider(name string, config ProviderBackendConfig) (Provider, error) {
	switch config.Backend {
	case BackendInMemory:
		return NewInMemoryCache(config.InMemory)
	case BackendRedis:
		client, err := NewRedisClient(name, config.Redis)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to create redis client"))
		}
		cache := NewRedisCache(name, client)
		if config.Layered {
			ttl, err := time.ParseDuration(config.LayeredTTL)
			if err != nil {
				ttl = 120 * time.Second
			}
			return NewCached(cache, name, ttl, config.InMemory)
		}
		return cache, nil
	default:
		return nil, errUnsupportedCacheBackend
	}
}
