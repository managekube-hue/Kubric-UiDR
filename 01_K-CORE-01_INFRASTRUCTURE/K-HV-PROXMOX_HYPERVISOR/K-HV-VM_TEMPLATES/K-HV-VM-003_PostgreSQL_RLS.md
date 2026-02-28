# K-HV-VM-003 — PostgreSQL with Row-Level Security

> **Template VM ID:** 9003  
> **Base Image:** Ubuntu 24.04 LTS cloud image  
> **vCPU:** 4 | **RAM:** 16 GB | **OS Disk:** 50 GB | **Data Disk:** 200 GB (Ceph RBD)  
> **Target Node:** pve-kubric-02  
> **VM ID:** 221 (primary)  
> **IP:** 10.0.50.22/24 (Workload), 10.0.200.22/24 (Storage/Replication)  
> **Ports:** 5432 (PostgreSQL), 9187 (postgres_exporter)

---

## 1. Create VM Template

```bash
qm create 9003 --name postgres-template --ostype l26 \
  --cpu cputype=host --cores 4 --sockets 1 \
  --memory 16384 --balloon 8192 \
  --net0 virtio,bridge=vmbr2,tag=50 \
  --net1 virtio,bridge=vmbr1,tag=200 \
  --scsihw virtio-scsi-single --machine q35 \
  --agent enabled=1

qm importdisk 9003 /var/lib/vz/template/iso/noble-server-cloudimg-amd64.raw kubric-ceph
qm set 9003 --scsi0 kubric-ceph:vm-9003-disk-0,iothread=1,discard=on,ssd=1
qm resize 9003 scsi0 50G

# Data disk for PG data directory
qm set 9003 --scsi1 kubric-ceph:200,iothread=1,discard=on,ssd=1

qm set 9003 --ide2 kubric-ceph:cloudinit
qm set 9003 --boot order=scsi0
qm set 9003 --serial0 socket --vga serial0
qm set 9003 --ciuser kubric --sshkeys /root/.ssh/kubric_deploy.pub
qm set 9003 --nameserver 10.0.100.1 --searchdomain kubric.local
```

---

## 2. Cloud-Init User Data

File: `/var/lib/vz/snippets/postgres-cloud.yml`

```yaml
#cloud-config
hostname: postgres-kubric
manage_etc_hosts: true
timezone: UTC

users:
  - name: kubric
    groups: [sudo]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... kubric-deploy

package_update: true
packages:
  - curl
  - wget
  - htop
  - jq
  - qemu-guest-agent

disk_setup:
  /dev/sdb:
    table_type: gpt
    layout: true
    overwrite: false

fs_setup:
  - label: pgdata
    filesystem: xfs
    device: /dev/sdb1
    opts: "-f -L pgdata"

mounts:
  - ["/dev/sdb1", "/var/lib/postgresql", "xfs", "noatime,nodiratime", "0", "2"]

runcmd:
  # Install PostgreSQL 16
  - |
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql-16 postgresql-16-pgvector

  # Fix ownership after mount
  - chown postgres:postgres /var/lib/postgresql
  - chmod 700 /var/lib/postgresql

  # Enable + start
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
  - systemctl enable postgresql

final_message: "PostgreSQL VM ready. Apply kubric configs then start."
```

```bash
qm set 9003 --cicustom "user=local:snippets/postgres-cloud.yml"
qm template 9003
```

---

## 3. Clone and Deploy

```bash
qm clone 9003 221 --name postgres-kubric --full --target pve-kubric-02 --storage kubric-ceph
qm set 221 \
  --ipconfig0 ip=10.0.50.22/24,gw=10.0.50.1 \
  --ipconfig1 ip=10.0.200.22/24
qm start 221

ssh kubric@10.0.50.22 "cloud-init status --wait"
```

---

## 4. PostgreSQL Configuration

### 4.1 postgresql.conf Tuning

