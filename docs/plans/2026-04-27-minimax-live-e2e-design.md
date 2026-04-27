# MiniMax Live E2E Design

Date: 2026-04-27
Status: Proposed and approved in-thread

## 1. Background

The repository currently validates most NL2SQL behavior with deterministic unit tests and fake planners, but it does not yet prove the complete production-like path:

- real Chinese natural language input
- real online planner call
- real MySQL schema and seed data
- real API request/response path
- real audit persistence

The next stage is to connect the planner to MiniMax Token Plan, initialize a repeatable MySQL dataset, and run default test suites against the live MiniMax planner plus a real MySQL database.

## 2. Goal

Build a production-like default verification path that:

- sends fixed Chinese questions to MiniMax
- receives validated `RawPlan` JSON from the model
- resolves, builds, guards, executes, formats, and audits queries end to end
- verifies that supported NL2SQL scenarios can successfully query MySQL data
- runs as part of default `go test ./...`

## 3. Intentional Constraint Change

This design intentionally changes the current repository constraint that real online LLM calls must not be part of the default test gate.

New rule for this feature slice:

- default test execution is allowed to call the live MiniMax planner
- missing live credentials are treated as failures, not skips
- online planner instability is accepted as part of the repository's default verification cost

This change is intentionally narrow to the MiniMax live planner path and must be documented in:

- `docs/project-constraints.md`
- `docs/plans/2026-04-27-nl2sql-design.md`

## 4. Chosen Approach

Chosen approach: dual-layer verification with live planner by default.

The repository keeps pure unit tests with fake planners for deterministic module coverage, but the default package test run also includes live end-to-end tests that call MiniMax and real MySQL.

Why this approach:

- it satisfies the user's requirement that default tests hit the online planner
- it preserves fast local fault isolation in resolver, builder, guard, and formatter
- it avoids replacing all deterministic tests with fragile network-bound tests

Rejected alternatives:

- live planner only, with no deterministic fallback: too fragile and poor at local diagnosis
- optional live planner behind env gating or separate command: conflicts with the requested default behavior

## 5. MiniMax Planner Design

The live planner will be implemented in `internal/planner` as a real client for the MiniMax Token Plan Anthropic-compatible messages API.

Key design points:

- endpoint: `https://api.minimaxi.com/anthropic/v1/messages`
- default model: `MiniMax-M2.7-highspeed`
- auth: Token Plan API key from environment variable
- prompt contract: the model must return `RawPlan` JSON only
- request settings: low-variance configuration with fixed prompt, deterministic output bias, and serial execution

The planner adapter is responsible for:

- building the system and user messages
- constraining output to the repository `RawPlan` schema
- extracting only text content blocks from the MiniMax response
- rejecting any non-JSON or schema-invalid output
- surfacing stable planner errors instead of leaking raw upstream payloads

The repository will keep `StaticClient` for deterministic tests, while adding a real `MiniMaxPlanner` for live execution.

## 6. MySQL Runtime and Bootstrap Design

The repository needs a repeatable MySQL bootstrap path for both integration tests and live end-to-end tests.

Bootstrap responsibilities:

- create the `ride_hailing` database if missing
- apply the audit table migration
- create business demo tables:
  - `trip_orders`
  - `drivers`
- insert deterministic demo drivers
- insert dynamic, time-relative order facts based on the current test execution time
- create and grant a readonly MySQL account for query execution

The seed dataset will be generated relative to runtime `now`, not frozen to a historical date. This is required because the default live planner tests use questions such as:

- 最近7天
- 最近30天
- 每天趋势

Without relative timestamps, the live planner tests would decay over time even if the code remained correct.

## 7. Seed Data Model

The dataset must be sufficient for all supported query modes with one coherent seed fixture.

Data requirements:

- at least 3 cities with stable code distribution
- orders across the last 7 days and last 30 days
- intentionally different cancellation rates by city for ranking assertions
- daily data points for trend assertions
- waiting-pickup rows for detail queries
- rows linked to drivers for detail join coverage
- some older-than-30-day rows to verify default time window behavior

Suggested semantic shape:

