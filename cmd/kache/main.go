package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/kacheio/kache/pkg/config"
	"github.com/kacheio/kache/pkg/kache"
	"github.com/kacheio/kache/pkg/utils/logger"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
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

	// Load config file.
	var configFile string

	flag.StringVar(&configFile, configFileOption, configFileName, "")
	flag.Parse()

	if configFile != "" {
		if err := LoadConfig(configFile, &cfg); err != nil {
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

// LoadConfig reads the YAML-formatted config from filename into config.
func LoadConfig(filename string, cfg *config.Configuration) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)

	return dec.Decode(cfg)
}

// DumpYaml dumps the config to stdout.
func DumpYaml(cfg *config.Configuration) {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Printf("%s\n", out)
	}
}