```bash
ssh kubric@10.0.50.22 "sudo tee /etc/postgresql/16/main/conf.d/kubric.conf" <<'EOF'
# ─── Connection ───
listen_addresses = '0.0.0.0'
port = 5432
max_connections = 200
superuser_reserved_connections = 5

# ─── Memory ───
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 32MB
maintenance_work_mem = 1GB
huge_pages = try

# ─── WAL ───
wal_level = replica
max_wal_senders = 5
wal_keep_size = 2GB
max_replication_slots = 5
wal_compression = zstd
checkpoint_timeout = 15min
checkpoint_completion_target = 0.9
max_wal_size = 4GB
min_wal_size = 1GB

# ─── Query Planner ───
random_page_cost = 1.1
effective_io_concurrency = 200
default_statistics_target = 200

# ─── Parallel Query ───
max_worker_processes = 8
max_parallel_workers_per_gather = 4
max_parallel_workers = 8
max_parallel_maintenance_workers = 4

# ─── Logging ───
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql-%Y-%m-%d.log'
log_rotation_age = 1d
log_rotation_size = 256MB
log_min_duration_statement = 500
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on
log_statement = 'ddl'
log_line_prefix = '%t [%p] %u@%d '

# ─── SSL ───
ssl = on
ssl_cert_file = '/etc/postgresql/16/main/certs/server.crt'
ssl_key_file = '/etc/postgresql/16/main/certs/server.key'

# ─── pgvector ───
shared_preload_libraries = 'pg_stat_statements,pgvector'

# ─── Autovacuum ───
autovacuum_max_workers = 4
autovacuum_naptime = 30s
autovacuum_vacuum_cost_delay = 2ms
EOF
```

### 4.2 pg_hba.conf

```bash
ssh kubric@10.0.50.22 "sudo tee /etc/postgresql/16/main/pg_hba.conf" <<'EOF'
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# Local
local   all             postgres                                peer
local   all             all                                     peer

# Kubric API servers
hostssl kubric_core     kubric_api      10.0.50.0/24            scram-sha-256
hostssl kubric_core     kubric_readonly 10.0.50.0/24            scram-sha-256

# KAI ML service (pgvector)
hostssl kubric_core     kubric_kai      10.0.50.0/24            scram-sha-256

# n8n workflow
hostssl kubric_core     kubric_n8n      10.0.50.33/32           scram-sha-256

# Replication
hostssl replication     kubric_repl     10.0.200.0/24           scram-sha-256

# Monitoring
host    all             kubric_monitor  10.0.50.0/24            scram-sha-256

# Deny everything else
host    all             all             0.0.0.0/0               reject
EOF
```

### 4.3 Start PostgreSQL

```bash
ssh kubric@10.0.50.22 "sudo systemctl restart postgresql"
```

---

## 5. Database & Role Creation

