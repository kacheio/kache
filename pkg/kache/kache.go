package kache

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/kacheio/kache/pkg/api"
	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/provider"
	"github.com/kacheio/kache/pkg/server"
	"github.com/rs/zerolog/log"
)

// Kache is the root data structure for Kache.
type Kache struct {
	Cfg config.Configuration

	API      *api.API
	Server   *server.Server
	Provider *provider.Provider
}

// New makes a new Kache.
func New(cfg config.Configuration) (*Kache, error) {
	kache := &Kache{
		Cfg: cfg,
	}

	if err := kache.setupModules(); err != nil {
		return nil, err
	}

	return kache, nil
}

// initAPI initializes the public API.
func (t *Kache) initAPI() (err error) {
	t.API, err = api.New(t.Cfg.API)
	if err != nil {
		return err
	}

	return nil
}

// initServer initializes the core server.
func (t *Kache) initServer() (err error) {
	t.Server, err = server.NewServer(t.Cfg, *t.Provider)
	if err != nil {
		return err
	}

	// Expose HTTP endpoints
	t.API.RegisterProxy(*t.Server)

	return nil
}

// initProvider initializes the cache provider.
func (t *Kache) initProvider() error {
	p, err := provider.NewSimpleCache(nil)
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
	modules := map[string]initFn{
		"API":      t.initAPI,
		"Provider": t.initProvider,
		"Proxy":    t.initServer,
	}

	for m, initFn := range modules {
		log.Info().Msgf("Initializing %s", m)
		if err := initFn(); err != nil {
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

	// Wait until shutdown signal received
	t.Server.Await()

	log.Info().Msg("Shutting down")
	return nil
}
