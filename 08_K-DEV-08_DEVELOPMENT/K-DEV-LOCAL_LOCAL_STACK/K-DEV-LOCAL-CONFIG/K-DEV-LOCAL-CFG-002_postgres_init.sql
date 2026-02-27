-- Local development PostgreSQL initialization
-- Module: K-DEV-LOCAL-CFG-002
-- Run: psql -U postgres -f K-DEV-LOCAL-CFG-002_postgres_init.sql
-- Purpose: Bootstraps local dev databases, roles, extensions, and seed data

-- ---------------------------------------------------------------------------
-- Databases
-- ---------------------------------------------------------------------------
SELECT 'Creating databases' AS step;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'kubric') THEN
        PERFORM dblink_exec('dbname=postgres', $$
            CREATE DATABASE kubric
                WITH OWNER      = postgres
                     ENCODING   = 'UTF8'
                     LC_COLLATE = 'en_US.UTF-8'
                     LC_CTYPE   = 'en_US.UTF-8'
                     TEMPLATE   = template0;
        $$);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'temporal') THEN
        PERFORM dblink_exec('dbname=postgres', $$
            CREATE DATABASE temporal
                WITH OWNER      = postgres
                     ENCODING   = 'UTF8'
                     LC_COLLATE = 'en_US.UTF-8'
                     LC_CTYPE   = 'en_US.UTF-8'
                     TEMPLATE   = template0;
        $$);
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Roles (idempotent)
-- ---------------------------------------------------------------------------
DO $$ BEGIN
    CREATE ROLE kubric_superuser BYPASSRLS NOLOGIN;
    EXCEPTION WHEN DUPLICATE_OBJECT THEN NULL;
END $$;

DO $$ BEGIN
    CREATE ROLE kubric_app NOINHERIT NOLOGIN;
    EXCEPTION WHEN DUPLICATE_OBJECT THEN NULL;
END $$;

DO $$ BEGIN
    CREATE ROLE kubric_readonly NOINHERIT NOLOGIN;
    EXCEPTION WHEN DUPLICATE_OBJECT THEN NULL;
END $$;

DO $$ BEGIN
    CREATE USER kubric WITH PASSWORD 'kubric' LOGIN;
    EXCEPTION WHEN DUPLICATE_OBJECT THEN NULL;
END $$;

DO $$ BEGIN
    CREATE USER kubric_ro WITH PASSWORD 'kubric_ro' LOGIN;
    EXCEPTION WHEN DUPLICATE_OBJECT THEN NULL;
END $$;

GRANT ALL PRIVILEGES ON DATABASE kubric TO kubric;
GRANT CONNECT         ON DATABASE kubric TO kubric_ro;
GRANT kubric_superuser TO kubric;
GRANT kubric_readonly  TO kubric_ro;

-- ---------------------------------------------------------------------------
-- Connect to kubric database and set up schema
-- ---------------------------------------------------------------------------
\c kubric

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Default session variable (used by RLS policies)
SET app.tenant_id = '';

-- Schema grants
GRANT ALL ON SCHEMA public TO kubric;
GRANT ALL ON SCHEMA public TO kubric_app;
GRANT USAGE ON SCHEMA public TO kubric_readonly;

-- Future table grants (auto-applied to tables created later)
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kubric_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO kubric_readonly;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO kubric_app;

-- ---------------------------------------------------------------------------
-- Seed: dev tenant (used in local tests)
-- ---------------------------------------------------------------------------
-- Ensure tenants table exists before inserting (migrations may not have run)
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    plan        TEXT        NOT NULL DEFAULT 'professional',
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'kubric:analyst',
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, email)
);

INSERT INTO tenants (id, name, email, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Dev Tenant',
    'dev@kubric.local',
    'professional',
    'active'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO tenants (id, name, email, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'Test Tenant Alpha',
    'alpha@kubric.local',
    'enterprise',
    'active'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, tenant_id, email, role)
VALUES (
    '10000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'admin@kubric.local',
    'kubric:admin'
) ON CONFLICT DO NOTHING;

INSERT INTO users (id, tenant_id, email, role)
VALUES (
    '10000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'analyst@kubric.local',
    'kubric:analyst'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Verify
-- ---------------------------------------------------------------------------
SELECT 'Setup complete' AS status,
       current_database() AS db,
       version() AS pg_version;

SELECT tablename, rowsecurity AS rls
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;
