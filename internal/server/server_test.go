package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nl2sql/internal/formatter"
	"nl2sql/internal/orchestrator"
)

func TestNewMuxRoutesQueriesToInjectedService(t *testing.T) {
	mux := NewMux(fakeQueryService{
		response: orchestrator.Response{
			RequestID: "req-001",
			Data: formatter.ResponseData{
				Summary:    "共返回1条聚合结果。",
				ResultKind: "aggregate",
				RowCount:   1,
			},
			Meta: orchestrator.Meta{
				QueryMode:  "aggregate_overview",
				ResultKind: "aggregate",
				RowCount:   1,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近30天取消率","domain":"ride_hailing"}`))
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"status":"success"`) {
		t.Fatalf("expected success payload, got %s", res.Body.String())
	}
}

type fakeQueryService struct {
	response orchestrator.Response
}

func (f fakeQueryService) Run(_ context.Context, _ orchestrator.QueryRequest) (orchestrator.Response, error) {
	return f.response, nil
}
