package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/toashd/kache/pkg/cache"
)

// ServeHTTP is the main handler, serving the response
// from cache, if it exists, or bypassing the request upstream.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Check if the response is already in the cache

	log.Println("serving request from proxy: ", r)

	cacheKey, _ := cache.NewKeyFromRequst(r)
	log.Println("Cache Key: ", cacheKey)

	if cached := p.cache.Get(cacheKey.String()); cached != nil {

		log.Println("serving from cache (hit)...")
		log.Println(cached)

		res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(cached.(*cache.Entry).Body)), r)
		if err != nil {
			log.Println(err)
		}

		// If the response is in the cache and hasn't been modified, serve it from the cache
		for k, v := range res.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(res.StatusCode)
		if _, err := io.Copy(w, res.Body); err != nil {
			log.Println(err)
		}
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxCacheKey{}, cacheKey.String())

	r = r.WithContext(ctx)

	// If the response isn't in the cache, forward the request to the backend servers
	p.proxy.ServeHTTP(w, r)
}

// modifyResponse modifies the response from the upstream server.
func (p *Proxy) modifyResponse(res *http.Response) error {
	log.Println("Upstream response: ", res)

	if response, err := httputil.DumpResponse(res, true); err == nil {
		log.Println("add response to cache")
		cacheKey := res.Request.Context().Value(ctxCacheKey{}).(string)

		entry := &cache.Entry{
			Body:         response,
			LastModified: time.Now(),
		}
		p.cache.Put(cacheKey, entry)

		log.Println("response added to cache: ", p.cache.Size(), p.cache)
	} else {
		log.Println(err)
	}

	return nil
}

/// Registered API handlers

// GetCacheKeysHandler renders all cache keys in JSON format.
func (p *Proxy) GetCacheKeysHandler(w http.ResponseWriter, r *http.Request) {
	// keys := p.cache.Keys()

	keys := []string{}

	it := p.cache.Iterator()
	for it.HasNext() {
		keys = append(keys, it.Next().Key().(string))
	}
	it.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// GetCacheKeyPurgeHandler deletes the given key from the cache.
func (p *Proxy) GetCacheKeyPurgeHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if ok := p.cache.Delete(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