```bash
ssh kubric@10.0.50.22 "sudo -u postgres psql" <<'EOSQL'

-- ═══════════════════════════════════════════
-- Roles
-- ═══════════════════════════════════════════
CREATE ROLE kubric_api LOGIN PASSWORD 'CHANGEME_PG_API_PASS';
CREATE ROLE kubric_readonly LOGIN PASSWORD 'CHANGEME_PG_READONLY_PASS';
CREATE ROLE kubric_kai LOGIN PASSWORD 'CHANGEME_PG_KAI_PASS';
CREATE ROLE kubric_n8n LOGIN PASSWORD 'CHANGEME_PG_N8N_PASS';
CREATE ROLE kubric_repl REPLICATION LOGIN PASSWORD 'CHANGEME_PG_REPL_PASS';
CREATE ROLE kubric_monitor LOGIN PASSWORD 'CHANGEME_PG_MONITOR_PASS';

-- ═══════════════════════════════════════════
-- Database
-- ═══════════════════════════════════════════
CREATE DATABASE kubric_core
  OWNER kubric_api
  ENCODING 'UTF8'
  LC_COLLATE 'en_US.UTF-8'
  LC_CTYPE 'en_US.UTF-8'
  TEMPLATE template0;

\c kubric_core

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS vector;           -- pgvector for KAI embeddings
CREATE EXTENSION IF NOT EXISTS pg_trgm;          -- trigram for fuzzy search

-- ═══════════════════════════════════════════
-- Schemas
-- ═══════════════════════════════════════════
CREATE SCHEMA IF NOT EXISTS core AUTHORIZATION kubric_api;
CREATE SCHEMA IF NOT EXISTS kai AUTHORIZATION kubric_kai;
CREATE SCHEMA IF NOT EXISTS audit AUTHORIZATION kubric_api;

-- ═══════════════════════════════════════════
-- Core Tables with RLS
-- ═══════════════════════════════════════════

-- Tenants
CREATE TABLE core.tenants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL UNIQUE,
    slug        TEXT NOT NULL UNIQUE,
    tier        TEXT NOT NULL DEFAULT 'standard' CHECK (tier IN ('free','standard','enterprise')),
    active      BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

-- Users
CREATE TABLE core.users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'analyst' CHECK (role IN ('admin','engineer','analyst','viewer')),
    password_hash TEXT NOT NULL,
    active      BOOLEAN DEFAULT true,
    last_login  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE (tenant_id, email)
);

-- Alerts
CREATE TABLE core.alerts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    severity        TEXT NOT NULL CHECK (severity IN ('critical','high','medium','low','info')),
    title           TEXT NOT NULL,
    description     TEXT,
    source          TEXT NOT NULL,    -- e.g. 'sigma', 'yara', 'ml-anomaly'
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','investigating','resolved','false_positive')),
    assigned_to     UUID REFERENCES core.users(id),
    raw_event_id    TEXT,             -- references ClickHouse event ID
    mitre_tactic    TEXT,
    mitre_technique TEXT,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_alerts_tenant_status ON core.alerts(tenant_id, status);
CREATE INDEX idx_alerts_severity ON core.alerts(severity);
CREATE INDEX idx_alerts_created ON core.alerts(created_at DESC);
CREATE INDEX idx_alerts_mitre ON core.alerts(mitre_technique);

-- Detection Rules
CREATE TABLE core.detection_rules (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    engine       TEXT NOT NULL CHECK (engine IN ('sigma','yara','suricata','custom')),
    content      TEXT NOT NULL,      -- rule body (YAML/YARA/Suricata)
    enabled      BOOLEAN DEFAULT true,
    severity     TEXT DEFAULT 'medium',
    tags         TEXT[] DEFAULT '{}',
    created_by   UUID REFERENCES core.users(id),
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now()
);

-- KAI Embeddings (pgvector)
CREATE TABLE kai.embeddings (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL,
    source_type TEXT NOT NULL,        -- 'alert', 'rule', 'document', 'incident'
    source_id   UUID NOT NULL,
    embedding   vector(1536),         -- OpenAI ada-002 or equivalent dimension
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_embeddings_ivfflat ON kai.embeddings
  USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Audit Log
CREATE TABLE audit.events (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    user_id     UUID,
    action      TEXT NOT NULL,        -- 'create', 'update', 'delete', 'login', 'export'
    resource    TEXT NOT NULL,        -- 'alert', 'rule', 'user', 'tenant'
    resource_id UUID,
    old_value   JSONB,
    new_value   JSONB,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_audit_tenant_time ON audit.events(tenant_id, created_at DESC);

-- ═══════════════════════════════════════════
-- Row-Level Security (RLS)
-- ═══════════════════════════════════════════

-- Enable RLS on all tenant-scoped tables
ALTER TABLE core.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE core.detection_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai.embeddings ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.events ENABLE ROW LEVEL SECURITY;

-- The API passes tenant_id via SET LOCAL:
--   SET LOCAL kubric.current_tenant_id = '<tenant-uuid>';

-- Tenant isolation policy
CREATE POLICY tenant_isolation_tenants ON core.tenants
  USING (id = current_setting('kubric.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_users ON core.users
  USING (tenant_id = current_setting('kubric.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_alerts ON core.alerts
  USING (tenant_id = current_setting('kubric.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_rules ON core.detection_rules
  USING (tenant_id = current_setting('kubric.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_embeddings ON kai.embeddings
  USING (tenant_id = current_setting('kubric.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_audit ON audit.events
  USING (tenant_id = current_setting('kubric.current_tenant_id')::UUID);

-- ═══════════════════════════════════════════
-- Grants
-- ═══════════════════════════════════════════

-- kubric_api: full CRUD on core + audit
GRANT USAGE ON SCHEMA core, audit TO kubric_api;
GRANT ALL ON ALL TABLES IN SCHEMA core TO kubric_api;
GRANT ALL ON ALL SEQUENCES IN SCHEMA core TO kubric_api;
GRANT INSERT, SELECT ON ALL TABLES IN SCHEMA audit TO kubric_api;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA audit TO kubric_api;

-- kubric_kai: full on kai schema, read on core
GRANT USAGE ON SCHEMA kai, core TO kubric_kai;
GRANT ALL ON ALL TABLES IN SCHEMA kai TO kubric_kai;
GRANT ALL ON ALL SEQUENCES IN SCHEMA kai TO kubric_kai;
GRANT SELECT ON ALL TABLES IN SCHEMA core TO kubric_kai;

-- kubric_readonly: SELECT-only on core
GRANT USAGE ON SCHEMA core TO kubric_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA core TO kubric_readonly;

-- kubric_monitor: pg_stat_statements + basic queries
GRANT pg_monitor TO kubric_monitor;
GRANT USAGE ON SCHEMA core TO kubric_monitor;
GRANT SELECT ON ALL TABLES IN SCHEMA core TO kubric_monitor;

-- Default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA core GRANT ALL ON TABLES TO kubric_api;
ALTER DEFAULT PRIVILEGES IN SCHEMA core GRANT SELECT ON TABLES TO kubric_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA kai GRANT ALL ON TABLES TO kubric_kai;

EOSQL
```

