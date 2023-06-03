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
func (t *Transport) RoundTrip(ireq *http.Request) (resp *http.Response, err error) {
	req := ireq // req is either the original request, or a modified fork.

	if !cache.IsCacheableRequest(req) {
		log.Debug().Msgf("Ignoring uncachable request: %v", req)
		return t.send(req)
	}

	lookupReq := cache.NewLookupRequest(req, t.currentTime())
	cachedRes := t.Cache.FetchResponse(context.Background(), *lookupReq)

	switch cachedRes.Status {
	case cache.EntryOk:
		cachedRes.Header().Set(XCache, HIT)
		return cachedRes.Response(), nil

	case cache.EntryRequiresValidation:
		cachedRes.Header().Set(XCache, HIT)

		// forkReq forks req into a shallow clone of ireq the first
		// time it's called.
		forkReq := func() {
			if ireq == req {
				req = new(http.Request)
				*req = *ireq // shallow clone
				// Copy the initial request's Header
				req.Header = make(http.Header)
				for k, vv := range ireq.Header {
					req.Header[k] = vv
				}
			}
		}

		// Inject validation headers.
		if etag := cachedRes.Header().Get(cache.HeaderEtag); etag != "" {
			forkReq()
			req.Header.Set(cache.HeaderIfNoneMatch, etag)
		}
		if lastModified := cachedRes.Header().Get(cache.HeaderLastModified); lastModified != "" {
			forkReq()
			req.Header.Set(cache.HeaderIfModifiedSince, lastModified)
		} else {
			forkReq()
			// Fallback to Date header if Last-Modified is missing or invalid.
			// https://httpwg.org/specs/rfc7232.html#header.if-modified-sinces
			date := cachedRes.Header().Get(cache.HeaderDate)
			req.Header.Set(cache.HeaderLastModified, date)
		}
	}

	// Send request to upstream.
	resp, err = t.send(req)
	if err != nil {
		log.Error().Err(err).Msgf("RoundTrip: error: %v", err)
		return resp, err
	}

	shouldUpdateCachedEntry := true
	if err == nil && resp.StatusCode == http.StatusNotModified &&
		cachedRes.Status == cache.EntryRequiresValidation {
		// Process successful validation.

		// If the 304 response contains a strong validator (etag) that does not match
		// the cached response, the cached response should not be updated.
		// https://httpwg.org/specs/rfc7234.html#freshening.responses
		resEtag := resp.Header.Get(cache.HeaderEtag)
		cacEtag := cachedRes.Header().Get(cache.HeaderEtag)
		shouldUpdateCachedEntry = (resEtag == "" || (cacEtag != "" && cacEtag == resEtag))

		// A response that has been validated should not contain an Age header
		// as it is equivalent to a freshly served response from the origin.
		cachedRes.Header().Del(cache.HeaderAge)

		// Add any missing headers from the 304 to the cached response.
		// Skip headers that should not be updated upon validation.
		// https://www.ietf.org/archive/id/draft-ietf-httpbis-cache-18.html (3.2)
		headersNotToUpdate := map[string]struct{}{
			"Content-Range":  {}, // should not be changed upon validation.
			"Content-Length": {}, // should never be updated.
			"Etag":           {},
			"Vary":           {},
		}
		for k, vv := range resp.Header {
			if _, ok := headersNotToUpdate[k]; !ok {
				cachedRes.Header()[k] = vv
			}
		}

		resp.Body.Close()
		resp = cachedRes.Response()
	}

	// Store new or update validated response.
	if cache.IsCacheableResponse(resp) && !lookupReq.ReqCacheControl.NoStore &&
		shouldUpdateCachedEntry && req.Method != "HEAD" {
		t.Cache.StoreResponse(context.TODO(), *lookupReq, resp)
	} else {
		t.Cache.Delete(context.TODO(), *lookupReq)
	}

	return resp, nil
}

// send issues an upstream request.
func (t *Transport) send(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}
