package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Listeners holds a map of Listeners.
type Listeners map[string]*Listener

// NewListeners creates new listeners.
func NewListeners(config Endpoints, handler http.Handler) (Listeners, error) {
	listeners := make(Listeners)
	for name, ep := range config {
		ctx := log.With().Str("listenerName", name).Logger().WithContext(context.Background())

		l, err := NewListener(ctx, ep, handler)
		if err != nil {
			return nil, err
		}

		listeners[name] = l
	}
	return listeners, nil
}

// Start starts the server listeners.
func (ls Listeners) Start() {
	for name, listener := range ls {
		ctx := log.With().Str("listenerName", name).Logger().WithContext(context.Background())
		go listener.Start(ctx)
	}
}

// Stops stops the server listeners.
func (ls Listeners) Stop() {
	// TODO
}

// Listener is the Listen server.
type Listener struct {
	listener   net.Listener
	httpServer *http.Server
}

func NewListener(ctx context.Context, config *EndpointConfig, handler http.Handler) (*Listener, error) {
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return nil, fmt.Errorf("error building listener: %w", err)
	}

	server := &http.Server{
		Handler: handler,
		// TODO: set ErrorLog
		// TODO: make timeouts configurable
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return &Listener{
		listener:   listener,
		httpServer: server,
	}, nil
}

// Start starts the listen server.
func (l *Listener) Start(ctx context.Context) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Start listening on %v", l.listener.Addr())
	err := l.httpServer.Serve(l.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error().Err(err).Msg("error while starting the listener server")
	}
}

// Shutdown stops the listener.
func (l *Listener) Shutdown(ctx context.Context) {
	// TODO
}
