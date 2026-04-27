# MiniMax Live E2E Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Connect the repository to the live MiniMax planner, bootstrap real MySQL demo data, wire a runnable server runtime, and make default `go test ./...` cover all supported NL2SQL scenarios end to end.

**Architecture:** Keep resolver, builder, guard, and formatter deterministic, but add a real planner adapter, real datasource opening, registry-backed executor routing, and test-side MySQL bootstrap helpers. Live tests call fixed Chinese questions against MiniMax and assert planner validity, query execution success, and audit persistence.

**Tech Stack:** Go, `net/http`, MiniMax Anthropic-compatible messages API, `database/sql`, MySQL, testcontainers-go, YAML config, existing orchestrator/api packages.

---

### Task 1: Update Repository Constraints and Record the Approved Design

**Files:**
- Modify: `docs/project-constraints.md`
- Modify: `docs/plans/2026-04-27-nl2sql-design.md`
- Create: `docs/plans/2026-04-27-minimax-live-e2e-design.md`

**Step 1: Write the failing documentation expectation test**

```go
func TestVerificationDocsMentionMiniMaxLiveDefault(t *testing.T) {
	raw, err := os.ReadFile("docs/project-constraints.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "default tests may call the live MiniMax planner") {
		t.Fatalf("expected live MiniMax default-test rule in project constraints")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/smoke -run TestVerificationDocsMentionMiniMaxLiveDefault -v`
Expected: `FAIL` because the repository docs do not yet describe the new default live rule.

**Step 3: Update the docs**

- Add the new live MiniMax default-test rule to `docs/project-constraints.md`
- Amend `docs/plans/2026-04-27-nl2sql-design.md` so it no longer says online LLM calls are excluded from the default gate
- Save the approved design text in `docs/plans/2026-04-27-minimax-live-e2e-design.md`

**Step 4: Run test to verify it passes**

Run: `go test ./tests/smoke -run TestVerificationDocsMentionMiniMaxLiveDefault -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add docs/project-constraints.md docs/plans/2026-04-27-nl2sql-design.md docs/plans/2026-04-27-minimax-live-e2e-design.md tests/smoke
git commit -m "文档: 明确 Minimax 默认联调约束"
```

### Task 2: Add a Real MiniMax Planner Client

**Files:**
- Create: `internal/planner/minimax.go`
- Create: `internal/planner/minimax_test.go`
- Modify: `internal/planner/schema.go`

**Step 1: Write the failing test**

```go
func TestMiniMaxPlannerBuildsAnthropicRequestAndParsesRawPlanJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/anthropic/v1/messages" {
			t.Fatalf("unexpected path %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": `{"query_mode":"ranking","metrics":["取消率"],"dimensions":["城市"],"limit":10}`},
			},
		})
	}))
	defer server.Close()

	client := NewMiniMaxPlanner(MiniMaxConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "MiniMax-M2.7-highspeed",
	})

	plan, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.NoError(t, err)
	require.Equal(t, "ranking", plan.QueryMode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/planner -run TestMiniMaxPlannerBuildsAnthropicRequestAndParsesRawPlanJSON -v`
Expected: `FAIL` because `MiniMaxPlanner` does not exist.

**Step 3: Write minimal implementation**

- Add config struct with `BaseURL`, `APIKey`, `Model`, and optional `HTTPClient`
- Build Anthropic-compatible request body
- Send request with `Authorization: Bearer <API key>`
- Parse only `content[].text`
- Validate the returned JSON using existing raw-plan schema validation before unmarshalling into `domain.RawPlan`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/planner -run TestMiniMaxPlannerBuildsAnthropicRequestAndParsesRawPlanJSON -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/planner/minimax.go internal/planner/minimax_test.go internal/planner/schema.go
git commit -m "功能: 接入 Minimax 规划客户端"
```

### Task 3: Open Datasources from Config and Route Execution by Datasource ID

**Files:**
- Create: `internal/datasource/open.go`
- Create: `internal/datasource/open_test.go`
- Modify: `internal/datasource/registry.go`
- Create: `internal/executor/registry_mysql.go`
- Create: `internal/executor/registry_mysql_test.go`

**Step 1: Write the failing tests**

```go
func TestOpenRegistryFromConfigRegistersConfiguredDatasource(t *testing.T) {
	cfg, _ := config.LoadFromDir("../config/testdata/basic")
	registry := NewRegistry()
	err := OpenAndRegister(context.Background(), registry, cfg.Datasources, openFuncStub)
	require.NoError(t, err)
	_, err = registry.ForDatasource("ride_hailing_ro")
	require.NoError(t, err)
}

