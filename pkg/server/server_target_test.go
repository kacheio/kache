package server

import (
	"net/http"
	"testing"

	"github.com/kacheio/kache/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTarget(t *testing.T) {
	targets, err := NewTargets([]*config.UpstreamConfig{
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
