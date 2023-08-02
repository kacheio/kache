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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/kacheio/kache/pkg/cache"
	"github.com/kacheio/kache/pkg/cluster"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/provider"
	"github.com/kacheio/kache/pkg/server/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	DefaultTimeout                = 30 * time.Second
	ServerGracefulShutdownTimeout = 5 * time.Second
)

var ErrMatchingTarget = fmt.Errorf("no matching target found")

// Server is the reverse proxy cache.
type Server struct {
	cfg *config.Configuration

	// proxy forwards requests to targets.
	proxy *httputil.ReverseProxy

	// listeners holds the downstream listeners.
	listeners Listeners

	// targets holds the upstream targets.
	targets Targets

	// cache is the caching backend provider.
	cache provider.Provider

	// httpcache holds the Http cache.
	httpcache *cache.HttpCache

	// cluster holds a custer connection.
	cluster cluster.Connection

	stopCh chan bool
}

// NewServer creates a new configured server.
func NewServer(
	cfg *config.Configuration,
	pdr provider.Provider,
	httpcache *cache.HttpCache,
	reg prometheus.Registerer,
) (*Server, error) {
	srv := &Server{
		cfg:       cfg,
		cache:     pdr,
		httpcache: httpcache,
		stopCh:    make(chan bool, 1),
	}

	// Build upstream targets.
	targets, err := NewTargets(cfg.Upstreams)
	if err != nil {
		return nil, err
	}
	srv.targets = targets

	// Build downstream listeners.
	listeners, err := NewListeners(cfg.Listeners, srv)
	if err != nil {
		return nil, err
	}
	srv.listeners = listeners

	if cfg.Cluster != nil {
		cc, err := cluster.NewConnection(cfg.Cluster)
		if err != nil {
			return nil, err
		}
		srv.cluster = cc
	}

	transport := middleware.NewCoalesced(
		middleware.NewCachedTransport(srv.httpcache, reg),
	)

	// Create the reverse proxy.
	proxy := &httputil.ReverseProxy{
		ErrorHandler: errorHandler,
		Director:     srv.Director(),
		Transport:    transport,
	}
	srv.proxy = proxy

	return srv, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timeout := DefaultTimeout
	http.TimeoutHandler(
		s.proxy,
		timeout,
		fmt.Sprintf("Request timeout after %v", timeout),
	).ServeHTTP(w, r)
}

// errorHandler is the proxy error handler.
func errorHandler(w http.ResponseWriter, req *http.Request, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, context.Canceled):
		ctx := req.Context()
		cErr := context.Cause(ctx)
		if errors.Is(cErr, ErrMatchingTarget) {
			status = http.StatusServiceUnavailable
			err = cErr
		} else { // client canceled request
			status = http.StatusBadGateway
		}
	case errors.Is(err, io.EOF):
		status = http.StatusBadGateway
	default: // connection error
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			status = http.StatusGatewayTimeout
		}
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			// unknown host or connection refused
			status = http.StatusServiceUnavailable
		}
	}

	logger := log.Ctx(req.Context())
	logger.Debug().Err(err).Msgf("Proxy error: status %d - %s", status, err.Error())

	w.WriteHeader(status)
	if _, wErr := w.Write([]byte(err.Error())); wErr != nil {
		logger.Debug().Err(wErr).Msg("Error writing error")
	}
}

// Director matches the incoming request to a specific target and sets
// the request object to be sent to the matched upstream server.
func (s *Server) Director() func(req *http.Request) {
	return func(req *http.Request) {
		// Find a matching target.
		target, ok := s.targets.MatchTarget(req)
		if !ok {
			log.Error().Str("request", req.URL.String()).Msg("no matching target found for request.")
			ctx, cancel := context.WithCancelCause(req.Context())
			*req = *req.WithContext(ctx)
			cancel(ErrMatchingTarget)
			return
		}
		upstream := target.upstream

		req.URL.Scheme = upstream.Scheme
		req.URL.Host = upstream.Host

		// Path is forwarded as-is.
		req.URL.Path = singleJoiningSlash(upstream.Path, req.URL.Path)

		// Pass host header
		req.Host = req.URL.Host

		// RequestURI should not be set in a HTTP client request
		req.RequestURI = ""

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "kache")
		}
	}
}

// Start starts the server.
func (s *Server) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		logger := log.Ctx(ctx)
		logger.Info().Msg("Received shutdown...")
		logger.Info().Msg("Stopping server gracefully")
		s.Stop()
	}()

	log.Debug().Msg("Starting server ...")

	s.listeners.Start()
}

// Await blocks until SIGTERM or Stop() is called.
func (s *Server) Await() {
	<-s.stopCh
}

// Stop stops the server.
func (s *Server) Stop() {
	defer log.Info().Msg("Server stopped")

	s.listeners.Stop()

	s.stopCh <- true
}

// Shutdown the server, gracefully. Should be defered after Start().
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), ServerGracefulShutdownTimeout)
	defer cancel()

	go func(ctx context.Context) {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			panic("Shutdown timeout exeeded, killing kache instance")
		}
	}(ctx)

	close(s.stopCh)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
