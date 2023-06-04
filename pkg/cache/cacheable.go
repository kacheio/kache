package cache

import (
	"net/http"
)

type (
	RequestHeaderMap  = map[string]string
	ResponseHeaderMap = map[string]string
)

const (
	// Common cache related HTTP headers

	HeaderCacheControl = "Cache-Control"
	HeaderDate         = "Date"

	HeaderAuthorization = "Authorization"

	// Request headers
	HeaderPragma            = "Pragma"
	HeaderIfRange           = "If-Range"
	HeaderIfMatch           = "If-Match"
	HeaderIfNoneMatch       = "If-None-Match"
	HeaderIfModifiedSince   = "If-Modified-Since"
	HeaderIfUnmodifiedSince = "If-Unmodified-Since"

	// Response headers
	HeaderAge          = "Age"
	HeaderEtag         = "Etag"
	HeaderExpires      = "Expires"
	HeaderLastModified = "Last-Modified"
)

var (
	// cacheableStatusCodes holds a set of cacheable status codes.
	// https://tools.ietf.org/html/rfc7231#section-6.1
	// https://tools.ietf.org/html/rfc7538#section-3
	// https://tools.ietf.org/html/rfc7725#section-3
	// TODO(toashd): this should be configurable.
	cacheableStatusCodes = map[int]struct{}{
		200: {},
		203: {},
		204: {},
		206: {},
		300: {},
		301: {},
		308: {},
		404: {},
		405: {},
		410: {},
		414: {},
		451: {},
		501: {},
	}

	// conditionalHeaders holds conditional headers.
	// https://httpwg.org/specs/rfc7232.html#preconditions.
	conditionalHeaders = []string{
		HeaderIfRange,
		HeaderIfMatch,
		HeaderIfNoneMatch,
		HeaderIfModifiedSince,
		HeaderIfUnmodifiedSince,
	}
)

// IsCacheableRequest checks if a request can be served from cache.
// This does not depend on cache-control headers as request cache-control
// headers only decide whether validation is required and whether the
// response can be cached.
func IsCacheableRequest(req *http.Request) bool {
	// Check if the request contains any conditional headers.
	// For now, requests with conditional headers bypass the cache.
	for _, h := range conditionalHeaders {
		if _, ok := req.Header[h]; ok {
			return false
		}
	}

	// Check if the request contains authorization headers.
	// For now, requests with authorization headers bypass the cache.
	// https://httpwg.org/specs/rfc7234.html#caching.authenticated.responses
	if _, ok := req.Header[HeaderAuthorization]; ok {
		return false
	}

	return req.URL.Path != "" && req.Host != "" &&
		(req.Method == http.MethodGet || req.Method == http.MethodHead)
}

// IsCacheableResponse checks if a response can be stored in cache.
// Note that if a request is not cacheable according to `CanServeRequestFromCache`
// then its response is also not cacheable. Hence, CanServeRequestFromCache and
// `IsCacheableResponse` together should cover the cacheability of the response.
func IsCacheableResponse(res *http.Response) bool {
	cacheControl := res.Header.Get(HeaderCacheControl)
	resCacheControl := ParseResponseCacheControl(cacheControl)

	// Only cache responses with enough data to calculate freshness lifetime:
	// https://httpwg.org/specs/rfc7234.html#calculating.freshness.lifetime
	// Either:
	//	'no-cache' cache-control directive (requires revalidation anyway)
	// 	'max-age' or 's-maxage' cache-control directives
	// 	Both 'Expires' and 'Date' headers
	hasValidationData := resCacheControl.MustValidate || resCacheControl.MaxAge >= 0 ||
		(res.Header.Get(HeaderDate) != "" && res.Header.Get(HeaderExpires) != "")

	return !resCacheControl.NoStore && Contains(cacheableStatusCodes, res.StatusCode) &&
		hasValidationData
}

// Contains checks if the given key is in the map.
func Contains[K comparable, V any](m map[K]V, k K) bool {
	if _, ok := m[k]; ok {
		return true
	}
	return false
}
