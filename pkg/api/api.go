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
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/server"
	"github.com/kacheio/kache/pkg/utils/version"
	"github.com/rs/zerolog/log"
)

const (
	ErrMsgUnauthorized = "Not authorized to access the requested resource"
)

// API is the root API structure.
type API struct {
	// config is the API configuration.
	config config.API

	// router is the API Router.
	router *mux.Router

	// allowedIPs is the access control list containing
	// the IPs allowed to access the API. If the list is empty,
	// the IP filter is not active and every request is allowed.
	allowedIPs map[string]struct{}
}

// New creates a new API.
func New(cfg config.API) (*API, error) {
	api := &API{
		config:     cfg,
		router:     mux.NewRouter(),
		allowedIPs: make(map[string]struct{}),
	}
	api.createRoutes()

	if cfg.Debug {
		DebugHandler{}.Append(api.router)
	}

	// Parse allowed IPs from config.
	if ips := strings.Trim(cfg.ACL, ","); len(ips) > 0 {
		for _, ip := range strings.Split(ips, ",") {
			if ipp := net.ParseIP(strings.TrimSpace(ip)); ipp != nil {
				api.allowedIPs[ipp.String()] = struct{}{}
			}
		}
	}

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

// RegisterRoute registers a new handler at the given path.
func (a *API) RegisterRoute(method string, path string, handler http.HandlerFunc) {
	a.router.HandleFunc(path, handler).Methods(method)
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Server) {
	// List cache keys
	a.RegisterRoute(http.MethodGet, "/api/cache/keys", a.ipFilter(p.CacheKeysHandler))
	// Delete cache key; /cache/keys/purge?key=...
	a.RegisterRoute(http.MethodDelete, "/api/cache/keys/purge", a.ipFilter((p.CacheKeyDeleteHandler)))
	// TODO: implement PURGE like this:
	// curl -v -X PURGE -H 'X-Purge-Regex: ^/assets/*.css' varnishserver:6081
}

func (a *API) createRoutes() {
	a.RegisterRoute(http.MethodGet, "/api/version", a.ipFilter(version.Handler))
}

// ipFilter is a middleware that checks the original IP against the
// configured access control list and allows or blocks the request.
func (a *API) ipFilter(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(a.allowedIPs) == 0 {
			next(w, r)
		}

		// Get the origianl client IP.
		ip := originalIP(r)

		// Validate if the IP is allowed or blocked.
		if _, ok := a.allowedIPs[ip]; !ok {
			http.Error(w, ErrMsgUnauthorized, http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}

// originalIP finds the originating client IP.
func originalIP(req *http.Request) string {
	addr := ""
	// The default is the originating IP. But we try to find better
	// options because this is almost never the right IP.
	if parts := strings.Split(req.RemoteAddr, ":"); len(parts) == 2 {
		addr = parts[0]
	}
	// If we have a forwarded-for header, take the address from there.
	if xff := strings.Trim(req.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		last := addrs[len(addrs)-1]
		if ip := net.ParseIP(last); ip != nil {
			return ip.String()
		}
	}
	// Otherwise, parse the X-Real-Ip header if it exists.
	if xri := req.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}
	return addr
}
