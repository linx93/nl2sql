# NL2SQL Design

Date: 2026-04-27
Status: Approved for implementation planning

## 1. Background

This project builds an internal NL2SQL backend service for non-SQL users. The system accepts natural language questions, converts them into controlled query plans, generates safe MySQL queries, executes them through readonly connections, and returns structured results.

The main engineering problem is not "can the model write SQL", but "can the system safely and repeatably turn natural language into executable queries under clear permission, performance, and audit boundaries".

## 2. Goals

- Build a Go-based backend service for controlled NL2SQL.
- Support both aggregate analytics queries and controlled detail-list queries.
- Support multiple MySQL datasources at runtime.
- Restrict each request to exactly one datasource.
- Keep semantic definitions explicit and auditable.
- Enforce TDD for core implementation work.

## 3. Non-Goals

- No arbitrary SQL editor.
- No write queries.
- No cross-datasource joins or federated query execution.
- No frontend BI or charting UI in v1.
- No unrestricted detail export.
- No external-user exposure in v1.

## 4. Hard Product Boundaries

- API-only in v1.
- MySQL-only in v1.
- Multiple datasource connections are allowed, but each request must resolve to one `domain`, and each `domain` maps to one `datasource_id`.
- Aggregate queries are allowed.
- Detail queries are allowed only through controlled `DetailViewSpec` definitions.
- Detail responses return only the first N rows and must expose truncation state.

## 5. Key Architecture Decisions

### 5.1 Model output is a plan, not SQL

The LLM produces a `RawPlan` in JSON, never executable SQL. The backend resolves that into a canonical `ResolvedPlan`, then builds SQL from controlled semantic definitions.

### 5.2 Semantic layer is explicit

The system uses versioned semantic config for:

- domains
- metrics
- dimensions
- detail views
- joins
- role policies
- aliases

### 5.3 Guard is a verifier, not the primary generator

The `Builder` generates SQL from approved semantic objects. The `Guard` verifies the generated SQL still respects statement, whitelist, complexity, and time-range rules.

### 5.4 Multiple datasources, single-datasource execution per request

The service keeps multiple readonly MySQL pools, but a single request always routes to one datasource through:

`request -> domain -> datasource_id -> *sql.DB`

### 5.5 TDD is mandatory

Core modules must be implemented test-first:

- resolver
- builder
- guard
- formatter
- catalog loader
- datasource registry
- scaffold
- orchestrator

## 6. System Architecture

```text
HTTP API
-> API Layer
-> Orchestrator
-> Catalog Loader
-> Planner (NL -> RawPlan)
-> Resolver (RawPlan -> ResolvedPlan)
-> Builder (ResolvedPlan -> SQL + Args)
-> Guard (AST + policy validation)
-> Executor (readonly MySQL)
-> Formatter
-> Audit Log
-> Response
```

Module responsibilities:

- `api`: HTTP transport, auth passthrough, request validation, request ID injection, response mapping
- `orchestrator`: end-to-end query flow coordination
- `catalog`: load and validate runtime semantic catalog from YAML
- `planner`: call LLM and validate `RawPlan` schema
- `resolver`: resolve aliases, permissions, limits, query mode, detail view, and time range
- `builder`: render controlled MySQL SQL and args
- `guard`: parse SQL AST and enforce final constraints
- `executor`: execute through readonly datasource pool
- `formatter`: produce `columns`, `rows`, `summary`, `meta`
- `audit`: persist end-to-end query log

## 7. Datasource and Domain Model

### 7.1 Datasources

The service supports many MySQL readonly connections simultaneously. Each datasource is defined by:

- `id`
- `driver`
- `dsn_env`
- `database`
- pool settings

Example:

```yaml
datasources:
  - id: ride_hailing_ro
    driver: mysql
    dsn_env: MYSQL_RIDE_HAILING_RO_DSN
    database: ride_hailing
    max_open_conns: 20
    max_idle_conns: 10
    conn_max_lifetime_sec: 1800
```

### 7.2 Domains

Each business domain binds to exactly one datasource:

```yaml
id: ride_hailing
display_name: 网约车订单域
datasource_id: ride_hailing_ro
default_timezone: Asia/Shanghai
enabled: true
```

This makes multi-datasource runtime support simple without turning v1 into a federated query engine.

## 8. Configuration Model

The configuration model is intentionally split into three layers.

### 8.1 Datasource layer

File:

- `configs/datasources.yaml`

Purpose:

- connection and pool config only

### 8.2 Generated schema layer

Files:

- `configs/schemas/<datasource>.generated.yaml`

Purpose:

- store table, column, type, comment, and index snapshot pulled from `information_schema`
- generated automatically
- no business semantics

### 8.3 Domain semantic layer

Files:

- `configs/domains/<domain>/domain.yaml`
- `configs/domains/<domain>/metrics.yaml`
- `configs/domains/<domain>/dimensions.yaml`
- `configs/domains/<domain>/detail_views.yaml`
- `configs/domains/<domain>/roles.yaml`
- `configs/domains/<domain>/aliases.yaml`

