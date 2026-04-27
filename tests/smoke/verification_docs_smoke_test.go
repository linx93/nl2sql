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

