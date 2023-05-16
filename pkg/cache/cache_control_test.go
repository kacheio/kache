package cache

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func duration[T int | int64 | float64](i T) time.Duration {
	return time.Duration(i * T(time.Second)) // i * 1e9 as durations are measured in nanoseconds
}

func TestParseRequestCacheControl(t *testing.T) {
	cases := []struct {
		name     string
		header   string
		expected RequestCacheControl
	}{
		{
			"Empty header",
			"",
			RequestCacheControl{MustValidate: false, NoStore: false, NoTransform: false,
				OnlyIfCached: false, MaxAge: -1, MinFresh: -1, MaxStale: -1},
		},
		{
			"Valid header",
			"max-age=3600, min-fresh=10, no-transform, only-if-cached, no-store",
			RequestCacheControl{false, true, true, true, duration(3600), duration(10), -1},
		},
		{
			"Valid header",
			"min-fresh=100, max-stale, no-cache",
			RequestCacheControl{true, false, false, false, -1, duration(100), math.MaxInt64},
		},
		{
			"Valid header",
			"max-age=10,  max-stale=40",
			RequestCacheControl{false, false, false, false, duration(10), -1, duration(40)},
		},
		{
			"Quoted args are valid",
			"max-age=\"3600\", min-fresh=\"10\", no-transform, only-if-cached, no-store",
			RequestCacheControl{false, true, true, true, duration(3600), duration(10), -1},
		},
		{
			"Unknown directives",
			"max-age=10, max-stale=40, unknown-directive",
			RequestCacheControl{false, false, false, false, duration(10), -1, duration(40)},
		},
		{
			"Unknown directives with arguments",
			"max-age=10, max-stale=40, unknown-directive=50",
			RequestCacheControl{false, false, false, false, duration(10), -1, duration(40)},
		},
		{
			"Unknown directives and with arguments",
			"max-age=10, max-stale=40, unknown-directive, unknown-with-argument=50",
			RequestCacheControl{false, false, false, false, duration(10), -1, duration(40)},
		},
		{
			"Unknown directives and quoted",
			"max-age=10, max-stale=40, unknown-directive, unknown-with-argument=50, unknown-qoted=\"70\"",
			RequestCacheControl{false, false, false, false, duration(10), -1, duration(40)},
		},
		{
			"Invalid durations (NaN)",
			"max-age=ten, min-fresh=20, max-stale=5",
			RequestCacheControl{false, false, false, false, -1, duration(20), duration(5)},
		},
		{
			"Invalid durations (negative)",
			"max-age=ten, min-fresh=20s, max-stale=-5",
			RequestCacheControl{false, false, false, false, -1, -1, -1},
		},
		{
			"Invalid durations (empty)",
			"max-age=, min-fresh=\"\"",
			RequestCacheControl{false, false, false, false, -1, -1, -1},
		},
		{
			"Invalid header parts (unknown)",
			"no-cache,,,asdf1337, max-age=20, min-fresh=30=40",
			RequestCacheControl{true, false, false, false, duration(20), duration(30), -1},
		},
		{
			"Invalid header parts (misplaced separator)",
			"no-cache, max-age=10,5, no-store, min-fresh=30",
			RequestCacheControl{true, true, false, false, duration(10), duration(30), -1},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ParseRequestCacheControl(c.header))
		})
	}
}

func TestParseResponseCacheControl(t *testing.T) {
	cases := []struct {
		name     string
		header   string
		expected ResponseCacheControl
	}{
		{
			"Empty header",
			"",
			ResponseCacheControl{MustValidate: false, NoStore: false, NoTransform: false,
				NoStale: false, IsPublic: false, MaxAge: -1},
		},
		{
			"Valid header",
			"s-maxage=100, max-age=200, proxy-revalidate, no-store",
			ResponseCacheControl{false, true, false, true, false, duration(100)},
		},
		{
			"Valid header",
			"s-maxage=100, private, no-cache",
			ResponseCacheControl{true, true, false, false, false, duration(100)},
		},
		{
			"Valid header",
			"max-age=50, must-revalidate, no-cache, no-transform",
			ResponseCacheControl{true, false, true, true, false, duration(50)},
		},
		{
			"Valid header",
			"private",
			ResponseCacheControl{false, true, false, false, false, -1},
		},
		{
			"Valid header",
			"public, max-age=0",
			ResponseCacheControl{false, false, false, false, true, duration(0)},
		},
		{
			"Quoted arg are valid",
			"s-maxage=\"100\", max-age=\"200\", public",
			ResponseCacheControl{false, false, false, false, true, duration(100)},
		},
		{
			"Unknown directives",
			"no-cache, private, max-age=20, unknown-directive",
			ResponseCacheControl{true, true, false, false, false, duration(20)},
		},
		{
			"Unknown directives with arguments",
			"no-cache, no-store, max-age=20, unknown-with-argument=arg",
			ResponseCacheControl{true, true, false, false, false, duration(20)},
		},
		{
			"Unknown directives and with arguments",
			"no-cache, private, max-age=20, unknown-directive, unknown-with-argument=50",
			ResponseCacheControl{true, true, false, false, false, duration(20)},
		},
		{
			"Unknown directives and quoted",
			"no-cache, private, max-age=20, unknown-directive, unknown-with-argument=50, unknown-qoted=\"arg\"",
			ResponseCacheControl{true, true, false, false, false, duration(20)},
		},
		{
			"Invalid durations (NaN)",
			"max-age=ten",
			ResponseCacheControl{false, false, false, false, false, -1},
		},
		{
			"Invalid durations (negative)",
			"max-age=-5",
			ResponseCacheControl{false, false, false, false, false, -1},
		},
		{
			"Invalid durations (s-maxage))",
			"s-maxage=zero, max-age=10",
			ResponseCacheControl{false, false, false, false, false, duration(10)},
		},
		{
			"Invalid durations (max-age))",
			"s-maxage=20, max-age=zero",
			ResponseCacheControl{false, false, false, false, false, duration(20)},
		},
		{
			"Invalid durations (missing argument))",
			"max-age=",
			ResponseCacheControl{false, false, false, false, false, -1},
		},
		{
			"Invalid durations (empty quotes)",
			"no-store, max-age=\"\"",
			ResponseCacheControl{false, true, false, false, false, -1},
		},
		{
			"Invalid durations (empty one quote)",
			"private, max-age=\"\"",
			ResponseCacheControl{false, true, false, false, false, -1},
		},
		{
			"Invalid header parts (unknown)",
			"no-cache,,,asdf1337, max-age=20",
			ResponseCacheControl{true, false, false, false, false, duration(20)},
		},
		{
			"Invalid header parts (misplaced separator)",
			"no-cache, max-age=10,5, no-store",
			ResponseCacheControl{true, true, false, false, false, duration(10)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ParseResponseCacheControl(c.header))
		})
	}
}
