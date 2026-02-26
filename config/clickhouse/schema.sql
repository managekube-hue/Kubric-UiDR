-- =============================================================================
-- Kubric ClickHouse Schema (L0-9)
-- =============================================================================
-- All tables use MergeTree family with tenant-level partitioning.
--
-- Design principles:
--   ORDER BY (tenant_id, toStartOfHour(timestamp), event_id)
--   PARTITION BY (tenant_id, toYYYYMM(timestamp))
--
-- This enables:
--   • Per-tenant data isolation (ALTER TABLE DROP PARTITION for GDPR deletion)
--   • Per-tenant retention via TTL expressions
--   • Efficient per-tenant queries without full-table scans
--
-- Run: clickhouse-client --multiquery < config/clickhouse/schema.sql
-- =============================================================================

-- ─── Database ─────────────────────────────────────────────────────────────────
CREATE DATABASE IF NOT EXISTS kubric;

-- =============================================================================
-- Table 1: kubric.events — raw OCSF event stream (all agents)
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.events
(
    event_id      String       NOT NULL,
    tenant_id     LowCardinality(String) NOT NULL,
    timestamp     DateTime64(3, 'UTC')   NOT NULL,
    class_uid     UInt32       NOT NULL,   -- OCSF class
    category_uid  UInt32       NOT NULL,   -- OCSF category
    severity_id   UInt8        NOT NULL,   -- 0=Unknown, 1=Info, 2=Low, 3=Med, 4=High, 5=Critical
    activity_id   UInt16       NOT NULL,
    agent_id      String       NOT NULL,
    hostname      String       NOT NULL,
    agent_type    LowCardinality(String) NOT NULL,
    payload       String       NOT NULL,   -- raw JSON blob
    blake3_hash   FixedString(64)         -- integrity check
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, toStartOfHour(timestamp), event_id)
TTL toDateTime(timestamp) + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 2: kubric.triage_results — KAI-TRIAGE enriched events
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.triage_results
(
    triage_id         String       NOT NULL,
    tenant_id         LowCardinality(String) NOT NULL,
    timestamp         DateTime64(3, 'UTC')   NOT NULL,
    source_event_id   String       NOT NULL,
    severity          LowCardinality(String) NOT NULL,  -- CRITICAL/HIGH/MEDIUM/LOW/INFO
    mitre_techniques  Array(String)          NOT NULL,
    summary           String       NOT NULL,
    recommended_action String      NOT NULL,
    confidence        Float32      NOT NULL DEFAULT 0.5,
    model_used        String       NOT NULL,
    kiss_score        Nullable(Float32)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, toStartOfHour(timestamp), triage_id)
