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

// API is the root API structure.
type API struct {
	// config is the API configuration.
	config config.API

	// router is the API Router.
	router *mux.Router

	// filter is the IP filter.
	filter *IPFilter
}

// New creates a new API.
func New(cfg config.API) (*API, error) {
	api := &API{
		config: cfg,
		router: mux.NewRouter(),
	}

	if len(cfg.ACL) > 0 {
		api.filter = NewIPFilter(cfg.ACL)
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
	path := a.config.Path
	log.Debug().Str("port", port).Str("prefix", path).Msg("Starting API server")

	if err := http.ListenAndServe(port, a); err != nil {
		log.Fatal().Err(err).Msg("Starting API server")
	}
}

// ServeHTTP serves the API requests.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *API) createRoutes() {
	a.RegisterRoute(http.MethodGet, "/api/version", a.filter.Wrap(version.Handler))
}

// RegisterRoute registers a new handler at the given path.
func (a *API) RegisterRoute(method string, path string, handler http.HandlerFunc) {
	a.router.HandleFunc(path, handler).Methods(method)
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Server) {
	// List cache keys
	a.RegisterRoute(http.MethodGet, "/api/cache/keys", a.filter.Wrap(p.CacheKeysHandler))
	// Delete cache key; /cache/keys/purge?key=...
	a.RegisterRoute(http.MethodDelete, "/api/cache/keys/purge", a.filter.Wrap((p.CacheKeyDeleteHandler)))
	// TODO: implement PURGE like this:
	// curl -v -X PURGE -H 'X-Purge-Regex: ^/assets/*.css' varnishserver:6081
}
