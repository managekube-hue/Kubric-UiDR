# K-DL-MIG-003 — Atlas CI Schema Sync

> **Tool:** Ariga Atlas v0.27+  
> **Schema Source:** `db/schema.hcl` (declarative) + `db/migrations/` (versioned)  
> **Database:** PostgreSQL 16 (kubric_core @ 10.0.50.22:5432)  
> **CI:** Woodpecker CI pipeline  
> **Use Case:** Declarative schema management, drift detection, CI-enforced schema changes

---

## 1. Install Atlas

### 1.1 CLI

```bash
# Linux (servers / CI)
curl -sSf https://atlasgo.sh | sh

# macOS (dev)
brew install ariga/tap/atlas

# Verify
atlas version
# atlas version v0.27.0
```

### 1.2 Go SDK (Optional)

```bash
go get ariga.io/atlas-go-sdk@latest
```

---

## 2. Atlas Project Configuration

File: `atlas.hcl` (project root)

```hcl
// atlas.hcl — Atlas project configuration for Kubric

variable "pg_url" {
  type    = string
  default = getenv("DATABASE_URL")
}

variable "pg_dev_url" {
  type    = string
  default = "docker://postgres/16/kubric_dev?search_path=public"
}

// ─── Environments ───

env "local" {
  src = "file://db/schema.hcl"
  url = var.pg_url
  dev = var.pg_dev_url

  migration {
    dir = "file://db/migrations"
  }

  diff {
    skip {
      drop_schema = true
      drop_table  = true
    }
  }

  lint {
    destructive {
      error = true
    }
    data_depend {
      error = true
    }
  }
}

env "production" {
  src = "file://db/schema.hcl"
  url = var.pg_url

  migration {
    dir    = "file://db/migrations"
    format = "golang-migrate"  // Compatible with golang-migrate
  }

  diff {
    skip {
      drop_schema = true
      drop_table  = true
      drop_column = true
    }
  }

  lint {
    destructive {
      error = true
    }
    data_depend {
      error = true
    }
    naming {
      error   = true
      match   = "^[a-z][a-z0-9_]*$"
      message = "Table/column names must be lowercase snake_case"
    }
  }
}

env "ci" {
  src = "file://db/schema.hcl"
  dev = var.pg_dev_url

  migration {
    dir    = "file://db/migrations"
    format = "golang-migrate"
  }

  lint {
    destructive {
      error = true
    }
    data_depend {
      error = true
    }
  }
}
```

---

## 3. Declarative Schema

File: `db/schema.hcl`

