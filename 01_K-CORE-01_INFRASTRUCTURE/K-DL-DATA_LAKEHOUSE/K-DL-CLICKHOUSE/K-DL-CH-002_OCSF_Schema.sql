-- ClickHouse OCSF v1.1 event tables
-- Module: K-DL-CH-002
-- Purpose: Store OCSF-normalized security events for all tenants with multi-tenant partitioning

-- Core OCSF events table (all classes)
CREATE TABLE IF NOT EXISTS ocsf_events
(
    class_uid              Int32,
    class_name             LowCardinality(String),
    category_uid           Int32,
    category_name          LowCardinality(String),
    activity_id            Int32,
    activity_name          LowCardinality(String),
    severity_id            Int32,
    severity               LowCardinality(String),
    status_id              Nullable(Int32),
    status                 LowCardinality(String),
    tenant_id              UUID,
    cloud_provider         LowCardinality(String) DEFAULT '',
    cloud_region           LowCardinality(String) DEFAULT '',
    src_endpoint_ip        String              DEFAULT '',
    src_endpoint_hostname  String              DEFAULT '',
    dst_endpoint_ip        String              DEFAULT '',
    dst_endpoint_hostname  String              DEFAULT '',
    actor_user_uid         String              DEFAULT '',
    actor_user_name        String              DEFAULT '',
    process_pid            Nullable(Int32),
    process_name           String              DEFAULT '',
    process_cmd_line       String              DEFAULT '',
    file_path              String              DEFAULT '',
    file_hash_sha256       String              DEFAULT '',
    network_src_ip         String              DEFAULT '',
    network_dst_ip         String              DEFAULT '',
    network_src_port       Nullable(Int32),
    network_dst_port       Nullable(Int32),
    network_protocol       LowCardinality(String) DEFAULT '',
    raw_data               String              DEFAULT '',
    unmapped               Map(String, String),
    time                   DateTime64(3)       DEFAULT now64(),
    event_time             DateTime64(3),
    ingested_at            DateTime64(3)       DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_time), tenant_id)
ORDER BY (tenant_id, class_uid, event_time)
TTL event_time + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

ALTER TABLE ocsf_events ADD INDEX IF NOT EXISTS idx_severity   (severity_id)      TYPE bloom_filter GRANULARITY 4;
ALTER TABLE ocsf_events ADD INDEX IF NOT EXISTS idx_src_ip     (src_endpoint_ip)  TYPE bloom_filter GRANULARITY 4;
ALTER TABLE ocsf_events ADD INDEX IF NOT EXISTS idx_actor_user (actor_user_uid)   TYPE bloom_filter GRANULARITY 4;
ALTER TABLE ocsf_events ADD INDEX IF NOT EXISTS idx_sha256     (file_hash_sha256) TYPE bloom_filter GRANULARITY 4;

-- Process activity events (OCSF class 1007)
CREATE TABLE IF NOT EXISTS ocsf_process_events
(
    tenant_id         UUID,
    activity_id       Int32,
    process_pid       Int32,
    process_name      String,
    process_cmd_line  String,
    process_uid       String,
    parent_pid        Nullable(Int32),
    parent_name       String DEFAULT '',
    actor_user        String DEFAULT '',
    src_hostname      String DEFAULT '',
    file_hash_sha256  String DEFAULT '',
    event_time        DateTime64(3),
    ingested_at       DateTime64(3) DEFAULT now64()
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (tenant_id, process_uid, event_time)
TTL event_time + INTERVAL 7 DAY
SETTINGS index_granularity = 8192;

-- Network activity events (OCSF class 4001)
CREATE TABLE IF NOT EXISTS ocsf_network_events
(
    tenant_id       UUID,
    activity_id     Int32,
    src_ip          String,
    dst_ip          String,
    src_port        Int32  DEFAULT 0,
    dst_port        Int32  DEFAULT 0,
    protocol        LowCardinality(String),
    bytes_in        Int64  DEFAULT 0,
    bytes_out       Int64  DEFAULT 0,
    duration_ms     Int64  DEFAULT 0,
    connection_uid  String DEFAULT '',
    event_time      DateTime64(3),
    ingested_at     DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_time))
ORDER BY (tenant_id, src_ip, dst_ip, event_time)
TTL event_time + INTERVAL 3 DAY
SETTINGS index_granularity = 8192;

-- Authentication events (OCSF class 3002)
CREATE TABLE IF NOT EXISTS ocsf_auth_events
(
    tenant_id        UUID,
    activity_id      Int32,
    actor_user_uid   String DEFAULT '',
    actor_user_name  String DEFAULT '',
    src_ip           String DEFAULT '',
    dst_ip           String DEFAULT '',
    auth_protocol    LowCardinality(String) DEFAULT '',
    mfa_used         UInt8  DEFAULT 0,
    is_remote        UInt8  DEFAULT 0,
    outcome          LowCardinality(String),
    failure_reason   String DEFAULT '',
    session_uid      String DEFAULT '',
    event_time       DateTime64(3),
    ingested_at      DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_time))
ORDER BY (tenant_id, actor_user_uid, event_time)
TTL event_time + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- File system events (OCSF class 1001)
CREATE TABLE IF NOT EXISTS ocsf_file_events
(
    tenant_id         UUID,
    activity_id       Int32,
    file_path         String DEFAULT '',
    file_name         String DEFAULT '',
    file_hash_sha256  String DEFAULT '',
    file_size         Nullable(Int64),
    actor_user        String DEFAULT '',
    src_hostname      String DEFAULT '',
    process_pid       Nullable(Int32),
    process_name      String DEFAULT '',
    event_time        DateTime64(3),
    ingested_at       DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_time))
ORDER BY (tenant_id, file_hash_sha256, event_time)
TTL event_time + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Materialized view: hourly event volume per tenant per class
CREATE MATERIALIZED VIEW IF NOT EXISTS ocsf_event_volume_hourly_mv
ENGINE = SummingMergeTree()
ORDER BY (tenant_id, class_uid, severity_id, hour)
AS SELECT
    tenant_id,
    class_uid,
    class_name,
    severity_id,
    severity,
    toStartOfHour(event_time) AS hour,
    count()                    AS event_count
FROM ocsf_events
GROUP BY tenant_id, class_uid, class_name, severity_id, severity, toStartOfHour(event_time);
