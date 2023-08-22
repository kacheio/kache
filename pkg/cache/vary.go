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
	"strings"
)

const (
	// headerVary is the canonical vary header name.
	headerVary = "Vary"

	// varySeparator marks the beginning of the vary key.
	varySeparator = "<varied>\n"

	// headerSeparator separates the values of different headers.
	headerSeparator = "\n"

	// valueSeparator separates multiple values of the same header.
	valueSeparator = "\r"
)

// varyAllowList contains the headers allowed to be varied.
// Values are specified in and populated by the cache config.
type varyAllowList map[string]struct{}

// allowsHeader checks whether the response header contains a permitted value in the Vary header.
func (l *varyAllowList) allowsHeader(header http.Header) bool {
	if !hasVary(header) {
		return true
	}

	varyHeaders := header.Values(headerVary)
	for _, vry := range varyHeaders {
		if vry == "*" {
			// "Vary: *" should never be cached:
			// https://tools.ietf.org/html/rfc7231#section-7.1.4
			return false
		}
		if !l.allowsValue(vry) {
			return false
		}
	}

	return true
}

// allowsValue checks whether this vary header value is allowed to vary cache entries.
func (l varyAllowList) allowsValue(val string) bool {
	return Contains(l, strings.ToLower(val))
}

// createVaryIdentifier creates a single string containing the values of the
// varied headers from the response. Returns an empty identifier if no valid
// vary key can be created and the response should not be cached (e.g. if
// invalid and not-allowed vary headers are present in the response).
func createVaryIdentifier(allowed varyAllowList, varyHeaderValues []string, requestHeader http.Header) (string, bool) {
	varyIdent := varySeparator

	if len(varyHeaderValues) == 0 {
		return varyIdent, true
	}

	for _, value := range varyHeaderValues {
		if len(value) == 0 {
			continue // ignore empty headers
		}

		if !allowed.allowsValue(value) {
			// The backend has tried to vary with a header that
			// is not allowed, hence this request cannot be cached.
			return "", false
		}

		// Create the vary identifier containing the corresponding request header values.
		values := strings.Join(requestHeader.Values(http.CanonicalHeaderKey(value)), valueSeparator)
		varyIdent += value + valueSeparator + values + headerSeparator
	}

	return varyIdent, true
}

// hasVary checks if the http header contains a non-empty vary header entry.
// Returns true if there is at least one non-empty vary entry available.
func hasVary(header http.Header) bool {
	for _, vv := range header.Values(headerVary) {
		if len(vv) > 0 {
			return true
		}
	}
	return false
}

// varyValues retrieves all the individual header values from the provided
// response header map across all vary header entries.
func varyValues(header http.Header) []string {
	vals := []string{}
	for _, vv := range header.Values(headerVary) {
		vals = append(vals, parseCommaDelimitedHeader(vv)...)
	}
	return vals // strings.Join(vals, ",")
}

// parseCommaDelimitedHeader parsed a comma separated header into a slice.
func parseCommaDelimitedHeader(header string) []string {
	vals := []string{}
	for _, token := range strings.Split(header, ",") {
		token = strings.Trim(token, "\"' ")
		if len(token) > 0 {
			vals = append(vals, token)
		}
	}
	return vals
}
