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

	"github.com/kacheio/kache/pkg/utils/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMeoryCache(t *testing.T) {
	cache, _ := NewInMemoryCache(DefaultInMemoryCacheConfig)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	cache.Set("A", []byte("Alice"), ttl)
	assert.Equal(t, "Alice", string(cache.Get(ctx, "A")))
	assert.Nil(t, cache.Get(ctx, "B"))
	assert.Equal(t, 1, cache.Size())

	cache.Set("B", []byte("Bob"), ttl)
	cache.Set("E", []byte("Eve"), ttl)
	cache.Set("G", []byte("Gopher"), ttl)
	assert.Equal(t, 4, cache.Size())

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

func TestInMemoryConcurrentAccess(t *testing.T) {
	data := map[string]string{
		"A": "Alice",
		"B": "Bob",
		"G": "Gopher",
		"E": "Eve",
	}

	cache, _ := NewInMemoryCache(DefaultInMemoryCacheConfig)

	ttl := time.Duration(120 * time.Second)

	for k, v := range data {
		cache.Set(k, []byte(v), ttl)
	}

	ch := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)

		// concurrent get and put
		go func() {
			defer wg.Done()

			<-ch

			for j := 0; j < 1000; j++ {
				cache.Get(context.Background(), "A")
				cache.Set("A", []byte("Arnie"), ttl)
			}
		}()
	}

	close(ch)
	wg.Wait()
}

func TestInMemoryCacheMaxSize(t *testing.T) {
	config := InMemoryCacheConfig{
		MaxSize:     2 * (sliceHeaderSize + 40), // 128
		MaxItemSize: 1 * (sliceHeaderSize + 40), // 64
	}
	cache, _ := NewInMemoryCache(config)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	// Item exceeds cache size.
	item := strings.Repeat("A", 129)
	cache.Set("Large Item", []byte(item), ttl)
	assert.Equal(t, 0, cache.Size())
	assert.Equal(t, 0, int(cache.(*inMemoryCache).curSize))

	// A in cache.
	itemA := strings.Repeat("A", 40)
	cache.Set("ItemA", []byte(itemA), ttl)
	assert.Equal(t, 1, cache.Size())
	assert.Equal(t, 64, int(cache.(*inMemoryCache).curSize))

	// B in cache.
	itemB := strings.Repeat("B", 40)
	cache.Set("ItemB", []byte(itemB), ttl)
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 128, int(cache.(*inMemoryCache).curSize))

	// C in cache, A evicted.
	itemC := strings.Repeat("C", 40)
	cache.Set("ItemC", []byte(itemC), ttl)
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 128, int(cache.(*inMemoryCache).curSize))

	assert.Equal(t, "", string(cache.Get(ctx, "ItemA")))
	assert.Equal(t, itemC, string(cache.Get(ctx, "ItemC")))

	// C updated with smaller item, no eviction.
	itemCm := strings.Repeat("c", 20)
	cache.Set("ItemC", []byte(itemCm), ttl)
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 108, int(cache.(*inMemoryCache).curSize))
	assert.Equal(t, itemCm, string(cache.Get(ctx, "ItemC")))

	// C updated with larger item, evction until fit.
	itemCM := strings.Repeat("C", 39)
	cache.Set("ItemC", []byte(itemCM), ttl)
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 127, int(cache.(*inMemoryCache).curSize))
	assert.Equal(t, itemCM, string(cache.Get(ctx, "ItemC")))
	assert.Equal(t, itemB, string(cache.Get(ctx, "ItemB")))

	// Reset
	cache.(*inMemoryCache).reset()
	assert.Equal(t, 0, cache.Size())
	assert.Equal(t, 0, int(cache.(*inMemoryCache).curSize))
}

func TestInMemoryCacheConfigMaxItemSizeTooBig(t *testing.T) {
	config := InMemoryCacheConfig{
		MaxSize:     1 * (sliceHeaderSize + 40), // 64
		MaxItemSize: 2 * (sliceHeaderSize + 40), // 128
	}
	_, err := NewInMemoryCache(config)
	assert.Error(t, err)
}

func TestKeys(t *testing.T) {
	cache, err := NewInMemoryCache(DefaultInMemoryCacheConfig)
	assert.NoError(t, err)

	ctx := context.Background()
	ttl := time.Duration(120 * time.Second)

	cache.Set("B", []byte("B"), ttl)
	cache.Set("E", []byte("E"), ttl)
	cache.Set("G", []byte("G"), ttl)

	assert.Equal(t, []string{"B", "E", "G"}, cache.Keys(ctx, ""))

	cache.Set("Foo:B", []byte("Foo:Bar"), ttl)
	cache.Set("Bar:F", []byte("Bar:Foo"), ttl)
	assert.Equal(t, []string{"Foo:B"}, cache.Keys(ctx, "Foo:"))
}

func TestTTL(t *testing.T) {
	ts := clock.NewEventTimeSource()
	ts.Update(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))

	cache, err := NewInMemoryCache(DefaultInMemoryCacheConfig)
	require.NoError(t, err)
	cache.(*inMemoryCache).currentTime = ts.Now

	cache.Set("A", []byte("Alice"), 120*time.Second)
	assert.Equal(t, 1, int(cache.Size()))

	// Advance time.
	ts.Update(ts.Now().Add(90 * time.Second))

	assert.Equal(t, "Alice", string(cache.Get(context.Background(), "A")))
	assert.Equal(t, 1, int(cache.Size()))

	// Advance time.
	ts.Update(ts.Now().Add(31 * time.Second)) // 121s

	assert.Equal(t, "", string(cache.Get(context.Background(), "A")))
	assert.Equal(t, 0, int(cache.Size()))
}

func TestTTLEvictionDisabled(t *testing.T) {
	ts := clock.NewEventTimeSource()
	ts.Update(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))

	config := DefaultInMemoryCacheConfig
	config.DefaultTTL = "-1" // disable TTL eviction

	cache, err := NewInMemoryCache(config)
	require.NoError(t, err)
	cache.(*inMemoryCache).currentTime = ts.Now

	assert.False(t, cache.(*inMemoryCache).ttlEviction)

	cache.Set("A", []byte("Alice"), 120*time.Second)
	assert.Equal(t, 1, int(cache.Size()))

	// Advance time.
	ts.Update(ts.Now().Add(90 * time.Second))

	assert.Equal(t, "Alice", string(cache.Get(context.Background(), "A")))
	assert.Equal(t, 1, int(cache.Size()))

	// Advance time.
	ts.Update(ts.Now().Add(31 * time.Second)) // 121s

	assert.Equal(t, "Alice", string(cache.Get(context.Background(), "A")))
	assert.Equal(t, 1, int(cache.Size()))
}
