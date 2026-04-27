package main

import (
	"log"
	"net/http"
	"os"

	internalserver "nl2sql/internal/server"
)

func main() {
	runtime, err := internalserver.BuildRuntime(internalserver.RuntimeConfig{
		ConfigDir:      "configs",
		MiniMaxAPIKey:  os.Getenv("MINIMAX_API_KEY"),
		MiniMaxModel:   os.Getenv("MINIMAX_MODEL"),
		MiniMaxBaseURL: os.Getenv("MINIMAX_BASE_URL"),
		AuditDSN:       os.Getenv("MYSQL_NL2SQL_AUDIT_DSN"),
	})
	if err != nil {
		log.Fatal(err)
	}

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	log.Printf("nl2sql server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, runtime.Mux))
}

