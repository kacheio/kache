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
