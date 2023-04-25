package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/toashd/kache/pkg/server"
)

// Config holds the API configuration.
type Config struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path,omitempty"`
}

// API is the root API structure.
type API struct {
	cfg    Config
	server *Server
}

func New(cfg Config) (*API, error) {
	server := NewServer()

	api := &API{
		cfg:    cfg,
		server: server,
	}

	return api, nil
}

func (a *API) Run() {
	port := fmt.Sprintf(":%d", a.cfg.Port)

	path := a.cfg.Path

	log.Printf("Starting API server on %s at /%s", port, path)
	log.Fatal(http.ListenAndServe(port, a.server))
}

// RegisterProxy registers the cache HTTP service.
func (a *API) RegisterProxy(p server.Proxy) {
	a.server.Get("/api/v1/cache/keys", p.GetCacheKeysHandler)
	a.server.Get("/api/v1/cache/keys/purge", p.GetCacheKeyPurgeHandler) // /cache/keys/purge?key=....
}

type Server struct {
	router *http.ServeMux
}

func NewServer() *Server {
	return &Server{
		router: http.NewServeMux(),
	}
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
