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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupRequest(t *testing.T) {
	testCases := []struct {
		name            string
		reqCacheControl string
		resCacheControl string
		reqTime         time.Time
		resTime         time.Time
		wantStatus      EntryStatus
		wantAge         string
	}{
		{
			"Request requires revalidation",
			"no-cache",
			"public, max-age=3600",
			currentTime(),
			currentTime(),
			EntryRequiresValidation,
			"0",
		},
		{
			"Response requires revaliation",
			"",
			"no-cache",
			currentTime(),
			currentTime(),
			EntryRequiresValidation,
			"0",
		},
		{
			"Request max age satisfied",
			"max-age=10",
			"public, max-age=3600",
			currentTime().Add(seconds(9)),
			currentTime(),
			EntryOk,
			"9",
		},
		{
			"Request max age unsatisfied",
			"max-age=10",
			"public, max-age=3600",
			currentTime().Add(seconds(11)),
			currentTime(),
			EntryRequiresValidation,
			"11",
		},
		{
			"Request min fresh satisfied",
			"min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(999)),
			currentTime(),
			EntryOk,
			"999",
		},
		{
			"Request min fresh unsatisfied",
			"min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max age satisfied but min fresh unsatisfied",
			"max-age=1500, min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max age satisfied but max stale unsatisfied",
			"max-age=1500, max-stale=400",
			"public, max-age=1000",
			currentTime().Add(seconds(1401)),
			currentTime(),
			EntryRequiresValidation,
			"1401",
		},
		{
			"Request max stale satisfied but min fresh unsatisfied",
			"min-fresh=1000, max-stale=500",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max stale satisfied but max age unsatisfied",
			"max-age=1200, max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1201)),
			currentTime(),
			EntryRequiresValidation,
			"1201",
		},
		{
			"Request min fresh satisfied but max age unsatisfied",
			"max-age=500, min-fresh=400",
			"public, max-age=1000",
			currentTime().Add(seconds(501)),
			currentTime(),
			EntryRequiresValidation,
			"501",
		},
		{
			"Expired",
			"",
			"public, max-age=1000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Expired but max stale satisfied",
			"max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1499)),
			currentTime(),
			EntryOk,
			"1499",
		},
		{
			"Expired and max stale unsatisfied",
			"max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1501)),
			currentTime(),
			EntryRequiresValidation,
			"1501",
		},
		{
			"Expired and max stale unsatisfied but response must revalidate",
			"max-stale=500",
			"public, max-age=1000, must-revalidate",
			currentTime().Add(seconds(1499)),
			currentTime(),
			EntryRequiresValidation,
			"1499",
		},
		{
			"Fresh but respsonse must revalidate",
			"",
			"public, max-age=1000, must-revalidate",
			currentTime().Add(seconds(999)),
			currentTime(),
			EntryOk,
			"999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add("Cache-Control", tc.reqCacheControl)

			lookup := NewLookupRequest(req, tc.reqTime)

			res := &http.Response{Request: req, Header: make(http.Header, 0)}
			res.Header.Add("Cache-Control", tc.resCacheControl)
			res.Header.Add("Date", tc.resTime.Format(http.TimeFormat))

			result := lookup.makeResult(res, tc.resTime)

			assert.Equal(t, tc.wantStatus, result.Status)
			assert.Equal(t, tc.wantAge, result.Header().Get(HeaderAge))
		})
	}
}

func TestExpiresFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderExpires, currentTime().Add(-seconds(5)).Format(http.TimeFormat))
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))

	result := lookup.makeResult(res, currentTime())

	assert.Equal(t, EntryRequiresValidation, result.Status)
}

func TestExpiresFallbackButFresh(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderExpires, currentTime().Add(seconds(5)).Format(http.TimeFormat))
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))

	result := lookup.makeResult(res, currentTime())

	assert.Equal(t, EntryOk, result.Status)
}

func TestNoCachePragmaFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "no-cache")

	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; But Pragma requires validation.
	assert.Equal(t, EntryRequiresValidation, result.Status)
}

func TestNonNoCachePragmaFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "max-age=0")

	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; Although Pragma is present, directives other than no-cache are ignored.
	assert.Equal(t, EntryOk, result.Status)
}

func TestNoCachePragmaFallbackIgnored(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "no-cache")
	req.Header.Add(HeaderCacheControl, "max-age=10")

	lookup := NewLookupRequest(req, currentTime().Add(seconds(5)))

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; Cache-Control prioritized over Pragma.
	assert.Equal(t, EntryOk, result.Status)
}

