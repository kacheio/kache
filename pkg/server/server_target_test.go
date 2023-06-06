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

package server

import (
	"net/http"
	"testing"

	"github.com/kacheio/kache/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTarget(t *testing.T) {
	targets, err := NewTargets([]*config.Upstream{
		{Name: "upstream 1", Addr: "http://example.com", Path: "/bot"},
		{Name: "upstream 2", Addr: "http://example.com", Path: "/bot"},
		{Name: "upstream 3", Addr: "http://example.com", Path: "/news/{id:[0-9]+}"},
		{Name: "upstream 4", Addr: "http://example.com", Path: "/"},
	})
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/bot", nil)
	target, _ := targets.MatchTarget(req)

	assert.Equal(t, "upstream 1", target.name)

	req, _ = http.NewRequest("GET", "/", nil)
	target, _ = targets.MatchTarget(req)

	assert.Equal(t, "upstream 4", target.name)

	req, _ = http.NewRequest("GET", "/news/2", nil)
	target, _ = targets.MatchTarget(req)

	assert.Equal(t, "upstream 3", target.name)
}
