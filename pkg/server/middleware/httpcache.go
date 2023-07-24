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
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/kacheio/kache/pkg/cache"
	"github.com/rs/zerolog/log"
)

// Transport is the http filter implementing the http caching logic.
type Transport struct {
	// The RoundTripper interface actually used to make requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// Cache is the http cache.
	Cache *cache.HttpCache

	// currentTime holds the time source.
	currentTime func() time.Time
}

// NewTransport returns a new Transport with the provided Cache implementation.
func NewCachedTransport(c *cache.HttpCache) *Transport {
	// Configure custom transport.
	// TODO: make some fields configurable via config.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &Transport{Transport: transport, Cache: c, currentTime: time.Now}
}

// RoundTrip issues a http roundtrip and applies the http caching logic.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ctx := req.Context()

	if t.Cache.IsExcludedPath(req.URL.Path) {
		log.Debug().Interface("path", req.URL.Path).Str("x-cache", "PASS").Msg("Ignoring excluded path")
		return t.send(req)
	}

	if t.Cache.IsExcludedHeader(req.Header) {
		log.Debug().Interface("header", req.Header).Str("x-cache", "PASS").Msg("Ignoring excluded header")
		return t.send(req)
	}

	if !cache.IsCacheableRequest(req) {
		log.Debug().Interface("header", req.Header).Str("x-cache", "PASS").Msg("Ignoring uncachable request")
		if t.Cache.MarkCachedResponses() {
			req.Header.Set(t.Cache.XCacheHeader(), cache.MISS)
		}
		return t.send(req)
	}

	lookup := cache.NewLookupRequest(req, t.currentTime(), t.Cache.Strict())
	cacheKey := lookup.Key.String()

	log.Debug().Str("cache-key", cacheKey).Msg("Lookup response")
	cached := t.Cache.FetchResponse(ctx, *lookup)

	switch cached.Status {
	case cache.EntryOk:
		return t.handleCacheHit(cacheKey, cached)

	case cache.EntryRequiresValidation:
		if t.Cache.MarkCachedResponses() {
			cached.Header().Set(t.Cache.XCacheHeader(), cache.HIT)
		}
		log.Debug().Str("cache-key", cacheKey).Interface("header", cached.Header()).
			Msg("Cache HIT with validation")
		req = t.injectValidationHeaders(lookup.Request, cached.Header())

	case cache.EntryInvalid:
		log.Debug().Str("cache-key", cacheKey).Str("x-cache", "MISS").Msg("Calling upstream")

	case cache.EntryLookupError:
		log.Error().Str("cache-key", cacheKey).Str("x-cache", "ERROR").Msg("Error while retrieving the response")
	}

	// Send request to upstream.
	resp, err = t.send(req)
	if err != nil {
		log.Error().Err(err).Msgf("RoundTrip: error: %v", err)
		return resp, err
	}

	shouldUpdateCachedEntry := true
	if resp.StatusCode == http.StatusNotModified {
		// If the 304 response contains a strong validator (etag) that does not match
		// the cached response, the cached response should not be updated.
		resEtag := resp.Header.Get(cache.HeaderEtag)
		cacEtag := cached.Header().Get(cache.HeaderEtag)
		shouldUpdateCachedEntry = (resEtag == "" || (cacEtag != "" && cacEtag == resEtag))

		// A response that has been validated should not contain an Age header
		// as it is equivalent to a freshly served response from the origin.
		cached.Header().Del(cache.HeaderAge)

		// Add any missing headers from the 304 to the cached response.
		cached.UpdateHeader(resp.Header)

		_ = resp.Body.Close()
		resp = cached.Response()
	}

	// Set or update custom cache control header.
	updateCacheControl(resp.Header, t.Cache.DefaultCacheControl(), t.Cache.ForceCacheControl())

	// Check cacheability depending on cache mode.
	cacheable := true
	if t.Cache.Strict() {
		cacheable = (cache.IsCacheableResponse(resp) && !lookup.ReqCacheControl.NoStore)
	}

	// Store new or update validated response.
	if cacheable && shouldUpdateCachedEntry && lookup.Request.Method != "HEAD" &&
		!t.Cache.IsExcludedContent(resp.Header.Get("Content-Type"), resp.ContentLength) {
		t.Cache.StoreResponse(context.Background(), lookup, resp, t.currentTime())
	} else {
		t.Cache.Delete(ctx, lookup)
	}

	return resp, nil
}

// handleCacheHit handles a cache hit and sends the cached response downstream.
func (t *Transport) handleCacheHit(key string, cached *cache.LookupResult) (*http.Response, error) {
	log.Debug().Str("cache-key", key).Interface("header", cached.Header()).Str("x-cache", "HIT").Send()
	cached.Header().Set(t.Cache.XCacheHeader(), cache.HIT)
	return cached.Response(), nil
}

// send issues an upstream request.
func (t *Transport) send(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}

// injectValidationHeaders injects validation headers.
// It either returns the original request or a modified fork.
func (t *Transport) injectValidationHeaders(ireq *http.Request, header http.Header) *http.Request {
	req := ireq // req is either the original request, or a modified fork.

	// forkReq forks req into a shallow clone of ireq
	// with copied headers the first time it's called.
	forkReq := func() {
		if ireq == req {
			req = new(http.Request)
			*req = *ireq // shallow clone
			req.Header = make(http.Header)
			for k, vv := range ireq.Header {
				req.Header[k] = vv
			}
		}
	}

	// Inject validation headers.
	if etag := header.Get(cache.HeaderEtag); etag != "" {
		forkReq()
		req.Header.Set(cache.HeaderIfNoneMatch, etag)
	}
	if lastModified := header.Get(cache.HeaderLastModified); lastModified != "" {
		forkReq()
		req.Header.Set(cache.HeaderIfModifiedSince, lastModified)
	} else {
		// Fallback to Date header if Last-Modified is missing.
		forkReq()
		date := header.Get(cache.HeaderDate)
		req.Header.Set(cache.HeaderLastModified, date)
	}

	return req
}

// updateCacheControl sets or updates the cache-control header.
func updateCacheControl(header http.Header, val string, force bool) {
	overwrite := force
	if _, presentcc := header["Cache-Control"]; !presentcc || overwrite {
		header["Cache-Control"] = []string{val}
	}
}
