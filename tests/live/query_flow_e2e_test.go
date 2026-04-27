package live_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	internalserver "nl2sql/internal/server"
	"nl2sql/tests/testsupport/mysqlbootstrap"
)

func TestLiveQueryFlowSupportsAggregateOverview(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近30天取消率是多少",
		Domain:   "ride_hailing",
		UserID:   "user-aggregate",
		UserRole: "analyst",
	})

	response.requireSuccess(t, http.StatusOK, "aggregate_overview")
	require.Len(t, response.DataRows(), 1)
	require.Equal(t, "success", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "aggregate_overview", fixture.latestAudit(t).QueryMode)
}

func TestLiveQueryFlowSupportsRanking(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近30天取消率最高的城市",
		Domain:   "ride_hailing",
		UserID:   "user-ranking",
		UserRole: "analyst",
	})

	response.requireSuccess(t, http.StatusOK, "ranking")
	require.Len(t, response.DataRows(), 3)
	require.Equal(t, "310000", response.DataRows()[0][0])
	require.Equal(t, "success", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "ranking", fixture.latestAudit(t).QueryMode)
}

func TestLiveQueryFlowSupportsTrend(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近7天每天的取消率趋势",
		Domain:   "ride_hailing",
		UserID:   "user-trend",
		UserRole: "analyst",
	})

	response.requireSuccess(t, http.StatusOK, "trend")
	require.Len(t, response.DataRows(), 7)
	require.Equal(t, "2026-04-21", response.DataRows()[0][0])
	require.Equal(t, "2026-04-27", response.DataRows()[6][0])
	require.Equal(t, "success", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "trend", fixture.latestAudit(t).QueryMode)
}

func TestLiveQueryFlowSupportsDetailList(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近7天上海待接驾订单明细",
		Domain:   "ride_hailing",
		UserID:   "user-detail",
		UserRole: "analyst",
	})

	response.requireSuccess(t, http.StatusOK, "detail_list")
	require.Len(t, response.DataRows(), 2)
	for _, row := range response.DataRows() {
		require.Equal(t, "310000", row[1])
		require.Equal(t, "waiting_pickup", row[3])
	}
	require.Equal(t, "success", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "detail_list", fixture.latestAudit(t).QueryMode)
}

func TestLiveQueryFlowRejectsDetailWithoutNarrowingFilter(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近7天待接驾订单明细",
		Domain:   "ride_hailing",
		UserID:   "user-broad-detail",
		UserRole: "analyst",
	})

	response.requireError(t, http.StatusBadRequest, "INVALID_QUERY")
	require.Equal(t, "failed", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "resolution", fixture.latestAudit(t).RejectionStage)
}

func TestLiveQueryFlowRejectsUnknownDomain(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近30天取消率是多少",
		Domain:   "unknown_domain",
		UserID:   "user-unknown-domain",
		UserRole: "analyst",
	})

	response.requireError(t, http.StatusBadRequest, "UNSUPPORTED_DOMAIN")
	require.Equal(t, "failed", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "request_validation", fixture.latestAudit(t).RejectionStage)
}

func TestLiveQueryFlowRejectsDetailForUnauthorizedRole(t *testing.T) {
	fixture := newLiveQueryFlowFixture(t)

	response := fixture.postQuery(t, liveQueryRequest{
		Query:    "最近7天上海待接驾订单明细",
		Domain:   "ride_hailing",
		UserID:   "user-viewer",
		UserRole: "viewer",
	})

	response.requireError(t, http.StatusForbidden, "PERMISSION_DENIED")
	require.Equal(t, "failed", fixture.latestAudit(t).ExecutionStatus)
	require.Equal(t, "resolution", fixture.latestAudit(t).RejectionStage)
}

type liveQueryFlowFixture struct {
	env     *mysqlbootstrap.Environment
	runtime internalserver.Runtime
}

type liveQueryRequest struct {
	Query    string `json:"query"`
	Domain   string `json:"domain"`
	UserID   string
	UserRole string
}

type liveQueryResponse struct {
	Status    string                 `json:"status"`
	RequestID string                 `json:"request_id"`
	Data      map[string]any         `json:"data"`
	Meta      map[string]any         `json:"meta"`
	Error     map[string]string      `json:"error"`
	HTTPCode  int                    `json:"-"`
	Body      map[string]any         `json:"-"`
	RawBody   map[string]interface{} `json:"-"`
}

type auditRecord struct {
	ExecutionStatus string
	RejectionStage  string
	QueryMode       string
	ErrorMessage    string
}

