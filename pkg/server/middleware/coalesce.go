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
	"bufio"
	"bytes"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/rs/zerolog/log"
)

// requestCoalescer allows concurrent requests for the same URL to share
// a single upstream request and response. Once a request is received for
// processing for the first time, it will execute a HTTP transaction, obtaining
// a response for a given request. If a duplicate request (same URL) comes in
// at the same time, those threads will wait for the original to complete and receive
// the same results. The original request's response is copied into the waiting
// threads, which will then resume their execution.
type requestCoalescer struct {
	sync.Mutex
	inflights map[string]*call

	// next holds a RoundTripper that executes a single HTTP
	// transaction, obtaining a response for a given request.
	next http.RoundTripper
}

// call is an in-flight or completed singleflight request.
type call struct {
	*sync.Cond // rendezvous point for goroutines.

	// coalesced indicates if there are any calls waiting
	// for the initial in-flight request's response.
	coalesced bool
	resp      []byte
	err       error
}

// NewCoalesced returns a coalesced http roundtripper.
func NewCoalesced(next http.RoundTripper) http.RoundTripper {
	return &requestCoalescer{
		inflights: make(map[string]*call),
		next:      next,
	}
}

// RoundTrip executes and returns the given request and coalesces
// concurrent GET requests. It ensures that only one execution call is
// in-flight for a given key at a time. Following duplicate or similar (same URL)
// requests are blocked until the original request completes. The resulting
// response is shared with all waiting requests.
func (coalescer *requestCoalescer) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only coalesce GET requests.
	if req.Method != http.MethodGet {
		return coalescer.next.RoundTrip(req)
	}

	key := req.URL.String()
	coalescer.Lock()
	inflight, ok := coalescer.inflights[key]
	if ok {
		// At this point a similar request is already in-flight. Wait for response.

		// Close the request body, as this request is put on hold here waiting for
		// the initial request's response.
		if req.Body != nil {
			defer req.Body.Close()
		}

		// Lock the inflight request to prevent it from being removed before
		// any waiting call routines are done processing.
		inflight.L.Lock()

		// Unlock the coalescer to allow other requests to use it.
		// Important: the inflight call must be locked before to
		// guarantee no other thread can delete it.
		coalescer.Unlock()

		// Mark that there is at least one call awaiting a shared response.
		inflight.coalesced = true

		// Suspend execution of current thread until inflight request is processed.
		inflight.Wait()

		// Resume execution of waiting thread (woken up by initial thread broadcast).
		inflight.L.Unlock()

		// Initial request completed.
		if inflight.err != nil {
			return nil, inflight.err
		}
		// Copy initial response to waiting caller.
		resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(inflight.resp)), nil)
		if err != nil {
			log.Error().Err(err).Str("key", key).Msg("Error loading response")
			return nil, err
		}
		return resp, nil
	}

	// No similar request in flight (common case).

	// Register a new call and execute the request.
	inflight = &call{Cond: sync.NewCond(&sync.Mutex{})}
	coalescer.inflights[key] = inflight
	coalescer.Unlock()

	// Execute e2e request.
	resp, err := coalescer.next.RoundTrip(req)

	// Upstream response received. Remove inflight request from the map.
	// Important: Removing the inflight request before waking up waiting
	// calls is crucial, otherwise race conditions and memory leaks occur.
	coalescer.Lock()
	delete(coalescer.inflights, key)
	coalescer.Unlock()

	// Wake up any suspended callers waiting for this response.
	inflight.L.Lock()
	if inflight.coalesced {
		if err != nil {
			inflight.resp, inflight.err = nil, err
		} else {
			// Copy the response before releasing it to waiter(s).
			inflight.resp, inflight.err = httputil.DumpResponse(resp, true)
		}
		// Wake up all waiting routines.
		inflight.Broadcast()
	}
	inflight.L.Unlock()

	if err != nil {
		log.Error().Err(err).Str("key", key).Send()
		return nil, err
	}

	return resp, nil
}
