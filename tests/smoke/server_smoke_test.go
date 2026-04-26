package smoke_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nl2sql/internal/formatter"
	"nl2sql/internal/orchestrator"
	internalserver "nl2sql/internal/server"
)

func TestServerBootsWithConfigAndRegistersRoutes(t *testing.T) {
	if _, err := internalserver.LoadAndValidateCatalog(filepath.Join("..", "..", "configs")); err != nil {
		t.Fatalf("LoadAndValidateCatalog returned error: %v", err)
	}

	srv := httptest.NewServer(internalserver.NewMux(smokeService{}))
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

type smokeService struct{}

func (smokeService) Run(_ context.Context, _ orchestrator.QueryRequest) (orchestrator.Response, error) {
	return orchestrator.Response{
		RequestID: "req-smoke",
		Data: formatter.ResponseData{
			Summary:    "共返回1条聚合结果。",
			ResultKind: "aggregate",
			RowCount:   1,
		},
		Meta: orchestrator.Meta{
			QueryMode:  "aggregate_overview",
			ResultKind: "aggregate",
			RowCount:   1,
		},
	}, nil
}
