package provider

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleCache(t *testing.T) {
	cache, _ := NewSimpleCache(nil)

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

func TestSimpleCacheConcurrentAccess(t *testing.T) {
	data := map[string]string{
		"A": "Alice",
		"B": "Bob",
		"G": "Gopher",
		"E": "Eve",
	}

	cache, _ := NewSimpleCache(nil)

	for k, v := range data {
		cache.Set(k, []byte(v))
	}

	ch := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)

		// concurrent get and Set
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
