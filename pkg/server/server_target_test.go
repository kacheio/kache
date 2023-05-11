package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTarget(t *testing.T) {
	targets, err := NewTargets([]*UpstreamConfig{
		{"upstream 1", "http://example.com", "/bot"},
		{"upstream 2", "http://example.com", "/bot"},
		{"upstream 3", "http://example.com", "/news/{id:[0-9]+}"},
		{"upstream 4", "http://example.com", "/"},
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
