package planner

import "testing"

func TestPlannerSchemaRejectsMissingQueryMode(t *testing.T) {
	err := ValidateRawPlanJSON([]byte(`{"metrics":["取消率"]}`))
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err.Error() != "query_mode is required" {
		t.Fatalf("expected query_mode validation error, got %q", err.Error())
	}
}
