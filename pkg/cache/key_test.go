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

func TestNewKey(t *testing.T) {
	url := "https://example.com/with/path"

	req, _ := http.NewRequest("GET", url, nil)
	key := NewKeyFromRequst(req)

	assert.Equal(t, "kache-"+url, key.String())
	assert.Equal(t, "https", key.Scheme)

	// Add trailing slash, expected be cleaned up.
	req, _ = http.NewRequest("GET", url+"/", nil)
	key = NewKeyFromRequst(req)

	assert.Equal(t, "kache-"+url, key.String())
}

func TestHashKey(t *testing.T) {

	key := Key{}
	key.Host = "example.com"

	hash := StableHashKey(key)

	expected := uint64(2492773929771664174)

	if hash != expected {
		t.Errorf("Expected hash %d, got %d", expected, hash)
	}

}
