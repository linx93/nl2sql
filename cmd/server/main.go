package main

import (
	"log"
	"net/http"

	internalserver "nl2sql/internal/server"
)

func main() {
	if err := http.ListenAndServe(":8080", internalserver.NewMux("configs")); err != nil {
		log.Fatal(err)
	}
}
