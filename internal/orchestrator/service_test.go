package orchestrator

import (
	"context"
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
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resp.Meta.QueryMode != "ranking" {
		t.Fatalf("expected ranking query mode, got %q", resp.Meta.QueryMode)
	}
}

func newServiceWithFakes(t *testing.T) Service {
	t.Helper()

	cat, err := catalog.Load("../catalog/testdata/catalog")
	if err != nil {
		t.Fatalf("catalog.Load returned error: %v", err)
	}

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
		executor: fakeExecutor{
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
		},
		auditor: fakeAuditor{},
	}
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

func (f fakeExecutor) Query(_ context.Context, _ string, _ []any) (formatter.QueryResult, error) {
	return f.result, nil
}

type fakeAuditor struct{}

func (fakeAuditor) Save(_ context.Context, _ audit.Entry) error {
	return nil
}
