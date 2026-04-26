package builder

import (
	"testing"
	"time"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

func TestBuildAggregateRankingCancelRateByCity(t *testing.T) {
	result, err := BuildAggregate(loadResolvedRankingPlanFixture(), loadCatalogFixture(t))
	if err != nil {
		t.Fatalf("BuildAggregate returned error: %v", err)
	}
	if !contains(result.SQL, "GROUP BY") {
		t.Fatalf("expected GROUP BY in SQL, got %q", result.SQL)
	}
	if !contains(result.SQL, "LIMIT ?") {
		t.Fatalf("expected LIMIT placeholder in SQL, got %q", result.SQL)
	}
	if len(result.Args) == 0 {
		t.Fatalf("expected SQL args to be populated")
	}
	lastArg := result.Args[len(result.Args)-1]
	if limit, ok := lastArg.(int); !ok || limit != 5 {
		t.Fatalf("expected last arg to be limit 5, got %#v", lastArg)
	}
}

func loadResolvedRankingPlanFixture() domain.ResolvedPlan {
	return domain.ResolvedPlan{
		QueryMode:    domain.QueryModeRanking,
		MetricIDs:    []string{"metric.cancel_rate"},
		DimensionIDs: []string{"dimension.city_code"},
		TimeRange: domain.TimeRange{
			Start: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
		},
		Limit:        5,
		DatasourceID: "ride_hailing_ro",
	}
}

func loadCatalogFixture(t *testing.T) catalog.Catalog {
	t.Helper()

	cat, err := catalog.Load("../catalog/testdata/catalog")
	if err != nil {
		t.Fatalf("catalog.Load returned error: %v", err)
	}

	return cat
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
