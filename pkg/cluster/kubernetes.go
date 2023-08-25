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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// client is the kubernetes client.
type client struct {
	clientset *kubernetes.Clientset
	broadcast *RequestQueue

	namespace string
	service   string
}

// Endpoint contains the kubernetes endpoint information.
type Endpoint struct {
	Name string
	Host string
	Port int
}

// NewKubernetesClient create a new kubernetes clientset.
func NewKubernetesClient(namespace string, service string) (Connection, error) {
	// Create the in-cluster config.
	config, err := rest.InClusterConfig()
	if err != nil {
		// Unable to load cluster config, fallback to kube config.
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubernetes config: %v", err)
		}
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	q := NewRequestQueue(RequestQueueOpts{
		Size:       30,
		MaxWorkers: 6,
		MaxRetries: 5,
		Backoff:    7 * time.Second,
	})

	return &client{
		clientset: c,
		broadcast: q,
		namespace: namespace,
		service:   service,
	}, nil
}

// Close closes the connection.
func (c *client) Close() {
	c.broadcast.Stop()
}

// Returns the endpoint addresses to a specific port name.
func (c *client) Endpoints(portname string) []Endpoint {
	eps, err := c.clientset.CoreV1().Endpoints(c.namespace).
		Get(context.Background(), c.service, v1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error getting kubernetes endpoints")
		return nil
	}

	var (
		port      int32
		endpoints []Endpoint
	)

	for _, e := range eps.Subsets {
		for _, p := range e.Ports {
			if p.Name != portname {
				continue
			}
			port = p.Port
		}
		if port == 0 {
			continue
		}
		for _, addr := range e.Addresses {
			// TODO: check endpoint/pod is healthy.
			endpoints = append(endpoints, Endpoint{
				Name: addr.TargetRef.Name,
				Host: addr.IP,
				Port: int(port),
			})
		}
	}

	return endpoints
}

// Broadcast broadcasts the given request to cluster endpoints specified by portname.
func (c *client) Broadcast(req *http.Request, portname, method, path string) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading body")
		return
	}

	endpoints := c.Endpoints(portname)
	log.Debug().Msgf("Cluster broadcast to: %v", endpoints)

	for _, ep := range endpoints {
		// Prepare endpoint request.
		url := fmt.Sprintf("%s://%s:%v%s", "http", ep.Host, ep.Port, path)
		out, err := http.NewRequest(method, url, io.NopCloser(bytes.NewBuffer(body)))
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}
		out.Header = req.Header.Clone()
		out.Header.Set("X-Forwarded-For", req.RemoteAddr)
		out.Header.Set("X-Kache-Cluster", "Broadcast")
		out.Host = req.Host

		// Dispatch message.
		c.broadcast.Queue <- Message{out, 0}
	}
}
