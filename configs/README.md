# Config Layout

The config model is split into three layers:

1. `datasources.yaml`
   Runtime datasource and pool settings.
2. `schemas/*.generated.yaml`
   Generated MySQL schema snapshots pulled from `information_schema`.
3. `domains/<domain>/*.yaml`
   Curated semantic config for metrics, dimensions, detail views, aliases, and role policies.

Generated schema files should be produced by tooling. Domain semantics are curated by engineers and reviewed before enablement.

