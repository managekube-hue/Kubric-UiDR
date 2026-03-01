-- K-DL-DUCK-002 — DuckDB ML Feature Computation
-- Embedded SQL for computing ML features from local OCSF events
-- Runs in-agent (Go via go-duckdb) or in KAI Python (duckdb module)

-- ============================================================
-- Feature Set 1: Process Behavior Baseline
-- Used by: CoreSec XGBoost malware scorer
-- ============================================================
CREATE OR REPLACE VIEW process_features AS
SELECT
    executable,
    count(*) AS exec_count,
    count(DISTINCT cmdline) AS unique_cmdlines,
    avg(length(cmdline)) AS avg_cmdline_len,
    max(length(cmdline)) AS max_cmdline_len,
    -- Entropy of command line (high entropy = obfuscation)
    sum(- (len / total) * log2(len / total)) AS cmdline_entropy,
    count(DISTINCT uid) AS unique_users,
    count(DISTINCT ppid) AS unique_parents,
    -- Ratio of unique parents to total executions (high = unusual)
    count(DISTINCT ppid)::FLOAT / NULLIF(count(*), 0) AS parent_diversity_ratio,
    min(timestamp_ns) AS first_seen,
    max(timestamp_ns) AS last_seen,
    -- Time span in hours
    (max(timestamp_ns) - min(timestamp_ns)) / 3.6e12 AS active_hours
FROM process_events
GROUP BY executable;

-- ============================================================
-- Feature Set 2: Network Flow Baseline
-- Used by: KAI-FORESIGHT LSTM network anomaly model
-- ============================================================
CREATE OR REPLACE VIEW network_flow_features AS
SELECT
    src_ip,
    dst_ip,
    dst_port,
    l7_protocol,
    count(*) AS flow_count,
    sum(bytes_sent) AS total_bytes_sent,
    sum(bytes_received) AS total_bytes_recv,
    avg(bytes_sent) AS avg_bytes_sent,
    stddev(bytes_sent) AS stddev_bytes_sent,
    -- Beacon detection features
    stddev(epoch_ms(end_time) - epoch_ms(start_time)) AS interval_stddev_ms,
    avg(epoch_ms(end_time) - epoch_ms(start_time)) AS avg_interval_ms,
    -- Connection regularity (low stddev/mean = beaconing)
    CASE WHEN avg(epoch_ms(end_time) - epoch_ms(start_time)) > 0
         THEN stddev(epoch_ms(end_time) - epoch_ms(start_time))
              / avg(epoch_ms(end_time) - epoch_ms(start_time))
         ELSE 0
    END AS interval_cv,  -- Coefficient of variation
    -- Payload size consistency (beacons have uniform sizes)
    CASE WHEN avg(bytes_sent) > 0
         THEN stddev(bytes_sent) / avg(bytes_sent)
         ELSE 0
    END AS payload_cv,
    min(start_time) AS first_seen,
    max(end_time) AS last_seen
FROM network_events
GROUP BY src_ip, dst_ip, dst_port, l7_protocol;

-- ============================================================
-- Feature Set 3: File Integrity Anomaly Features
-- Used by: CoreSec FIM drift detection
-- ============================================================
CREATE OR REPLACE VIEW file_integrity_features AS
SELECT
    path,
    count(*) AS modification_count,
    count(DISTINCT operation) AS unique_operations,
    count(DISTINCT pid) AS unique_modifiers,
    -- Files modified outside business hours (suspicious)
    count(*) FILTER (WHERE hour(timestamp) < 6 OR hour(timestamp) > 22)
        AS offhours_modifications,
    -- Rapid modification (many changes in short window)
    count(*) FILTER (
        WHERE timestamp >= now() - INTERVAL '1 hour'
    ) AS modifications_last_hour,
    min(timestamp) AS first_modified,
    max(timestamp) AS last_modified
FROM file_events
GROUP BY path;

-- ============================================================
-- Feature Set 4: Authentication Anomaly Features
-- Used by: KAI-TRIAGE ITDR behavioral baseline
-- ============================================================
CREATE OR REPLACE VIEW auth_features AS
SELECT
    "user",
    src_ip,
    auth_protocol,
    count(*) AS total_attempts,
    count(*) FILTER (WHERE success = true) AS successful,
    count(*) FILTER (WHERE success = false) AS failed,
    -- Failure ratio (high = brute force or credential stuffing)
    count(*) FILTER (WHERE success = false)::FLOAT
        / NULLIF(count(*), 0) AS failure_ratio,
    count(DISTINCT src_ip) AS unique_source_ips,
    count(DISTINCT domain) AS unique_domains,
    -- Geo-diversity (impossible travel indicator)
    count(DISTINCT src_ip) FILTER (
        WHERE src_ip NOT LIKE '10.%'
          AND src_ip NOT LIKE '192.168.%'
          AND src_ip NOT LIKE '172.1%'
    ) AS external_source_ips,
    min(timestamp) AS first_attempt,
    max(timestamp) AS last_attempt
FROM auth_events
GROUP BY "user", src_ip, auth_protocol;

-- ============================================================
-- Export features to Parquet for model training
-- ============================================================
-- COPY (SELECT * FROM process_features) TO 'features/process.parquet' (FORMAT PARQUET);
-- COPY (SELECT * FROM network_flow_features) TO 'features/network.parquet' (FORMAT PARQUET);
-- COPY (SELECT * FROM file_integrity_features) TO 'features/file.parquet' (FORMAT PARQUET);
-- COPY (SELECT * FROM auth_features) TO 'features/auth.parquet' (FORMAT PARQUET);
