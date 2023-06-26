package api

import (
	"fmt"
	"net"
	"net/http"
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
	fmt.Fprintln(w, ErrMsgUnauthorized)
})

// IPFilter implements a simple IP filter. It is configured with an access control list
// containing IPs that are allowed to access the specified resource. If the list of
// allowed IPs is empty, any request is allowed and bypasses the filter.
type IPFilter struct {
	// allowedIPs is the list of IPs allowed to access a resource.
	// If empty, the IP filter is disabled and every request is allowed.
	allowedIPs map[string]struct{}
}

// NewIPFilter create a new IP filter.
func NewIPFilter(whitelist string) *IPFilter {
	f := &IPFilter{
		allowedIPs: make(map[string]struct{}),
	}

	// Parse allowed IPs from config.
	if ips := strings.Trim(whitelist, ","); len(ips) > 0 {
		for _, ip := range strings.Split(ips, ",") {
			if ipp := net.ParseIP(strings.TrimSpace(ip)); ipp != nil {
				f.allowedIPs[ipp.String()] = struct{}{}
			}
		}
	}

	return f
}

// Wrap wraps the specified handler with an IP filter. It filters the request
// based on the configured access control list and allows or blocks the request
// according to the original IP. If the list of allowed IPs is empty, any
// request bypasses the filter.
func (f *IPFilter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(f.allowedIPs) == 0 {
			next(w, r)
		}

		// Get the original client IP.
		ip := originalIP(r)

		// Check if the IP is allowed or blocked.
		if !f.IsAllowed(ip) {
			defaultBlockedHandler.ServeHTTP(w, r)
			return
		}

		next(w, r)
	})
}

// IsAllowed checks if the given IP is allowed.
func (f *IPFilter) IsAllowed(ip string) bool {
	if _, ok := f.allowedIPs[ip]; ok {
		return true
	}
	return false
}

// originalIP finds the originating client IP.
func originalIP(req *http.Request) string {
	addr := ""
	// The default is the originating IP. But, we try to find better
	// options as this is almost never the right IP we're looking for.
	if parts := strings.Split(req.RemoteAddr, ":"); len(parts) == 2 {
		addr = parts[0]
	}
	// If we have a forwarded-for header, take the address from there.
	if xff := strings.Trim(req.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		last := addrs[len(addrs)-1]
		if ip := net.ParseIP(last); ip != nil {
			return ip.String()
		}
	}
	// Otherwise, parse the X-Real-Ip header if it exists.
	if xri := req.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}
	return addr
}
