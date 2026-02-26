# K-DL-MIG-003 — Atlas CI Sync

## Overview

[Atlas](https://atlasgo.io/) by Ariga provides declarative schema management — define schema in HCL, Atlas diffs and migrates. Integrates with CI/CD for automated schema sync.

## Workflow

```
Developer writes HCL schema (atlas_schema.hcl)
         │
         ▼
CI: atlas schema diff --from "postgres://..." --to "file://atlas_schema.hcl"
         │
         ▼
Review generated SQL migration
         │
         ▼
CD: atlas schema apply --url "postgres://..." --to "file://atlas_schema.hcl"
```

## CLI Commands

```bash
# Install Atlas
curl -sSf https://atlasgo.sh | sh

# Inspect current database schema
atlas schema inspect --url "postgres://kubric:pass@localhost:5432/kubric?sslmode=disable"

# Generate diff between current DB and desired HCL
atlas schema diff \
  --from "postgres://kubric:pass@localhost:5432/kubric?sslmode=disable" \
  --to "file://migrations/postgres/atlas_schema.hcl"

# Apply schema changes (with approval)
atlas schema apply \
  --url "postgres://kubric:pass@localhost:5432/kubric?sslmode=disable" \
  --to "file://migrations/postgres/atlas_schema.hcl"

# Dry run (show SQL without executing)
atlas schema apply \
  --url "postgres://..." \
  --to "file://atlas_schema.hcl" \
  --dry-run
```

## CI Pipeline Integration

```yaml
# .woodpecker.yml or .github/workflows/atlas-ci.yml
steps:
  - name: atlas-lint
    image: arigaio/atlas:latest
    commands:
      - atlas schema inspect --url "$DATABASE_URL" > current.hcl
      - atlas schema diff --from file://current.hcl --to file://migrations/postgres/atlas_schema.hcl
    environment:
      DATABASE_URL:
        from_secret: test_database_url

  - name: atlas-apply
    image: arigaio/atlas:latest
    commands:
      - atlas schema apply --url "$DATABASE_URL" --to file://migrations/postgres/atlas_schema.hcl --auto-approve
    environment:
      DATABASE_URL:
        from_secret: database_url
    when:
      branch: main
      event: push
```

## Relationship to golang-migrate

| Tool | Approach | Best For |
|------|----------|----------|
| golang-migrate | Imperative (sequential SQL files) | Simple migrations, rollback support |
| Atlas | Declarative (HCL schema diff) | Schema drift detection, CI validation |
| Liquibase | Imperative (XML/YAML changelogs) | K8s init containers, enterprise auditing |

Use golang-migrate for day-to-day migrations. Use Atlas in CI to verify the running schema matches the declared HCL — catches drift caused by manual DDL changes.
