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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kacheio/kache/pkg/api"
	"github.com/kacheio/kache/pkg/cache"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/provider"
	"github.com/kacheio/kache/pkg/server"
	"github.com/kacheio/kache/pkg/utils/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Kache is the root data structure for Kache.
type Kache struct {
	Config *config.Configuration
	loader *config.Loader

	Registerer prometheus.Registerer

	API      *api.API
	Server   *server.Server
	Cache    *cache.HttpCache
	Provider *provider.Provider
}

// New makes a new Kache.
func New(loader *config.Loader, registerer prometheus.Registerer) (*Kache, error) {
	kache := &Kache{
		loader: loader,
		Config: loader.Config(),
		Registerer: registerer,
	}

	if err := kache.setupModules(); err != nil {
		return nil, err
	}

	return kache, nil
}

// initAPI initializes the public API.
func (t *Kache) initAPI() (err error) {
	t.API, err = api.New(*t.Config.API, t.Server)
	if err != nil {
		return err
	}
	return nil
}

// initServer initializes the core server.
func (t *Kache) initServer() (err error) {
	t.Server, err = server.NewServer(t.Config, *t.Provider, t.Cache)
	if err != nil {
		return err
	}
	return nil
}

// initHTTPCache initializes the HTTP cache with a caching provider.
func (t *Kache) initHTTPCache() error {
	c, err := cache.NewHttpCache(t.Config.HttpCache, *t.Provider)
	if err != nil {
		return err
	}
	t.Cache = c
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
		{"Provider", t.initProvider},
		{"HTTPCache", t.initHTTPCache},
		{"Server", t.initServer},
		{"API", t.initAPI},
	}

	for _, m := range modules {
		log.Debug().Msgf("Initializing %s", m.Name)
		if err := m.Init(); err != nil {
			return err
		}
	}

	return nil
}

// reloadConfig reloads the config, triggered by SIGHUP event.
func (t *Kache) reloadConfig(ctx context.Context) error {
	reloaded, err := t.loader.Load(ctx)
	if err != nil {
		return err
	}
	if !reloaded {
		log.Info().Msg("Config not reloaded, no changes detected")
		return nil
	}
	t.applyConfig()
	log.Info().Msg("Config reloaded")
	return nil
}

// applyConfig applies the config to modules.
func (t *Kache) applyConfig() {
	t.Config = t.loader.Config()
	t.Cache.UpdateConfig(t.Config.HttpCache)
}

// Run starts the Kache and its services.
func (t *Kache) Run() error {
	// Watch and reload config.
	if t.loader.AutoReload() {
		if err := t.loader.Watch(context.Background()); err != nil {
			return err
		}
		defer t.loader.Close()
		go func() {
			for changed := range t.loader.Events {
				if !changed {
					continue
				}
				log.Info().Msg("Config file changed, reloading config")
				t.applyConfig()
			}
		}()
	}

	// Reload config on SIGHUP.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)
	stop := make(chan struct{})
	defer func() {
		close(stop)
	}()
	go func() {
		for {
			select {
			case s := <-signals:
				if s == syscall.SIGHUP {
					log.Info().Msg("Received SIGHUP, reloading config")
					if err := t.reloadConfig(context.Background()); err != nil {
						log.Error().Err(err).Msg("Error reloading config")
					}
					continue
				}
			case <-stop:
				return
			}
		}
	}()

	// Start API server
	go func() {
		t.API.Run()
	}()

	// Setup signals to gracefully shutdown on SIGTERM or SIGINT
	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)

	// Start core server
	t.Server.Start(ctx)
	defer t.Server.Shutdown()

	time.Sleep(120 * time.Millisecond)
	log.Info().Str("version", version.Info()).Msg("Kache just started")

	// Wait until shutdown signal received
	t.Server.Await()

	log.Info().Msg("Shutting down")
	return nil
}
