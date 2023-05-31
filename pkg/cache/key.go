package cache

import (
	"fmt"
	"net/http"
	"net/url"

	xxhash "github.com/cespare/xxhash/v2"
)

// Key is the cache key.
type Key struct {
	ClusterName string
	Host        string
	Path        string
	Query       string
	Scheme      string
}

// NewFromRequest creates a cache key from the given request.
func NewKeyFromRequst(req *http.Request) *Key {
	key := &Key{
		ClusterName: "kache-",
		Host:        req.Host,
		Path:        req.URL.Path,
		Query:       req.URL.Query().Encode(),
		Scheme:      req.URL.Scheme,
	}
	if key.Scheme == "" {
		if req.TLS == nil {
			key.Scheme = "http"
		} else {
			key.Scheme = "https"
		}
	}
	return key
}

// String encodes the cache key as string.
func (k Key) String() string {
	url := url.URL{
		Scheme:   k.Scheme,
		Host:     k.Host,
		Path:     k.Path,
		RawQuery: k.Query,
	}
	return fmt.Sprintf("%s%s", k.ClusterName, url.String())
}

// Hash produces a stable hash of key.
func (k Key) Hash() uint64 {
	return StableHashKey(k)
}

// Produces a hash of key that is consistent across restarts, architectures,
// builds, and configurations. Caches that store persistent entries based on a
// 64-bit hash should (but are not required to) use stableHashKey.
func StableHashKey(k Key) uint64 {
	// TODO(toashd): performance; use proto marshal instead?
	return xxhash.Sum64([]byte(k.String()))
}
