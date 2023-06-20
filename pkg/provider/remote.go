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
	"time"
)

// RedisCache is a Redis-based cache.
type RedisCache struct {
	*remoteCache
}

// NewRedisCache makes a new RedisCache.
func NewRedisCache(name string, client RemoteCacheClient) *RedisCache {
	return &RedisCache{
		remoteCache: newRemoteCache(name, client),
	}
}

// remoteCache holds the remote cache client.
type remoteCache struct {
	// client is the remote cache client.
	client RemoteCacheClient
	// name identifies the remote client.
	name string
}

// newRemoteCache creates a new remote cache for the provided client.
func newRemoteCache(name string, client RemoteCacheClient) *remoteCache {
	return &remoteCache{
		client: client,
		name:   name,
	}
}

// Get retrieves an element based on a key, returning nil if the element
// does not exist.
func (c *remoteCache) Get(ctx context.Context, key string) []byte {
	return c.client.Fetch(ctx, key)
}

// Set adds an item to the cache.
func (c *remoteCache) Set(key string, value []byte, ttl time.Duration) {
	_ = c.client.StoreAsync(key, value, ttl)
}

// Delete deletes an item from the cache.
func (c *remoteCache) Delete(ctx context.Context, key string) bool {
	return c.client.Delete(ctx, key) == nil
}

// Size returns the number of entries currently stored in the Cache.
// TODO: not implemented yet.
func (c *remoteCache) Size() int { return -1 }

// Keys returns a slice of cache keys.
// TODO: not implement yet.
func (c *remoteCache) Keys(ctx context.Context, prefix string) []string {
	return c.client.Keys(ctx, prefix)
}