- city `310000`: highest recent cancellation rate
- city `110000`: mid cancellation rate
- city `440100`: lowest cancellation rate
- driver `张三`: appears in waiting-pickup detail rows
- driver `李四`: appears in non-detail rows to avoid accidental overfitting

## 8. Supported Scenario Coverage

The live test suite must cover every currently supported design-time query scenario.

Successful scenarios:

- `aggregate_overview`
  - example: `最近30天取消率是多少`
- `ranking`
  - example: `最近30天取消率最高的城市`
- `trend`
  - example: `最近7天每天的取消率趋势`
- `detail_list`
  - example: `最近7天上海待接驾订单明细`
- `detail_list` with join filter
  - example: `最近7天司机张三的待接驾订单明细`

Rejection scenarios:

- detail query missing narrowing filter
- unknown domain
- denied role attempting detail access
- invalid planner output, if induced through planner-specific tests

Assertions must focus on business contracts, not exact model phrasing. The suite verifies:

- valid `RawPlan`
- expected `query_mode`
- non-empty or correctly rejected MySQL results
- stable API error codes
- persisted audit records

## 9. Runtime Wiring Design

The current server package is not yet a runnable query service. This design upgrades it into a full runtime composition root.

Runtime composition order:

1. load config
2. load and validate catalog
3. open datasource registry from datasource config
4. open audit repository connection
5. construct MiniMax planner
6. construct executor backed by datasource registry
7. construct orchestrator service
8. expose HTTP handler and server mux

The executor must stop pretending to be multi-datasource by interface only. It must resolve the incoming `datasource_id` into the correct readonly `*sql.DB` at execution time.

## 10. Test Layout

Production code remains in `internal/...`. Test-only helpers live under `tests/...`.

Planned layout:

- `tests/testsupport/mysqlbootstrap`
  - database creation
  - migration application
  - seed insertion
  - readonly user setup
- `tests/live/minimax_planner_e2e_test.go`
  - planner-only live contract tests
- `tests/live/query_flow_e2e_test.go`
  - full API or orchestrator round-trip tests with MiniMax + MySQL

This preserves DDD boundaries while still enabling true live verification.

## 11. Environment Variables

Default test execution will require the following environment variables:

- `MINIMAX_API_KEY`
- `MINIMAX_MODEL`
  - default: `MiniMax-M2.7-highspeed`
- `MYSQL_RIDE_HAILING_ROOT_DSN`
  - used for bootstrap, schema setup, and seed insertion
- `MYSQL_RIDE_HAILING_RO_DSN`
  - used for readonly query execution
- `MYSQL_NL2SQL_AUDIT_DSN`
  - optional only if audit storage is split from the business database

Missing required live variables will fail the default test run.

## 12. Failure Policy

The live path distinguishes between hard failures and bounded transient failures.

Hard failures:

- missing credentials
- authentication failures
- MySQL bootstrap failure
- schema drift
- planner output that violates `RawPlan` schema
- query results that do not satisfy expected contracts

Bounded transient handling:

- a very small retry budget for MiniMax timeout or 5xx responses
- serial request execution
- no unbounded backoff loops

The repository must fail fast after the bounded retry window, rather than silently hiding instability.

## 13. Risks

Known risks accepted by this design:

- default tests become network-dependent
- live MiniMax availability can fail otherwise-correct builds
- token consumption becomes part of routine verification cost
- result drift is possible if prompts are too loose

Mitigations:

- fixed question set
- strict system prompt
- JSON-only planner contract
- low-variance request settings
- serial execution
- assertions on contract and data shape rather than prose wording

## 14. Implementation Outline

Implementation proceeds in six stages:

1. align design and repository constraints documents
2. add live MiniMax planner client and tests
3. make datasource opening and executor routing real
4. add MySQL bootstrap and seed-data support
5. wire a real runnable server runtime
6. add default live MiniMax + MySQL end-to-end tests

## 15. Final Decision

The repository will adopt a hybrid testing model:

- deterministic unit tests remain for core modules
- default package tests also call live MiniMax and real MySQL
- test data is dynamic and time-relative
- supported query scenarios are covered by fixed Chinese question sets
- production runtime wiring is upgraded to a real deployable service path

This gives the project an actual proof path for the end-to-end NL2SQL contract instead of only validating isolated modules.
