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
	"github.com/rs/zerolog/log"
)

// API is the root API structure.
type API struct {
	config config.API
	server *Server
}

// New creates a new API.
func New(cfg config.API) (*API, error) {
	srv := NewServer(cfg)

	api := &API{
		config: cfg,
		server: srv,
	}

	return api, nil
}

// Run starts the API server.
func (a *API) Run() {
	port := fmt.Sprintf(":%d", a.config.Port)

	path := a.config.Path

	log.Info().Msgf("Starting API server on %s at /%s", port, path)
	if err := http.ListenAndServe(port, a.server); err != nil {
		log.Fatal().Err(err).Msg("starting API server")
	}
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Server) {
	a.server.Get("/api/v1/cache/keys", p.CacheKeysHandler)
	a.server.Get("/api/v1/cache/keys/purge", p.CacheKeyPurgeHandler) // /cache/keys/purge?key=....
}

type Server struct {
	router *mux.Router
}

func NewServer(cfg config.API) *Server {
	srv := &Server{
		router: mux.NewRouter(),
	}
	if cfg.Debug {
		DebugHandler{}.Append(srv.router)
	}
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) RegisterRoute(method string, path string, handler http.HandlerFunc) {
	s.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	})
}

func (s *Server) Get(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodGet, path, handler)
}

func (s *Server) Post(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodPost, path, handler)
}

func (s *Server) Purge(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodDelete, path, handler)
}
