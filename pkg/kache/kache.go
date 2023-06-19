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

package kache

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/kacheio/kache/pkg/api"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/provider"
	"github.com/kacheio/kache/pkg/server"
	"github.com/rs/zerolog/log"
)

// Kache is the root data structure for Kache.
type Kache struct {
	Config config.Configuration

	API      *api.API
	Server   *server.Server
	Provider *provider.Provider
}

// New makes a new Kache.
func New(cfg config.Configuration) (*Kache, error) {
	kache := &Kache{
		Config: cfg,
	}

	if err := kache.setupModules(); err != nil {
		return nil, err
	}

	return kache, nil
}

// initAPI initializes the public API.
func (t *Kache) initAPI() (err error) {
	t.API, err = api.New(*t.Config.API)
	if err != nil {
		return err
	}

	return nil
}

// initServer initializes the core server.
func (t *Kache) initServer() (err error) {
	t.Server, err = server.NewServer(&t.Config, *t.Provider)
	if err != nil {
		return err
	}

	// Expose HTTP endpoints
	t.API.RegisterProxy(*t.Server)

	return nil
}

// initProvider initializes the cache provider.
func (t *Kache) initProvider() error {
	p, err := provider.CreateCacheProvider("kache", *t.Config.Provider)
	if err != nil {
		return err
	}
	t.Provider = &p

	return nil
}

// setupModules initializes the modules.
func (t *Kache) setupModules() error {
	// Register module init functions
	type initFn func() error
	modules := [...]struct {
		Name string
		Init initFn
	}{
		{"API", t.initAPI},
		{"Provider", t.initProvider},
		{"Server", t.initServer},
	}

	for _, m := range modules {
		log.Debug().Msgf("Initializing %s", m.Name)
		if err := m.Init(); err != nil {
			return err
		}
	}

	return nil
}

// Run starts the Kache and its services.
func (t *Kache) Run() error {
	// Start API server
	go func() {
		t.API.Run()
	}()

	// Setup signals to gracefully shutdown in response to SIGTERM or SIGINT
	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)

	// Start core server
	t.Server.Start(ctx)
	defer t.Server.Shutdown()

	time.Sleep(120 * time.Millisecond)
	log.Info().Str("version", "0.0.1").Msg("Kache just started")

	// Wait until shutdown signal received
	t.Server.Await()

	log.Info().Msg("Shutting down")
	return nil
}
