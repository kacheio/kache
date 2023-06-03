package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/kacheio/kache/pkg/cache"
	"github.com/kacheio/kache/pkg/provider"
	"github.com/rs/zerolog/log"
)

const (
	XCache = "X-Kache"
	HIT    = "HIT"
)

// Transport is the http filter implementing the http caching logic.
type Transport struct {
	// The RoundTripper interface actually used to make requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// Cache is the http cache.
	Cache *cache.HttpCache

	// If true, responses returned from the cache will be given an extra header.
	MarkCachedResponses bool

	// currentTime holds the time source.
	currentTime func() time.Time
}

// NewTransport returns a new Transport with the provided Cache implementation.
func NewCachedTransport(p provider.Provider) *Transport {
	c, err := cache.NewHttpCache(p)
	if err != nil {
		return nil
	}
	return &Transport{Cache: c, MarkCachedResponses: true, currentTime: time.Now}
}

// RoundTrip issues a http roundtrip and applies the http caching logic.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {

	if !cache.IsCacheableRequest(req) {
		log.Debug().Msgf("Ignoring uncachable request: %v", req)
		return t.send(req)
	}

	lookup := cache.NewLookupRequest(req, t.currentTime())
	cached := t.Cache.FetchResponse(context.Background(), *lookup)

	switch cached.Status {
	case cache.EntryOk:
		return t.handleCacheHit(cached)

	case cache.EntryRequiresValidation:
		log.Debug().Msgf("Cache HIT with validation: %v", cached.Response())
		cached.Header().Set(XCache, HIT)
		req = t.injectValidationHeaders(lookup.Request, cached.Header())
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

	// Store new or update validated response.
	if cache.IsCacheableResponse(resp) && shouldUpdateCachedEntry &&
		!lookup.ReqCacheControl.NoStore && lookup.Request.Method != "HEAD" {
		t.Cache.StoreResponse(context.TODO(), lookup, resp)
	} else {
		t.Cache.Delete(context.TODO(), lookup)
	}

	return resp, nil
}

// handleCacheHit handles a cache hit and sends the cached response downstream.
func (t *Transport) handleCacheHit(cached *cache.LookupResult) (*http.Response, error) {
	log.Debug().Msgf("Cache HIT: %v", cached.Response())
	cached.Header().Set(XCache, HIT)
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
