# Verification Checklist

## Required Commands

- Unit and package tests: `go test ./...`
- MySQL executor integration: `go test ./tests/integration/mysql -run TestExecutorRunsReadonlyQueryAgainstMySQL -v`
- Smoke tests: `go test ./tests/smoke -v`
- Config validation: `go run ./cmd/nl2sqlctl config validate`

## Required Environment

- `MYSQL_RIDE_HAILING_RO_DSN` for runtime datasource wiring
- Docker access for MySQL integration tests that start containers

## Pre-Commit Expectations

- Local git hook path should point to `.githooks`
- `go test ./...` must pass
- `scripts/check-encoding.ps1` must report no invalid UTF-8 or obvious Chinese garbling

## Config Maintenance

- Schema snapshot availability check: `go run ./cmd/nl2sqlctl schema pull --datasource ride_hailing_ro`
- Conservative scaffold preview: `go run ./cmd/nl2sqlctl scaffold domain --domain ride_hailing --datasource ride_hailing_ro --tables trip_orders`
