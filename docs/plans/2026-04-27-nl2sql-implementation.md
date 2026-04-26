# NL2SQL Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go-based NL2SQL backend that supports aggregate queries and controlled detail queries over multiple readonly MySQL datasources, with single-datasource execution per request.

**Architecture:** The service loads generated schema plus curated semantic YAML into a runtime catalog, uses an LLM only for `RawPlan` generation, resolves that plan into canonical objects, builds controlled MySQL SQL, validates it with a guard layer, executes through readonly datasource pools, and audits every request. Core modules stay deterministic and are implemented with @superpowers:test-driven-development.

**Tech Stack:** Go, `database/sql`, MySQL readonly connections, YAML config, OpenTelemetry, testcontainers-go for MySQL integration tests, a MySQL-compatible AST parser, standard `net/http`.

---

### Task 1: Bootstrap Module and Runtime Config Loader

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `configs/datasources.yaml`
- Create: `configs/domains/ride_hailing/domain.yaml`
- Create: `internal/config/types.go`
- Create: `internal/config/loader.go`
- Test: `internal/config/loader_test.go`

**Step 1: Write the failing test**

```go
func TestLoadConfigReadsDatasourcesAndDomainRouting(t *testing.T) {
	cfg, err := LoadFromDir("testdata/basic")
	require.NoError(t, err)
	require.Equal(t, "ride_hailing_ro", cfg.Domains["ride_hailing"].DatasourceID)
	require.Equal(t, "mysql", cfg.Datasources["ride_hailing_ro"].Driver)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestLoadConfigReadsDatasourcesAndDomainRouting -v`  
Expected: `FAIL` because `LoadFromDir` and config structs do not exist yet.

**Step 3: Write minimal implementation**

```go
type DatasourceConfig struct {
	ID       string `yaml:"id"`
	Driver   string `yaml:"driver"`
	DsnEnv   string `yaml:"dsn_env"`
	Database string `yaml:"database"`
}

type DomainConfig struct {
	ID           string `yaml:"id"`
	DatasourceID string `yaml:"datasource_id"`
}
```

