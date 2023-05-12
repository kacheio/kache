package config

import (
	"errors"
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
