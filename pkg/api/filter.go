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
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	ErrMsgUnauthorized = "Not authorized to access the requested resource"
)

// defaultBlockedHandler is the default error handler sent when the request IP is blocked.
var defaultBlockedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprintln(w, ErrMsgUnauthorized)
})

// IPFilter implements a simple IP filter. It is configured with an access control list
// containing IP and CIDR addresses that are allowed to access the specified resource.
// If the list of allowed addresses is empty, any request is granted access and bypasses
// the filter without any restrictions.
type IPFilter struct {
	// allowedIPs and allowedCIDRs are the lists of IP and network addresses allowed to
	// access a resource. If empty, the IP filter is disabled and every request is allowed.
	allowedIPs   map[netip.Addr]struct{}
	allowedCIDRs []*net.IPNet
}

// NewIPFilter creates a new IP filter.
func NewIPFilter(whitelist string) (*IPFilter, error) {
	allowedIPs := make(map[netip.Addr]struct{})
	allowedCIDRs := make([]*net.IPNet, 0, len(whitelist))

	// Parse allowed IP and CIDR addresses.
	if ips := strings.Trim(whitelist, ","); len(ips) > 0 {
		for _, ip := range strings.Split(ips, ",") {
			ip = strings.TrimSpace(ip)
			if _, cidr, err := net.ParseCIDR(ip); err == nil {
				allowedCIDRs = append(allowedCIDRs, cidr)
				continue
			}
			if addr, err := netip.ParseAddr(ip); err == nil {
				allowedIPs[addr] = struct{}{}
				continue
			}
			return nil, fmt.Errorf("malformed IP or CIDR address: %v", ip)
		}
	}

	return &IPFilter{
		allowedIPs:   allowedIPs,
		allowedCIDRs: allowedCIDRs,
	}, nil
}

// Wrap wraps the specified handler with an IP filter. It filters the request
// based on the configured access control list and allows or blocks the request
// according to the original IP. If the list of allowed IPs is empty, any
// request bypasses the filter.
func (f *IPFilter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(f.allowedIPs) == 0 {
			next(w, r)
			return
		}

		// Get the original client IP.
		ip, err := originalIP(r)
		if err != nil {
			defaultBlockedHandler.ServeHTTP(w, r)
			return
		}

		// Check if the IP is allowed or blocked.
		if !f.IsAllowed(ip) {
			defaultBlockedHandler.ServeHTTP(w, r)
			return
		}

		next(w, r)
	})
}

// IsAllowed checks if the given IP is allowed.
func (f *IPFilter) IsAllowed(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}
	if _, ok := f.allowedIPs[ip]; ok {
		return true
	}
	for _, cidr := range f.allowedCIDRs {
		if cidr.Contains(ip.AsSlice()) {
			return true
		}
	}
	return false
}

// originalIP finds the originating client IP.
func originalIP(req *http.Request) (netip.Addr, error) {
	// The default is the originating IP. But, we try to find better
	// options as this is almost never the right IP we're looking for.
	addr := ""
	if parts := strings.Split(req.RemoteAddr, ":"); len(parts) == 2 {
		addr = parts[0]
	}

	// If we have a forwarded-for header, take the address from there.
	if xff := strings.Trim(req.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		last := strings.TrimSpace(addrs[len(addrs)-1])
		return netip.ParseAddr(last)
	}

	// Otherwise, parse the X-Real-Ip header if it exists.
	if xri := req.Header.Get("X-Real-Ip"); len(xri) > 0 {
		return netip.ParseAddr(xri)
	}

	return netip.ParseAddr(addr)
}
