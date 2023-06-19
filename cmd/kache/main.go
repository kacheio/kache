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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/kache"
	"github.com/kacheio/kache/pkg/utils/logger"
	"github.com/rs/zerolog/log"
)

const (
	configFileOption = "config.file"
	configFileName   = "kache.yml"
)

func main() {
	// Cleanup all flags registered via init() methods of 3rd-party libraries.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// TODO: handle config via flags and env

	cfg := config.Configuration{}
	ldr := config.NewFileLoader()

	// Load config file.
	var configFile string

	flag.StringVar(&configFile, configFileOption, configFileName, "")
	flag.Parse()

	if configFile != "" {
		if err := ldr.Load(configFile, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error loading config from %s: %v\n", configFile, err)
			os.Exit(1)
		}
	}

	if err := cfg.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error validating config:\n%v\n", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.Log)

	log.Info().Msg("Kache is starting")
	log.Info().Str("config", configFile).Msg("Kache initializing application")

	t, err := kache.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Initializing application")
	}

	err = t.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Running application")
	}
}
