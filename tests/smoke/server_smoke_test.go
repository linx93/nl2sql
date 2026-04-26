package smoke_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	internalserver "nl2sql/internal/server"
)

func TestServerBootsWithConfigAndRegistersRoutes(t *testing.T) {
	srv := httptest.NewServer(internalserver.NewMux(filepath.Join("..", "..", "configs")))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("http.Get returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestVerificationChecklistExists(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "plans", "verification-checklist.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected verification checklist file to exist: %v", err)
	}
}
