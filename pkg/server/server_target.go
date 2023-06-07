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
func NewTarget(cfg *config.Upstream) (*Target, error) {
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
