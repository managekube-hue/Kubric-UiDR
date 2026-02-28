# K-DL-MIG-001 — golang-migrate Setup

> **Tool:** golang-migrate v4.17+  
> **Database:** PostgreSQL 16 (kubric_core @ 10.0.50.22:5432)  
> **Migration Dir:** `db/migrations/`  
> **Naming Convention:** `{version}_{description}.up.sql` / `{version}_{description}.down.sql`  
> **CI Integration:** Woodpecker CI pipeline (.woodpecker/migrate.yml)

---

## 1. Install golang-migrate

### 1.1 CLI Binary

```bash
# Linux (servers / CI)
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz | \
  tar xz && mv migrate /usr/local/bin/migrate

# macOS (dev)
brew install golang-migrate

# Verify
migrate --version
# 4.17.1
```

### 1.2 Go Library (Embedded in API)

```bash
go get -u github.com/golang-migrate/migrate/v4
go get -u github.com/golang-migrate/migrate/v4/database/postgres
go get -u github.com/golang-migrate/migrate/v4/source/file
go get -u github.com/golang-migrate/migrate/v4/source/iofs
```

---

## 2. Migration Directory Structure

```
db/
├── migrations/
│   ├── 000001_create_tenants.up.sql
│   ├── 000001_create_tenants.down.sql
│   ├── 000002_create_users.up.sql
│   ├── 000002_create_users.down.sql
│   ├── 000003_create_alerts.up.sql
│   ├── 000003_create_alerts.down.sql
│   ├── 000004_create_detection_rules.up.sql
│   ├── 000004_create_detection_rules.down.sql
│   ├── 000005_create_kai_embeddings.up.sql
│   ├── 000005_create_kai_embeddings.down.sql
│   ├── 000006_create_audit_events.up.sql
│   ├── 000006_create_audit_events.down.sql
│   ├── 000007_enable_rls.up.sql
│   ├── 000007_enable_rls.down.sql
│   ├── 000008_create_indexes.up.sql
│   ├── 000008_create_indexes.down.sql
│   ├── 000009_seed_default_tenant.up.sql
│   └── 000009_seed_default_tenant.down.sql
└── seed/
    └── dev_seed.sql
```

---

## 3. Example Migrations

### 3.1 000001_create_tenants.up.sql

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS core;

CREATE TABLE core.tenants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL UNIQUE,
    slug        TEXT NOT NULL UNIQUE,
    tier        TEXT NOT NULL DEFAULT 'standard'
                CHECK (tier IN ('free', 'standard', 'enterprise')),
    active      BOOLEAN DEFAULT true,
    settings    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_tenants_slug ON core.tenants(slug);
CREATE INDEX idx_tenants_active ON core.tenants(active) WHERE active = true;

-- Updated_at trigger
CREATE OR REPLACE FUNCTION core.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tenants_updated_at
    BEFORE UPDATE ON core.tenants
    FOR EACH ROW
    EXECUTE FUNCTION core.update_updated_at();
```

### 3.2 000001_create_tenants.down.sql

```sql
DROP TRIGGER IF EXISTS tenants_updated_at ON core.tenants;
DROP FUNCTION IF EXISTS core.update_updated_at();
DROP TABLE IF EXISTS core.tenants;
DROP SCHEMA IF EXISTS core CASCADE;
```

### 3.3 000007_enable_rls.up.sql

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE core.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.detection_rules ENABLE ROW LEVEL SECURITY;

-- Tenant isolation policies
-- API sets: SET LOCAL kubric.current_tenant_id = '<uuid>';

CREATE POLICY tenant_isolation ON core.tenants
    USING (id = current_setting('kubric.current_tenant_id', true)::UUID);

CREATE POLICY tenant_isolation ON core.users
    USING (tenant_id = current_setting('kubric.current_tenant_id', true)::UUID);

CREATE POLICY tenant_isolation ON core.alerts
    USING (tenant_id = current_setting('kubric.current_tenant_id', true)::UUID);

CREATE POLICY tenant_isolation ON core.detection_rules
    USING (tenant_id = current_setting('kubric.current_tenant_id', true)::UUID);

-- Allow kubric_api to bypass RLS (service account)
ALTER TABLE core.tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE core.users FORCE ROW LEVEL SECURITY;
ALTER TABLE core.alerts FORCE ROW LEVEL SECURITY;
ALTER TABLE core.detection_rules FORCE ROW LEVEL SECURITY;
```

### 3.4 000007_enable_rls.down.sql

```sql
DROP POLICY IF EXISTS tenant_isolation ON core.tenants;
DROP POLICY IF EXISTS tenant_isolation ON core.users;
DROP POLICY IF EXISTS tenant_isolation ON core.alerts;
DROP POLICY IF EXISTS tenant_isolation ON core.detection_rules;

ALTER TABLE core.tenants DISABLE ROW LEVEL SECURITY;
ALTER TABLE core.users DISABLE ROW LEVEL SECURITY;
ALTER TABLE core.alerts DISABLE ROW LEVEL SECURITY;
ALTER TABLE core.detection_rules DISABLE ROW LEVEL SECURITY;
```

---

## 4. CLI Usage

### 4.1 Database Connection URL

```bash
# Set connection string
export DATABASE_URL="postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_core?sslmode=require"
```

### 4.2 Create a New Migration

```bash
migrate create -ext sql -dir db/migrations -seq create_incidents
# Creates:
#   db/migrations/000010_create_incidents.up.sql
#   db/migrations/000010_create_incidents.down.sql
```

### 4.3 Run Migrations

