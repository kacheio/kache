package cache

import (
	"math"
	"net/http"
	"strings"
	"time"
)

// RequestCacheControl holds the parsed request cache-control header.
// https://httpwg.org/specs/rfc7234.html#cache-request-directive
type RequestCacheControl struct {
	// MustValidate is true if 'no-cache' directive is present.
	// A cached response must not be served without successful validation on the origin.
	MustValidate bool

	// NoStore is true if the 'no-store' directive is present.
	// Any part of either the request or any response to this request must not be cached (stored).
	NoStore bool

	// NoTransform is true if the 'no-transform' directive is present.
	// No transformations should be done to the response of this request, as defined by:
	// https://httpwg.org/specs/rfc7230.html#message.transformations
	NoTransform bool

	// OnlyIfCached is true if the 'only-if-cached' directive is present.
	// The request should be satisfied using a cached response only, or respond with 504 (Gateway Timeout).
	OnlyIfCached bool

	// MaxAge indicates that the client is unwilling to accept a response whose age exceeds the
	// specified duration.
	MaxAge time.Duration

	// MinFresh indicates that the client is willing to accept a response whose freshness lifetime
	// is no less than its current age plus the specified time, i.e. the client is unwilling to
	// receive a cached response that satisfies:
	//   expiration_time - now < min-fresh
	MinFresh time.Duration

	// MaxStale indicates that the client is willing to accept a response that has exceeded its
	// freshness lifetime, i.e. the client is willing to receive a stale response that satisfies:
	//   now - expiration_time < max-stale
	// If max-stale is assigned no value, the client is willing to accept any stale response.
	MaxStale time.Duration
}

// SetDefaults sets default values.
func (cc *RequestCacheControl) SetDefaults() {
	cc.MaxAge = time.Duration(-1)
	cc.MinFresh = time.Duration(-1)
	cc.MaxStale = time.Duration(-1)
}

// ParseRequestCacheControl parses the cache-control header into a RequestCacheControl.
func ParseRequestCacheControl(header string) RequestCacheControl {
	var cc RequestCacheControl
	cc.SetDefaults()

	directives := strings.Split(header, ",")
	for _, directive := range directives {
		dir, arg := splitDirective(directive)
		switch dir {
		case "no-cache":
			cc.MustValidate = true
		case "no-store":
			cc.NoStore = true
		case "no-transform":
			cc.NoTransform = true
		case "only-if-cached":
			cc.OnlyIfCached = true
		case "max-age":
			cc.MaxAge = parseDuration(arg)
		case "min-fresh":
			cc.MinFresh = parseDuration(arg)
		case "max-stale":
			if arg != "" {
				cc.MaxStale = parseDuration(arg)
			} else {
				cc.MaxStale = math.MaxInt64
			}
		}
	}
	return cc
}

// ResponseCacheControl holds the parsed response cache-control header.
// https://httpwg.org/specs/rfc7234.html#cache-response-directive
type ResponseCacheControl struct {
	// MustValidate is true if 'no-cache' directive is present; arguments are ignored for now.
	// This response must not be used to satisfy subsequent requests without successful validation
	// on the origin.
	MustValidate bool

	// NoStore is true if any of 'no-store' or 'private' directives is present.
	// 'private' arguments are ignored for now so it is equivalent to 'no-store'.
	// Any part of either the immediate request or response must not be cached (stored).
	NoStore bool

	// NoTransform is true if the 'no-transform' directive is present.
	// No transformations should be applied to the response, as defined by:
	// https://httpwg.org/specs/rfc7230.html#message.transformations
	NoTransform bool

	// NoStale is true if any of 'must-revalidate' or 'proxy-revalidate' directives is present.
	// This response must not be served stale without successful validation on the origin.
	NoStale bool

	// IsPublic is true if the 'public' directive is present.
	// This response may be stored, even if the response would normally be non-cacheable or
	// cacheable only within a private cache, see:
	// https://httpwg.org/specs/rfc7234.html#cache-response-directive.public
	IsPublic bool

	// MaxAge is set if to 's-maxage' if present, otherwise is set to 'max-age' if present.
	// Indicates the maximum time after which this response will be considered stale.
	MaxAge time.Duration
}

