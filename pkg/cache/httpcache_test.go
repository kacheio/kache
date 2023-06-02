package cache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookupRequest(t *testing.T) {
	testCases := []struct {
		name            string
		reqCacheControl string
		resCacheControl string
		reqTime         time.Time
		resTime         time.Time
		wantStatus      EntryStatus
		wantAge         string
	}{
		{
			"Request requires revalidation",
			"no-cache",
			"public, max-age=3600",
			currentTime(),
			currentTime(),
			EntryRequiresValidation,
			"0",
		},
		{
			"Response requires revaliation",
			"",
			"no-cache",
			currentTime(),
			currentTime(),
			EntryRequiresValidation,
			"0",
		},
		{
			"Request max age satisfied",
			"max-age=10",
			"public, max-age=3600",
			currentTime().Add(seconds(9)),
			currentTime(),
			EntryOk,
			"9",
		},
		{
			"Request max age unsatisfied",
			"max-age=10",
			"public, max-age=3600",
			currentTime().Add(seconds(11)),
			currentTime(),
			EntryRequiresValidation,
			"11",
		},
		{
			"Request min fresh satisfied",
			"min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(999)),
			currentTime(),
			EntryOk,
			"999",
		},
		{
			"Request min fresh unsatisfied",
			"min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max age satisfied but min fresh unsatisfied",
			"max-age=1500, min-fresh=1000",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max age satisfied but max stale unsatisfied",
			"max-age=1500, max-stale=400",
			"public, max-age=1000",
			currentTime().Add(seconds(1401)),
			currentTime(),
			EntryRequiresValidation,
			"1401",
		},
		{
			"Request max stale satisfied but min fresh unsatisfied",
			"min-fresh=1000, max-stale=500",
			"public, max-age=2000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Request max stale satisfied but max age unsatisfied",
			"max-age=1200, max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1201)),
			currentTime(),
			EntryRequiresValidation,
			"1201",
		},
		{
			"Request min fresh satisfied but max age unsatisfied",
			"max-age=500, min-fresh=400",
			"public, max-age=1000",
			currentTime().Add(seconds(501)),
			currentTime(),
			EntryRequiresValidation,
			"501",
		},
		{
			"Expired",
			"",
			"public, max-age=1000",
			currentTime().Add(seconds(1001)),
			currentTime(),
			EntryRequiresValidation,
			"1001",
		},
		{
			"Expired but max stale satisfied",
			"max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1499)),
			currentTime(),
			EntryOk,
			"1499",
		},
		{
			"Expired and max stale unsatisfied",
			"max-stale=500",
			"public, max-age=1000",
			currentTime().Add(seconds(1501)),
			currentTime(),
			EntryRequiresValidation,
			"1501",
		},
		{
			"Expired and max stale unsatisfied but response must revalidate",
			"max-stale=500",
			"public, max-age=1000, must-revalidate",
			currentTime().Add(seconds(1499)),
			currentTime(),
			EntryRequiresValidation,
			"1499",
		},
		{
			"Fresh but respsonse must revalidate",
			"",
			"public, max-age=1000, must-revalidate",
			currentTime().Add(seconds(999)),
			currentTime(),
			EntryOk,
			"999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add("Cache-Control", tc.reqCacheControl)

			lookup := NewLookupRequest(req, tc.reqTime)

			res := &http.Response{Request: req, Header: make(http.Header, 0)}
			res.Header.Add("Cache-Control", tc.resCacheControl)
			res.Header.Add("Date", tc.resTime.Format(http.TimeFormat))

			result := lookup.makeResult(res, tc.resTime)

			assert.Equal(t, tc.wantStatus, result.Status)
			assert.Equal(t, tc.wantAge, result.Header().Get(HeaderAge))
		})
	}
}

func TestExpiresFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderExpires, currentTime().Add(-seconds(5)).Format(http.TimeFormat))
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))

	result := lookup.makeResult(res, currentTime())

	assert.Equal(t, EntryRequiresValidation, result.Status)
}

func TestExpiresFallbackButFresh(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderExpires, currentTime().Add(seconds(5)).Format(http.TimeFormat))
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))

	result := lookup.makeResult(res, currentTime())

	assert.Equal(t, EntryOk, result.Status)
}

func TestNoCachePragmaFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "no-cache")

	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; But Pragma requires validation.
	assert.Equal(t, EntryRequiresValidation, result.Status)
}

func TestNonNoCachePragmaFallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "max-age=0")

	lookup := NewLookupRequest(req, currentTime())

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; Although Pragma is present, directives other than no-cache are ignored.
	assert.Equal(t, EntryOk, result.Status)
}

func TestNoCachePragmaFallbackIgnored(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(HeaderPragma, "no-cache")
	req.Header.Add(HeaderCacheControl, "max-age=10")

	lookup := NewLookupRequest(req, currentTime().Add(seconds(5)))

	res := &http.Response{Request: req, Header: make(http.Header, 0)}
	res.Header.Add(HeaderDate, currentTime().Format(http.TimeFormat))
	res.Header.Add(HeaderCacheControl, "public, max-age=3600")

	result := lookup.makeResult(res, currentTime())

	// Response is fresh; Cache-Control prioritized over Pragma.
	assert.Equal(t, EntryOk, result.Status)
}
