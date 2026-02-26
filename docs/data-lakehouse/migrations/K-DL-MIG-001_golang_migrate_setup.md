# K-DL-MIG-001 — golang-migrate Setup

## Overview

[golang-migrate](https://github.com/golang-migrate/migrate) manages PostgreSQL schema versioning for K-SVC and K-DATA services. Migrations run on service startup via `migrate.Up()`.

## Directory Structure

```
db/migrations/
├── 001_layer0_foundation.up.sql
├── 001_layer0_foundation.down.sql
├── 002_tenant_rls.up.sql
├── 002_tenant_rls.down.sql
├── 003_oscal_ingestion.up.sql     (K-DL-PG-004)
├── 003_oscal_ingestion.down.sql
├── 004_contract_rates.up.sql      (K-DL-PG-003)
└── 004_contract_rates.down.sql
```

## Go Integration

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) error {
    m, err := migrate.New("file://db/migrations", dbURL)
    if err != nil {
        return fmt.Errorf("migration init: %w", err)
    }
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migration up: %w", err)
    }
    version, dirty, _ := m.Version()
    log.Printf("Database at migration version %d (dirty=%v)", version, dirty)
    return nil
}
```

## CLI Usage

```bash
# Create a new migration
migrate create -ext sql -dir db/migrations -seq <name>

# Apply all pending migrations
migrate -path db/migrations -database "$DATABASE_URL" up

# Rollback last migration
migrate -path db/migrations -database "$DATABASE_URL" down 1

# Check current version
migrate -path db/migrations -database "$DATABASE_URL" version

# Force version (recovery from dirty state)
migrate -path db/migrations -database "$DATABASE_URL" force <version>
```

## K8s Init Container

```yaml
initContainers:
  - name: migrate
    image: migrate/migrate:v4.17.0
    command: ["migrate"]
    args:
      - "-path=/migrations"
      - "-database=$(DATABASE_URL)"
      - "up"
    envFrom:
      - secretRef:
          name: postgresql-credentials
    volumeMounts:
      - name: migrations
        mountPath: /migrations
```

## CI/CD Integration

Run `migrate -path db/migrations -database "$TEST_DB_URL" up` in the test job before integration tests. Verify migration reversibility with `down` + `up` cycle.
