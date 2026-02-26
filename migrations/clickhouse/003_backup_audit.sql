-- migrations/clickhouse/003_backup_audit.sql
-- Backup audit log table — records every backup run result.
-- Referenced by scripts/backup/clickhouse.go INSERT INTO kubric.backup_audit

CREATE TABLE IF NOT EXISTS kubric.backup_audit
(
    backup_type LowCardinality(String) NOT NULL,   -- clickhouse | vault | postgres | neo4j
    backup_name String                 NOT NULL,   -- e.g. "clickhouse/2026-02-26/kubric-2026-02-26"
    status      LowCardinality(String) NOT NULL,   -- success | failure | partial
    error_msg   String                 DEFAULT '',
    created_at  DateTime64(0, 'UTC')  NOT NULL
)
ENGINE = MergeTree()
ORDER BY (created_at, backup_type)
PARTITION BY toYYYYMM(created_at)
TTL created_at + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
