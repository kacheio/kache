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
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

// Loader is the loader interface.
type Loader interface {
	Load(ctx context.Context) (bool, error)
	Config() *Configuration
	Path() string
}

// fileLoader loads a configuration from file.
type fileLoader struct {
	path       string
	config     atomic.Pointer[Configuration]
	configHash []byte
}

// NewFileLoader creates a new config Loader.
func NewFileLoader(path string) (Loader, error) {
	ldr := &fileLoader{path: path}
	if _, err := ldr.Load(context.Background()); err != nil {
		return nil, err
	}
	return ldr, nil
}

// Load reads the YAML-formatted config.
func (l *fileLoader) Load(ctx context.Context) (bool, error) {
	buf, err := os.ReadFile(l.path)
	if err != nil {
		return false, err
	}

	sum := md5.Sum(buf)
	hash := sum[:]
	if bytes.Equal(l.configHash, hash) {
		return false, nil
	}
	l.configHash = hash

	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)

	config := &Configuration{}
	if err := dec.Decode(config); err != nil {
		return false, err
	}
	l.config.Store(config)

	return true, nil
}

// Config returns the loaded config.
func (l *fileLoader) Config() *Configuration {
	return l.config.Load()
}

// Path returns the file path.
func (l *fileLoader) Path() string {
	return l.path
}

// Checksum return the calculated checksum of the config.
func (l *fileLoader) Checksum() string {
	return hex.EncodeToString(l.configHash)
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
