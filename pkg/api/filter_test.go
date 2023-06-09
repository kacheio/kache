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
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPFilter(t *testing.T) {
	// Valid addresses.
	_, err := NewIPFilter("192.0.2.0/24, 192.0.2.255")
	require.NoError(t, err)

	// Invalid IP.
	_, err = NewIPFilter("192.0.2.0/24, 192.0.2.")
	require.Error(t, err)

	// Invalid CIDR.
	_, err = NewIPFilter("192.0.2.0/33, 192.0.2.1")
	require.Error(t, err)

	// No addresses. Valid, but inactive.
	_, err = NewIPFilter("")
	require.NoError(t, err)
}

func TestIsAllowed(t *testing.T) {
	allowed := []string{
		"192.0.2.1/32",
		"192.0.2.1",
		"10.0.0.0/16",
	}
	whitelist := strings.Join(allowed, ",")
	f, err := NewIPFilter(whitelist)
	require.NoError(t, err)

	cases := []struct {
		IP        string
		IsAllowed bool
	}{
		{"192.0.2.1", true},
		{"192.0.2.2", false},
		{"10.0.0.1", true},
		{"10.0.30.1", true},
		{"10.20.0.1", false},
	}
	for _, c := range cases {
		t.Run(c.IP, func(t *testing.T) {
			assert.Equal(t, c.IsAllowed, f.IsAllowed(netip.MustParseAddr(c.IP)))
		})
	}
}

func TestOriginalIP(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	req.RemoteAddr = "192.0.2.1:6087"

	var ip netip.Addr

	// Get IP form request.
	ip, _ = originalIP(req)
	assert.Equal(t, "192.0.2.1", ip.String())

	// Get IP from X-Forwarded-For header.
	req.Header.Set("X-Forwarded-For", "192.0.2.1, 10.0.10.1, 10.20.2.1")
	ip, _ = originalIP(req)
	assert.Equal(t, "10.20.2.1", ip.String())

	// Get IP from X-Real-Ip header.
	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-Ip", "10.20.2.2")
	ip, _ = originalIP(req)
	assert.Equal(t, "10.20.2.2", ip.String())
}

func TestFilterHandler(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	cases := []struct {
		name   string
		addr   string
		status int
	}{
		{"Access granted", "192.0.2.1:6087", 202},
		{"Access denied", "192.0.20.1:6087", 401},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = c.addr

			filter, _ := NewIPFilter("192.0.2.1")
			handler := filter.Wrap(h)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, c.status, rr.Result().StatusCode)
		})
	}
}