Purpose:

- define what NL2SQL may use
- define labels, aliases, rules, and permissions
- kept conservative by default

This solves the maintainability issue: schema is auto-generated, semantics are manually curated.

## 9. Query Planning Model

### 9.1 RawPlan

Generated by the LLM with user-facing vocabulary only.

Example fields:

- `query_mode`
- `metrics`
- `dimensions`
- `detail_subject`
- `select_fields`
- `filters`
- `time_range`
- `order_by`
- `limit`
- `explanation`

### 9.2 ResolvedPlan

Generated by backend resolver with canonical IDs only.

Example fields:

- `query_mode`
- `metric_ids`
- `dimension_ids`
- `detail_view_id`
- `select_column_ids`
- `filters`
- `time_range.start`
- `time_range.end`
- `sort`
- `limit`
- `datasource_id`

This split is required to separate language understanding from execution semantics.

## 10. Query Modes

The final v1 query modes are:

- `aggregate_overview`
- `ranking`
- `trend`
- `detail_list`

Routing:

- aggregate modes use metric/dimension semantics
- detail mode uses `DetailViewSpec`

## 11. Semantic Objects

### 11.1 Aggregate path

- `MetricSpec`
- `DimensionSpec`
- `JoinSpec`
- `TimeFieldSpec`

`MetricSpec` is not just documentation. It must carry the data required to render SQL safely.

### 11.2 Detail path

- `DetailViewSpec`
- allowed columns
- allowed filters
- required time field
- default sort
- row policy key
- masked columns

This is the control point that makes detail queries possible without opening arbitrary row browsing.

## 12. Aggregate Query Rules

Supported:

- aggregate overview
- ranking
- trend

Constraints:

- at most 2 metrics
- ranking supports exactly 1 grouping dimension
- trend supports `day`, `week`, or `month`
- at most 2 joins
- default time range required
- default limit 10
- max limit 100

Not supported:

- arbitrary subqueries
- UNION
- arbitrary HAVING
- window functions
- free-form expressions from model output

## 13. Detail Query Rules

Detail query support is intentionally narrow.

Supported:

- controlled detail views only
- first N rows only
- fixed allowed columns only
- fixed allowed join path only
- fixed allowed filter fields only

Required:

- time range
- role permission for detail mode
- allowed detail view
- allowed selected columns
- narrowing filter when configured

Not supported:

- unrestricted detail export
- aggregate + detail mixed response in one request
- arbitrary selected columns
- arbitrary joins
- unrestricted pagination or scrolling through entire tables

### 13.1 DetailViewSpec example

```yaml
id: detail.waiting_pickup_orders
display_name: 待接驾订单明细
base_table: trip_orders
allowed_joins:
  - drivers
default_select_columns:
  - trip_orders.order_id
  - trip_orders.city_code
  - trip_orders.service_type
  - trip_orders.order_status
  - trip_orders.called_at
allowed_select_columns:
  - trip_orders.order_id
  - trip_orders.city_code
  - trip_orders.service_type
  - trip_orders.order_status
  - trip_orders.called_at
  - drivers.driver_name
allowed_filter_fields:
  - trip_orders.city_code
  - trip_orders.service_type
  - trip_orders.order_status
  - drivers.driver_name
required_time_field: trip_orders.called_at
default_sort:
  field: trip_orders.called_at
  direction: desc
max_limit: 50
max_time_range_days: 30
require_narrowing_filter: true
row_policy_key: city_scope
masked_columns: []
enabled: true
```

## 14. SQL Builder Design

The builder accepts only `ResolvedPlan`.

Output:

- `SQL`
- `Args`
- `ReferencedTables`
- `ReferencedCols`
- `MetricIDs`
- `DimensionIDs`
- `TimeRangeDays`
- `JoinCount`

The builder is not a generic SQL composer. It is a controlled renderer for approved query shapes.

## 15. SQL Guard Design

Guard verifies final SQL against policy.

Validation classes:

1. statement type
2. AST structure
3. table whitelist
4. column whitelist
5. limit and complexity
6. time range
7. parameterization

Allowed auto-fixes:

- inject default limit if missing
- clamp limit to policy maximum

Everything else should reject instead of rewriting silently.

## 16. API Contract

Main endpoints:

- `POST /api/v1/nl2sql/queries`
- `GET /api/v1/nl2sql/queries`
- `GET /api/v1/nl2sql/queries/{request_id}`
- `GET /api/v1/nl2sql/capabilities?domain=<domain>`

Response shape:

- `request_id`
- `status`
- `data.columns`
- `data.rows`
- `data.summary`
- `meta.query_mode`
- `meta.result_kind`
- `meta.row_count`
- `meta.truncated`

Error response shape:

- `error.code`
- `error.message`
- `error.suggestion`

