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
