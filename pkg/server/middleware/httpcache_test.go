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

package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/kacheio/kache/pkg/utils/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var s struct {
	client    http.Client
	mux       *http.ServeMux
	server    *httptest.Server
	transport *Transport
}

var (
	// Fake time source.
	ts *clock.EventTime
)

// currentTime returns the fake time.
func currentTime() time.Time {
	return ts.Now()
}

// advanceTime advances time forward by the specified duration.
func advanceTime(d time.Duration) {
	ts.Update(ts.Now().Add(d))
}

func setup(t *testing.T) {
	ts = clock.NewEventTimeSource()
	ts.Update(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))

	p, _ := provider.NewSimpleCache(nil)
	tp := NewCachedTransport(p)
	tp.currentTime = currentTime

	client := http.Client{Transport: tp}

	s.transport = tp
	s.client = client

	s.mux = http.NewServeMux()
	s.server = httptest.NewServer(s.mux)
}

func teardown(t *testing.T) {
	s.server.Close()
}

func TestMissInsertHit(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_miss_insert_hit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", currentTime().Format(http.TimeFormat))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Content-length", "2")
		_, _ = w.Write([]byte("42"))
	}))

	req, err := http.NewRequest("GET", s.server.URL+"/test_miss_insert_hit", nil)
	require.NoError(t, err)

	// Send first request, get response from upstream.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "", resp.Header.Get(XCache))
		assert.Equal(t, "", resp.Header.Get("Age"))
		assert.Equal(t, "42", string(body))
	}

	// Advance time.
	advanceTime(10 * time.Second)

	// Send second request, get response from cache.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "HIT", resp.Header.Get(XCache))
		assert.Equal(t, "10", resp.Header.Get("Age"))
		assert.Equal(t, "42", string(body))
	}
}

func TestExpiredValidated(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_expired_validated", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("if-none-match") == "abc123" {
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Date", currentTime().Format(http.TimeFormat))
		w.Header().Set("Cache-Control", "max-age=10")
		w.Header().Set("Content-length", "2")
		w.Header().Set("Etag", "abc123")
		_, _ = w.Write([]byte("42"))
	}))

	req, err := http.NewRequest("GET", s.server.URL+"/test_expired_validated", nil)
	require.NoError(t, err)

	// Send first request, get response from upstream.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "", resp.Header.Get("Age"))
		assert.Equal(t, "42", string(body))
	}

	// Advance time for the cached response to be stale (expired).
	// Ensure response date header gets updated with the 304 date.
	advanceTime(11 * time.Second)

	// Send second request, cached response should be validated, then served.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check that the served response is the cached response.
		assert.Equal(t, "HIT", resp.Header.Get(XCache))

		// A response that has been validated should not contain an Age header
		// as it is equivalent to a freshly served response from the origin.
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time to get a fresh cached response.
	advanceTime(1 * time.Second)

	// Send third request. The cached response was validated, thus it should have an Age header.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "1", resp.Header.Get("Age"))
	}
}

func TestExpiredFetchedNewResponse(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_expired_fetched_new_response",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("if-none-match") == "a1" {
				w.Header().Set("Cache-Control", "max-age=10")
				w.Header().Set("Etag", "a2")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("aaaaaaaaa"))
				return
			}
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "max-age=10")
			w.Header().Set("Content-length", "1")
			w.Header().Set("Etag", "a1")
			_, _ = w.Write([]byte("a"))
		}))

	req, err := http.NewRequest("GET", s.server.URL+"/test_expired_fetched_new_response", nil)
	require.NoError(t, err)

	// Send first request, and get response from upstream.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time for the cached response to be stale (expired).
	// Ensure response date header gets updated with the 304 date.
	advanceTime(11 * time.Second)

	// Send second request, attempted validation of the cached response should fail. The new
	// response should be served.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		var buf bytes.Buffer
		_, err = io.Copy(&buf, resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check that the served response is the new response.
		assert.Equal(t, "aaaaaaaaa", buf.String())

		// Check that age header does not exist as this is not a cached response.
		assert.Equal(t, "", resp.Header.Get("Age"))
		assert.Equal(t, "", resp.Header.Get(XCache))
	}
}

