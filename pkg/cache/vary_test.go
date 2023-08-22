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
