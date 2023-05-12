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

	t, err := kache.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("initializing application")
	}

	log.Info().Msgf("Starting application version 0.0.1")

	err = t.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("running application")
	}
}