```bash
# Apply all pending
migrate -database "${DATABASE_URL}" -path db/migrations up

# Apply specific number of migrations
migrate -database "${DATABASE_URL}" -path db/migrations up 3

# Rollback last migration
migrate -database "${DATABASE_URL}" -path db/migrations down 1

# Rollback all
migrate -database "${DATABASE_URL}" -path db/migrations down

# Check current version
migrate -database "${DATABASE_URL}" -path db/migrations version
# 7

# Force version (after fixing a dirty migration)
migrate -database "${DATABASE_URL}" -path db/migrations force 7
```

### 4.4 Validate Migrations

```bash
# Check for missing down migrations or naming issues
migrate -database "${DATABASE_URL}" -path db/migrations validate
```

---

## 5. Go Embedded Migrations

Run migrations automatically on API startup:

File: `internal/database/migrate.go`

```go
package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending database migrations.
func RunMigrations(pool *pgxpool.Pool, logger *zap.Logger) error {
	// Get *sql.DB from pgxpool for migrate compatibility
	db := stdlib.OpenDBFromPool(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "schema_migrations",
		SchemaName:      "public",
	})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNoChange {
		logger.Info("no existing migrations")
	} else {
		logger.Info("current migration version",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("database schema is up to date")
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	newVersion, _, _ := m.Version()
	logger.Info("migrations applied successfully",
		zap.Uint("from_version", version),
		zap.Uint("to_version", newVersion),
	)

	return nil
}
```

### 5.1 Call from API Main

```go
// cmd/api/main.go
func main() {
    // ... pool setup ...

    logger.Info("running database migrations")
    if err := database.RunMigrations(pool, logger); err != nil {
        logger.Fatal("migration failed", zap.Error(err))
    }

    // ... start server ...
}
```

---

## 6. Makefile Targets

```makefile
# Makefile additions

DB_URL ?= postgres://kubric_api:$(PG_API_PASS)@10.0.50.22:5432/kubric_core?sslmode=require
MIGRATE_DIR := db/migrations

.PHONY: migrate-up migrate-down migrate-create migrate-version migrate-force

migrate-up:
	migrate -database "$(DB_URL)" -path $(MIGRATE_DIR) up

migrate-down:
	migrate -database "$(DB_URL)" -path $(MIGRATE_DIR) down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATE_DIR) -seq $$name

migrate-version:
	migrate -database "$(DB_URL)" -path $(MIGRATE_DIR) version

migrate-force:
	@read -p "Force version: " ver; \
	migrate -database "$(DB_URL)" -path $(MIGRATE_DIR) force $$ver

migrate-validate:
	@echo "Checking migration files..."
	@ls -la $(MIGRATE_DIR)/*.sql | wc -l
	@echo "Verifying pairs..."
	@for f in $(MIGRATE_DIR)/*.up.sql; do \
		down=$${f%.up.sql}.down.sql; \
		if [ ! -f "$$down" ]; then \
			echo "MISSING: $$down"; \
			exit 1; \
		fi; \
	done
	@echo "All migrations have up/down pairs ✓"
```

---

## 7. CI Pipeline Integration

File: `.woodpecker/migrate.yml`

```yaml
when:
  event: [push, pull_request]
  path:
    include: ["db/migrations/**"]

steps:
  # Validate migration files have up/down pairs
  validate-migrations:
    image: alpine:3.20
    commands:
      - |
        echo "=== Checking migration pairs ==="
        for f in db/migrations/*.up.sql; do
          down="${f%.up.sql}.down.sql"
          if [ ! -f "$down" ]; then
            echo "ERROR: Missing down migration: $down"
            exit 1
          fi
        done
        echo "All migrations have up/down pairs ✓"

  # Dry run against test database
  test-migrations:
    image: migrate/migrate:v4.17.1
    commands:
      - |
        # Create test database
        PGPASSWORD=${PG_ADMIN_PASS} psql -h 10.0.50.22 -U postgres -c "DROP DATABASE IF EXISTS kubric_test"
        PGPASSWORD=${PG_ADMIN_PASS} psql -h 10.0.50.22 -U postgres -c "CREATE DATABASE kubric_test OWNER kubric_api"
        
        # Apply all migrations
        migrate -database "postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_test?sslmode=disable" \
          -path db/migrations up
        
        # Rollback all
        migrate -database "postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_test?sslmode=disable" \
          -path db/migrations down -all
        
        # Re-apply (idempotency check)
        migrate -database "postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_test?sslmode=disable" \
          -path db/migrations up
        
        # Cleanup
        PGPASSWORD=${PG_ADMIN_PASS} psql -h 10.0.50.22 -U postgres -c "DROP DATABASE kubric_test"
        
        echo "Migration test passed ✓"
    secrets: [pg_admin_pass, pg_api_pass]

  # Apply to production (main branch only)
  apply-production:
    image: migrate/migrate:v4.17.1
    commands:
      - migrate -database "postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_core?sslmode=require" -path db/migrations up
      - migrate -database "postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_core?sslmode=require" -path db/migrations version
    secrets: [pg_api_pass]
    when:
      event: push
      branch: main
```

---

## 8. Verification

```bash
# Current migration version
migrate -database "${DATABASE_URL}" -path db/migrations version

# Schema_migrations table
psql "${DATABASE_URL}" -c "SELECT * FROM schema_migrations;"
#  version | dirty
# ---------+-------
#        9 | f

# List all tables created by migrations
psql "${DATABASE_URL}" -c "
  SELECT schemaname, tablename
  FROM pg_tables
  WHERE schemaname IN ('core', 'kai', 'audit')
  ORDER BY schemaname, tablename;"

# Verify RLS is enabled
psql "${DATABASE_URL}" -c "
  SELECT schemaname, tablename, rowsecurity
  FROM pg_tables
  WHERE schemaname = 'core' AND rowsecurity = true;"

# Migration files integrity
ls -la db/migrations/ | grep -c ".sql"
# Should be even number (up + down pairs)
```
