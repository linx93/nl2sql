# AGENTS.md

## Project Purpose

This repository builds an internal NL2SQL backend service in Go.

The system accepts natural language questions, converts them into controlled query plans, builds safe MySQL queries, executes them through readonly connections, and returns structured results.

## Product Scope

- v1 is backend API only.
- v1 is MySQL only.
- The service may connect to multiple readonly MySQL datasources.
- A single request must execute against exactly one datasource.
- v1 supports aggregate queries and controlled detail-list queries.
- v1 does not support arbitrary SQL.
- v1 does not support cross-datasource queries.
- v1 does not support unrestricted detail export.

## Core Architecture Rules

- The LLM may generate `RawPlan` JSON only. It must not generate executable SQL for production use.
- Backend code resolves `RawPlan` into canonical `ResolvedPlan`.
- Backend code builds executable SQL from curated semantic config.
- Every generated SQL statement must pass the guard layer before execution.
- Query execution must use readonly MySQL credentials only.
- Audit logging is mandatory for both success and failure paths.

## Query Modes

Supported query modes:

- `aggregate_overview`
- `ranking`
- `trend`
- `detail_list`

Rules:

- Aggregate queries use metric and dimension semantics.
- Detail queries must map to an approved `DetailViewSpec`.
- Detail queries return only the first N rows and must expose truncation state.

## Configuration Model

Keep config split into three layers:

1. `configs/datasources.yaml`
   Runtime datasource and pool configuration.
2. `configs/schemas/*.generated.yaml`
   Generated MySQL schema snapshots from `information_schema`.
3. `configs/domains/<domain>/*.yaml`
   Curated semantic config for metrics, dimensions, detail views, aliases, and roles.

Rules:

- Generated schema files are facts, not hand-written business semantics.
- Domain semantic config must stay conservative by default.
- Do not enable tables, columns, metrics, or detail views without explicit review.

## Code Organization

Expected package layout:

- `cmd/server`: HTTP service entry point
- `cmd/nl2sqlctl`: internal tooling for datasource, schema, scaffold, and config validation
- `internal/api`: transport layer
- `internal/orchestrator`: end-to-end flow coordination
- `internal/planner`: LLM plan generation and schema validation
- `internal/resolver`: `RawPlan` to `ResolvedPlan`
- `internal/builder`: controlled SQL generation
- `internal/guard`: AST and policy validation
- `internal/executor`: readonly MySQL execution
- `internal/formatter`: response formatting
- `internal/audit`: query log persistence
- `internal/catalog`: runtime semantic catalog loading and validation
- `internal/datasource`: datasource registry and pool management
- `internal/schema`: schema pull logic
- `internal/scaffold`: semantic config scaffold generation

Keep `resolver`, `builder`, `guard`, and `formatter` as pure as practical. Keep external effects at the edges.

## DDD Rules

This repository uses domain-driven design for core business code.

Requirements:

- Model the core domain explicitly instead of pushing business rules into handlers or infrastructure code.
- Keep ubiquitous language consistent across code, config, tests, and docs.
- Core concepts such as `Domain`, `Datasource`, `RawPlan`, `ResolvedPlan`, `MetricSpec`, `DimensionSpec`, `DetailViewSpec`, and `RolePolicy` must be represented as first-class domain concepts.
- Put domain invariants and business rules in domain-centric packages, not in transport or persistence layers.
- Application orchestration belongs in `internal/orchestrator`; transport concerns belong in `internal/api`; persistence and external integrations belong in adapter-style packages such as `internal/executor`, `internal/planner`, and `internal/audit`.
- Infrastructure code must not own business rules that should live in domain or application services.
- Prefer clear bounded contexts such as catalog, planning, resolution, SQL generation, execution, and audit instead of a generic all-in-one service layer.

## TDD Rules

This repository uses strict test-driven development for production behavior.

Mandatory rule:

> No production code without a failing test first.

Requirements:

- Write one failing test first.
- Run it and confirm it fails for the expected reason.
- Write the minimal code to make it pass.
- Re-run the test and confirm it passes.
- Refactor only after green.

Do not:

- write implementation before the test
- add tests after implementation and call it TDD
- skip the failing-test step
- rely on manual testing for core logic

Exceptions are limited to low-risk scaffolding or plain config/document files.

TDD and DDD are both mandatory for production behavior. Domain models and domain services must also be developed test-first.

## Git Hook Rules

- The repository must provide a git pre-commit hook for local commits.
- Before every commit, the hook must run the required test command and fail the commit if tests do not pass.
- Before every commit, the hook must also check for encoding problems and obvious Chinese garbling.
- Commits must not bypass the hook unless there is explicit approval for an exceptional emergency case.
- Hook logic must be versioned in the repository instead of relying only on undocumented local machine setup.

## Git Commit Rules

- Git commit messages in this repository must use Chinese.
- Commit messages should clearly describe the intent of the change, not just repeat filenames.
- Prefer concise Chinese summaries in conventional formats such as `功能: ...`、`修复: ...`、`文档: ...`、`重构: ...`、`测试: ...`、`初始化: ...`.
- Do not mix English-only commit subjects into normal project history unless there is explicit approval.

## Encoding and Chinese Text Rules

- All source code, config, SQL, Markdown, and hook files must use UTF-8 encoding.
- Do not introduce mixed encodings or platform-dependent garbled Chinese text.
- Any newly added or modified Chinese text must be reviewed in file form to ensure it is readable and not mojibake.
- If tooling generates non-UTF-8 output, it must be converted before commit.

## Chinese Comment Rules

- Key business code must include detailed Chinese comments.
- Core business structs, interfaces, methods, functions, variables, constants, fields, and important branches must have Chinese comments that explain business meaning, constraints, and usage.
- Do not write meaningless comments that only restate syntax; comments must explain domain intent, boundary, or reasoning.
- Comments for domain objects should prefer business semantics over low-level implementation narration.

## SQL Comment Rules

- All database files must use Chinese comments for SQL statements.
- This applies to create, update, query, delete, migration, rollback, seed, and maintenance SQL.
- Every significant SQL statement should have a Chinese comment immediately above it explaining purpose, affected object, and important constraints.
- Complex joins, filters, data repair logic, and destructive operations require more detailed Chinese comments.

## Testing Strategy

Prefer deterministic tests for core logic:

- unit tests for resolver, builder, guard, formatter
- catalog and config validation tests
- datasource registry tests
- orchestrator tests with fake planner and fake executor
- MySQL integration tests for executor only

Do not make real online LLM calls part of the default CI gate.

## Security and Safety Constraints

- Never use write-capable database credentials for query execution.
- Never allow arbitrary SQL passthrough.
- Never bypass the guard layer.
- Never expose internal SQL errors directly to end users.
- Never add cross-datasource execution in v1.
- Never turn detail queries into unrestricted exports.

## Implementation Priorities

When in doubt, prioritize:

1. safety
2. correctness
3. auditability
4. deterministic behavior
5. TDD compliance
6. DDD clarity
7. narrow scope over broad capability

## Source of Truth

Read these files before major changes:

- `docs/project-constraints.md`
- `docs/plans/2026-04-27-nl2sql-design.md`
- `docs/plans/2026-04-27-nl2sql-implementation.md`

If code or behavior conflicts with those documents, update the design intentionally instead of drifting silently.
