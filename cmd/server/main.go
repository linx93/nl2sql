package main

import (
	"log"

	internalserver "nl2sql/internal/server"
)

func main() {
	if _, err := internalserver.LoadAndValidateCatalog("configs"); err != nil {
		log.Fatal(err)
	}

	log.Fatal("query service wiring is not configured; construct a real orchestrator service before starting the server")
}
