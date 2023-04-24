package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/toashd/kache/pkg/provider"
)

func TestNewProxy(t *testing.T) {
	// Create a test HTTP server that returns a simple response
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world!"))
	}))
	defer testServer.Close()

	cache, _ := provider.NewSimpleCache(nil)

	// Create a new reverse proxy with the test server as the only backend
	proxy, _ := NewProxy([]string{testServer.URL}, cache)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	proxyURL, _ := url.Parse((proxyServer.URL))

	// Create a test HTTP client that uses the proxy
	testClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	// Make a request through the proxy
	resp, err := testClient.Get(testServer.URL)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is 200 OK
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check if the response body is correct
	expectedBody := "Hello, world!"
	body, _ := io.ReadAll(resp.Body)
	if string(body) != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, string(body))
	}
}
