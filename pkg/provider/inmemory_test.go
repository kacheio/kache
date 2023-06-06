package provider

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMeoryCache(t *testing.T) {
	cache, _ := NewInMemoryCache(DefaultInMemoryCacheConfig)

	cache.Set("A", []byte("Alice"))
	assert.Equal(t, "Alice", string(cache.Get("A")))
	assert.Nil(t, cache.Get("B"))
	assert.Equal(t, 1, cache.Size())

	cache.Set("B", []byte("Bob"))
	cache.Set("E", []byte("Eve"))
	cache.Set("G", []byte("Gopher"))
	assert.Equal(t, 4, cache.Size())

	assert.Equal(t, "Bob", string(cache.Get("B")))
	assert.Equal(t, "Eve", string(cache.Get("E")))
	assert.Equal(t, "Gopher", string(cache.Get("G")))

	cache.Set("A", []byte("Foo"))
	assert.Equal(t, "Foo", string(cache.Get("A")))

	cache.Set("B", []byte("Bar"))
	assert.Equal(t, "Bar", string(cache.Get("B")))
	assert.Equal(t, "Foo", string(cache.Get("A")))

	cache.Delete("A")
	assert.Nil(t, cache.Get("A"))
}

func TestInMemoryConcurrentAccess(t *testing.T) {
	data := map[string]string{
		"A": "Alice",
		"B": "Bob",
		"G": "Gopher",
		"E": "Eve",
	}

	cache, _ := NewInMemoryCache(DefaultInMemoryCacheConfig)

	for k, v := range data {
		cache.Set(k, []byte(v))
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
				cache.Get("A")
				cache.Set("A", []byte("Arnie"))
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

	// Item exceeds cache size.
	item := strings.Repeat("A", 129)
	cache.Set("Large Item", []byte(item))
	assert.Equal(t, 0, cache.Size())
	assert.Equal(t, 0, int(cache.(*inMemoryCache).curSize))

	// A in cache.
	itemA := strings.Repeat("A", 40)
	cache.Set("ItemA", []byte(itemA))
	assert.Equal(t, 1, cache.Size())
	assert.Equal(t, 64, int(cache.(*inMemoryCache).curSize))

	// B in cache.
	itemB := strings.Repeat("B", 40)
	cache.Set("ItemB", []byte(itemB))
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 128, int(cache.(*inMemoryCache).curSize))

	// C in cache, A evicted.
	itemC := strings.Repeat("C", 40)
	cache.Set("ItemC", []byte(itemC))
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 128, int(cache.(*inMemoryCache).curSize))

	assert.Equal(t, "", string(cache.Get("ItemA")))
	assert.Equal(t, itemC, string(cache.Get("ItemC")))

	// C updated with smaller item, no eviction.
	itemCm := strings.Repeat("c", 20)
	cache.Set("ItemC", []byte(itemCm))
	assert.Equal(t, 2, cache.Size())
	assert.Equal(t, 108, int(cache.(*inMemoryCache).curSize))
	assert.Equal(t, itemCm, string(cache.Get("ItemC")))

	// C updated with larger item, evction until fit.
	itemCM := strings.Repeat("C", 64)
	cache.Set("ItemC", []byte(itemCM))
	assert.Equal(t, 1, cache.Size())
	assert.Equal(t, 88, int(cache.(*inMemoryCache).curSize))
	assert.Equal(t, itemCM, string(cache.Get("ItemC")))

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