// SetDefaults sets default values.
func (cc *ResponseCacheControl) SetDefaults() {
	cc.MaxAge = time.Duration(-1)
}

// ParseResponseCacheControl parses the Cache Control header into a ResponseCacheControl.
func ParseResponseCacheControl(header string) ResponseCacheControl {
	var cc ResponseCacheControl
	cc.SetDefaults()

	directives := strings.Split(header, ",")
	for _, directive := range directives {
		dir, arg := splitDirective(directive)
		switch dir {
		case "no-cache":
			cc.MustValidate = true
		case "no-store", "private":
			cc.NoStore = true
		case "no-transform":
			cc.NoTransform = true
		case "must-revalidate", "proxy-revalidate":
			cc.NoStale = true
		case "public":
			cc.IsPublic = true
		case "s-maxage":
			cc.MaxAge = parseDuration(arg)
		case "max-age":
			if cc.MaxAge < 0 {
				cc.MaxAge = parseDuration(arg)
			}
		}
	}
	return cc
}

// splitDirective splits the cache control directive into its token and optional argument.
// Grammar (https://httpwg.org/specs/rfc7234.html#header.cache-control):
//
//	Cache-Control   = 1#cache-directive
//	cache-directive = token [ "=" ( token / quoted-string ) ]
func splitDirective(s string) (dir string, arg string) {
	if strings.ContainsRune(s, '=') {
		split := strings.SplitN(strings.TrimSpace(s), "=", 3)
		dir, arg = split[0], split[1]
	} else {
		dir = strings.TrimSpace(s)
	}
	return dir, arg
}

// parseDuration parses a directive argument and returns a valid duration in
// seconds. Any invalid duration is ignored, returning a negative duration.
// https://httpwg.org/specs/rfc7234.html#delta-seconds
func parseDuration(s string) time.Duration {
	s = strings.Trim(s, "\"'") // trim quotes
	d, err := time.ParseDuration(s + "s")
	if err != nil || d < 0 {
		return time.Duration(-1)
	}
	return d
}

// parseHttpTime parse a datetime http header value.
func parseHttpTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Acceptable Date/Time Formats per:
	// https://datatracker.ietf.org/doc/html/rfc7231#section-7.1.1.1
	//
	// Preferred format:
	// Sun, 06 Nov 1994 08:49:37 GMT    ; IMF-fixdate. RFC1123
	//
	// Obsolete formats:
	// Sunday, 06-Nov-94 08:49:37 GMT   ; obsolete RFC 850 format.
	// Sun Nov  6 08:49:37 1994         ; ANSI C's asctime() format.
	//
	// A recipient that parses a timestamp value in an HTTP header field
	// MUST accept all three HTTP-date formats.
	//
	httpRFC850 := "Monday, 02-Jan-06 15:04:05 GMT" // time.RFC1123 but hard-coded GMT time zone.
	for _, fmt := range [...]string{http.TimeFormat, httpRFC850, time.ANSIC} {
		if t, err := time.Parse(fmt, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// CalculateAge calculates the value of Age headers.
// https://httpwg.org/specs/rfc7234.html#age.calculations
func CalculateAge(headers *http.Header, responseTime time.Time, now time.Time) time.Duration {
	// Calculate apparent age.
	date := parseHttpTime(headers.Get(HeaderDate))
	apparentAge := Max(0, int64(responseTime.Sub(date)))

	// Set corrected age to the value Age header,
	// as response delay (response_time - request_time) is assumed to be negligible.
	age, err := time.ParseDuration(headers.Get(HeaderAge) + "s")
	if err != nil {
		age = time.Duration(0 * time.Second)
	}
	correctedAge := age
	correctedInitialAge := Max(int64(apparentAge), int64(correctedAge))

	// Calculate current age by adding the amount of time (seconds)
	// since the response was last validated by the origin server.
	residentTime := now.Sub(responseTime)
	currentAge := correctedInitialAge + int64(residentTime)

	return time.Duration(currentAge)
}

// Max returns the max of the given values.
func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