```hcl
// db/schema.hcl — Kubric Core Database Schema (Atlas HCL)

// ─── Extensions ───
schema "public" {}

schema "core" {}
schema "kai" {}
schema "audit" {}

extension "uuid-ossp" {
  schema  = schema.public
  version = "1.1"
}

extension "pgcrypto" {
  schema  = schema.public
  version = "1.3"
}

extension "pg_stat_statements" {
  schema  = schema.public
}

extension "vector" {
  schema  = schema.public
  version = "0.7.4"
}

extension "pg_trgm" {
  schema  = schema.public
}

// ─── Functions ───
function "update_updated_at" {
  schema = schema.core
  lang   = PLpgSQL
  return = trigger
  as     = <<-SQL
    BEGIN
      NEW.updated_at = now();
      RETURN NEW;
    END;
  SQL
}

// ─── Tables: core ───

table "tenants" {
  schema = schema.core

  column "id" {
    type    = uuid
    default = sql("uuid_generate_v4()")
  }
  column "name" {
    type = text
  }
  column "slug" {
    type = text
  }
  column "tier" {
    type    = text
    default = "standard"
  }
  column "active" {
    type    = boolean
    default = true
  }
  column "settings" {
    type    = jsonb
    default = "{}"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_tenants_slug" {
    columns = [column.slug]
    unique  = true
  }

  index "idx_tenants_active" {
    columns = [column.active]
    where   = "active = true"
  }

  check "tier_check" {
    expr = "tier IN ('free', 'standard', 'enterprise')"
  }
}

table "users" {
  schema = schema.core

  column "id" {
    type    = uuid
    default = sql("uuid_generate_v4()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "email" {
    type = text
  }
  column "name" {
    type = text
  }
  column "role" {
    type    = text
    default = "analyst"
  }
  column "password_hash" {
    type = text
  }
  column "active" {
    type    = boolean
    default = true
  }
  column "last_login" {
    type = timestamptz
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_users_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
    on_delete   = CASCADE
  }

  index "idx_users_tenant_email" {
    columns = [column.tenant_id, column.email]
    unique  = true
  }

  check "role_check" {
    expr = "role IN ('admin', 'engineer', 'analyst', 'viewer')"
  }
}

table "alerts" {
  schema = schema.core

  column "id" {
    type    = uuid
    default = sql("uuid_generate_v4()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "severity" {
    type = text
  }
  column "title" {
    type = text
  }
  column "description" {
    type = text
    null = true
  }
  column "source" {
    type = text
  }
  column "status" {
    type    = text
    default = "open"
  }
  column "assigned_to" {
    type = uuid
    null = true
  }
  column "raw_event_id" {
    type = text
    null = true
  }
  column "mitre_tactic" {
    type = text
    null = true
  }
  column "mitre_technique" {
    type = text
    null = true
  }
  column "metadata" {
    type    = jsonb
    default = "{}"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_alerts_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
    on_delete   = CASCADE
  }

  foreign_key "fk_alerts_assignee" {
    columns     = [column.assigned_to]
    ref_columns = [table.users.column.id]
    on_delete   = SET_NULL
  }

  index "idx_alerts_tenant_status" {
    columns = [column.tenant_id, column.status]
  }

  index "idx_alerts_severity" {
    columns = [column.severity]
  }

  index "idx_alerts_created" {
    columns = [column.created_at]
    type    = BTREE
  }

  index "idx_alerts_mitre" {
    columns = [column.mitre_technique]
  }

  check "severity_check" {
    expr = "severity IN ('critical', 'high', 'medium', 'low', 'info')"
  }

  check "status_check" {
    expr = "status IN ('open', 'acknowledged', 'investigating', 'resolved', 'false_positive')"
  }
}

table "detection_rules" {
  schema = schema.core

  column "id" {
    type    = uuid
    default = sql("uuid_generate_v4()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "name" {
    type = text
  }
  column "engine" {
    type = text
  }
  column "content" {
    type = text
  }
  column "enabled" {
    type    = boolean
    default = true
  }
  column "severity" {
    type    = text
    default = "medium"
  }
  column "tags" {
    type    = sql("text[]")
    default = "{}"
  }
  column "created_by" {
    type = uuid
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_rules_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
    on_delete   = CASCADE
  }

  check "engine_check" {
    expr = "engine IN ('sigma', 'yara', 'suricata', 'custom')"
  }
}

// ─── Tables: kai ───

table "embeddings" {
  schema = schema.kai

  column "id" {
    type    = uuid
    default = sql("uuid_generate_v4()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "source_type" {
    type = text
  }
  column "source_id" {
    type = uuid
  }
  column "embedding" {
    type = sql("vector(1536)")
  }
  column "metadata" {
    type    = jsonb
    default = "{}"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
}

// ─── Tables: audit ───

table "events" {
  schema = schema.audit

  column "id" {
    type = bigserial
  }
  column "tenant_id" {
    type = uuid
  }
  column "user_id" {
    type = uuid
    null = true
  }
  column "action" {
    type = text
  }
  column "resource" {
    type = text
  }
  column "resource_id" {
    type = uuid
    null = true
  }
  column "old_value" {
    type = jsonb
    null = true
  }
  column "new_value" {
    type = jsonb
    null = true
  }
  column "ip_address" {
    type = inet
    null = true
  }
  column "user_agent" {
    type = text
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_audit_tenant_time" {
    columns = [column.tenant_id, column.created_at]
  }
}
```

---

## 4. Atlas CLI Usage

### 4.1 Inspect Current Schema

```bash
# Inspect live database
atlas schema inspect -u "${DATABASE_URL}"

# Output as HCL
atlas schema inspect -u "${DATABASE_URL}" --format '{{ hcl . }}'

# Output as SQL
atlas schema inspect -u "${DATABASE_URL}" --format '{{ sql . }}'
```

### 4.2 Diff (Schema vs Database)

```bash
# Compare desired schema (HCL) against live database
atlas schema diff \
  --from "file://db/schema.hcl" \
  --to "${DATABASE_URL}" \
  --dev-url "docker://postgres/16/dev"

# Diff output is the SQL needed to bring database in sync
```

### 4.3 Apply Changes (Declarative)

```bash
# Preview changes (dry run)
atlas schema apply \
  --env production \
  --url "${DATABASE_URL}" \
  --dry-run

# Apply changes
atlas schema apply \
  --env production \
  --url "${DATABASE_URL}" \
  --auto-approve
```

### 4.4 Generate Migration from Diff

```bash
# Auto-generate a golang-migrate compatible migration from schema changes
atlas migrate diff create_incidents \
  --env production \
  --dir "file://db/migrations" \
  --dev-url "docker://postgres/16/dev"

# This creates:
#   db/migrations/000010_create_incidents.up.sql
#   db/migrations/000010_create_incidents.down.sql
```

### 4.5 Validate Migrations

```bash
# Check migration directory integrity
atlas migrate validate --env ci

# Lint migrations for issues
atlas migrate lint \
  --env ci \
  --dir "file://db/migrations" \
  --dev-url "docker://postgres/16/dev" \
  --latest 1
```

---

## 5. CI Pipeline — Schema Drift Detection

File: `.woodpecker/atlas-ci.yml`

