package resolver

import (
	"testing"
	"time"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	pkgclock "nl2sql/pkg/clock"
)

func TestResolveAggregateRankingMapsAliasesAndClampsLimit(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:  "ranking",
		Metrics:    []string{"取消率"},
		Dimensions: []string{"城市"},
		Limit:      500,
	}

	plan, err := ResolveAggregate(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveAggregate returned error: %v", err)
	}
	if len(plan.MetricIDs) != 1 || plan.MetricIDs[0] != "metric.cancel_rate" {
		t.Fatalf("expected metric.cancel_rate, got %#v", plan.MetricIDs)
	}
	if len(plan.DimensionIDs) != 1 || plan.DimensionIDs[0] != "dimension.city_code" {
		t.Fatalf("expected dimension.city_code, got %#v", plan.DimensionIDs)
	}
	if plan.Limit != 100 {
		t.Fatalf("expected limit to be clamped to 100, got %d", plan.Limit)
	}
	if plan.DatasourceID != "ride_hailing_ro" {
		t.Fatalf("expected datasource_id ride_hailing_ro, got %q", plan.DatasourceID)
	}
}

func TestResolveAggregateAppliesDefaultTimeRange(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:  "aggregate_overview",
		Metrics:    []string{"取消率"},
		Dimensions: nil,
	}

	plan, err := ResolveAggregate(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveAggregate returned error: %v", err)
	}

	wantStart := time.Date(2026, 3, 29, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	wantEnd := time.Date(2026, 4, 27, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	if !plan.TimeRange.Start.Equal(wantStart) {
		t.Fatalf("expected default start %v, got %v", wantStart, plan.TimeRange.Start)
	}
	if !plan.TimeRange.End.Equal(wantEnd) {
		t.Fatalf("expected default end %v, got %v", wantEnd, plan.TimeRange.End)
	}
}

func TestResolveAggregateRejectsQueryModeOutsideRolePolicy(t *testing.T) {
	role := loadRoleFixture(t)
	role.AllowedQueryModes = []string{"aggregate_overview"}

	raw := domain.RawPlan{
		QueryMode:  "ranking",
		Metrics:    []string{"取消率"},
		Dimensions: []string{"城市"},
	}

	_, err := ResolveAggregate(raw, loadCatalogFixture(t), role, fixedClock())
	if err == nil {
		t.Fatalf("expected permission error")
	}
	if err.Error() != "query mode not allowed: ranking" {
		t.Fatalf("expected query mode rejection, got %q", err.Error())
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

func loadRoleFixture(t *testing.T) catalog.RolePolicy {
	t.Helper()

	cat := loadCatalogFixture(t)
	role, ok := cat.Roles["analyst"]
	if !ok {
		t.Fatalf("expected analyst role fixture to exist")
	}

	return role
}

func fixedClock() pkgclock.Clock {
	return stubClock{
		now: time.Date(2026, 4, 27, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
	}
}

type stubClock struct {
	now time.Time
}

func (c stubClock) Now() time.Time {
	return c.now
}
