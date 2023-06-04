package server

import (
	"encoding/json"

	"net/http"
)

// CacheKeysHandler renders all cache keys in JSON format.
func (s *Server) CacheKeysHandler(w http.ResponseWriter, r *http.Request) {
	// keys := p.cache.Keys()

	keys := []string{}

	it := s.cache.Iterator()
	for it.HasNext() {
		keys = append(keys, it.Next().Key().(string))
	}
	it.Close()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CacheKeyPurgeHandler deletes the given key from the cache.
func (s *Server) CacheKeyPurgeHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if ok := s.cache.Delete(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