The API must not expose internal SQL, database driver errors, or raw parser errors to normal users.

## 17. Error Model

The pipeline should explicitly record rejection stage:

- `request_validation`
- `planning`
- `resolution`
- `guard`
- `execution`

Recommended error codes:

- `INVALID_REQUEST`
- `UNSUPPORTED_DOMAIN`
- `LLM_TIMEOUT`
- `LLM_INVALID_OUTPUT`
- `UNSUPPORTED_QUERY_MODE`
- `UNKNOWN_METRIC`
- `UNKNOWN_DIMENSION`
- `DETAIL_VIEW_NOT_ALLOWED`
- `DETAIL_COLUMN_NOT_ALLOWED`
- `DETAIL_QUERY_REQUIRES_NARROWING_FILTER`
- `PERMISSION_DENIED`
- `QUERY_TOO_COMPLEX`
- `INVALID_SQL`
- `DB_TIMEOUT`
- `DB_EXECUTION_ERROR`
- `INTERNAL_ERROR`

## 18. Audit and Observability

### 18.1 Audit log

Suggested fields:

- `request_id`
- `user_id`
- `user_role`
- `domain`
- `datasource_id`
- `natural_language_query`
- `raw_plan_json`
- `resolved_plan_json`
- `built_sql`
- `validated_sql`
- `query_mode`
- `result_kind`
- `detail_view_id`
- `rejection_stage`
- `execution_status`
- `error_code`
- `error_message_internal`
- `result_columns_json`
- `result_preview_json`
- `result_row_count`
- `latency_ms`
- `llm_model`
- `prompt_version`
- `sql_fingerprint`
- `created_at`

### 18.2 Metrics

Minimum required:

- request count
- success rate
- rejection rate by stage
- LLM timeout rate
- DB execution failure rate
- p50/p95 latency
- token usage
- top queries

### 18.3 Tracing

Recommended spans:

- `query.request`
- `planner.generate_plan`
- `resolver.resolve_plan`
- `builder.build_sql`
- `guard.validate_sql`
- `executor.run_query`
- `audit.persist_log`

## 19. Package Structure

```text
/cmd/server
/cmd/nl2sqlctl
/internal
  /api
  /audit
  /builder
  /catalog
  /config
  /datasource
  /domain
  /executor
  /formatter
  /guard
  /observability
  /orchestrator
  /planner
  /resolver
  /schema
  /scaffold
/pkg
  /clock
  /idgen
/configs
  /datasources.yaml
  /schemas
  /domains
/db/migrations
/tests
```

Core rule:

- `resolver`, `builder`, and `guard` should remain as pure as practical
- external effects belong at the edges

## 20. TDD and Test Strategy

Mandatory principle:

> No production code without a failing test first.

Priority order:

1. resolver
2. builder
3. guard
4. formatter
5. catalog loader
6. datasource registry
7. scaffold
8. orchestrator
9. executor integration
10. planner contract validation

Test layers:

- pure unit tests for resolver/builder/guard/formatter
- config and catalog validation tests
- datasource registry tests
- MySQL integration tests for executor
- end-to-end orchestration tests with fake planner

Real online LLM calls must not be part of the core CI gate.

## 21. Tooling Workflow for Config Maintenance

To keep semantic config maintainable, v1 should include an internal CLI:

- `nl2sql datasource test --id <datasource>`
- `nl2sql schema pull --datasource <id>`
- `nl2sql scaffold domain --domain <name> --datasource <id> --tables ...`
- `nl2sql config validate`

Purpose:

- test readonly connectivity
- pull generated schema snapshots
- scaffold conservative semantic config
- validate config consistency before runtime

This addresses the practical concern that manual YAML authoring alone would not scale.

## 22. Rollout Gates

Before internal rollout:

- readonly MySQL account verified
- timeout enforcement working
- guard rules covering whitelist, joins, limit, and time range
- detail queries restricted to detail views
- at least one fully configured domain
- audit log persisted end-to-end
- stable error codes
- baseline pressure test completed
- regression question set prepared

## 23. Implementation Order

Recommended order:

1. datasource config + schema pull
2. catalog loader + validation
3. minimal ride_hailing domain config
4. resolver
5. aggregate builder
6. detail builder
7. guard
8. formatter
9. orchestrator
10. executor
11. API
12. real planner integration

This order keeps high-signal, high-test-value work first and unstable dependencies last.

## 24. Final Summary

The approved v1 design is:

- Go backend
- MySQL-only
- multi-datasource runtime support
- single-datasource execution per request
- aggregate queries plus controlled detail queries
- semantic config split into generated schema and curated domain YAML
- LLM outputs plans, not SQL
- backend builds and guards executable SQL
- auditability and permissions are first-class concerns
- TDD is a delivery requirement, not a nice-to-have

This gives the project a realistic path to a controlled internal NL2SQL service without turning v1 into a federated query engine or a free-form database browser.

