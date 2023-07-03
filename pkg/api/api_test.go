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

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kacheio/kache/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIPrefix(t *testing.T) {
	api, err := New(config.API{
		Prefix: "/test-api",
	}, nil)
	require.NoError(t, err)

	api.RegisterRoute("GET", "/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name   string
		path   string
		status int
	}{
		{"Valid prefix", "/test-api/healthz", 200},
		{"Invalid prefix", "/invalid/healthz", 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest("GET", c.path, nil)
			require.NoError(t, err)

			api.ServeHTTP(rr, req)

			assert.Equal(t, c.status, rr.Result().StatusCode)
		})
	}
}

func TestAPIAccessControl(t *testing.T) {
	api, err := New(config.API{
		ACL: "192.0.2.1",
	}, nil)
	require.NoError(t, err)

	api.RegisterRoute("GET", "/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name   string
		addr   string
		status int
	}{
		{"Access granted", "192.0.2.1:6087", 200},
		{"Access denied", "192.0.20.1:6087", 401},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest("GET", "/api/healthz", nil)
			require.NoError(t, err)
			req.RemoteAddr = c.addr

			api.ServeHTTP(rr, req)

			assert.Equal(t, c.status, rr.Result().StatusCode)
		})
	}
}
