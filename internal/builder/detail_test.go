package builder

import (
	"testing"
	"time"

	"nl2sql/internal/domain"
)

func TestBuildDetailListUsesAllowedColumnsAndDefaultSort(t *testing.T) {
	result, err := BuildDetail(loadResolvedDetailPlanFixture(), loadCatalogFixture(t))
	if err != nil {
		t.Fatalf("BuildDetail returned error: %v", err)
	}
	if contains(result.SQL, "*") {
		t.Fatalf("expected explicit columns instead of wildcard, got %q", result.SQL)
	}
	if !contains(result.SQL, "ORDER BY trip_orders.called_at DESC") {
		t.Fatalf("expected default sort in SQL, got %q", result.SQL)
	}
	if !contains(result.SQL, "LIMIT ?") {
		t.Fatalf("expected LIMIT placeholder in SQL, got %q", result.SQL)
	}
}

func loadResolvedDetailPlanFixture() domain.ResolvedPlan {
	return domain.ResolvedPlan{
		QueryMode:    domain.QueryModeDetailList,
		DetailViewID: "detail.waiting_pickup_orders",
		SelectColumnIDs: []string{
			"trip_orders.order_id",
			"trip_orders.city_code",
			"trip_orders.service_type",
			"trip_orders.order_status",
			"trip_orders.called_at",
		},
		TimeRange: domain.TimeRange{
			Start: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 27, 23, 59, 59, 0, time.UTC),
		},
		Limit:        50,
		DatasourceID: "ride_hailing_ro",
	}
}
