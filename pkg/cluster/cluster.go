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

package cluster

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Config contains the cluster configuration.
type Config struct {
	Discovery string `yaml:"discovery"`
	Namespace string `yaml:"namespace"`
	Service   string `yaml:"service"`
}

// Connection is the cluster connection interface.
type Connection interface {
	Endpoints(portname string) []Endpoint
	Broadcast(req *http.Request, portname, method, path string)
	Close()
}

// NewConnection create a new cluster connection.
func NewConnection(config *Config) (Connection, error) {
	if config.Discovery == "kubernetes" {
		kc, err := NewKubernetesClient(config.Namespace, config.Service)
		if err != nil {
			log.Error().Err(err).
				Str("cluster", config.Discovery).
				Msg("Error creating cluster connection")
		}
		return kc, err
	}
	return nil, fmt.Errorf("unknown cluster discovery provider: %v", config.Discovery)
}
