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

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/server"
	"github.com/kacheio/kache/pkg/utils/version"
	"github.com/rs/zerolog/log"
)

const (
	defaultPrefix = "/api"
)

// API is the root API structure.
type API struct {
	// config is the API configuration.
	config config.API

	// router is the API Router.
	router *mux.Router

	// filter is the IP filter.
	filter *IPFilter

	// prefix is the custom path prefix for all API routes.
	// Has no effect on purge and debug routes. Default is '/api'.
	prefix string
}

// New creates a new API.
func New(cfg config.API) (*API, error) {
	api := &API{
		config: cfg,
		router: mux.NewRouter(),
		filter: NewIPFilter(cfg.ACL),
		prefix: defaultPrefix,
	}

	if len(cfg.Prefix) > 0 {
		api.prefix = sanitzePrefix(cfg.Prefix)
	}

	if cfg.Debug {
		DebugHandler{}.Append(api.router)
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

func (a *API) createRoutes() {
	a.RegisterRoute(http.MethodGet, a.prefix+"/version", a.filter.Wrap(version.Handler))
}

// RegisterRoute registers a new handler at the given path.
func (a *API) RegisterRoute(method string, path string, handler http.HandlerFunc) {
	a.router.HandleFunc(path, handler).Methods(method)
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Server) {
	// List all keys in the cache.
	a.RegisterRoute(http.MethodGet, a.prefix+"/cache/keys", a.filter.Wrap(p.CacheKeysHandler))
	// Delete cache key; /cache/keys/purge?key=...
	a.RegisterRoute(http.MethodDelete, a.prefix+"/cache/keys/purge", a.filter.Wrap((p.CacheKeyDeleteHandler)))
	// Purge cache key: curl -v -X PURGE -H 'X-Purge-Key: <cache-key>' kacheserver:PORT
	a.RegisterRoute("PURGE", "/", a.filter.Wrap(p.CacheKeyPurgeHandler))
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