TTL toDateTime(timestamp) + INTERVAL 180 DAY
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 3: kubric.vuln_findings — VDR findings mirrored from Postgres
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.vuln_findings
(
    id          String       NOT NULL,
    tenant_id   LowCardinality(String) NOT NULL,
    target      String       NOT NULL,
    scanner     LowCardinality(String) NOT NULL,
    severity    LowCardinality(String) NOT NULL,
    cve_id      String       NOT NULL,
    title       String       NOT NULL,
    status      LowCardinality(String) NOT NULL,
    epss_score  Nullable(Float32),
    created_at  DateTime64(3, 'UTC')   NOT NULL,
    updated_at  DateTime64(3, 'UTC')   NOT NULL
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY (tenant_id, toYYYYMM(created_at))
ORDER BY (tenant_id, id)
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 4: kubric.threat_intel — MISP/STIX/TAXII TI feed ingestion
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.threat_intel
(
    ioc_id       String       NOT NULL,
    tenant_id    LowCardinality(String) NOT NULL DEFAULT '__global__',
    ioc_type     LowCardinality(String) NOT NULL,  -- ip, domain, hash, url, email
    ioc_value    String       NOT NULL,
    threat_type  LowCardinality(String) NOT NULL,  -- malware, phishing, c2, scanner, other
    source       LowCardinality(String) NOT NULL,  -- misp, alienvault, greynoise, shodan
    confidence   UInt8        NOT NULL DEFAULT 50,  -- 0-100
    tlp          LowCardinality(String) NOT NULL DEFAULT 'white',
    tags         Array(String) NOT NULL,
    first_seen   DateTime     NOT NULL,
    last_seen    DateTime     NOT NULL,
    expires_at   Nullable(DateTime),
    raw_json     String       NOT NULL DEFAULT ''
)
ENGINE = MergeTree()
PARTITION BY (ioc_type, toYYYYMM(last_seen))
ORDER BY (ioc_type, ioc_value, last_seen)
TTL last_seen + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 5: kubric.compliance_findings — KIC assessments mirrored from Postgres
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.compliance_findings
(
    id            String       NOT NULL,
    tenant_id     LowCardinality(String) NOT NULL,
    framework     LowCardinality(String) NOT NULL,
    control_id    String       NOT NULL,
    title         String       NOT NULL,
    status        LowCardinality(String) NOT NULL,
    assessed_by   LowCardinality(String) NOT NULL,
    assessed_at   DateTime64(3, 'UTC')   NOT NULL,
    updated_at    DateTime64(3, 'UTC')   NOT NULL
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY (tenant_id, toYYYYMM(assessed_at))
ORDER BY (tenant_id, framework, control_id, id)
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 6: kubric.agent_metrics — PerfTrace host metrics (high-volume)
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.agent_metrics
(
    metric_id    String       NOT NULL DEFAULT generateUUIDv4(),
    tenant_id    LowCardinality(String) NOT NULL,
    agent_id     String       NOT NULL,
    hostname     String       NOT NULL,
    timestamp    DateTime64(3, 'UTC')   NOT NULL,
    -- CPU
    cpu_pct      Float32      NOT NULL DEFAULT 0.0,
    -- Memory
    mem_used_mb  UInt32       NOT NULL DEFAULT 0,
    mem_total_mb UInt32       NOT NULL DEFAULT 0,
    -- Disk I/O
    disk_read_bps  UInt64     NOT NULL DEFAULT 0,
    disk_write_bps UInt64     NOT NULL DEFAULT 0,
    -- Network I/O
    net_rx_bps   UInt64       NOT NULL DEFAULT 0,
    net_tx_bps   UInt64       NOT NULL DEFAULT 0,
    -- Process count
    proc_count   UInt32       NOT NULL DEFAULT 0
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMMDD(timestamp))
ORDER BY (tenant_id, agent_id, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Table 7: kubric.network_flows — NetGuard captured flows
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.network_flows
(
    flow_id      String       NOT NULL DEFAULT generateUUIDv4(),
    tenant_id    LowCardinality(String) NOT NULL,
    agent_id     String       NOT NULL,
    timestamp    DateTime64(3, 'UTC')   NOT NULL,
    src_ip       IPv4         NOT NULL,
    dst_ip       IPv4         NOT NULL,
    src_port     UInt16       NOT NULL,
    dst_port     UInt16       NOT NULL,
    protocol     LowCardinality(String) NOT NULL,
    app_proto    LowCardinality(String) NOT NULL DEFAULT 'unknown',
    bytes_in     UInt64       NOT NULL DEFAULT 0,
    bytes_out    UInt64       NOT NULL DEFAULT 0,
    packets_in   UInt32       NOT NULL DEFAULT 0,
    packets_out  UInt32       NOT NULL DEFAULT 0,
    duration_ms  UInt32       NOT NULL DEFAULT 0,
    flags        Array(String) NOT NULL,
    is_beacon    UInt8        NOT NULL DEFAULT 0,
    beacon_score Float32
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMMDD(timestamp))
ORDER BY (tenant_id, toStartOfMinute(timestamp), src_ip, dst_ip)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- =============================================================================
-- Materialized view: kubric.kiss_scores_hourly — KiSS score trend
-- =============================================================================
CREATE TABLE IF NOT EXISTS kubric.kiss_scores_hourly
(
    tenant_id    LowCardinality(String) NOT NULL,
    hour         DateTime              NOT NULL,
    kiss_score   Float32               NOT NULL,
    vuln_score   Float32               NOT NULL,
    compliance_score Float32           NOT NULL,
    detection_score  Float32           NOT NULL,
    response_score   Float32           NOT NULL,
    sample_count UInt32                NOT NULL
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, hour);

CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.kiss_scores_hourly_mv
TO kubric.kiss_scores_hourly
AS SELECT
    tenant_id,
    toStartOfHour(timestamp) AS hour,
    avg(kiss_score)           AS kiss_score,
    avg(kiss_score)           AS vuln_score,
    avg(kiss_score)           AS compliance_score,
    avg(kiss_score)           AS detection_score,
    avg(kiss_score)           AS response_score,
    count()                   AS sample_count
FROM kubric.triage_results
WHERE kiss_score IS NOT NULL
GROUP BY tenant_id, hour;
