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
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Loader loads a configuration from file.
type Loader struct {
	path string

	watch         bool
	watchInterval time.Duration

	config     atomic.Pointer[Configuration]
	configHash []byte

	Events chan bool
	done   chan struct{}
}

// NewFileLoader creates a new config Loader.
func NewLoader(path string, watch bool, interval time.Duration) (*Loader, error) {
	ldr := &Loader{
		path:          path,
		watch:         watch,
		watchInterval: interval,
		Events:        make(chan bool),
		done:          make(chan struct{}),
	}
	if _, err := ldr.Load(context.Background()); err != nil {
		return nil, err
	}
	return ldr, nil
}

// Load reads the YAML-formatted config.
func (l *Loader) Load(ctx context.Context) (bool, error) {
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
func (l *Loader) Config() *Configuration {
	return l.config.Load()
}

// Path returns the file path.
func (l *Loader) Path() string {
	return l.path
}

// Checksum returns the calculated checksum of the config.
func (l *Loader) Checksum() string {
	return hex.EncodeToString(l.configHash)
}

// AutoReload returns true if auto-reloading is enabled.
func (l *Loader) AutoReload() bool {
	return l.watch
}

// Watch watches and reloads the config file if changed.
func (l *Loader) Watch(ctx context.Context) error {
	if _, err := l.Load(ctx); err != nil {
		return err
	}
	go func() {
		tick := time.NewTicker(l.watchInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}

			changed, err := l.Load(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Error reloading config file")
			}
			if changed {
				l.notifyChange()
			}
		}
	}()
	return nil
}

// Close closes the events channel.
func (l *Loader) Close() {
	close(l.done)
}

// notifyChange sends to the Event channel.
func (l *Loader) notifyChange() bool {
	select {
	case l.Events <- true:
		return true
	case <-l.done:
	}
	return false
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
