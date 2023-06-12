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
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint: s.Addr(),
	}
	client, err := NewRedisClient("test", config)
	require.NoError(t, err)

	cache := NewRedisCache("test", client)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	cache.Set("A", []byte("Alice"), ttl)
	assert.Equal(t, "Alice", string(cache.Get(ctx, "A")))
	assert.Nil(t, cache.Get(ctx, "B"))

	cache.Set("B", []byte("Bob"), ttl)
	cache.Set("E", []byte("Eve"), ttl)
	cache.Set("G", []byte("Gopher"), ttl)

	assert.Equal(t, "Bob", string(cache.Get(ctx, "B")))
	assert.Equal(t, "Eve", string(cache.Get(ctx, "E")))
	assert.Equal(t, "Gopher", string(cache.Get(ctx, "G")))

	cache.Set("A", []byte("Foo"), ttl)
	assert.Equal(t, "Foo", string(cache.Get(ctx, "A")))

	cache.Set("B", []byte("Bar"), ttl)
	assert.Equal(t, "Bar", string(cache.Get(ctx, "B")))
	assert.Equal(t, "Foo", string(cache.Get(ctx, "A")))

	cache.Delete(ctx, "A")
	assert.Nil(t, cache.Get(ctx, "A"))
}
