package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Listeners holds a map of Listeners.
type Listeners map[string]*Listener

// NewListeners creates new listeners.
func NewListeners(config Endpoints, handler http.Handler) (Listeners, error) {
	listeners := make(Listeners)
	for name, endpoint := range config {
		ctx := log.With().Str("listenerName", name).Logger().WithContext(context.Background())

		l, err := NewListener(ctx, endpoint, handler)
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
	var wg sync.WaitGroup

	for name, listener := range ls {
		wg.Add(1)

		go func(listenerName string, listener *Listener) {
			defer wg.Done()

			logger := log.With().Str("listenerName", listenerName).Logger()
			listener.Shutdown(logger.WithContext(context.Background()))

			logger.Debug().Msg("Listener stopped")
		}(name, listener)
	}

	wg.Wait()
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
		logger.Error().Err(err).Msg("Error while starting the listener server")
	}
}

// Shutdown stops the listener.
func (l *Listener) Shutdown(ctx context.Context) {
	logger := log.Ctx(ctx)

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	logger.Debug().Msgf("Waiting %s seconds before closing listeners", timeout)

	var wg sync.WaitGroup

	shutdown := func(server *http.Server) {
		defer wg.Done()

		err := server.Shutdown(ctx)
		if err == nil {
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			logger.Debug().Err(err).Msg("Timout exceeded while closing listener server")
			if err = server.Close(); err != nil {
				logger.Error().Err(err).Send()
			}
			return
		}
		logger.Error().Err(err).Msg("Failed to shut down listener server")

		if err = server.Close(); err != nil {
			logger.Error().Err(err).Send()
		}
	}

	if l.httpServer != nil {
		wg.Add(1)
		go shutdown(l.httpServer)
	}

	wg.Wait()
	cancel()
}
