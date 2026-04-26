# NL2SQL

Internal NL2SQL backend service implemented in Go.

## Scope

- API-only in v1
- MySQL-only in v1
- Multiple readonly datasources at runtime
- Single datasource per request
- Aggregate queries and controlled detail-list queries
- No arbitrary SQL
- No cross-datasource execution

## Delivery Rules

- Core implementation follows test-driven development.
- The LLM generates `RawPlan`, not SQL.
- SQL is built by backend code from curated semantic config.
- Detail queries must go through approved `DetailViewSpec` definitions.

## Project Docs

- `docs/plans/2026-04-27-nl2sql-design.md`
- `docs/plans/2026-04-27-nl2sql-implementation.md`
- `docs/project-constraints.md`

## Initial Layout

- `cmd/server`: HTTP service entry point
- `cmd/nl2sqlctl`: internal config and schema tooling
- `configs`: datasource, generated schema, and domain semantics
- `db/migrations`: MySQL schema migrations for NL2SQL system tables
- `internal`: application code
- `pkg`: small reusable utility packages
- `tests`: smoke and integration tests