```yaml
when:
  event: [push, pull_request]
  path:
    include: ["db/**", "atlas.hcl"]

steps:
  # 1. Lint migration files
  atlas-lint:
    image: arigaio/atlas:0.27
    commands:
      - |
        atlas migrate lint \
          --dir "file://db/migrations" \
          --dev-url "docker://postgres/16/dev" \
          --latest 5 \
          --format '{{ range .Steps }}{{ .Name }}: {{ range .Reports }}{{ .Text }}{{ end }}{{ end }}'
    when:
      path:
        include: ["db/migrations/**"]

  # 2. Validate migration checksums
  atlas-validate:
    image: arigaio/atlas:0.27
    commands:
      - atlas migrate validate --dir "file://db/migrations" --dev-url "docker://postgres/16/dev"
      - echo "Migration validation passed ✓"

  # 3. Dry-run against test database
  atlas-dry-run:
    image: arigaio/atlas:0.27
    environment:
      - DATABASE_URL=postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_test?sslmode=disable
    commands:
      - |
        # Create test database
        PGPASSWORD=${PG_ADMIN_PASS} psql -h 10.0.50.22 -U postgres \
          -c "DROP DATABASE IF EXISTS kubric_test" \
          -c "CREATE DATABASE kubric_test OWNER kubric_api"

        # Apply all migrations
        atlas migrate apply \
          --url "${DATABASE_URL}" \
          --dir "file://db/migrations"

        # Check schema drift
        atlas schema diff \
          --from "file://db/schema.hcl" \
          --to "${DATABASE_URL}" \
          --dev-url "docker://postgres/16/dev" \
          --format '{{ sql . }}'

        # Cleanup
        PGPASSWORD=${PG_ADMIN_PASS} psql -h 10.0.50.22 -U postgres \
          -c "DROP DATABASE kubric_test"

        echo "Dry run passed ✓"
    secrets: [pg_admin_pass, pg_api_pass]

  # 4. Apply to production (main branch, push only)
  atlas-apply-production:
    image: arigaio/atlas:0.27
    environment:
      - DATABASE_URL=postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_core?sslmode=require
    commands:
      - |
        # Show pending migrations
        atlas migrate status --url "${DATABASE_URL}" --dir "file://db/migrations"

        # Apply
        atlas migrate apply --url "${DATABASE_URL}" --dir "file://db/migrations"

        # Verify
        atlas migrate status --url "${DATABASE_URL}" --dir "file://db/migrations"

        echo "Production migration applied ✓"
    secrets: [pg_api_pass]
    when:
      event: push
      branch: main

  # 5. Schema drift alert (scheduled, optional)
  drift-detection:
    image: arigaio/atlas:0.27
    environment:
      - DATABASE_URL=postgres://kubric_api:${PG_API_PASS}@10.0.50.22:5432/kubric_core?sslmode=require
    commands:
      - |
        DRIFT=$(atlas schema diff \
          --from "file://db/schema.hcl" \
          --to "${DATABASE_URL}" \
          --dev-url "docker://postgres/16/dev" \
          --format '{{ sql . }}' 2>&1)

        if [ -n "$DRIFT" ]; then
          echo "⚠️ SCHEMA DRIFT DETECTED:"
          echo "$DRIFT"
          # Could send Slack/Teams alert here
          exit 1
        fi
        echo "No schema drift detected ✓"
    secrets: [pg_api_pass]
    when:
      event: cron
      cron: "@daily"
```

---

## 6. Atlas + golang-migrate Coexistence

Atlas can generate migrations in golang-migrate format:

```bash
# Set format in atlas.hcl (already done):
#   format = "golang-migrate"

# Generate new migration from schema.hcl changes
atlas migrate diff add_incidents_table \
  --env production \
  --dir "file://db/migrations"

# Files created are compatible with `migrate` CLI:
#   db/migrations/000010_add_incidents_table.up.sql
#   db/migrations/000010_add_incidents_table.down.sql

# Apply with either tool:
migrate -database "${DATABASE_URL}" -path db/migrations up
# OR
atlas migrate apply --url "${DATABASE_URL}" --dir "file://db/migrations"
```

---

## 7. Verification

```bash
# Atlas version
atlas version

# Inspect current database schema
atlas schema inspect -u "${DATABASE_URL}" | head -50

# Migration status
atlas migrate status --url "${DATABASE_URL}" --dir "file://db/migrations"
# Expected:
#   Migration Status: OK
#   -- Current Version: 9
#   -- Total Applied:   9
#   -- Pending:         0

# Lint latest migration
atlas migrate lint \
  --dir "file://db/migrations" \
  --dev-url "docker://postgres/16/dev" \
  --latest 1

# Validate migration directory
atlas migrate validate \
  --dir "file://db/migrations" \
  --dev-url "docker://postgres/16/dev"

# Check for drift
atlas schema diff \
  --from "file://db/schema.hcl" \
  --to "${DATABASE_URL}" \
  --dev-url "docker://postgres/16/dev"
# Expected: "Schemas are synced, no changes to be made."
```
