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

	"github.com/stretchr/testify/assert"
)

func TestHasVary(t *testing.T) {
	header := http.Header{}

	// Empty header
	assert.False(t, hasVary(header))

	// Empty value
	header.Set("Vary", "")
	assert.False(t, hasVary(header))

	// Non-empty value
	header.Set("Vary", "accept-encoding")
	assert.True(t, hasVary(header))

	// Empty and non-empty value
	header.Set("Vary", "")
	header.Add("Vary", "accept-encoding")
	assert.True(t, hasVary(header))
}

func TestVaryValues(t *testing.T) {
	header := http.Header{}

	// No vary header
	assert.Equal(t, 0, len(varyValues(header)))

	// Empty value
	header.Set("Vary", "")
	assert.Equal(t, 0, len(varyValues(header)))

	// Non-empty single value
	header.Set("Vary", "Accept-Encoding")
	assert.Equal(t, []string{"Accept-Encoding"}, varyValues(header))

	// Non-empty multi value
	header.Add("Vary", "Accept, Width")
	assert.Equal(t, []string{"Accept-Encoding", "Accept", "Width"}, varyValues(header))
}

func TestVaryAllowList(t *testing.T) {
	allowList := varyAllowList{
		"accept":          struct{}{},
		"accept-language": struct{}{},
		"width":           struct{}{},
	}

	assert.True(t, allowList.allowsValue("Accept-Language"))
	assert.False(t, allowList.allowsValue("Cookie"))

	header := http.Header{}

	// No header
	assert.True(t, allowList.allowsHeader(header))

	// Empty header
	header.Set("Vary", "")
	assert.True(t, allowList.allowsHeader(header))

	// Allowed header
	header.Set("Vary", "Accept")
	assert.True(t, allowList.allowsHeader(header))

	// Allowed header multi
	header.Add("Vary", "Accept-Language")
	assert.True(t, allowList.allowsHeader(header))

	// Not-allowed wildcard
	header.Set("Vary", "*")
	assert.False(t, allowList.allowsHeader(header))

	// Not allowed single
	header.Set("Vary", "Cookie")
	assert.False(t, allowList.allowsHeader(header))

	// Not allowed multi
	header.Set("Vary", "Accept")
	header.Add("Vary", "Not-Allowed")
	assert.False(t, allowList.allowsHeader(header))
}

func TestParseCommaDelimitedHeader(t *testing.T) {
	cases := []struct {
		name    string
		headers *http.Header
		want    []string
	}{
		{
			"empty",
			headers(headerMap{}),
			[]string{},
		},
		{
			"single value",
			headers(headerMap{"Vary": "accept"}),
			[]string{"accept"},
		},
		{
			"multi value",
			headers(headerMap{"Vary": "accept, accept-encoding"}),
			[]string{"accept", "accept-encoding"},
		},
		{
			"multi value with spaces",
			headers(headerMap{"Vary": "  accept  , accept-encoding "}),
			[]string{"accept", "accept-encoding"},
		},
		{
			"multi entries",
			headers(headerMap{"Vary": "\"accept\", \"accept-encoding\""}),
			[]string{"accept", "accept-encoding"},
		},
		{
			"multi entries with multi values",
			headers(headerMap{"Vary": "\"accept, accept-encoding\", \"accept-language\""}),
			[]string{"accept", "accept-encoding", "accept-language"},
		},
		{
			"multi entries with multi values and spaces",
			headers(headerMap{"Vary": "\"  accept,   accept-encoding\",  \"  accept-language\""}),
			[]string{"accept", "accept-encoding", "accept-language"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, parseCommaDelimitedHeader(c.headers.Get("Vary")))
		})
	}
}

