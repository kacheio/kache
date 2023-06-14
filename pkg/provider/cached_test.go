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

func TestCached(t *testing.T) {
	s := miniredis.RunT(t)
	client, err := NewRedisClient("redis", RedisClientConfig{
		Endpoint: s.Addr(),
	})
	require.NoError(t, err)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	cache, err := NewCached(NewRedisCache("inner", client), "cached", ttl, DefaultInMemoryCacheConfig)
	require.NoError(t, err)

	cache.Set("A", []byte("Alice"), ttl)
	assert.Equal(t, "Alice", string(cache.Get(ctx, "A")))
	assert.Nil(t, cache.Get(ctx, "B"))

	assert.Equal(t, 1, len(cache.outer.Keys(ctx, "")))
	assert.Equal(t, 1, len(cache.inner.Keys(ctx, "")))

	s.Del("A")
	// After removing item from layer 2, it is still in layer 1.
	assert.Equal(t, "Alice", string(cache.Get(ctx, "A")))

	cache.Delete(ctx, "A")

	_ = s.Set("B", "Bob")
	_ = s.Set("I", "Ida")
	assert.Equal(t, "Bob", string(cache.Get(ctx, "B")))
	// After getting B, Bob is in the layer one cache, but Ida not.
	assert.Equal(t, 1, len(cache.outer.Keys(ctx, "")))
	assert.Equal(t, 2, len(cache.inner.Keys(ctx, "")))
}
