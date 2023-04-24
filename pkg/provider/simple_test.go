package provider

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleCache(t *testing.T) {
	cache, _ := NewSimpleCache(nil)

	cache.Put("A", "Alice")
	assert.Equal(t, "Alice", cache.Get("A"))
	assert.Nil(t, cache.Get("B"))
	assert.Equal(t, 1, cache.Size())

	cache.Put("B", "Bob")
	cache.Put("E", "Eve")
	cache.Put("G", "Gopher")
	assert.Equal(t, 4, cache.Size())

	assert.Equal(t, "Bob", cache.Get("B"))
	assert.Equal(t, "Eve", cache.Get("E"))
	assert.Equal(t, "Gopher", cache.Get("G"))

	cache.Put("A", "Foo")
	assert.Equal(t, "Foo", cache.Get("A"))

	cache.Put("B", "Bar")
	assert.Equal(t, "Bar", cache.Get("B"))
	assert.Equal(t, "Foo", cache.Get("A"))

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
		cache.Put(k, v)
	}

	ch := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)

		// concurrent get and put
		go func() {
			defer wg.Done()

			<-ch

			for j := 0; j < 1000; j++ {
				cache.Get("A")
				cache.Put("A", "Arnie")
			}
		}()

		// concurrent iteration
		go func() {
			defer wg.Done()

			<-ch

			for j := 0; j < 50; j++ {
				it := cache.Iterator()
				for it.HasNext() {
					_ = it.Next()
				}
				it.Close()
			}
		}()
	}

	close(ch)
	wg.Wait()
}

func TestSimpleIterator(t *testing.T) {
	expected := map[string]string{
		"A": "Alice",
		"B": "Bob",
		"G": "Gopher",
		"E": "Eve",
	}

	cache, _ := NewSimpleCache(nil)

	for k, v := range expected {
		cache.Put(k, v)
	}

	got := map[string]string{}

	it := cache.Iterator()
	for it.HasNext() {
		entry := it.Next()
		got[entry.Key().(string)] = entry.Value().(string)
	}
	it.Close()
	assert.Equal(t, expected, got)

	it = cache.Iterator()
	for i := 0; i < len(expected); i++ {
		entry := it.Next()
		got[entry.Key().(string)] = entry.Value().(string)
	}
	it.Close()
	assert.Equal(t, expected, got)
}
