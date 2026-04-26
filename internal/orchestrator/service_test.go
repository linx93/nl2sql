package orchestrator

import (
	"context"
	"errors"
	"testing"

	"nl2sql/internal/audit"
	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	"nl2sql/internal/formatter"
)

func TestOrchestratorRunsAggregateQueryEndToEnd(t *testing.T) {
	svc := newServiceWithFakes(t)

	resp, err := svc.Run(context.Background(), QueryRequest{
		Query:  "最近30天取消率最高的前5个城市",
		Domain: "ride_hailing",
		UserRole: "analyst",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resp.Meta.QueryMode != "ranking" {
		t.Fatalf("expected ranking query mode, got %q", resp.Meta.QueryMode)
	}
}

func TestOrchestratorRejectsUnknownRole(t *testing.T) {
	svc := newServiceWithFakes(t)

	_, err := svc.Run(context.Background(), QueryRequest{
		Query:    "最近30天取消率最高的前5个城市",
		Domain:   "ride_hailing",
		UserRole: "guest",
	})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestOrchestratorPassesResolvedDatasourceToExecutor(t *testing.T) {
	executor := &capturingExecutor{
		result: formatter.QueryResult{
			Columns: []formatter.Column{{Name: "cancel_rate", Label: "取消率"}},
			Rows:    [][]any{{0.12}},
		},
	}
	svc := newServiceWithExecutor(t, executor)

	_, err := svc.Run(context.Background(), QueryRequest{
		Query:    "最近30天取消率",
		Domain:   "ride_hailing",
		UserRole: "analyst",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if executor.datasourceID != "ride_hailing_ro" {
		t.Fatalf("expected executor to receive datasource_id ride_hailing_ro, got %q", executor.datasourceID)
	}
}

func TestScopeCatalogToDomainKeepsOnlyRequestedDomain(t *testing.T) {
	cat := loadCatalogFixture(t)
	cat.Domains["finance"] = catalog.DomainSpec{
		ID:           "finance",
		DisplayName:  "财务域",
		DatasourceID: "finance_ro",
		Enabled:      true,
	}
	cat.AliasesByDomain["finance"] = catalog.AliasSet{
		Metrics: map[string]string{
			"营收": "metric.revenue",
		},
	}
	cat.Metrics["metric.revenue"] = catalog.MetricSpec{
		ID:            "metric.revenue",
		DomainID:      "finance",
		DisplayName:   "营收",
		BaseTable:     "trip_orders",
		SQLExpression: "SUM(trip_orders.gmv_amount)",
		TimeField:     "called_at",
		Enabled:       true,
	}

	scoped, err := scopeCatalogToDomain(cat, "finance")
	if err != nil {
		t.Fatalf("scopeCatalogToDomain returned error: %v", err)
	}
	if len(scoped.Domains) != 1 {
		t.Fatalf("expected one scoped domain, got %#v", scoped.Domains)
	}
	if _, ok := scoped.Domains["finance"]; !ok {
		t.Fatalf("expected finance domain in scoped catalog")
	}
	if _, ok := scoped.Domains["ride_hailing"]; ok {
		t.Fatalf("did not expect ride_hailing domain in scoped catalog")
	}
	if _, ok := scoped.Metrics["metric.revenue"]; !ok {
		t.Fatalf("expected finance metric in scoped catalog")
	}
	if _, ok := scoped.Metrics["metric.cancel_rate"]; ok {
		t.Fatalf("did not expect ride_hailing metric in scoped catalog")
	}
}

func newServiceWithFakes(t *testing.T) Service {
	t.Helper()

	cat := loadCatalogFixture(t)
	return newServiceWithCatalog(cat, fakeExecutor{
		result: formatter.QueryResult{
			Columns: []formatter.Column{
				{Name: "city_code", Label: "城市"},
				{Name: "cancel_rate", Label: "取消率"},
			},
			Rows: [][]any{
				{"310000", 0.12},
			},
			ResultKind: "aggregate",
		},
	})
}

func newServiceWithExecutor(t *testing.T, executor Executor) Service {
	t.Helper()

	cat := loadCatalogFixture(t)
	return newServiceWithCatalog(cat, executor)
}

func newServiceWithCatalog(cat catalog.Catalog, executor Executor) Service {
	return Service{
		catalog: cat,
		planner: fakePlanner{
			rawPlan: domain.RawPlan{
				QueryMode:  "ranking",
				Metrics:    []string{"取消率"},
				Dimensions: []string{"城市"},
				Limit:      5,
			},
		},
		executor: executor,
		auditor:  fakeAuditor{},
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

type fakePlanner struct {
	rawPlan domain.RawPlan
}

func (f fakePlanner) Plan(_ context.Context, _ string, _ string) (domain.RawPlan, error) {
	return f.rawPlan, nil
}

type fakeExecutor struct {
	result formatter.QueryResult
}

func (f fakeExecutor) Query(_ context.Context, _ string, _ string, _ []any) (formatter.QueryResult, error) {
	return f.result, nil
}

type capturingExecutor struct {
	datasourceID string
	result       formatter.QueryResult
}

func (e *capturingExecutor) Query(_ context.Context, datasourceID string, _ string, _ []any) (formatter.QueryResult, error) {
	e.datasourceID = datasourceID
	return e.result, nil
}

type fakeAuditor struct{}

func (fakeAuditor) Save(_ context.Context, _ audit.Entry) error {
	return nil
}
