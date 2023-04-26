package server

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kacheio/kache/pkg/provider"
)

type ctxCacheKey struct{}

// Connfig holds the server configuration.
type Config struct {
	Upstreams []UpstreamConfig `yaml:"upstreams"`
}

type UpstreamConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
}

type Proxy struct {
	// The list of backend servers to load balance between
	backends []*url.URL

	// The reverse proxy for forwarding requests to the backends
	proxy *httputil.ReverseProxy

	// The cache for storing responses from the backends
	cache provider.Provider
}

// NewProxy creates a new proxy.
func NewProxy(config Config, cache provider.Provider) (*Proxy, error) {

	backends := config.Upstreams

	backendURLs := make([]*url.URL, len(backends))
	for i, backend := range backends {
		backendURL, err := url.Parse(backend.Addr)
		if err != nil {
			log.Fatal(err)
		}
		backendURLs[i] = backendURL
	}

	proxy := &Proxy{
		backends: backendURLs,
		cache:    cache,
	}

	proxy.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Choose a backend server to forward the request to
			target := proxy.chooseBackend()
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "kache")
			}
		},
		Transport: &http.Transport{
			// Disable SSL verification for self-signed certificates
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		ModifyResponse: proxy.modifyResponse,
	}
	return proxy, nil
}

func (p *Proxy) chooseBackend() *url.URL {
	return p.backends[time.Now().UnixNano()%int64(len(p.backends))]
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