Implement `LoadFromDir` to read `datasources.yaml` and `domains/*/domain.yaml`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestLoadConfigReadsDatasourcesAndDomainRouting -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add go.mod .gitignore configs/datasources.yaml configs/domains/ride_hailing/domain.yaml internal/config/types.go internal/config/loader.go internal/config/loader_test.go
git commit -m "feat: add runtime config loader"
```

### Task 2: Add Generated Schema Catalog Loader

**Files:**
- Create: `configs/schemas/ride_hailing_ro.generated.yaml`
- Create: `internal/catalog/types.go`
- Create: `internal/catalog/loader.go`
- Test: `internal/catalog/loader_test.go`

**Step 1: Write the failing test**

```go
func TestCatalogLoaderBuildsTableAndColumnIndex(t *testing.T) {
	catalog, err := Load("testdata/catalog")
	require.NoError(t, err)
	require.Contains(t, catalog.TablesByName, "trip_orders")
	require.Contains(t, catalog.ColumnsByTable["trip_orders"], "called_at")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog -run TestCatalogLoaderBuildsTableAndColumnIndex -v`  
Expected: `FAIL` because the catalog loader does not exist.

**Step 3: Write minimal implementation**

```go
type TableSpec struct {
	Name    string       `yaml:"name"`
	Columns []ColumnSpec `yaml:"columns"`
}

type ColumnSpec struct {
	Name string `yaml:"name"`
	Type string `yaml:"data_type"`
}
```

Load the generated schema YAML and index tables and columns by name.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog -run TestCatalogLoaderBuildsTableAndColumnIndex -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add configs/schemas/ride_hailing_ro.generated.yaml internal/catalog/types.go internal/catalog/loader.go internal/catalog/loader_test.go
git commit -m "feat: load generated schema catalog"
```

### Task 3: Validate Semantic Domain Config Against Schema

**Files:**
- Create: `configs/domains/ride_hailing/dimensions.yaml`
- Create: `configs/domains/ride_hailing/metrics.yaml`
- Create: `configs/domains/ride_hailing/detail_views.yaml`
- Create: `configs/domains/ride_hailing/roles.yaml`
- Create: `configs/domains/ride_hailing/aliases.yaml`
- Create: `internal/catalog/validate.go`
- Test: `internal/catalog/validate_test.go`

**Step 1: Write the failing test**

```go
func TestCatalogValidationRejectsUnknownMetricColumn(t *testing.T) {
	err := Validate(loadBrokenCatalogFixture(t))
	require.ErrorContains(t, err, "unknown column")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog -run TestCatalogValidationRejectsUnknownMetricColumn -v`  
Expected: `FAIL` because no validation exists yet.

**Step 3: Write minimal implementation**

```go
func Validate(c Catalog) error {
	for _, metric := range c.Metrics {
		if !c.HasColumn(metric.BaseTable, metric.TimeField) {
			return fmt.Errorf("unknown column %s.%s", metric.BaseTable, metric.TimeField)
		}
	}
	return nil
}
```

Add the first-pass validation for metric columns, dimension columns, detail view columns, and role references.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog -run TestCatalogValidationRejectsUnknownMetricColumn -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add configs/domains/ride_hailing/dimensions.yaml configs/domains/ride_hailing/metrics.yaml configs/domains/ride_hailing/detail_views.yaml configs/domains/ride_hailing/roles.yaml configs/domains/ride_hailing/aliases.yaml internal/catalog/validate.go internal/catalog/validate_test.go
git commit -m "feat: validate semantic config against schema"
```

### Task 4: Add Datasource Registry and Domain Routing

**Files:**
- Create: `internal/datasource/registry.go`
- Test: `internal/datasource/registry_test.go`

**Step 1: Write the failing test**

```go
func TestDatasourceRegistryReturnsPoolByDomain(t *testing.T) {
	registry := NewRegistry()
	registry.Register("ride_hailing_ro", &sql.DB{})
	db, err := registry.ForDomain("ride_hailing", loadDomainMapFixture())
	require.NoError(t, err)
	require.NotNil(t, db)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/datasource -run TestDatasourceRegistryReturnsPoolByDomain -v`  
Expected: `FAIL` because the registry does not exist.

**Step 3: Write minimal implementation**

```go
type Registry struct {
	pools map[string]*sql.DB
}
```

Implement `Register` and `ForDomain`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/datasource -run TestDatasourceRegistryReturnsPoolByDomain -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/datasource/registry.go internal/datasource/registry_test.go
git commit -m "feat: add datasource registry"
```

### Task 5: Add Core Domain Types for RawPlan and ResolvedPlan

**Files:**
- Create: `internal/domain/plan.go`
- Test: `internal/domain/plan_test.go`

**Step 1: Write the failing test**

```go
func TestResolvedPlanRejectsMissingDatasourceID(t *testing.T) {
	plan := ResolvedPlan{QueryMode: QueryModeRanking}
	err := plan.Validate()
	require.ErrorContains(t, err, "datasource_id")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain -run TestResolvedPlanRejectsMissingDatasourceID -v`  
Expected: `FAIL` because plan validation does not exist.

**Step 3: Write minimal implementation**

```go
func (p ResolvedPlan) Validate() error {
	if p.DatasourceID == "" {
		return errors.New("datasource_id is required")
	}
	return nil
}
```

Add query mode constants and the shared plan structs.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain -run TestResolvedPlanRejectsMissingDatasourceID -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/domain/plan.go internal/domain/plan_test.go
git commit -m "feat: add query plan domain types"
```

### Task 6: Implement Aggregate Resolver

**Files:**
- Create: `internal/resolver/aggregate.go`
- Test: `internal/resolver/aggregate_test.go`

**Step 1: Write the failing test**

```go
func TestResolveAggregateRankingMapsAliasesAndClampsLimit(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode: "ranking",
		Metrics:   []string{"取消率"},
		Dimensions: []string{"城市"},
		Limit:     500,
	}
	plan, err := ResolveAggregate(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	require.NoError(t, err)
	require.Equal(t, []string{"metric.cancel_rate"}, plan.MetricIDs)
	require.Equal(t, 100, plan.Limit)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resolver -run TestResolveAggregateRankingMapsAliasesAndClampsLimit -v`  
Expected: `FAIL` because aggregate resolution does not exist.

**Step 3: Write minimal implementation**

```go
func ResolveAggregate(raw domain.RawPlan, catalog Catalog, role RolePolicy, clk clock.Clock) (domain.ResolvedPlan, error) {
	return domain.ResolvedPlan{
		QueryMode:    domain.QueryModeRanking,
		MetricIDs:    []string{"metric.cancel_rate"},
		DimensionIDs: []string{"dimension.city_name"},
		Limit:        min(raw.Limit, role.MaxLimit),
		DatasourceID: catalog.Domain.DatasourceID,
	}, nil
}
```

Expand incrementally for aliases, time range expansion, and unsupported mode rejection.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resolver -run TestResolveAggregateRankingMapsAliasesAndClampsLimit -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/resolver/aggregate.go internal/resolver/aggregate_test.go
git commit -m "feat: resolve aggregate query plans"
```

### Task 7: Implement Detail Resolver

**Files:**
- Create: `internal/resolver/detail.go`
- Test: `internal/resolver/detail_test.go`

**Step 1: Write the failing test**

```go
func TestResolveDetailMapsSubjectToDetailViewAndRequiresNarrowingFilter(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		TimeRange:     domain.RawTimeRange{Type: "relative", Value: "last_7_days"},
	}
	_, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	require.ErrorContains(t, err, "narrowing filter")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resolver -run TestResolveDetailMapsSubjectToDetailViewAndRequiresNarrowingFilter -v`  
Expected: `FAIL` because detail resolution does not exist.

**Step 3: Write minimal implementation**

```go
func ResolveDetail(raw domain.RawPlan, catalog Catalog, role RolePolicy, clk clock.Clock) (domain.ResolvedPlan, error) {
	return domain.ResolvedPlan{}, errors.New("detail query requires narrowing filter")
}
```

Then make the test pass by resolving the subject to `detail.waiting_pickup_orders` and enforcing required detail rules.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resolver -run TestResolveDetailMapsSubjectToDetailViewAndRequiresNarrowingFilter -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/resolver/detail.go internal/resolver/detail_test.go
git commit -m "feat: resolve detail query plans"
```

### Task 8: Implement Aggregate SQL Builder

**Files:**
- Create: `internal/builder/aggregate.go`
- Create: `internal/builder/result.go`
- Test: `internal/builder/aggregate_test.go`

**Step 1: Write the failing test**

```go
func TestBuildAggregateRankingCancelRateByCity(t *testing.T) {
	plan := loadResolvedRankingPlanFixture(t)
	result, err := BuildAggregate(plan, loadCatalogFixture(t))
	require.NoError(t, err)
	require.Contains(t, result.SQL, "GROUP BY")
	require.Contains(t, result.SQL, "LIMIT ?")
	require.Equal(t, 5, result.Args[len(result.Args)-1])
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/builder -run TestBuildAggregateRankingCancelRateByCity -v`  
Expected: `FAIL` because aggregate SQL builder does not exist.

**Step 3: Write minimal implementation**

```go
func BuildAggregate(plan domain.ResolvedPlan, catalog Catalog) (BuildResult, error) {
	return BuildResult{
		SQL:  "SELECT trip_orders.city_code, SUM(...) AS cancel_rate FROM ... GROUP BY trip_orders.city_code ORDER BY cancel_rate DESC LIMIT ?",
		Args: []any{plan.Limit},
	}, nil
}
```

Then replace placeholders with catalog-driven rendering.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/builder -run TestBuildAggregateRankingCancelRateByCity -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/builder/aggregate.go internal/builder/result.go internal/builder/aggregate_test.go
git commit -m "feat: build aggregate SQL"
```

### Task 9: Implement Detail SQL Builder

**Files:**
- Create: `internal/builder/detail.go`
- Test: `internal/builder/detail_test.go`

**Step 1: Write the failing test**

```go
func TestBuildDetailListUsesAllowedColumnsAndDefaultSort(t *testing.T) {
	plan := loadResolvedDetailPlanFixture(t)
	result, err := BuildDetail(plan, loadCatalogFixture(t))
	require.NoError(t, err)
	require.NotContains(t, result.SQL, "*")
	require.Contains(t, result.SQL, "ORDER BY trip_orders.called_at DESC")
	require.Contains(t, result.SQL, "LIMIT ?")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/builder -run TestBuildDetailListUsesAllowedColumnsAndDefaultSort -v`  
Expected: `FAIL` because detail SQL builder does not exist.

**Step 3: Write minimal implementation**

```go
func BuildDetail(plan domain.ResolvedPlan, catalog Catalog) (BuildResult, error) {
	return BuildResult{
		SQL:  "SELECT trip_orders.order_id, trip_orders.called_at FROM trip_orders WHERE trip_orders.called_at BETWEEN ? AND ? ORDER BY trip_orders.called_at DESC LIMIT ?",
		Args: []any{plan.TimeRange.Start, plan.TimeRange.End, plan.Limit},
	}, nil
}
```

Then extend to catalog-driven column selection and joins.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/builder -run TestBuildDetailListUsesAllowedColumnsAndDefaultSort -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/builder/detail.go internal/builder/detail_test.go
git commit -m "feat: build detail SQL"
```

### Task 10: Implement SQL Guard

**Files:**
- Create: `internal/guard/guard.go`
- Test: `internal/guard/guard_test.go`

**Step 1: Write the failing test**

```go
func TestGuardRejectsDetailQueryWithoutAllowedColumns(t *testing.T) {
	input := loadGuardInputWithSensitiveColumnFixture(t)
	_, err := Validate(input)
	require.ErrorContains(t, err, "column not allowed")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/guard -run TestGuardRejectsDetailQueryWithoutAllowedColumns -v`  
Expected: `FAIL` because the guard does not exist.

**Step 3: Write minimal implementation**

```go
func Validate(input GuardInput) (BuildResult, error) {
	for _, col := range input.BuildResult.ReferencedCols {
		if !input.Domain.AllowsColumn(col, input.RolePolicy) {
			return BuildResult{}, fmt.Errorf("column not allowed: %s", col)
		}
	}
	return input.BuildResult, nil
}
```

Add statement-type, limit, join-count, and time-range checks next.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/guard -run TestGuardRejectsDetailQueryWithoutAllowedColumns -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/guard/guard.go internal/guard/guard_test.go
git commit -m "feat: validate generated SQL with guard rules"
```

### Task 11: Implement Result Formatter

**Files:**
- Create: `internal/formatter/formatter.go`
- Test: `internal/formatter/formatter_test.go`

**Step 1: Write the failing test**

```go
func TestFormatterBuildsFactSummaryForDetailResult(t *testing.T) {
	resp := Format(loadDetailResultFixture(t))
	require.Contains(t, resp.Summary, "前50条")
	require.True(t, resp.Truncated)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/formatter -run TestFormatterBuildsFactSummaryForDetailResult -v`  
Expected: `FAIL` because the formatter does not exist.

**Step 3: Write minimal implementation**

```go
func Format(result QueryResult) ResponseData {
	return ResponseData{
		Summary:   "返回前50条结果。",
		Truncated: true,
	}
}
```

Then expand to columns, rows, and aggregate/detail summary templates.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/formatter -run TestFormatterBuildsFactSummaryForDetailResult -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/formatter/formatter.go internal/formatter/formatter_test.go
git commit -m "feat: format query results"
```

### Task 12: Implement Audit Persistence and Migration

**Files:**
- Create: `db/migrations/0001_create_nl2sql_query_log.sql`
- Create: `internal/audit/repository.go`
- Test: `internal/audit/repository_test.go`

**Step 1: Write the failing test**

```go
func TestAuditRepositoryBuildsInsertForResolvedPlanAndSQL(t *testing.T) {
	entry := loadAuditEntryFixture(t)
	query, args := BuildInsert(entry)
	require.Contains(t, query, "INSERT INTO nl2sql_query_log")
	require.NotEmpty(t, args)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/audit -run TestAuditRepositoryBuildsInsertForResolvedPlanAndSQL -v`  
Expected: `FAIL` because the repository does not exist.

**Step 3: Write minimal implementation**

```go
func BuildInsert(entry Entry) (string, []any) {
	return "INSERT INTO nl2sql_query_log (request_id, domain) VALUES (?, ?)", []any{entry.RequestID, entry.Domain}
}
```

Add the full insert shape after the first test goes green.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/audit -run TestAuditRepositoryBuildsInsertForResolvedPlanAndSQL -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add db/migrations/0001_create_nl2sql_query_log.sql internal/audit/repository.go internal/audit/repository_test.go
git commit -m "feat: add audit persistence"
```

### Task 13: Implement Orchestrator with Fake Planner and Fake Executor

**Files:**
- Create: `internal/orchestrator/service.go`
- Test: `internal/orchestrator/service_test.go`

**Step 1: Write the failing test**

```go
func TestOrchestratorRunsAggregateQueryEndToEnd(t *testing.T) {
	svc := newServiceWithFakes(t)
	resp, err := svc.Run(context.Background(), QueryRequest{
		Query:  "最近30天取消率最高的前5个城市",
		Domain: "ride_hailing",
	})
	require.NoError(t, err)
	require.Equal(t, "ranking", resp.Meta.QueryMode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestOrchestratorRunsAggregateQueryEndToEnd -v`  
Expected: `FAIL` because the orchestrator does not exist.

**Step 3: Write minimal implementation**

```go
func (s *Service) Run(ctx context.Context, req QueryRequest) (Response, error) {
	raw, err := s.planner.Plan(ctx, req.Query, req.Domain)
	if err != nil {
		return Response{}, err
	}
	_ = raw
	return Response{Meta: Meta{QueryMode: "ranking"}}, nil
}
```

Then add resolver, builder, guard, executor, formatter, and audit wiring incrementally.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestOrchestratorRunsAggregateQueryEndToEnd -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/orchestrator/service.go internal/orchestrator/service_test.go
git commit -m "feat: add orchestrator service"
```

### Task 14: Implement Readonly MySQL Executor Integration

**Files:**
- Create: `internal/executor/mysql.go`
- Test: `tests/integration/mysql/executor_test.go`

**Step 1: Write the failing test**

```go
func TestExecutorRunsReadonlyQueryAgainstMySQL(t *testing.T) {
	env := startMySQLContainer(t)
	exec := NewMySQLExecutor(env.DB)
	result, err := exec.Query(context.Background(), "SELECT 1 AS value", nil)
	require.NoError(t, err)
	require.Equal(t, [][]any{{int64(1)}}, result.Rows)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/integration/mysql -run TestExecutorRunsReadonlyQueryAgainstMySQL -v`  
Expected: `FAIL` because the executor does not exist.

**Step 3: Write minimal implementation**

```go
func (e *MySQLExecutor) Query(ctx context.Context, sql string, args []any) (Result, error) {
	rows, err := e.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	return scanRows(rows)
}
```

Add timeout handling and row-limit enforcement after the first green test.

**Step 4: Run test to verify it passes**

Run: `go test ./tests/integration/mysql -run TestExecutorRunsReadonlyQueryAgainstMySQL -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/executor/mysql.go tests/integration/mysql/executor_test.go
git commit -m "feat: add mysql executor"
```

### Task 15: Implement HTTP API

**Files:**
- Create: `internal/api/http.go`
- Create: `internal/api/query_handler.go`
- Test: `internal/api/query_handler_test.go`

**Step 1: Write the failing test**

```go
func TestPostQueriesReturnsSuccessPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl2sql/queries", strings.NewReader(`{"query":"最近7天完单数","domain":"ride_hailing"}`))
	res := httptest.NewRecorder()
	handler := newHandlerWithFakeService(t)
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), `"status":"success"`)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run TestPostQueriesReturnsSuccessPayload -v`  
Expected: `FAIL` because the HTTP handler does not exist.

**Step 3: Write minimal implementation**

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}
```

Then replace the placeholder with real request decoding, service invocation, and response encoding.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api -run TestPostQueriesReturnsSuccessPayload -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/api/http.go internal/api/query_handler.go internal/api/query_handler_test.go
git commit -m "feat: add query API"
```

### Task 16: Implement Planner Contract and CLI Tooling

**Files:**
- Create: `internal/planner/client.go`
- Create: `internal/planner/schema.go`
- Create: `internal/schema/pull.go`
- Create: `internal/scaffold/domain.go`
- Create: `cmd/nl2sqlctl/main.go`
- Test: `internal/planner/schema_test.go`
- Test: `internal/scaffold/domain_test.go`

**Step 1: Write the failing tests**

```go
func TestPlannerSchemaRejectsMissingQueryMode(t *testing.T) {
	err := ValidateRawPlanJSON([]byte(`{"metrics":["取消率"]}`))
	require.ErrorContains(t, err, "query_mode")
}

func TestScaffoldDomainCreatesDisabledDimensionSpecsByDefault(t *testing.T) {
	files := ScaffoldDomain(loadSchemaFixture(t), "ride_hailing", []string{"trip_orders"})
	require.Contains(t, files["dimensions.yaml"], "enabled: false")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/planner -run TestPlannerSchemaRejectsMissingQueryMode -v`  
Expected: `FAIL`

Run: `go test ./internal/scaffold -run TestScaffoldDomainCreatesDisabledDimensionSpecsByDefault -v`  
Expected: `FAIL`

**Step 3: Write minimal implementation**

```go
func ValidateRawPlanJSON(raw []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if _, ok := payload["query_mode"]; !ok {
		return errors.New("query_mode is required")
	}
	return nil
}
```

Implement the first `scaffold` function to output conservative YAML with `enabled: false`.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/planner -run TestPlannerSchemaRejectsMissingQueryMode -v`  
Expected: `PASS`

Run: `go test ./internal/scaffold -run TestScaffoldDomainCreatesDisabledDimensionSpecsByDefault -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/planner/client.go internal/planner/schema.go internal/planner/schema_test.go internal/schema/pull.go internal/scaffold/domain.go internal/scaffold/domain_test.go cmd/nl2sqlctl/main.go
git commit -m "feat: add planner contract and config tooling"
```

### Task 17: Wire the Server Entry Point and Smoke Test

**Files:**
- Create: `cmd/server/main.go`
- Test: `tests/smoke/server_smoke_test.go`

**Step 1: Write the failing test**

```go
func TestServerBootsWithConfigAndRegistersRoutes(t *testing.T) {
	addr, shutdown := startServer(t)
	defer shutdown()
	resp, err := http.Get(addr + "/healthz")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/smoke -run TestServerBootsWithConfigAndRegistersRoutes -v`  
Expected: `FAIL` because the server entry point does not exist.

**Step 3: Write minimal implementation**

```go
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

Then wire real config loading and handler registration.

**Step 4: Run test to verify it passes**

Run: `go test ./tests/smoke -run TestServerBootsWithConfigAndRegistersRoutes -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add cmd/server/main.go tests/smoke/server_smoke_test.go
git commit -m "feat: wire server entry point"
```

### Task 18: Run Full Verification Before Any Release Candidate

**Files:**
- Modify: `README.md`
- Create: `docs/plans/verification-checklist.md`

**Step 1: Write the failing verification checklist test**

```go
func TestVerificationChecklistExists(t *testing.T) {
	_, err := os.Stat("docs/plans/verification-checklist.md")
	require.NoError(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/smoke -run TestVerificationChecklistExists -v`  
Expected: `FAIL` because the checklist file does not exist.

**Step 3: Write minimal implementation**

Document:
- unit test commands
- integration test commands
- required env vars
- schema pull command
- config validate command
- smoke test command

**Step 4: Run test to verify it passes**

Run: `go test ./tests/smoke -run TestVerificationChecklistExists -v`  
Expected: `PASS`

**Step 5: Commit**

```bash
git add README.md docs/plans/verification-checklist.md
git commit -m "docs: add verification checklist"
```

## Execution Notes

- Implement every task with @superpowers:test-driven-development.
- Before marking any task done, run the exact verification command for that task.
- Keep `resolver`, `builder`, and `guard` free from database and HTTP dependencies.
- Do not wire the real LLM client into the main CI gate.
- Use fake planner and fake executor for orchestrator tests.
- Use testcontainers-go only for MySQL integration tests.
- If a test passes on the first run, rewrite the test because it did not prove the behavior.

