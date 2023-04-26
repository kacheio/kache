package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kacheio/kache/pkg/kache"
	"gopkg.in/yaml.v3"
)

const (
	configFileOption = "config.file"
)

func main() {
	// Cleanup all flags registered via init() methods of 3rd-party libraries.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// TODO: handle config via flags and env

	config := kache.Config{}

	// Load config file.
	var configFile string

	flag.StringVar(&configFile, configFileOption, configFileOption, "")

	flag.Parse()

	if configFile != "" {
		if err := LoadConfig(configFile, &config); err != nil {
			fmt.Fprintf(os.Stderr, "error loading config from %s: %v\n", configFile, err)
			os.Exit(1)
		}
	}

	// TODO: validate config.

	t, err := kache.New(config)
	if err != nil {
		log.Fatal("initializing application", err)
	}

	log.Println("Starting application", "version", "0.0.1")

	err = t.Run()
	if err != nil {
		log.Fatal("running application", err)
	}
}

// LoadConfig reads the YAML-formatted config from filename into config.
func LoadConfig(filename string, config *kache.Config) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)

	if err := dec.Decode(config); err != nil {
		return err
	}

	return nil
}

// DumpYaml dumps the config to stdout.
func DumpYaml(config *kache.Config) {
	out, err := yaml.Marshal(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Printf("%s\n", out)
	}
}