func newLiveQueryFlowFixture(t *testing.T) liveQueryFlowFixture {
	t.Helper()

	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Fatal("MINIMAX_API_KEY is required")
	}

	env := mysqlbootstrap.StartMySQLContainer(t)
	t.Cleanup(func() {
		env.Terminate(t)
	})

	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	require.NoError(t, mysqlbootstrap.Bootstrap(context.Background(), env.RootDB, now))
	truncateAuditLog(t, env)

	t.Setenv("MYSQL_RIDE_HAILING_RO_DSN", env.ReadonlyDSN)

	runtimeConfig := internalserver.RuntimeConfig{
		ConfigDir:      filepath.Join(repoRoot(), "configs"),
		MiniMaxAPIKey:  apiKey,
		MiniMaxModel:   os.Getenv("MINIMAX_MODEL"),
		MiniMaxBaseURL: os.Getenv("MINIMAX_BASE_URL"),
		AuditDSN:       swapDatabaseInDSN(env.RootDSN, "ride_hailing"),
		Clock: fixedClock{now: now},
	}

	appRuntime, err := internalserver.BuildRuntime(runtimeConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		if db, dbErr := appRuntime.Registry.ForDatasource("ride_hailing_ro"); dbErr == nil {
			_ = db.Close()
		}
		if appRuntime.AuditDB != nil {
			_ = appRuntime.AuditDB.Close()
		}
	})

	return liveQueryFlowFixture{
		env:     env,
		runtime: appRuntime,
	}
}

func (f liveQueryFlowFixture) postQuery(t *testing.T, req liveQueryRequest) liveQueryResponse {
	t.Helper()

	body, err := json.Marshal(map[string]string{
		"query":  req.Query,
		"domain": req.Domain,
	})
	require.NoError(t, err)

	httpRequest := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-User-ID", req.UserID)
	httpRequest.Header.Set("X-User-Role", req.UserRole)

	recorder := httptest.NewRecorder()
	f.runtime.Mux.ServeHTTP(recorder, httpRequest)

	var payload liveQueryResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	payload.HTTPCode = recorder.Code

	return payload
}

func (r liveQueryResponse) requireSuccess(t *testing.T, wantHTTPCode int, wantQueryMode string) {
	t.Helper()

	require.Equalf(t, wantHTTPCode, r.HTTPCode, "response=%#v", r)
	require.Equalf(t, "success", r.Status, "response=%#v", r)
	require.Equalf(t, wantQueryMode, r.Meta["query_mode"], "response=%#v", r)
	require.NotEmpty(t, r.RequestID)
}

func (r liveQueryResponse) requireError(t *testing.T, wantHTTPCode int, wantCode string) {
	t.Helper()

	require.Equalf(t, wantHTTPCode, r.HTTPCode, "response=%#v", r)
	require.Equalf(t, "error", r.Status, "response=%#v", r)
	require.Equalf(t, wantCode, r.Error["code"], "response=%#v", r)
}

func (r liveQueryResponse) DataRows() [][]any {
	rowsValue, ok := r.Data["Rows"]
	if !ok {
		return nil
	}

	rowItems, ok := rowsValue.([]any)
	if !ok {
		return nil
	}

	rows := make([][]any, 0, len(rowItems))
	for _, rowItem := range rowItems {
		columns, ok := rowItem.([]any)
		if !ok {
			continue
		}

		rows = append(rows, columns)
	}

	return rows
}

func (f liveQueryFlowFixture) latestAudit(t *testing.T) auditRecord {
	t.Helper()

	var record auditRecord
	err := f.env.RootDB.QueryRow(
		`SELECT execution_status, rejection_stage, query_mode, error_message_internal
		FROM ride_hailing.nl2sql_query_log
		ORDER BY created_at DESC, request_id DESC
		LIMIT 1`,
	).Scan(&record.ExecutionStatus, &record.RejectionStage, &record.QueryMode, &record.ErrorMessage)
	require.NoError(t, err)

	return record
}

func truncateAuditLog(t *testing.T, env *mysqlbootstrap.Environment) {
	t.Helper()

	_, err := env.RootDB.Exec("TRUNCATE TABLE ride_hailing.nl2sql_query_log")
	require.NoError(t, err)
}

func swapDatabaseInDSN(dsn string, database string) string {
	return strings.Replace(dsn, "/mysql?", "/"+database+"?", 1)
}

func repoRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve repo root: runtime.Caller failed")
	}

	root := filepath.Dir(currentFile)
	for i := 0; i < 2; i++ {
		root = filepath.Dir(root)
	}

	return root
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}
