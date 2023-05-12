package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/config"
)

// Targets holds an array of targets.
type Targets []*Target

// NewTargets creates new upstream targets.
func NewTargets(upstreamsConfig config.Upstreams) (Targets, error) {
	targets := make(Targets, len(upstreamsConfig))
	for i, config := range upstreamsConfig {
		t, err := NewTarget(config)
		if err != nil {
			return nil, fmt.Errorf("error while creating target %s: %w", config.Name, err)
		}
		targets[i] = t
	}
	return targets, nil
}

// MatchTarget matches a given request to a registered route.
func (tgs Targets) MatchTarget(req *http.Request) (*Target, bool) {
	for _, t := range tgs {
		m := &mux.RouteMatch{}
		if t.router.Match(req, m) { // find first
			return t, true
		}
	}
	return nil, false
}

// Target holds information about the upstream target.
type Target struct {
	name     string
	upstream *url.URL
	router   *mux.Router
}

// NewTarget creates a new upstream target.
func NewTarget(cfg *config.UpstreamConfig) (*Target, error) {
	u, err := url.Parse(cfg.Addr)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter()
	r.PathPrefix(cfg.Path) // TODO: add host?

	t := &Target{
		name:     cfg.Name,
		upstream: u,
		router:   r,
	}
	return t, nil
}
