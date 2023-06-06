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
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kacheio/kache/pkg/config"
	"github.com/rs/zerolog/log"
)

// Listeners holds a map of Listeners.
type Listeners map[string]*Listener

// NewListeners creates new listeners.
func NewListeners(cfg config.Listeners, handler http.Handler) (Listeners, error) {
	listeners := make(Listeners)
	for name, listener := range cfg {
		ctx := log.With().Str("listenerName", name).Logger().WithContext(context.Background())

		l, err := NewListener(ctx, listener, handler)
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

func NewListener(ctx context.Context, cfg *config.Listener, handler http.Handler) (*Listener, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
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
