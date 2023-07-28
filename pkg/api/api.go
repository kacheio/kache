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

package api

import (
	"fmt"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/server"
	"github.com/rs/zerolog/log"
)

const (
	defaultPrefix = "/api"
)

// API is the root API structure.
type API struct {
	// config is the API configuration.
	config config.API

	// server is the core server.
	server *server.Server

	// router is the API Router.
	router *mux.Router

	// filter is the IP filter.
	filter *IPFilter

	// prefix is the custom path prefix for all API routes.
	// Has no effect on purge and debug routes. Default is '/api'.
	prefix string
}

// New creates a new API.
func New(cfg config.API, srv *server.Server) (*API, error) {
	prefix := defaultPrefix

	filter, err := NewIPFilter(cfg.ACL)
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filter.Wrap(next.ServeHTTP).ServeHTTP(w, r)
		})
	})

	VersionHandler{}.Append(router)

	if cfg.Debug {
		DebugHandler{}.Append(router)
	}

	if len(cfg.Prefix) > 0 {
		prefix = sanitzePrefix(cfg.Prefix)
	}

	api := &API{
		config: cfg,
		server: srv,
		router: router,
		filter: filter,
		prefix: prefix,
	}
	api.createRoutes()

	return api, nil
}

// Run starts the API server.
func (a *API) Run() {
	port := fmt.Sprintf(":%d", a.config.Port)
	log.Debug().Str("port", port).Str("prefix", a.prefix).Msg("Starting API server")

	if err := http.ListenAndServe(port, a); err != nil {
		log.Fatal().Err(err).Msg("Starting API server")
	}
}

// ServeHTTP serves the API requests.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// RegisterRoute registers a new handler at the given path.
func (a *API) RegisterRoute(method string, p string, handler http.HandlerFunc) {
	a.router.HandleFunc(path.Join(a.prefix, p), handler).Methods(method)
}

// createRoutes registers the core service endpoints.
func (a *API) createRoutes() {
	// Purge cache key: curl -v -X PURGE -H 'X-Purge-Key: <cache-key>' kacheserver:PORT
	a.router.Methods("PURGE").Path("/").
		HandlerFunc(a.filter.Wrap(a.server.CacheKeyPurgeHandler))

	// List all keys in the cache.
	a.router.Methods(http.MethodGet).
		PathPrefix(path.Join(a.prefix, "/cache/keys")).
		HandlerFunc(a.server.CacheKeysHandler)

	// Render the current cache config.
	a.router.Methods(http.MethodGet).
		PathPrefix(path.Join(a.prefix, "/cache/config")).
		HandlerFunc(a.server.CacheConfigHandler)

	// Update the current cache config.
	a.router.Methods(http.MethodPut).
		PathPrefix(path.Join(a.prefix, "/cache/config/update")).
		HandlerFunc(a.server.CacheConfigUpdateHandler)

	// Invalidates a key in the cache.
	a.router.Methods(http.MethodDelete).
		PathPrefix(path.Join(a.prefix, "/cache/invalidate")).
		HandlerFunc(a.server.CacheInvalidateHandler)

	// Flush all keys from the cache.
	a.router.Methods(http.MethodDelete).
		PathPrefix(path.Join(a.prefix, "/cache/flush")).
		HandlerFunc(a.server.CacheFlushHandler)
}

// sanitizePrefix ensures that the specified prefix contains a leading and no trailing '/'.
func sanitzePrefix(prefix string) string {
	p := prefix
	if p[0] != '/' {
		p = "/" + p
	}
	if p[len(p)-1] == '/' {
		p = p[0 : len(p)-1]
	}
	return p
}
