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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upstream is the test transport holding a request counter that gets
// incremented on each request hitting the upstream. Depending on the 
// `Coalesced` http request header, it either responds instantly or 
// supends its operation until signaled to resume.
type upstream struct {
	wait chan struct{}

	mu   sync.Mutex
	hits map[string]int
}

func (t *upstream) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.hits[req.URL.Path] += 1
	t.mu.Unlock()

	if _, ok := req.Header["Coalesced"]; ok {
		<-t.wait
	}

	return &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(req.URL.Path)),
	}, nil
}

func TestCoalescedRoundTrip(t *testing.T) {
	// For concurrent requests to the same resource,
	// only 1 request should hit the upstream.

	upstream := &upstream{
		hits: make(map[string]int),
		wait: make(chan struct{}),
	}
	coalesced := NewCoalesced(upstream)

	n := 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := doRequest(t, coalesced, "/coalesced", true)
			require.NoError(t, err)
		}()
	}

	// Add some grace time to wait for all requests to be made.
	time.Sleep(100 * time.Millisecond)

	// Requests to other resources should not be blocked.
	_, err := doRequest(t, coalesced, "/non-coalesced", false)
	require.NoError(t, err)

	// Resume upstream processing.
	close(upstream.wait)

	// Singleflight requests should all hit the upstream.
	_, err = doRequest(t, coalesced, "/non-coalesced", false)
	require.NoError(t, err)

	wg.Wait()

	expected := map[string]int{
		"/coalesced":     1,
		"/non-coalesced": 2,
	}
	assert.Equal(t, expected, upstream.hits)
}

//nolint:revive
func doRequest(t *testing.T, rt http.RoundTripper, path string, coalesced bool) (*http.Response, error) {
	u, err := url.Parse("http://test.com" + path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if coalesced {
		req.Header.Set("coalesced", "1")
	}

	done := make(chan struct{})
	var resp *http.Response
	go func() {
		defer close(done)
		resp, err = rt.RoundTrip(req)
		if err == nil {
			if body, ioErr := io.ReadAll(resp.Body); ioErr != nil {
				err = ioErr
			} else if string(body) != path {
				err = errors.New("unexpected response value")
			}
		}
	}()

	select {
	case <-time.After(3 * time.Second):
		return nil, fmt.Errorf("request for %q timed out", path)
	case <-done:
		return resp, err
	}
}
