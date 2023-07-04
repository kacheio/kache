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

package cache

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kacheio/kache/pkg/provider"
	"github.com/rs/zerolog/log"
)

// TODO: add interface.

const (
	xCache = "X-Kache"
	HIT    = "HIT"
	MISS   = "MISS"
)

// DefaultTTL is the default time-to-live for cache entries.
var DefaultTTL = 120 * time.Second

// HttpCacheConfig holds the http cache configuration.
type HttpCacheConfig struct {
	// XCache specifies if the XCache debug header should be attached to responses.
	// If the response exists in the cache the header value is HIT, MISS otherwise.
	XCache bool `yaml:"x_header" json:"x_header"`

	// XCacheName is the name of the X-Cache header.
	XCacheName string `yaml:"x_header_name" json:"x_header_name"`

	// Default TTL is the default TTL for cache entries. Overrides 'DefaultTTL'.
	DefaultTTL string `yaml:"default_ttl" json:"default_ttl"`

	// DefaultCacheControl specifies a default cache-control header.
	DefaultCacheControl string `yaml:"default_cache_control" json:"default_cache_control"`

	// ForceCacheControl specifies whether to overwrite an existing cache-control header.
	ForceCacheControl bool `yaml:"force_cache_control" json:"force_cache_control"`

	// Timeouts holds the TTLs per path/resource.
	Timeouts []Timeout `yaml:"timeouts" json:"timeouts"`

	// Exclude contains the cache exclude configuration.
	Exclude *Exclude `yaml:"exclude" json:"exclude"`
}

// Timeout holds the custom TTL configuration
type Timeout struct {
	// Path is the path the ttl is applied to. String or Regex.
	Path string `yaml:"path" json:"path"`
	// TTL is the corresponing resource ttl.
	TTL time.Duration `yaml:"ttl" json:"ttl"`
	// Matcher holds the compiled regex.
	Matcher *regexp.Regexp `json:"-"`
}

// Exclude holds the cache ignore information.
type Exclude struct {
	// Path contains the paths to be ignored by the cache.
	Path []string `yaml:"path" json:"path"`

	// PathMatcher contains the compile `Path` patterns.
	PathMatcher []*regexp.Regexp `json:"-"`

	// Header contains the headers to be ignored by the cache.
	Header map[string]string `yaml:"header" json:"header"`

	// Content contains the content types to be ignored by the cache.
	Content []Content `yaml:"content" json:"content"`
}

// Content holds the specific content-type and max content size used for excluding responses from
// caching. Every response matching the specified content type regex and exceeding the max content
// size is excluded from cache. If max size is not specified, only type is used to decide wethter
// to cache the response or not.
type Content struct {
	// Type is the content type to be ignored by the cache.
	Type string `yaml:"type" json:"type"`

	// TypeMatcher contains the compiled `Type` patterns.
	TypeMatcher *regexp.Regexp `json:"-"`

	// Size is the max content size in bytes.
	Size int `yaml:"size,omitempty" json:"size,omitempty"`
}

// HttpCache is the http cache.
type HttpCache struct {
	// config is the http cache configuration.
	config atomic.Pointer[HttpCacheConfig]

	// cache holds the inner caching provider.
	cache provider.Provider
}

// NewHttpCache creates a new http cache.
func NewHttpCache(config *HttpCacheConfig, pdr provider.Provider) (*HttpCache, error) {
	cfg := &HttpCacheConfig{}
	if config != nil {
		cfg = config

	}
	c := &HttpCache{
		cache: pdr,
	}
	c.UpdateConfig(cfg)
	return c, nil
}

// Config returns the current cache config.
func (c *HttpCache) Config() *HttpCacheConfig {
	return c.loadConfig()
}

func (c *HttpCache) loadConfig() *HttpCacheConfig {
	if c := c.config.Load(); c != nil {
		return c
	}
	return &HttpCacheConfig{}
}

// UpdateConfig updates the cache config in a concurrent safe way.
func (c *HttpCache) UpdateConfig(config *HttpCacheConfig) {
	// Compile custom timeout matchers.
	for i, t := range config.Timeouts {
		r, err := regexp.Compile(t.Path)
		if err != nil {
			log.Error().Err(err).Str("path", t.Path).Msg("Invalid timeout path regex")
		}
		config.Timeouts[i].Matcher = r
	}

	// Compile cache exclude matchers.
	if config.Exclude != nil {
		config.Exclude.PathMatcher = make([]*regexp.Regexp, len(config.Exclude.Path))
		for i, p := range config.Exclude.Path {
			r, err := regexp.Compile(p)
			if err != nil {
				log.Error().Err(err).Str("path", p).Msg("Invalid exclude path regex")
			}
			config.Exclude.PathMatcher[i] = r
		}
		for i, co := range config.Exclude.Content {
			r, err := regexp.Compile(co.Type)
			if err != nil {
				log.Error().Err(err).Str("content", co.Type).Msg("Invalid exclude content type regex")
			}
			config.Exclude.Content[i].TypeMatcher = r
		}
	}

	// Safely update config.
	c.config.Store(config)
}

