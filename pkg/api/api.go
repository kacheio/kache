package api

import (
	"log"
	"net/http"

	"github.com/toashd/kache/pkg/server"
)

type Config struct {
	port int
}

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
	log.Fatal(http.ListenAndServe(":1338", a.server))
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
