package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyNoHost(t *testing.T) {
	// Setup test server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	}))
	defer testServer.Close()

	// Setup proxy server.

	config := Config{
		Upstreams: []*UpstreamConfig{
			// {"test", testServer.URL, ""},
		},
	}
	cache, _ := provider.NewSimpleCache(nil)
	proxy, _ := NewServer(config, cache)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Run tests.

	assert.HTTPError(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil)
	assert.HTTPStatusCode(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil, 503)
	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil, "no matching target found")
}

func TestProxySingleHost(t *testing.T) {
	// Setup test server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	}))
	defer testServer.Close()

	// Setup proxy server.

	config := Config{
		Upstreams: []*UpstreamConfig{
			{"test", testServer.URL, ""},
		},
	}
	cache, _ := provider.NewSimpleCache(nil)
	proxy, _ := NewServer(config, cache)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Run tests.

	assert.HTTPStatusCode(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil, 200)
	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil, "Test Server")
	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/with-path", nil, "Test Server")
}

func TestProxyMultiHost(t *testing.T) {
	// Setup test servers
	testServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server 1"))
	}))
	defer testServer1.Close()

	testServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server 2"))
	}))
	defer testServer2.Close()

	testServer3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server 3"))
	}))
	defer testServer3.Close()

	// Setup proxy server.

	config := Config{
		Upstreams: []*UpstreamConfig{
			{"test 1", testServer1.URL, "/bot"},
			{"test 2", testServer2.URL, "/api/test"},
			{"test 3", testServer3.URL, "/api"},
		},
	}
	cache, _ := provider.NewSimpleCache(nil)
	proxy, _ := NewServer(config, cache)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Run tests.

	assert.HTTPError(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil)
	assert.HTTPError(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/invalid-path", nil)
	assert.HTTPStatusCode(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/api", nil, 200)

	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/bot", nil, "Test Server 1")
	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/api", nil, "Test Server 3")
	assert.HTTPBodyContains(t, proxy.ServeHTTP, "GET", proxyServer.URL+"/api/test", nil, "Test Server 2")
}

func TestProxyMultiListener(t *testing.T) {
	// Setup test server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	}))
	defer testServer.Close()

	// Setup proxy server.

	config := Config{
		Upstreams: []*UpstreamConfig{
			{"Backend", testServer.URL, "/"},
		},
		Endpoints: map[string]*EndpointConfig{
			"ep1": {":1337"},
			"ep2": {":1338"},
		},
	}
	cache, _ := provider.NewSimpleCache(nil)
	proxy, _ := NewServer(config, cache)
	proxy.Start(context.Background())
	defer proxy.Stop()

	// Run tests.

	// Dial :1337
	resp, err := http.Get("http://localhost:1337")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "Test Server", string(body))

	// Dial :1338
	resp, err = http.Get("http://localhost:1338")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "Test Server", string(body))

	// Dial :4242 (not exposed)
	_, err = http.Get("http://localhost:4242")
	assert.Error(t, err)
}
