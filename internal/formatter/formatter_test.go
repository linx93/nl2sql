package formatter

import (
	"testing"

	"nl2sql/internal/domain"
)

func TestFormatterBuildsFactSummaryForDetailResult(t *testing.T) {
	resp := Format(loadDetailResultFixture())

	if !contains(resp.Summary, "前50条") {
		t.Fatalf("expected summary to mention 前50条, got %q", resp.Summary)
	}
	if !resp.Truncated {
		t.Fatalf("expected response to be marked truncated")
	}
}

func loadDetailResultFixture() QueryResult {
	return QueryResult{
		QueryMode: domain.QueryModeDetailList,
		Columns: []Column{
			{Name: "order_id", Label: "订单号"},
		},
		Rows: [][]any{
			{"A1001"},
		},
		Limit:      50,
		Truncated:  true,
		TotalRows:  50,
		ResultKind: "detail",
	}
}

func contains(text string, want string) bool {
	return len(want) == 0 || (len(text) >= len(want) && index(text, want) >= 0)
}

func index(text string, want string) int {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}