func TestHttpCacheDefaultConfig(t *testing.T) {
	c, err := NewHttpCache(nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "", c.config.DefaultTTL)
	assert.Equal(t, time.Duration(DefaultTTL), c.DefaultTTL())

	assert.Equal(t, false, c.config.XCache)
	assert.Equal(t, xCache, c.XCacheHeader())

	assert.Equal(t, false, c.MarkCachedResponses())
}

func TestHttpCacheDefaultTTL(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		DefaultTTL: "3600s",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "3600s", c.config.DefaultTTL)
	assert.Equal(t, time.Duration(3600*time.Second), c.DefaultTTL())
}

func TestHttpCacheXCacheHeader(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		XCacheName: "X-Test",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "X-Test", c.XCacheHeader())
}

func TestHttpCacheCustomTTL(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		DefaultTTL: "3600s",
		Timeouts: []Timeout{
			{Path: "/test", TTL: time.Duration(120 * time.Second)},
			{Path: "/news", TTL: time.Duration(10 * time.Second)},
			{Path: "^/assets/([a-z0-9].*).css", TTL: time.Duration(180 * time.Second)},
		},
	}, nil)
	require.NoError(t, err)
	// simple match
	assert.Equal(t, time.Duration(120*time.Second), c.PathTTL("/test"))
	// regex match
	assert.Equal(t, time.Duration(180*time.Second), c.PathTTL("/assets/style54.css"))
	// sub-paths
	assert.Equal(t, time.Duration(10*time.Second), c.PathTTL("/news"))
	assert.Equal(t, time.Duration(10*time.Second), c.PathTTL("/news/latest"))
	// no match
	assert.Equal(t, time.Duration(3600*time.Second), c.PathTTL("/no-match"))
}

func TestExcludePath(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		Exclude: &Exclude{
			Path: []string{
				"^/admin",
				"^/.well-known/acme-challenge/(.*)",
			},
		},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, false, c.IsExcludedPath("/"))
	assert.Equal(t, false, c.IsExcludedPath("/api"))
	assert.Equal(t, true, c.IsExcludedPath("/admin/config"))
	assert.Equal(t, true, c.IsExcludedPath("/.well-known/acme-challenge/m4g1C-t0k3n"))
}

func TestExcludeHeader(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		Exclude: &Exclude{
			Header: map[string]string{
				"x_requested_with": "XMLHttpRequest",
			},
		},
	}, nil)
	require.NoError(t, err)

	h := http.Header{}
	h.Set("X-Requested-With", "XMLHttpRequest")
	assert.Equal(t, true, c.IsExcludedHeader(h))

	h = http.Header{}
	h.Set("X-Request-ID", "12345")
	assert.Equal(t, false, c.IsExcludedHeader(h))

	h = http.Header{}
	h.Set("Accept-Encoding", "gzip, deflate, br")
	h.Set("Accept-Language", "de-DE,de;q=0.9")
	assert.Equal(t, false, c.IsExcludedHeader(h))
}

func TestIsExcludedContent(t *testing.T) {
	c, err := NewHttpCache(&HttpCacheConfig{
		Exclude: &Exclude{
			Content: []Content{
				{Type: "application/javascript|text/css|image/.*", Size: 1024},
				{Type: "application/vnd.*", Size: 1024},
				{Type: "audio/.*"},
			},
		},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, false, c.IsExcludedContent("", 100))

	assert.Equal(t, false, c.IsExcludedContent("text/html; charset=utf-8", 1024))
	assert.Equal(t, false, c.IsExcludedContent("text/html; charset=utf-8", -1))

	assert.Equal(t, true, c.IsExcludedContent("application/javascript; charset=UTF-8", 1025))
	assert.Equal(t, false, c.IsExcludedContent("application/javascript; charset=UTF-8", 1023))
	assert.Equal(t, false, c.IsExcludedContent("application/javascript; charset=UTF-8", -1))

	assert.Equal(t, true, c.IsExcludedContent("image/jpeg", 3495))
	assert.Equal(t, false, c.IsExcludedContent("video/mp4", 10485))
	assert.Equal(t, true, c.IsExcludedContent("audio/x-wav", 8096))

	assert.Equal(t, true, c.IsExcludedContent("application/vnd.mozilla.xul+xml", 2024))
}
