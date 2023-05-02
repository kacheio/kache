package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/stretchr/testify/assert"
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
			//  {"test", testServer.URL, ""},
		},
	}
	cache, _ := provider.NewSimpleCache(nil)
	proxy, _ := NewServer(config, cache)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Run tests.

	assert.HTTPError(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil)
	assert.HTTPStatusCode(t, proxy.ServeHTTP, "GET", proxyServer.URL, nil, 502)
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

	// proxyURL, _ := url.Parse(proxyServer.URL)

	// // Create a test HTTP client that uses the proxy
	// testClient := &http.Client{
	// 	Transport: &http.Transport{
	// 		Proxy: http.ProxyURL(proxyURL),
	// 	},
	// }

	// // testClient := &http.Client{}

	// // Make a request through the proxy
	// resp, err := testClient.Get(proxyServer.URL + "/epi/test")
	// if err != nil {
	// 	t.Errorf("Error: %v", err)
	// }
	// defer resp.Body.Close()

	// // Check if the response status code is 200 OK
	// if resp.StatusCode != http.StatusOK {
	// 	t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	// }

	// // Check if the response body is correct
	// expectedBody := "Test Server"
	// body, _ := io.ReadAll(resp.Body)
	// if string(body) != expectedBody {
	// 	t.Errorf("Expected body %q, got %q", expectedBody, string(body))
	// }
}

// TestMultiListener