func TestServeHeadRequest(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_serve_head_request",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.Header().Set("Content-length", "3")
			w.Header().Set("Etag", "a1")
			_, _ = w.Write([]byte("aaa"))
		}))

	req, err := http.NewRequest("HEAD", s.server.URL+"/test_serve_head_request", nil)
	require.NoError(t, err)

	// Send first request, and get response from upstream.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time, to verify the original date header is preserved.
	advanceTime(10 * time.Second)

	// Send second request, and get response from upstream, since head requests are not stored
	// in cache.
	{
		resp, err := s.client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}
}

func TestServeHeadFromCacheAfterGetRequest(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_serve_head_from_cache_after_get",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.Header().Set("Content-length", "3")
			w.Header().Set("Etag", "a3")
			_, _ = w.Write([]byte("aaa"))
		}))

	url := s.server.URL + "/test_serve_head_from_cache_after_get"

	// Send GET request, and get response from upstream.
	{
		resp, err := s.client.Get(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "aaa", string(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time, to verify the original date header is preserved.
	advanceTime(10 * time.Second)

	// Send HEAD request, and get response from cache.
	{
		resp, err := s.client.Head(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))
		assert.Equal(t, "10", resp.Header.Get("Age"))
		assert.Equal(t, "HIT", resp.Header.Get(XCache))
	}
}

func TestServeGetFromUpstreamAfterHead(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_serve_get_from_upstream_after_head",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.Header().Set("Content-length", "3")
			w.Header().Set("Etag", "a1")
			_, _ = w.Write([]byte("aaa"))
		}))

	url := s.server.URL + "/test_serve_get_from_upstream_after_head"

	// Send HEAD request, and get response from upstream.
	{
		resp, err := s.client.Head(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Send GET request, and get response from upstream.
	{
		resp, err := s.client.Get(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "aaa", string(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}
}

func TestServeGetFollowedByHead304WithValidation(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_serve_get_head_validation",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("if-none-match") == "abc123" {
				w.Header().Set("Date", currentTime().Format(http.TimeFormat))
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "max-age=10")
			w.Header().Set("Content-length", "3")
			w.Header().Set("Etag", "abc123")
			_, _ = w.Write([]byte("aaa"))
		}))

	url := s.server.URL + "/test_serve_get_head_validation"

	// Send GET request, and get response from upstream.
	{
		resp, err := s.client.Get(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "aaa", string(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time for the cached response to be stale (expired).
	// Ensure response date header gets updated with the 304 date.
	advanceTime(11 * time.Second)

	// Send HEAD request, the cached response should be validated, then served.
	{
		resp, err := s.client.Head(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))
		assert.Equal(t, "HIT", resp.Header.Get(XCache))

		// A response that has been validated should not contain an Age header.
		assert.Equal(t, "", resp.Header.Get("Age"))
	}
}

func TestServeGetFollowedByHead200WithValidation(t *testing.T) {
	setup(t)
	t.Cleanup(func() { teardown(t) })

	s.mux.HandleFunc("/test_serve_get_head_200_validation",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("if-none-match") == "a3" {
				w.Header().Set("Date", currentTime().Format(http.TimeFormat))
				w.Header().Set("Cache-Control", "max-age=10")
				w.Header().Set("Content-length", "9")
				w.Header().Set("Etag", "a9")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("aaaaaaaaa"))
				return
			}
			w.Header().Set("Date", currentTime().Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "max-age=10")
			w.Header().Set("Content-length", "3")
			w.Header().Set("Etag", "a3")
			_, _ = w.Write([]byte("aaa"))
		}))

	url := s.server.URL + "/test_serve_get_head_200_validation"

	// Send GET request, and get response from upstream.
	{
		resp, err := s.client.Get(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "aaa", string(body))
		assert.Equal(t, "", resp.Header.Get("Age"))
	}

	// Advance time for the cached response to be stale (expired).
	advanceTime(11 * time.Second)

	// Send HEAD request, attempted validation of the cached response should fail.
	{
		resp, err := s.client.Head(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 0, len(body))

		// Ensure this is not a cached response.
		assert.Equal(t, "", resp.Header.Get("Age"))
		assert.Equal(t, "", resp.Header.Get(XCache))
	}
}
