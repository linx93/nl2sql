package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nl2sql/internal/formatter"
	"nl2sql/internal/orchestrator"
)

func TestPostQueriesReturnsSuccessPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近7天完单数","domain":"ride_hailing"}`))
	res := httptest.NewRecorder()
	handler := newHandlerWithFakeService()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"status":"success"`) {
		t.Fatalf("expected success payload, got %s", res.Body.String())
	}
}

func TestPostQueriesPassesIdentityHeadersToService(t *testing.T) {
	service := &capturingService{
		response: orchestrator.Response{
			RequestID: "req-001",
			Data: formatter.ResponseData{
				Summary:    "共返回1条聚合结果。",
				ResultKind: "aggregate",
				RowCount:   1,
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近7天完单数","domain":"ride_hailing"}`))
	req.Header.Set("X-User-ID", "user-001")
	req.Header.Set("X-User-Role", "analyst")
	res := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(res, req)

	if service.request.UserID != "user-001" {
		t.Fatalf("expected user id to be forwarded, got %q", service.request.UserID)
	}
	if service.request.UserRole != "analyst" {
		t.Fatalf("expected user role to be forwarded, got %q", service.request.UserRole)
	}
}

func TestPostQueriesReturnsForbiddenForPermissionError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近7天完单数","domain":"ride_hailing"}`))
	res := httptest.NewRecorder()

	NewHandler(errorService{err: orchestrator.ErrPermissionDenied}).ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestPostQueriesReturnsBadRequestForInvalidQueryError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近7天待接驾订单明细","domain":"ride_hailing"}`))
	res := httptest.NewRecorder()

	NewHandler(errorService{err: orchestrator.ErrInvalidQuery}).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"INVALID_QUERY"`) {
		t.Fatalf("expected INVALID_QUERY error code, got %s", res.Body.String())
	}
}

func newHandlerWithFakeService() *Handler {
	return NewHandler(fakeService{
		response: orchestrator.Response{
			RequestID: "req-001",
			Data: formatter.ResponseData{
				Summary:   "共返回1条聚合结果。",
				ResultKind: "aggregate",
				RowCount:  1,
			},
			Meta: orchestrator.Meta{
				QueryMode:  "aggregate_overview",
				ResultKind: "aggregate",
				RowCount:   1,
			},
		},
	})
}

type fakeService struct {
	response orchestrator.Response
}

func (f fakeService) Run(_ context.Context, _ orchestrator.QueryRequest) (orchestrator.Response, error) {
	return f.response, nil
}

type capturingService struct {
	request  orchestrator.QueryRequest
	response orchestrator.Response
}

func (s *capturingService) Run(_ context.Context, req orchestrator.QueryRequest) (orchestrator.Response, error) {
	s.request = req
	return s.response, nil
}

type errorService struct {
	err error
}

func (s errorService) Run(_ context.Context, _ orchestrator.QueryRequest) (orchestrator.Response, error) {
	return orchestrator.Response{}, fmt.Errorf("%w", s.err)
}
