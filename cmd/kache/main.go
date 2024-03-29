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
	"time"

	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/kache"
	"github.com/kacheio/kache/pkg/utils/logger"
	"github.com/kacheio/kache/pkg/utils/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	configFileName = "kache.yml"

	configFileOption          = "config.file"
	configAutoReloadOption    = "config.auto-reload"
	configWatchIntervalOption = "config.watch-interval"

	versionOption = "version"
	versionUsage  = "Print application version and exit."
)

func init() {
	prometheus.MustRegister(version.NewCollector("kache"))
}

func main() {
	// Cleanup all flags registered via init() methods of 3rd-party libraries.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	var printVersion bool
	flag.BoolVar(&printVersion, versionOption, false, versionUsage)

	var configAutoReload bool
	flag.BoolVar(&configAutoReload, configAutoReloadOption, false, "")

	var configWatchInterval time.Duration
	flag.DurationVar(&configWatchInterval, configWatchIntervalOption, 10*time.Second, "")

	var configFile string
	flag.StringVar(&configFile, configFileOption, configFileName, "")

	flag.Parse()

	if printVersion {
		_, _ = fmt.Fprintln(os.Stdout, version.Print("Kache"))
		return
	}

	// Load config file.
	ldr, err := config.NewLoader(configFile, configAutoReload, configWatchInterval)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error loading config from %s: %v\n", configFile, err)
		os.Exit(1)
	}

	cfg := ldr.Config()

	if err := cfg.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error validating config:\n%v\n", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.Log)

	log.Info().Msg("Kache is starting")
	log.Info().Str("config", configFile).Msg("Kache initializing application")

	t, err := kache.New(ldr, prometheus.DefaultRegisterer)
	if err != nil {
		log.Fatal().Err(err).Msg("Initializing application")
	}

	err = t.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Running application")
	}
}