func TestRegistryExecutorUsesDatasourceIDFromResolvedPlan(t *testing.T) {
	exec := NewRegistryExecutor(stubRegistry{db: &sql.DB{}})
	_, err := exec.Query(context.Background(), "ride_hailing_ro", "SELECT 1", nil)
	require.NoError(t, err)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/datasource -run TestOpenRegistryFromConfigRegistersConfiguredDatasource -v`
Expected: `FAIL`

Run: `go test ./internal/executor -run TestRegistryExecutorUsesDatasourceIDFromResolvedPlan -v`
Expected: `FAIL`

**Step 3: Write minimal implementation**

- Add `ForDatasource` to registry
- Add `OpenAndRegister` to open pools from datasource config
- Add a registry-backed executor that resolves `datasource_id` to a readonly DB and reuses the current scan logic

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/datasource -run TestOpenRegistryFromConfigRegistersConfiguredDatasource -v`
Expected: `PASS`

Run: `go test ./internal/executor -run TestRegistryExecutorUsesDatasourceIDFromResolvedPlan -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/datasource/open.go internal/datasource/open_test.go internal/datasource/registry.go internal/executor/registry_mysql.go internal/executor/registry_mysql_test.go
git commit -m "功能: 打通真实数据源注册与执行路由"
```

### Task 4: Add MySQL Bootstrap and Dynamic Seed Data Support

**Files:**
- Create: `db/migrations/0002_create_ride_hailing_demo_tables.sql`
- Create: `tests/testsupport/mysqlbootstrap/bootstrap.go`
- Create: `tests/testsupport/mysqlbootstrap/bootstrap_test.go`
- Modify: `tests/integration/mysql/executor_test.go`

**Step 1: Write the failing test**

```go
func TestBootstrapCreatesDemoTablesAndSeedsQueryableData(t *testing.T) {
	env := startMySQLContainer(t)
	defer env.Terminate(t)

	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	err := Bootstrap(context.Background(), env.RootDB, now)
	require.NoError(t, err)

	var count int
	err = env.RootDB.QueryRow("SELECT COUNT(*) FROM ride_hailing.trip_orders").Scan(&count)
	require.NoError(t, err)
	require.Greater(t, count, 0)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/testsupport/mysqlbootstrap -run TestBootstrapCreatesDemoTablesAndSeedsQueryableData -v`
Expected: `FAIL` because bootstrap helpers and business tables do not exist.

**Step 3: Write minimal implementation**

- Add migration SQL for `trip_orders` and `drivers`
- Add bootstrap code to:
  - create database
  - apply audit migration
  - apply business-table migration
  - create readonly user and grants
  - insert dynamic seed data relative to `now`
- Expose helper output needed by live tests

**Step 4: Run test to verify it passes**

Run: `go test ./tests/testsupport/mysqlbootstrap -run TestBootstrapCreatesDemoTablesAndSeedsQueryableData -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add db/migrations/0002_create_ride_hailing_demo_tables.sql tests/testsupport/mysqlbootstrap/bootstrap.go tests/testsupport/mysqlbootstrap/bootstrap_test.go tests/integration/mysql/executor_test.go
git commit -m "功能: 增加测试库初始化与演示数据"
```

### Task 5: Build a Real Server Runtime

**Files:**
- Create: `internal/server/runtime.go`
- Create: `internal/server/runtime_test.go`
- Modify: `internal/server/server.go`
- Modify: `cmd/server/main.go`

**Step 1: Write the failing test**

```go
func TestBuildRuntimeWiresPlannerExecutorAndAudit(t *testing.T) {
	runtime, err := BuildRuntime(RuntimeConfig{
		ConfigDir: "testdata/configs",
		MiniMaxAPIKey: "test-key",
	})
	require.NoError(t, err)
	require.NotNil(t, runtime.Service)
	require.NotNil(t, runtime.Mux)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestBuildRuntimeWiresPlannerExecutorAndAudit -v`
Expected: `FAIL` because no runtime builder exists.

**Step 3: Write minimal implementation**

- Add runtime builder that:
  - loads config and catalog
  - opens datasource registry
  - creates audit repository
  - creates MiniMax planner
  - creates registry-backed executor
  - creates orchestrator service
  - returns mux from the real service
- Make `cmd/server/main.go` use runtime builder instead of exiting intentionally

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestBuildRuntimeWiresPlannerExecutorAndAudit -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/server/runtime.go internal/server/runtime_test.go internal/server/server.go cmd/server/main.go
git commit -m "功能: 装配真实服务运行时"
```

### Task 6: Add Live Planner Contract Tests

**Files:**
- Create: `tests/live/minimax_planner_e2e_test.go`

**Step 1: Write the failing test**

```go
func TestMiniMaxPlannerReturnsValidRankingRawPlan(t *testing.T) {
	client := newLivePlannerFromEnv(t)
	plan, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.NoError(t, err)
	require.Equal(t, "ranking", plan.QueryMode)
	require.Contains(t, plan.Metrics, "取消率")
	require.Contains(t, plan.Dimensions, "城市")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/live -run TestMiniMaxPlannerReturnsValidRankingRawPlan -v`
Expected: `FAIL` because live planner env wiring and live test helpers do not yet exist.

**Step 3: Write minimal implementation**

- Add live test helper to construct a MiniMax planner from env vars
- Fail when `MINIMAX_API_KEY` is missing
- Use the real planner client and assert raw-plan contract only

**Step 4: Run test to verify it passes**

Run: `go test ./tests/live -run TestMiniMaxPlannerReturnsValidRankingRawPlan -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add tests/live/minimax_planner_e2e_test.go
git commit -m "测试: 增加 Minimax 在线规划契约验证"
```

### Task 7: Add Full Query-Flow Live Tests for All Supported Scenarios

**Files:**
- Create: `tests/live/query_flow_e2e_test.go`
- Modify: `internal/server/runtime.go` as needed
- Modify: `internal/orchestrator/service.go` as needed

**Step 1: Write the failing tests**

```go
func TestLiveQueryFlowSupportsAggregateOverview(t *testing.T) {}
func TestLiveQueryFlowSupportsRanking(t *testing.T) {}
func TestLiveQueryFlowSupportsTrend(t *testing.T) {}
func TestLiveQueryFlowSupportsDetailList(t *testing.T) {}
func TestLiveQueryFlowRejectsDetailWithoutNarrowingFilter(t *testing.T) {}
func TestLiveQueryFlowRejectsUnknownDomain(t *testing.T) {}
func TestLiveQueryFlowRejectsDetailForUnauthorizedRole(t *testing.T) {}
```

Each success test should:

- bootstrap a real MySQL dataset
- build the real runtime
- send a fixed Chinese request through the HTTP handler
- assert `status=success`
- assert expected `query_mode`
- assert returned rows or summaries match contract
- assert audit rows are persisted

Each rejection test should:

- send the request through the real HTTP handler
- assert stable error code and HTTP status
- assert audit rows are still persisted

**Step 2: Run tests to verify they fail**

Run: `go test ./tests/live -run TestLiveQueryFlow -v`
Expected: `FAIL` because the full live path and bootstrap support are not yet complete.

**Step 3: Write minimal implementation**

- Add live test helpers for:
  - bootstrap
  - runtime build
  - HTTP request execution
  - audit inspection
- Use the fixed Chinese question set from the design
- Keep tests serial and low-volume

**Step 4: Run tests to verify they pass**

Run: `go test ./tests/live -run TestLiveQueryFlow -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add tests/live/query_flow_e2e_test.go internal/server/runtime.go internal/orchestrator/service.go
git commit -m "测试: 覆盖在线全链路查询场景"
```

### Task 8: Make Default Verification Explicit

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/verification-checklist.md`
- Modify: `.githooks/pre-commit` if needed

**Step 1: Write the failing test**

```go
func TestVerificationDocsDescribeLiveMiniMaxRequirements(t *testing.T) {
	raw, err := os.ReadFile("docs/plans/verification-checklist.md")
	require.NoError(t, err)
	require.Contains(t, string(raw), "MINIMAX_API_KEY")
	require.Contains(t, string(raw), "go test ./...")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/smoke -run TestVerificationDocsDescribeLiveMiniMaxRequirements -v`
Expected: `FAIL` because the verification docs do not yet describe the live-default setup.

**Step 3: Write minimal implementation**

- Update README with required env vars and live default test behavior
- Update verification checklist with:
  - MiniMax credentials
  - root and readonly MySQL DSNs
  - default `go test ./...` expectations
- Only change the hook if the current hook needs extra live setup messaging

**Step 4: Run test to verify it passes**

Run: `go test ./tests/smoke -run TestVerificationDocsDescribeLiveMiniMaxRequirements -v`
Expected: `PASS`

**Step 5: Commit**

```bash
git add README.md docs/plans/verification-checklist.md tests/smoke
git commit -m "文档: 补充在线联调验证说明"
```

### Task 9: Run Final Verification

**Files:**
- Verify only

**Step 1: Run the full package test suite**

Run: `go test ./...`
Expected: all packages pass, including live MiniMax and live MySQL tests.

**Step 2: Run config validation**

Run: `go run ./cmd/nl2sqlctl config validate`
Expected: `config validation passed`

**Step 3: Run encoding checks**

Run: `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-encoding.ps1`
Expected: `encoding check passed`

**Step 4: Commit final cleanup if needed**

```bash
git add -A
git commit -m "测试: 完成 Minimax 在线全链路验证"
```

## Execution Notes

- Implement each task with @superpowers:test-driven-development
- Keep `resolver`, `builder`, `guard`, and `formatter` pure
- Do not let live-test helpers own business rules
- Use fixed Chinese question strings in live tests
- Keep MiniMax live tests serial to control quota usage
- Prefer contract assertions over exact free-form explanation text
- Fail when live credentials are missing, because default tests are intentionally live

## Expected Question Set

- `最近30天取消率是多少`
- `最近30天取消率最高的城市`
- `最近7天每天的取消率趋势`
- `最近7天上海待接驾订单明细`
- `最近7天司机张三的待接驾订单明细`
- `最近7天待接驾订单明细`

## Expected Environment Variables

- `MINIMAX_API_KEY`
- `MINIMAX_MODEL`
- `MYSQL_RIDE_HAILING_ROOT_DSN`
- `MYSQL_RIDE_HAILING_RO_DSN`
- `MYSQL_NL2SQL_AUDIT_DSN` when audit storage is split
