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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisClientCache(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint: s.Addr(),
	}
	cache, err := NewRedisClient("test", config)
	require.NoError(t, err)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	_ = cache.Store("A", []byte("Alice"), ttl)
	assert.Equal(t, "Alice", string(cache.Fetch(ctx, "A")))
	assert.Nil(t, cache.Fetch(ctx, "B"))

	_ = cache.Store("B", []byte("Bob"), ttl)
	_ = cache.Store("E", []byte("Eve"), ttl)
	_ = cache.Store("G", []byte("Gopher"), ttl)

	assert.Equal(t, "Bob", string(cache.Fetch(ctx, "B")))
	assert.Equal(t, "Eve", string(cache.Fetch(ctx, "E")))
	assert.Equal(t, "Gopher", string(cache.Fetch(ctx, "G")))

	_ = cache.Store("A", []byte("Foo"), ttl)
	assert.Equal(t, "Foo", string(cache.Fetch(ctx, "A")))

	_ = cache.Store("B", []byte("Bar"), ttl)
	assert.Equal(t, "Bar", string(cache.Fetch(ctx, "B")))
	assert.Equal(t, "Foo", string(cache.Fetch(ctx, "A")))

	_ = cache.Delete(ctx, "A")
	assert.Nil(t, cache.Fetch(ctx, "A"))

	assert.Equal(t, []string{"B", "E", "G"}, cache.Keys(ctx, ""))

	_ = cache.Store("Foo:B", []byte("Foo:Bar"), ttl)
	_ = cache.Store("Bar:F", []byte("Bar:Foo"), ttl)
	assert.Equal(t, []string{"Foo:B"}, cache.Keys(ctx, "Foo:"))
}

func TestRedisClientConcurrentAccess(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint: s.Addr(),
	}
	cache, err := NewRedisClient("test", config)
	require.NoError(t, err)

	data := map[string]string{
		"A": "Alice",
		"B": "Bob",
		"G": "Gopher",
		"E": "Eve",
	}

	ttl := time.Duration(120 * time.Second)

	for k, v := range data {
		_ = cache.Store(k, []byte(v), ttl)
	}

	ch := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)

		// concurrent Fetch and put
		go func() {
			defer wg.Done()

			<-ch

			for j := 0; j < 1000; j++ {
				cache.Fetch(context.Background(), "A")
				_ = cache.Store("A", []byte("Arnie"), ttl)
			}
		}()
	}

	close(ch)
	wg.Wait()
}

func TestRedisClientMaxItemSize(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint:    s.Addr(),
		MaxItemSize: 128,
	}
	cache, err := NewRedisClient("test", config)
	require.NoError(t, err)

	smallItem := strings.Repeat("A", 127)
	assert.NoError(t, cache.Store("A", []byte(smallItem), 120*time.Second))

	largeItem := strings.Repeat("A", 129)
	assert.Error(t, ErrRedisMaxItemSize, cache.Store("A", []byte(largeItem), 120*time.Second))
}

func TestRedisClientJobQueue(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint: s.Addr(),
	}
	cache, err := NewRedisClient("test", config)
	require.NoError(t, err)

	smallItem := strings.Repeat("A", 127)
	assert.Error(t, ErrRedisJobQueueFull, cache.Store("A", []byte(smallItem), 120*time.Second))
}

func TestRedisPurgeAndFlush(t *testing.T) {
	s := miniredis.RunT(t)
	config := RedisClientConfig{
		Endpoint: s.Addr(),
	}
	cache, err := NewRedisClient("test", config)
	require.NoError(t, err)

	items := []string{
		"https://www.example.com/resources/assets/901.css",
		"https://www.example.com/resources/assets/fonts/TheSansC5s-4_SemiLight.woff2",
		"https://www.example.com/resources/assets/fonts/TheSansC5s-4_SemiLightItalic.woff2",
		"https://www.example.com/resources/assets/image/lazy-image-placeholder.jpg",
		"https://www.example.com/resources/assets/382-38c6d9b600def80a4c9bea1def6d6ec2b134644f.bundle.js",
		"https://www.example.com/news",
		"https://www.example.com/news/article",
		"https://www.example.com/news/article/asff",
	}
	for _, item := range items {
		_ = cache.Store(item, []byte("test"), 120*time.Second)
	}

	_ = cache.Purge(context.Background(), "*fonts*")
	assert.Nil(t, cache.Fetch(context.Background(), items[2]))

	_ = cache.Purge(context.Background(), "*/news*")
	assert.Nil(t, cache.Fetch(context.Background(), items[5]))
	assert.Nil(t, cache.Fetch(context.Background(), items[6]))
	assert.Nil(t, cache.Fetch(context.Background(), items[7]))

	// Flush DB.
	_ = cache.Flush(context.Background())
	assert.Equal(t, 0, len(cache.Keys(context.Background(), "")))
}
