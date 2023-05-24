package cache

import (
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/kacheio/kache/pkg/utils/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seconds[T int | int64 | float64](i T) time.Duration {
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
			RequestCacheControl{false, true, true, true, seconds(3600), seconds(10), -1},
		},
		{
			"Valid header",
			"min-fresh=100, max-stale, no-cache",
			RequestCacheControl{true, false, false, false, -1, seconds(100), math.MaxInt64},
		},
		{
			"Valid header",
			"max-age=10,  max-stale=40",
			RequestCacheControl{false, false, false, false, seconds(10), -1, seconds(40)},
		},
		{
			"Quoted args are valid",
			"max-age=\"3600\", min-fresh=\"10\", no-transform, only-if-cached, no-store",
			RequestCacheControl{false, true, true, true, seconds(3600), seconds(10), -1},
		},
		{
			"Unknown directives",
			"max-age=10, max-stale=40, unknown-directive",
			RequestCacheControl{false, false, false, false, seconds(10), -1, seconds(40)},
		},
		{
			"Unknown directives with arguments",
			"max-age=10, max-stale=40, unknown-directive=50",
			RequestCacheControl{false, false, false, false, seconds(10), -1, seconds(40)},
		},
		{
			"Unknown directives and with arguments",
			"max-age=10, max-stale=40, unknown-directive, unknown-with-argument=50",
			RequestCacheControl{false, false, false, false, seconds(10), -1, seconds(40)},
		},
		{
			"Unknown directives and quoted",
			"max-age=10, max-stale=40, unknown-directive, unknown-with-argument=50, unknown-qoted=\"70\"",
			RequestCacheControl{false, false, false, false, seconds(10), -1, seconds(40)},
		},
		{
			"Invalid durations (NaN)",
			"max-age=ten, min-fresh=20, max-stale=5",
			RequestCacheControl{false, false, false, false, -1, seconds(20), seconds(5)},
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
			RequestCacheControl{true, false, false, false, seconds(20), seconds(30), -1},
		},
		{
			"Invalid header parts (misplaced separator)",
			"no-cache, max-age=10,5, no-store, min-fresh=30",
			RequestCacheControl{true, true, false, false, seconds(10), seconds(30), -1},
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
			ResponseCacheControl{false, true, false, true, false, seconds(100)},
		},
		{
			"Valid header",
			"s-maxage=100, private, no-cache",
			ResponseCacheControl{true, true, false, false, false, seconds(100)},
		},
		{
			"Valid header",
			"max-age=50, must-revalidate, no-cache, no-transform",
			ResponseCacheControl{true, false, true, true, false, seconds(50)},
		},
		{
			"Valid header",
			"private",
			ResponseCacheControl{false, true, false, false, false, -1},
		},
		{
			"Valid header",
			"public, max-age=0",
			ResponseCacheControl{false, false, false, false, true, seconds(0)},
		},
		{
			"Quoted arg are valid",
			"s-maxage=\"100\", max-age=\"200\", public",
			ResponseCacheControl{false, false, false, false, true, seconds(100)},
		},
		{
			"Unknown directives",
			"no-cache, private, max-age=20, unknown-directive",
			ResponseCacheControl{true, true, false, false, false, seconds(20)},
		},
		{
			"Unknown directives with arguments",
			"no-cache, no-store, max-age=20, unknown-with-argument=arg",
			ResponseCacheControl{true, true, false, false, false, seconds(20)},
		},
		{
			"Unknown directives and with arguments",
			"no-cache, private, max-age=20, unknown-directive, unknown-with-argument=50",
			ResponseCacheControl{true, true, false, false, false, seconds(20)},
		},
		{
			"Unknown directives and quoted",
			"no-cache, private, max-age=20, unknown-directive, unknown-with-argument=50, unknown-qoted=\"arg\"",
			ResponseCacheControl{true, true, false, false, false, seconds(20)},
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
			ResponseCacheControl{false, false, false, false, false, seconds(10)},
		},
		{
			"Invalid durations (max-age))",
			"s-maxage=20, max-age=zero",
			ResponseCacheControl{false, false, false, false, false, seconds(20)},
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
			ResponseCacheControl{true, false, false, false, false, seconds(20)},
		},
		{
			"Invalid header parts (misplaced separator)",
			"no-cache, max-age=10,5, no-store",
			ResponseCacheControl{true, true, false, false, false, seconds(10)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ParseResponseCacheControl(c.header))
		})
	}
}

func TestParseHttpTime(t *testing.T) {
	// Header format: {Date: [Fri, 19 May 2023 17:45:39 GMT]}
	validDateFormats := [...]string{
		"Sun, 06 Nov 1994 08:49:37 GMT",
		"Sunday, 06-Nov-94 08:49:37 GMT",
		"Sun Nov  6 08:49:37 1994",
	}
	want, err := time.Parse(http.TimeFormat, validDateFormats[0])
	require.NoError(t, err)
	for _, dt := range validDateFormats {
		assert.Equal(t, want, parseHttpTime(dt))
	}

	invalidDateFormat := "Fri, 05-19-2023 17:45:39"
	assert.Equal(t, time.Time{}, parseHttpTime(invalidDateFormat))
	assert.Equal(t, time.Time{}, parseHttpTime(""))
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC1123)
}

func currentTime() time.Time {
	ts := clock.NewEventTimeSource()
	ts.Update(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))
	return ts.Now()
}

func headers(hdrs headerMap) *http.Header {
	h := &http.Header{}
	for k, v := range hdrs {
		h.Set(k, v)
	}
	return h
}

type headerMap map[string]string

func TestCalculateAge(t *testing.T) {
	cases := []struct {
		name         string
		headers      *http.Header
		responseTime time.Time
		now          time.Time
		want         time.Duration
	}{
		{
			"no age and all times equal",
			headers(headerMap{"Date": formatTime(currentTime())}),
			currentTime(),
			currentTime(),
			seconds(0),
		},
		{
			"zero age and all times equal",
			headers(headerMap{"Date": formatTime(currentTime()), "Age": "0"}),
			currentTime(),
			currentTime(),
			seconds(0),
		},
		{
			"initial age and all times equal",
			headers(headerMap{"Date": formatTime(currentTime()), "Age": "50"}),
			currentTime(),
			currentTime(),
			seconds(50),
		},
		{
			"date behind response time",
			headers(headerMap{"Date": formatTime(currentTime().Add(5 * time.Second))}),
			currentTime(),
			currentTime().Add(10 * time.Second),
			seconds(10),
		},
		{
			"initial age and date behind response time",
			headers(headerMap{"Date": formatTime(currentTime().Add(10 * time.Second)), "Age": "5"}),
			currentTime(),
			currentTime().Add(10 * time.Second),
			seconds(15),
		},
		{
			"apparent age equals initial age",
			headers(headerMap{"Date": formatTime(currentTime()), "Age": "1"}),
			currentTime().Add(1 * time.Second),
			currentTime().Add(5 * time.Second),
			seconds(5),
		},
		{
			"apparent age less than initial age",
			headers(headerMap{"Date": formatTime(currentTime()), "Age": "3"}),
			currentTime().Add(1 * time.Second),
			currentTime().Add(5 * time.Second),
			seconds(7),
		},
		{
			"apparent age greater than initial age",
			headers(headerMap{"Date": formatTime(currentTime()), "Age": "1"}),
			currentTime().Add(3 * time.Second),
			currentTime().Add(5 * time.Second),
			seconds(5),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, CalculateAge(c.headers, c.responseTime, c.now))
		})
	}
}
