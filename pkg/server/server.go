package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/rs/zerolog/log"
)

type ctxCacheKey struct{}

// Config holds the server configuration.
type Config struct {
	HTTPAddr string `yaml:"host"`
	HTTPPort int    `yaml:"http_port"`

	Upstreams Upstreams `yaml:"upstreams"`
}

type Upstreams []*UpstreamConfig

type UpstreamConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	Path string `yaml:"path"`
}

// Server is the reverse proxy cache.
type Server struct {
	cfg Config

	// proxy forwards requests to targets.
	proxy *httputil.ReverseProxy

	// targets holds the list of upstream targets.
	targets Targets

	// cache holds the provider for storing responses.
	cache provider.Provider
}

// NewServer creates a new configured server.
func NewServer(cfg Config, pdr provider.Provider) (*Server, error) {
	srv := &Server{
		cfg:   cfg,
		cache: pdr,
	}

	// Create upstream targets.
	targets, err := NewTargets(cfg.Upstreams)
	if err != nil {
		return nil, err
	}
	srv.targets = targets

	// Create the reverse proxy.
	proxy := &httputil.ReverseProxy{
		Director: srv.Director,
		Transport: &http.Transport{
			// Disable SSL verification for self-signed certificates
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		ModifyResponse: srv.modifyResponse,
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				ctx := req.Context()
				err := context.Cause(ctx)
				if errors.Is(err, ErrMatchingTarget) {
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(err.Error()))
					return
				}
			}
		},
	}
	srv.proxy = proxy

	return srv, nil
}

var ErrMatchingTarget = fmt.Errorf("no matching target found")

// Director matches the incoming request to a specific target and sets
// the request object to be sent to the matched upstream server.
func (s *Server) Director(req *http.Request) {
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
	req.URL.Path = singleJoiningSlash(upstream.Path, req.URL.Path)
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "kache")
	}
}

// Start starts the server.
func (s *Server) Start() {
	// Start proxy server
	log.Info().Msg("Starting server on :1337")
	if err := http.ListenAndServe(":1337", s); err != nil {
		log.Fatal().Err(err).Msg("starting server")
	}
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