// IsExcludedPath checks whether a specific path is excluded from caching.
func (c *HttpCache) IsExcludedPath(p string) bool {
	config := c.loadConfig()
	if config.Exclude == nil {
		return false
	}
	for _, m := range config.Exclude.PathMatcher {
		if m != nil && m.MatchString(p) {
			return true
		}
	}
	return false
}

// IsExcludedHeader checks whether a specific HTTP header is excluded from caching.
func (c *HttpCache) IsExcludedHeader(h http.Header) bool {
	config := c.loadConfig()
	if config.Exclude == nil {
		return false
	}
	for k, vv := range config.Exclude.Header {
		if h.Get(strings.ReplaceAll(k, "_", "-")) == vv {
			return true
		}
	}
	return false
}

// IsExcludedContent checks if the specific responses content-type and size is excluded from caching.
func (c *HttpCache) IsExcludedContent(content string, length int64) bool {
	config := c.loadConfig()
	if config.Exclude == nil || len(content) == 0 {
		return false
	}
	for _, t := range config.Exclude.Content {
		if t.TypeMatcher != nil && t.TypeMatcher.MatchString(content) {
			if t.Size > 0 {
				// if content exceeds the allowed size, do not cache.
				return t.Size < int(length)
			}
			return true
		}
	}
	return false
}

// MarkCachedResponses returns true if cached responses should be marked.
func (c *HttpCache) MarkCachedResponses() bool {
	config := c.loadConfig()
	return config.XCache
}

// XCacheHeader returns the XCache debug header key.
func (c *HttpCache) XCacheHeader() string {
	config := c.loadConfig()
	if config.XCacheName == "" {
		return xCache
	}
	return config.XCacheName
}

// DefaultCacheControl returns the default cache control.
func (c *HttpCache) DefaultCacheControl() string {
	config := c.loadConfig()
	return config.DefaultCacheControl
}

// ForceCacheControl specifies whether to overwrite an existing cache-control header.
func (c *HttpCache) ForceCacheControl() bool {
	config := c.loadConfig()
	return config.ForceCacheControl
}

// DefaultTTL returns the TTL as specified in the configuration as a valid duration
// in seconds. If not specified, the default value is returned.
func (c *HttpCache) DefaultTTL() time.Duration {
	config := c.loadConfig()
	if len(config.DefaultTTL) == 0 {
		return DefaultTTL
	}
	t, err := time.ParseDuration(config.DefaultTTL)
	if err != nil {
		return DefaultTTL
	}
	return t
}

// PathTTL matches a path with the configured path regex. If a match
// is found, the corresponding TTL is returned, otherwise DefaultTTL.
func (c *HttpCache) PathTTL(p string) time.Duration {
	config := c.loadConfig()
	for _, t := range config.Timeouts {
		if t.Matcher != nil && t.Matcher.MatchString(p) {
			return t.TTL
		}
	}
	return c.DefaultTTL()
}

// FetchResponse fetches a response matching the given request.
func (c *HttpCache) FetchResponse(ctx context.Context, lookup LookupRequest) *LookupResult {
	if cached := c.cache.Get(ctx, lookup.Key.String()); cached != nil {
		entry, err := DecodeEntry(cached)
		if err != nil {
			return &LookupResult{}
		}
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(entry.Body)), lookup.Request)
		if err != nil {
			log.Error().Err(err).Send()
			return &LookupResult{}
		}
		return lookup.makeResult(res, time.Unix(entry.Timestamp, 0))
	}
	return &LookupResult{}
}

// StoreResponse stores a response in the cache.
func (c *HttpCache) StoreResponse(_ context.Context, lookup *LookupRequest,
	response *http.Response, responseTime time.Time) {
	resp, err := httputil.DumpResponse(response, true)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	entry := &Entry{
		Body:      resp,
		Timestamp: responseTime.Unix(),
	}
	enc, err := entry.Encode()
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	c.cache.Set(lookup.Key.String(), enc, c.PathTTL(lookup.Request.URL.Path))
}

// Deletes deletes the response matching the request key from the cache.
func (c *HttpCache) Delete(ctx context.Context, lookup *LookupRequest) {
	c.cache.Delete(ctx, lookup.Key.String())
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

// UpdateHeader add any headers to the cached response.
func (r *LookupResult) UpdateHeader(header http.Header) {
	cachedHeader := r.cachedResponse.Header

	// Skip headers that should not be updated upon validation.
	// // https://www.ietf.org/archive/id/draft-ietf-httpbis-cache-18.html (3.2)
	headersNotToUpdate := map[string]struct{}{
		"Content-Range":  {}, // should not be changed upon validation.
		"Content-Length": {}, // should never be updated.
		"Etag":           {},
		"Vary":           {},
	}
	for k, vv := range header {
		if _, ok := headersNotToUpdate[k]; !ok {
			cachedHeader[k] = vv
		}
	}
}
