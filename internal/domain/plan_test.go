package domain

import "testing"

func TestResolvedPlanRejectsMissingDatasourceID(t *testing.T) {
	plan := ResolvedPlan{QueryMode: QueryModeRanking}

	err := plan.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err.Error() != "datasource_id is required" {
		t.Fatalf("expected datasource_id validation error, got %q", err.Error())
	}
}
