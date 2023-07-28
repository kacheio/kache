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

package server

import (
	"context"
	"encoding/json"
	"path"

	"net/http"

	"github.com/kacheio/kache/pkg/cache"
	"gopkg.in/yaml.v3"
)

// CacheKeysHandler renders all cache keys in JSON format.
func (s *Server) CacheKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys := s.cache.Keys(r.Context(), "")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CacheKeyPurgeHandler handles a PURGE request and deletes the given key from the
// cache. The cache key is obtained from a custom request header 'X-Purge-Key'.
// When running in a cluster a invalidation signal gets broadcasted to other instances.
func (s *Server) CacheKeyPurgeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PURGE" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// TODO: implement regex header, e.g. 'X-Purge-Regex: ^/assets/*.css'.
	key := r.Header.Get("X-Purge-Key")
	if err := s.cache.Purge(context.Background(), key); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.broadcastPurge(r)

	w.WriteHeader(http.StatusOK)
}

// CacheInvalidateHandler handles the DELETE request to invalidate the provided key in the cache.
// When running in a cluster, this does not broadcast to other kache instances.
func (s *Server) CacheInvalidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	key := r.Header.Get("X-Purge-Key")
	if err := s.cache.Purge(context.Background(), key); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CacheFlushHandler handles the DELETE request to flush all keys from the cache.
func (s *Server) CacheFlushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if err := s.cache.Flush(context.Background()); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CacheConfigHandler renders the current cache config.
func (s *Server) CacheConfigHandler(w http.ResponseWriter, r *http.Request) {
	config := s.httpcache.Config()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// CacheConfigHandler renders the current cache config.
func (s *Server) CacheConfigUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var c cache.HttpCacheConfig
	dec := yaml.NewDecoder(r.Body)
	if err := dec.Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.httpcache.UpdateConfig(&c)
	w.WriteHeader(http.StatusOK)
}

// broadcastPurge broadcasts a purge request to other nodes in the cluster.
func (s *Server) broadcastPurge(req *http.Request) {
	if s.cfg.Cluster == nil || !s.cfg.Provider.Layered {
		return
	}

	// TODO: fix this prefix hack.
	prefix := "/api"
	if len(s.cfg.API.Prefix) > 0 {
		prefix = s.cfg.API.Prefix
	}

	s.cluster.Broadcast(req, "api", path.Join(prefix, "/cache/invalidate"))
}
