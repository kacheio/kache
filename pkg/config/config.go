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

package config

import (
	"errors"

	"github.com/kacheio/kache/pkg/cache"
	"github.com/kacheio/kache/pkg/provider"
)

var (
	errInvalidListenersConfig = errors.New("invalid listeners config")
	errInvalidUpstreamsConfig = errors.New("invalid upstreams config")
)

// TODO: set defaults

// Configuration is the root configuration.
type Configuration struct {
	Listeners Listeners `yaml:"listeners"`
	Upstreams Upstreams `yaml:"upstreams"`

	HttpCache *cache.HttpCacheConfig          `yaml:"cache"`
	Provider  *provider.ProviderBackendConfig `yaml:"provider"`

	API *API `yaml:"api"`
	Log *Log `yaml:"logging"`
}

// Validate validates the configuration.
func (c *Configuration) Validate() error {
	return errors.Join(
		c.Listeners.Validate(),
		c.Upstreams.Validate(),
	)
}

// Global holds the global configuration.
type Global struct {
	ApplicationName string `yaml:"-"`
	HTTPAddr        string `yaml:"host"`
}

// Listeners holds the listener configs.
type Listeners map[string]*Listener

// Listener holds the listener config.
type Listener struct {
	Addr string `yaml:"addr"`
}

// Validate validates the listener config.
func (l Listeners) Validate() error {
	if len(l) < 1 {
		return errInvalidListenersConfig
	}
	return nil
}

// Upstreams holds the upstream configs.
type Upstreams []*Upstream

// Upstream holds the upstream target config.
type Upstream struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	Path string `yaml:"path"`
}

// Validate validates the upstream config.
func (u Upstreams) Validate() error {
	if len(u) < 1 {
		return errInvalidUpstreamsConfig
	}
	return nil
}

// API holds the API configuration.
type API struct {
	Port  int    `yaml:"port"`
	Path  string `yaml:"path,omitempty"`
	Debug bool   `yaml:"debug,omitempty"`
}

// Log holds the logger configuration.
type Log struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`

	FilePath   string `yaml:"filePath,omitempty"`
	MaxSize    int    `yaml:"maxSize,omitempty"`
	MaxAge     int    `yaml:"maxAge,omitempty"`
	MaxBackups int    `yaml:"maxBackups,omitempty"`
	Compress   bool   `yaml:"compress,omitempty"`
}
