package smoke_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerificationDocsMentionMiniMaxLiveDefault(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "project-constraints.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "default tests may call the live MiniMax planner") {
		t.Fatalf("expected live MiniMax default-test rule in project constraints")
	}
}

func TestVerificationDocsDescribeLiveMiniMaxRequirements(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "plans", "verification-checklist.md"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(raw)
	if !strings.Contains(content, "MINIMAX_API_KEY") {
		t.Fatalf("expected verification checklist to mention MINIMAX_API_KEY")
	}
	if !strings.Contains(content, "MYSQL_RIDE_HAILING_ROOT_DSN") {
		t.Fatalf("expected verification checklist to mention MYSQL_RIDE_HAILING_ROOT_DSN")
	}
	if !strings.Contains(content, "MYSQL_RIDE_HAILING_RO_DSN") {
		t.Fatalf("expected verification checklist to mention MYSQL_RIDE_HAILING_RO_DSN")
	}
	if !strings.Contains(content, "go test ./...") {
		t.Fatalf("expected verification checklist to mention go test ./...")
	}
}