func TestCreateVaryIdentifier(t *testing.T) {
	cases := []struct {
		name    string
		reqHdr  *http.Header
		resHdr  *http.Header
		allowed varyAllowList
		want    string
	}{
		{
			"Empty response vary",
			headers(headerMap{"accept": "images/*"}),
			headers(headerMap{}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\n",
		},
		{
			"Single request vary",
			headers(headerMap{"accept": "images/*"}),
			headers(headerMap{"Vary": "accept"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\rimages/*\n",
		},
		{
			"No request vary",
			headers(headerMap{}),
			headers(headerMap{"Vary": "accept"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\r\n",
		},
		{
			"Multi request vary",
			headers(headerMap{"accept": "images/*", "accept-language": "en-us", "width": "640"}),
			headers(headerMap{"Vary": "accept, accept-language, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\rimages/*\naccept-language\ren-us\nwidth\r640\n",
		},
		{
			"Multi request vary XXX",
			headers(headerMap{"accept": "images/*", "width": "640"}),
			headers(headerMap{"Vary": "accept, accept-language, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\rimages/*\naccept-language\r\nwidth\r640\n",
		},
		{
			"Multi request vary with additional one",
			headers(headerMap{"accept": "images/*", "width": "640", "height": "1280"}),
			headers(headerMap{"Vary": "accept, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\rimages/*\nwidth\r640\n",
		},
		{
			"Multi request vary with additional one",
			headers(headerMap{"accept": "images/*", "width": "640", "height": "1280"}),
			headers(headerMap{"Vary": "accept, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\naccept\rimages/*\nwidth\r640\n",
		},
		{
			"Multi request vary with same header and different value",
			&http.Header{"Width": []string{"640", "1280"}},
			headers(headerMap{"Vary": "width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"<vry>\nwidth\r640\r1280\n",
		},
		{
			"Not allowed",
			headers(headerMap{"width": "640"}),
			headers(headerMap{"Vary": "not-allowed"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"",
		},
		{
			"Not allowed (list)",
			headers(headerMap{"width": "640"}),
			headers(headerMap{"Vary": "width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {},
			},
			"",
		},
		{
			"Not allowed and allowd",
			headers(headerMap{"width": "640"}),
			headers(headerMap{"Vary": "not-allowed, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {}, "width": {},
			},
			"",
		},
		{
			"Not allowed (list) and allowed",
			headers(headerMap{"accept-language": "en-us", "width": "640"}),
			headers(headerMap{"Vary": "accept-language, width"}),
			map[string]struct{}{
				"accept": {}, "accept-language": {},
			},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ident, _ := createVaryIdentifier(c.allowed, *c.resHdr, *c.reqHdr)
			assert.Equal(t, c.want, ident)
		})
	}

	// Identical, but out-of-order allow lists should create equal identifiers.
	{
		reqHeader := headers(headerMap{"accept-language": "en-us", "width": "640"})
		respHeader := headers(headerMap{"Vary": "accept-language, width"})
		allowed1 := map[string]struct{}{
			"accept": {}, "accept-language": {}, "width": {},
		}
		allowed2 := map[string]struct{}{
			"width": {}, "accept-language": {}, "accept": {},
		}
		ident1, _ := createVaryIdentifier(allowed1, *respHeader, *reqHeader)
		ident2, _ := createVaryIdentifier(allowed2, *respHeader, *reqHeader)
		assert.Equal(t, ident1, ident2)
	}

	// Requests with different header but same value, should create non-equal identifiers.
	{
		reqHeader1 := headers(headerMap{"accept-language": "en-us"})
		reqHeader2 := headers(headerMap{"accept": "en-us"})
		respHeader := headers(headerMap{"Vary": "accept-language, width"})
		allowed := map[string]struct{}{
			"accept": {}, "accept-language": {}, "width": {},
		}
		ident1, _ := createVaryIdentifier(allowed, *respHeader, *reqHeader1)
		ident2, _ := createVaryIdentifier(allowed, *respHeader, *reqHeader2)
		assert.NotEqual(t, ident1, ident2)
	}
}
