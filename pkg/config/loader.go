package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader is the loader interface.
type Loader interface {
	Load(string, *Configuration) error
}

// fileLoader loads a configuration from file.
type fileLoader struct{}

// NewFileLoader creates a new config Loader.
func NewFileLoader() Loader {
	return &fileLoader{}
}

// LoadConfig reads the YAML-formatted config from filename into config.
func (l *fileLoader) Load(filename string, config *Configuration) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)

	return dec.Decode(config)
}

// DumpYaml dumps the config to stdout.
func DumpYaml(config *Configuration) {
	out, err := yaml.Marshal(config)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	} else {
		_, _ = fmt.Printf("%s\n", out)
	}
}

// TODO: Find configuration.
