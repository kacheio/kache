package cache

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/rs/zerolog/log"
)

// TODO: add interface and config.

// HttpCache is the http cache.
type HttpCache struct {
	// cache holds the inner caching provider.
	cache provider.Provider
}

// NewHttpCache creates a new http cache.
func NewHttpCache(pdr provider.Provider) (*HttpCache, error) {
	return &HttpCache{pdr}, nil
}

// FetchResponse fetches a response matching the given request.
func (c *HttpCache) FetchResponse(_ context.Context, lookup LookupRequest) *LookupResult {
	if cached := c.cache.Get(lookup.Key.String()); cached != nil {
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(cached.(*Entry).Body)), lookup.Request)
		if err != nil {
			log.Error().Err(err).Send()
			return &LookupResult{}
		}
		return lookup.makeResult(res, cached.(*Entry).LastModified)
	}
	return &LookupResult{}
}

// StoreResponse stores a response in the cache.
func (c *HttpCache) StoreResponse(_ context.Context, lookup LookupRequest, response *http.Response) {
	res, err := httputil.DumpResponse(response, true)
	if err != nil {
		// TODO: handle errors
		return
	}
	entry := &Entry{
		Body:         res,
		LastModified: lookup.Timestamp,
	}
	c.cache.Put(lookup.Key.String(), entry)
}

// Deletes deletes the response matching the request key from the cache.
func (c *HttpCache) Delete(_ context.Context, lookup LookupRequest) {
	c.cache.Delete(lookup.Key.String())
}

// LookupRequest holds the context for looking up a request.
type LookupRequest struct {
	// Request is the original request.
	Request *http.Request

	// ReqCacheControl holds the parsed request cache control.
	ReqCacheControl RequestCacheControl

	// Key is the cache key generated from the request.
	Key *Key

	// Timestamp is the time this lookup was created.
	Timestamp time.Time
}

// NewLookupRequest creates a new lookup request structure.
func NewLookupRequest(req *http.Request, timestamp time.Time) *LookupRequest {
	var requestCacheControl RequestCacheControl
	requestCacheControl.SetDefaults()
	cacheControl := req.Header.Get(HeaderCacheControl)

	if cacheControl != "" {
		requestCacheControl = ParseRequestCacheControl(cacheControl)
	} else {
		// Fallback to Pragma header, if Cache-Control is missing.
		// According to https://httpwg.org/specs/rfc7234.html#header.pragma, when the Cache-Control
		// header is not set, the "Pragma:no-cache" directive is equivalent to "Cache-Control:no-cache",
		// any other directives are ignored.
		pragma := req.Header.Get(HeaderPragma)
		requestCacheControl.MustValidate = ParseRequestCacheControl(pragma).MustValidate
	}

	return &LookupRequest{
		Request:         req,
		Timestamp:       timestamp,
		ReqCacheControl: requestCacheControl,
		Key:             NewKeyFromRequst(req),
	}
}

// MakeResult prepares and creates the cache result. Specifically, it sets the cache entry status
// according to the HTTP caching validation logic, takes care of response headers, parts, and ranges.
// TODO: incomplete implementation.
func (l *LookupRequest) makeResult(res *http.Response, resTime time.Time) *LookupResult {
	age := CalculateAge(&res.Header, resTime, l.Timestamp)
	res.Header.Set(HeaderAge, fmt.Sprintf("%.0f", age.Seconds()))

	var status EntryStatus
	if l.requiresValidation(&res.Header, age) {
		status = EntryRequiresValidation
	} else {
		status = EntryOk
	}

	return &LookupResult{
		cachedResponse: res,
		Status:         status,
	}
}

// requiresValidation checks if the cached response needs to be validated by the origin.
func (l *LookupRequest) requiresValidation(header *http.Header, age time.Duration) bool {
	resCacheControl := ParseResponseCacheControl(header.Get(HeaderCacheControl))
	reqCacheControl := l.ReqCacheControl

	maxAgeExceeded := reqCacheControl.MaxAge >= 0 && reqCacheControl.MaxAge < age
	if resCacheControl.MustValidate || reqCacheControl.MustValidate || maxAgeExceeded {
		return true
	}

	// Valid expiration data is ensured by `IsCachableResponse(..)`.

	// Calculate freshness lifetime.
	var freshness time.Duration
	if resCacheControl.MaxAge >= 0 {
		freshness = resCacheControl.MaxAge
	} else {
		expires := parseHttpTime(header.Get(HeaderExpires))
		date := parseHttpTime(header.Get(HeaderDate))
		freshness = expires.Sub(date)
	}

	if age > freshness { // Stale response.
		// Check if the response is allowed being served stale,
		// or if the request max-stale directive prevents it.
		allowStale := reqCacheControl.MaxStale >= 0 &&
			reqCacheControl.MaxStale > age-freshness

		return resCacheControl.NoStale || !allowStale
	}

	// Fresh response. Only requires validation if min-fresh requirement is not satisfied.
	return reqCacheControl.MinFresh >= 0 && reqCacheControl.MinFresh > freshness-age
}

// LookupResult wraps the cached response.
type LookupResult struct {
	// Status holds the status of the cached entry.
	Status EntryStatus

	// cachedResponse is the response fetched from the cache.
	cachedResponse *http.Response
}

// Header returns the cached response header.
func (r *LookupResult) Header() http.Header {
	return r.cachedResponse.Header
}

// Response returns the cached response.
func (r *LookupResult) Response() *http.Response {
	return r.cachedResponse
}
