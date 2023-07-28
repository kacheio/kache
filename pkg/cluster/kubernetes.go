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
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// client is the kubernetes client.
type client struct {
	clientset *kubernetes.Clientset

	namespace string
	service   string
}

// Endpoint contains kubernetes endpoint information.
type Endpoint struct {
	Name string
	Host string
	Port int
}

// type Signal struct {
// 	req *http.Request
// 	portname string
// 	path string
// }

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

	return &client{
		clientset: c,
		namespace: namespace,
		service:   service,
	}, nil
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

// Broadcast broadcast a request to endpoints.
func (c *client) Broadcast(req *http.Request, portname string, path string) {
	endpoints := c.Endpoints(portname)
	log.Info().Msgf("Cluster broadcast to: %v", endpoints)

	for _, ep := range endpoints {
		url := fmt.Sprintf("%s://%s:%v%s", "http", ep.Host, ep.Port, path)
		out, err := http.NewRequest(http.MethodDelete, url, req.Body)
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}
		out.Header = req.Header.Clone()
		out.Header.Set("X-Forwarded-For", req.RemoteAddr)
		out.Host = req.Host

		// TODO: enque this in an async queue; handle error, retry.
		client := &http.Client{}
		resp, err := client.Do(out)
		if err != nil {
			log.Error().Err(err).Str("url", url).Msg("Error signaling node")
			continue
		}

		log.Info().Msgf("%v: %v", url, resp)
	}
}

// use asynch queue
// b.signalQueue <- Signal{request, 0}
