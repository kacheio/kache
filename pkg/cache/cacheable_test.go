package cache

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	req *http.Request
	res *http.Response
)

func setupRequest(t *testing.T) {
	r, err := http.NewRequest("GET", "http://example.com/", nil)
	require.NoError(t, err)
	req = r
}

func setupResponse(t *testing.T) {
	resHeaders := http.Header{}
	resHeaders.Set("Cache-Control", "max-age=7200")
	resHeaders.Set("Date", "Mon, 15 May 2023 13:42:52 GMT")
	res = &http.Response{StatusCode: 200, Header: resHeaders}
}

func TestCanServeRequestFromCache(t *testing.T) {
	t.Run("Cacheable request", func(t *testing.T) {
		setupRequest(t)
		assert.True(t, IsCacheableRequest(req))
	})
	t.Run("Path", func(t *testing.T) {
		setupRequest(t)
		assert.True(t, IsCacheableRequest(req))
		req.URL.Path = ""
		assert.False(t, IsCacheableRequest(req))
	})
	t.Run("Host", func(t *testing.T) {
		setupRequest(t)
		assert.True(t, IsCacheableRequest(req))
		req.Host = ""
		assert.False(t, IsCacheableRequest(req))
	})
	t.Run("Method", func(t *testing.T) {
		setupRequest(t)
		req.Method = http.MethodGet
		assert.True(t, IsCacheableRequest(req))
		req.Method = http.MethodHead
		assert.True(t, IsCacheableRequest(req))
		req.Method = http.MethodPost
		assert.False(t, IsCacheableRequest(req))
		req.Method = http.MethodPut
		assert.False(t, IsCacheableRequest(req))
		req.Method = ""
		assert.False(t, IsCacheableRequest(req))
	})
	t.Run("Authorization header", func(t *testing.T) {
		setupRequest(t)
		assert.True(t, IsCacheableRequest(req))
		req.Header.Set(HeaderAuthorization, "Basic bmFtZTpwYXNzd29yZA==")
		assert.False(t, IsCacheableRequest(req))
	})
	t.Run("Conditional headers", func(t *testing.T) {
		setupRequest(t)
		headers := [...]string{"if-match", "if-none-match", "if-modified-since",
			"if-unmodified-since", "if-range"}
		for _, header := range headers {
			t.Run(header, func(t *testing.T) {
				assert.True(t, IsCacheableRequest(req))
				req.Header.Set(header, "some value")
				assert.False(t, IsCacheableRequest(req))
			})
			req.Header.Del(header)
		}
	})
}

func TestIsCacheableResponse(t *testing.T) {
	t.Run("Cacheable response", func(t *testing.T) {
		setupResponse(t)
		assert.True(t, IsCacheableResponse(res))
	})
	t.Run("Non cacheable status code", func(t *testing.T) {
		setupResponse(t)
		assert.True(t, IsCacheableResponse(res))
		res.StatusCode = 600
		assert.False(t, IsCacheableResponse(res))
	})
	t.Run("Validation data", func(t *testing.T) {
		setupResponse(t)
		assert.True(t, IsCacheableResponse(res))
		// w/o cache-control
		res.Header.Del(HeaderCacheControl)
		assert.False(t, IsCacheableResponse(res))
		// no max-age or expires
		res.Header.Set(HeaderCacheControl, "public, no-transform")
		assert.False(t, IsCacheableResponse(res))
		// max-age provided
		res.Header.Set(HeaderCacheControl, "s-maxage=1337")
		assert.True(t, IsCacheableResponse(res))
		// no max-age, but must validate
		res.Header.Set(HeaderCacheControl, "no-cache")
		assert.True(t, IsCacheableResponse(res))
		// no cache-control, but expires
		res.Header.Del(HeaderCacheControl)
		res.Header.Set(HeaderExpires, "Thu, 04 May 2023 13:42:52 GMT")
		assert.True(t, IsCacheableResponse(res))
	})
	t.Run("NoStore directive", func(t *testing.T) {
		setupResponse(t)
		assert.True(t, IsCacheableResponse(res))
		res.Header.Set(HeaderCacheControl, res.Header.Get(HeaderCacheControl)+", no-store")
		assert.False(t, IsCacheableResponse(res))
	})
	t.Run("Private directive", func(t *testing.T) {
		setupResponse(t)
		assert.True(t, IsCacheableResponse(res))
		res.Header.Set(HeaderCacheControl, res.Header.Get(HeaderCacheControl)+", private")
		assert.False(t, IsCacheableResponse(res))
	})
}
