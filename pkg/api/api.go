package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/server"
	"github.com/rs/zerolog/log"
)

// Config holds the API configuration.
type Config struct {
	Port  int    `yaml:"port"`
	Path  string `yaml:"path,omitempty"`
	Debug bool   `yaml:"debug,omitempty"`
}

// API is the root API structure.
type API struct {
	config Config
	server *Server
}

// New creates a new API.
func New(cfg Config) (*API, error) {
	srv := NewServer(cfg)

	api := &API{
		config: cfg,
		server: srv,
	}

	return api, nil
}

// Run starts the API server.
func (a *API) Run() {
	port := fmt.Sprintf(":%d", a.config.Port)

	path := a.config.Path

	log.Info().Msgf("Starting API server on %s at /%s", port, path)
	if err := http.ListenAndServe(port, a.server); err != nil {
		log.Fatal().Err(err).Msg("starting API server")
	}
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Server) {
	a.server.Get("/api/v1/cache/keys", p.CacheKeysHandler)
	a.server.Get("/api/v1/cache/keys/purge", p.CacheKeyPurgeHandler) // /cache/keys/purge?key=....
}

type Server struct {
	router *mux.Router
}

func NewServer(cfg Config) *Server {
	srv := &Server{
		router: mux.NewRouter(),
	}
	if cfg.Debug {
		DebugHandler{}.Append(srv.router)
	}
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) RegisterRoute(method string, path string, handler http.HandlerFunc) {
	s.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	})
}

func (s *Server) Get(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodGet, path, handler)
}

func (s *Server) Post(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodPost, path, handler)
}

func (s *Server) Purge(path string, handler http.HandlerFunc) {
	s.RegisterRoute(http.MethodDelete, path, handler)
}
