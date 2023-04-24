package main

import (
	"log"

	"github.com/toashd/kache/pkg/kache"
)

func main() {

	config := kache.Config{}

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
