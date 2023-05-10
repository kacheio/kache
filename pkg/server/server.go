package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/rs/zerolog/log"
)

const (
	ServerGracefulShutdownTimeout = 5 * time.Second
)

type ctxCacheKey struct{}

var ErrMatchingTarget = fmt.Errorf("no matching target found")

// Config holds the server configuration.
type Config struct {
	HTTPAddr string `yaml:"host"`
	HTTPPort int    `yaml:"http_port"`

	Endpoints Endpoints `yaml:"endpoints"`
	Upstreams Upstreams `yaml:"upstreams"`
}

type Upstreams []*UpstreamConfig

type UpstreamConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	Path string `yaml:"path"`
}

type Endpoints map[string]*EndpointConfig

type EndpointConfig struct {
	Addr string `yaml:"addr"`
}

// Server is the reverse proxy cache.
type Server struct {
	cfg Config

	// proxy forwards requests to targets.
	proxy *httputil.ReverseProxy

	// listeners holds the downstream listeners.
	listeners Listeners

	// targets holds the upstream targets.
	targets Targets

	// cache holds the provider for storing responses.
	cache provider.Provider

	stopCh chan bool
}

// NewServer creates a new configured server.
func NewServer(cfg Config, pdr provider.Provider) (*Server, error) {
	srv := &Server{
		cfg:    cfg,
		cache:  pdr,
		stopCh: make(chan bool, 1),
	}

	// Build upstream targets.
	targets, err := NewTargets(cfg.Upstreams)
	if err != nil {
		return nil, err
	}
	srv.targets = targets

	// Build downstream listeners.
	listeners, err := NewListeners(cfg.Endpoints, srv)
	if err != nil {
		return nil, err
	}
	srv.listeners = listeners

	// Create the reverse proxy.
	proxy := &httputil.ReverseProxy{
		Director: srv.Director(),
		Transport: &http.Transport{
			// Disable SSL verification for self-signed certificates
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		ModifyResponse: srv.modifyResponse,
		ErrorHandler:   errorHandler,
	}
	srv.proxy = proxy

	return srv, nil
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

	log.Info().Msg("Starting server ...")

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
