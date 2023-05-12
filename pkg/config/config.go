package config

// Configuration is the root configuration.
type Configuration struct {
	Endpoints Endpoints `yaml:"listeners"`
	Upstreams Upstreams `yaml:"upstreams"`

	API API `yaml:"api"`
	Log Log `yaml:"logging"`
}

// Global holds the global configuration.
type Global struct {
	ApplicationName string `yaml:"-"`
	HTTPAddr        string `yaml:"host"`
}

// Endpoints holds the listeners.
type Endpoints map[string]*EndpointConfig

type EndpointConfig struct {
	Addr string `yaml:"addr"`
}

// Upstreams holds the upstreams
type Upstreams []*UpstreamConfig

// UpstreamConfig ....
type UpstreamConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	Path string `yaml:"path"`
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
