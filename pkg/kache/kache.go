package kache

import (
	"log"
	"net/http"

	"github.com/toashd/kache/pkg/api"
	"github.com/toashd/kache/pkg/provider"
	server "github.com/toashd/kache/pkg/server"
)

// Config is the root config for kache.
type Config struct {
	ApplicationName string `yaml:"-"`

	API    api.Config    `yaml:"api"`
	Server server.Config `yaml:"server"`
	// Storage 	storage.Config ´yaml:"storage"´
}

// Kache is the root data structure for Kache.
type Kache struct {
	Cfg Config

	API      *api.API
	Server   *server.Proxy
	Provider *provider.Provider
}

// New makes a new Kache.
func New(cfg Config) (*Kache, error) {
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
	t.Server, err = server.NewProxy(t.Cfg.Server, *t.Provider)
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
	// Register and setup all modules here.

	// RegisterModule(name string, initFn func() error)
	type initFn func() error
	modules := map[string]initFn{
		"API":      t.initAPI,
		"Provider": t.initProvider,
		"Proxy":    t.initServer,
	}

	for m, initFn := range modules {
		log.Println("Initializing", m)
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

	// Start proxy server
	log.Println("Starting server on :1337")
	if err := http.ListenAndServe(":1337", t.Server); err != nil {
		log.Fatal(err)
	}

	// Start secure server
	tlsEnabled := false
	if tlsEnabled {
		// Start the server with TLS termination
		log.Println("Starting server on :443")
		if err := http.ListenAndServeTLS(":443", "server.crt", "server.key", t.Server); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
