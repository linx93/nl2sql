# Project Constraints

Date: 2026-04-27

## Product Constraints

- v1 is backend API only.
- v1 supports MySQL only.
- The service may connect to multiple readonly MySQL datasources.
- A single request must execute against exactly one datasource.
- v1 supports aggregate queries and controlled detail-list queries.
- v1 does not support arbitrary SQL.
- v1 does not support cross-datasource joins or federation.
- v1 does not support unrestricted detail export.

## Architecture Constraints

- The model may produce `RawPlan` JSON only.
- The backend must resolve `RawPlan` into canonical `ResolvedPlan`.
- The backend, not the model, generates executable SQL.
- Generated SQL must pass a guard layer before execution.
- Execution must use readonly MySQL credentials only.
- Query history and audit logging are mandatory.

## Configuration Constraints

- Datasources are defined in `configs/datasources.yaml`.
- Physical schema snapshots are generated under `configs/schemas/*.generated.yaml`.
- Domain semantics are curated under `configs/domains/<domain>/`.
- Generated schema files are facts, not hand-maintained business semantics.
- Semantic config must default to conservative enablement.

## Delivery Constraints

- Production behavior starts with a failing test first.
- Production business logic must follow both TDD and DDD.
- Pure logic should stay in resolver, builder, guard, and formatter packages.
- External effects should stay at the edges: API, planner, executor, audit.
- Real online LLM calls must not be part of the default CI gate.
- Domain rules must not drift into transport, persistence, or infrastructure layers.
- The repository must provide a pre-commit hook that runs required tests before each commit.
- The repository must block commits when tests fail or when encoding checks detect obvious Chinese garbling.
- Source code, config, SQL, Markdown, and hook files must use UTF-8 encoding.
- Key business code, structs, interfaces, methods, functions, variables, constants, and fields must include detailed Chinese comments.
- Database SQL files must include Chinese comments for create, update, query, delete, migration, rollback, seed, and maintenance statements.