---

## 6. Go API RLS Usage Pattern

The Go API must set the tenant context before every query:

```go
// internal/database/rls.go
package database

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant executes fn within a transaction that has RLS tenant set.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(tx pgx.Tx) error) error {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    // Set the tenant context for RLS policies
    _, err = tx.Exec(ctx, "SET LOCAL kubric.current_tenant_id = $1", tenantID)
    if err != nil {
        return fmt.Errorf("set tenant context: %w", err)
    }

    if err := fn(tx); err != nil {
        return err
    }

    return tx.Commit(ctx)
}

// Usage:
// database.WithTenant(ctx, pool, req.TenantID, func(tx pgx.Tx) error {
//     rows, err := tx.Query(ctx, "SELECT * FROM core.alerts WHERE status = 'open'")
//     // RLS automatically filters to current tenant
//     ...
// })
```

---

## 7. Prometheus Exporter

```bash
ssh kubric@10.0.50.22 <<'EOF'
# Install postgres_exporter
wget https://github.com/prometheus-community/postgres_exporter/releases/download/v0.16.0/postgres_exporter-0.16.0.linux-amd64.tar.gz
tar xzf postgres_exporter-0.16.0.linux-amd64.tar.gz
sudo mv postgres_exporter-0.16.0.linux-amd64/postgres_exporter /usr/local/bin/

# Systemd unit
sudo tee /etc/systemd/system/postgres-exporter.service <<'UNIT'
[Unit]
Description=Prometheus PostgreSQL Exporter
After=postgresql.service

[Service]
User=kubric
Environment=DATA_SOURCE_NAME=postgresql://kubric_monitor:CHANGEME_PG_MONITOR_PASS@localhost:5432/kubric_core?sslmode=disable
ExecStart=/usr/local/bin/postgres_exporter --web.listen-address=:9187
Restart=on-failure

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable --now postgres-exporter
EOF
```

---

## 8. Backup to MinIO

```bash
# Daily pg_dump to MinIO
ssh kubric@10.0.50.22 "sudo tee /opt/kubric/pg-backup.sh" <<'SCRIPT'
#!/bin/bash
set -euo pipefail

DATE=$(date +%Y%m%d_%H%M%S)
DUMP_FILE="/tmp/kubric_core_${DATE}.sql.gz"

pg_dump -U postgres -d kubric_core -Fc | gzip > "${DUMP_FILE}"

mc alias set kubric http://10.0.50.23:9000 kubric-admin "${MINIO_ROOT_PASS}"
mc cp "${DUMP_FILE}" kubric/kubric-backups/postgres/kubric_core_${DATE}.sql.gz

# Keep last 30 days
mc rm --recursive --force --older-than 30d kubric/kubric-backups/postgres/

rm -f "${DUMP_FILE}"
echo "[$(date)] Backup complete: kubric_core_${DATE}.sql.gz"
SCRIPT

ssh kubric@10.0.50.22 "sudo chmod +x /opt/kubric/pg-backup.sh"
ssh kubric@10.0.50.22 "echo '0 2 * * * /opt/kubric/pg-backup.sh >> /var/log/kubric/pg-backup.log 2>&1' | sudo crontab -u postgres -"
```

---

## 9. Verification

```bash
# PostgreSQL running
ssh kubric@10.0.50.22 "sudo systemctl status postgresql"
ssh kubric@10.0.50.22 "sudo -u postgres psql -c 'SELECT version();'"

# RLS test
ssh kubric@10.0.50.22 "sudo -u postgres psql kubric_core" <<'EOSQL'
-- Insert test tenant
INSERT INTO core.tenants (id, name, slug) VALUES
  ('11111111-1111-1111-1111-111111111111', 'Acme Corp', 'acme'),
  ('22222222-2222-2222-2222-222222222222', 'Globex Inc', 'globex');

-- Set tenant context and verify isolation
SET LOCAL kubric.current_tenant_id = '11111111-1111-1111-1111-111111111111';
SELECT * FROM core.tenants;
-- Should show only 'Acme Corp'

RESET kubric.current_tenant_id;
EOSQL

# Exporter metrics
curl -s http://10.0.50.22:9187/metrics | grep pg_stat

# Connection count
ssh kubric@10.0.50.22 "sudo -u postgres psql -c 'SELECT count(*) FROM pg_stat_activity;'"
```
