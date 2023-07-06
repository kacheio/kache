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

package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kacheio/kache/pkg/cache"
	"github.com/kacheio/kache/pkg/config"
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

	cfg := &config.Configuration{
		Upstreams: []*config.Upstream{
			// {"test", testServer.URL, ""},
		},
	}
	p, _ := provider.NewSimpleCache(nil)
	c, _ := cache.NewHttpCache(nil, p)
	proxy, _ := NewServer(cfg, c)
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

	cfg := &config.Configuration{
		Upstreams: []*config.Upstream{
			{Name: "test", Addr: testServer.URL, Path: ""},
		},
	}
	p, _ := provider.NewSimpleCache(nil)
	c, _ := cache.NewHttpCache(nil, p)
	proxy, _ := NewServer(cfg, c)
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

	cfg := &config.Configuration{
		Upstreams: []*config.Upstream{
			{Name: "test 1", Addr: testServer1.URL, Path: "/bot"},
			{Name: "test 2", Addr: testServer2.URL, Path: "/api/test"},
			{Name: "test 3", Addr: testServer3.URL, Path: "/api"},
		},
	}
	p, _ := provider.NewSimpleCache(nil)
	c, _ := cache.NewHttpCache(nil, p)
	proxy, _ := NewServer(cfg, c)
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

	cfg := &config.Configuration{
		Upstreams: []*config.Upstream{
			{Name: "Backend", Addr: testServer.URL, Path: "/"},
		},
		Listeners: map[string]*config.Listener{
			"ep1": {Addr: ":1337"},
			"ep2": {Addr: ":1338"},
		},
	}
	p, _ := provider.NewSimpleCache(nil)
	c, _ := cache.NewHttpCache(nil, p)
	proxy, _ := NewServer(cfg, c)
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
