-- =============================================================================
-- Kubric ClickHouse DDL — L0-9 Schema
-- All tenant-partitioned OCSF class tables, TTLs, and materialized views.
-- Idempotent: all DDL uses CREATE ... IF NOT EXISTS.
-- =============================================================================
-- Apply with:
--   clickhouse-client --host localhost --port 9000 \
--     --multiquery < migrations/clickhouse/001_ocsf_events.sql
-- =============================================================================

-- ─── Database ─────────────────────────────────────────────────────────────────

CREATE DATABASE IF NOT EXISTS kubric;

-- =============================================================================
-- TABLE 1 — kubric.ocsf_events
-- Canonical fan-in table for ALL OCSF events from the NATS bridge.
-- OCSF class_uid values:
--   1001 = Threat Finding        (KAI-TRIAGE, TI)
--   2002 = Vulnerability Finding (VDR)
--   3002 = Authentication        (ITDR)
--   4001 = Network Activity      (NDR — ocsf_network_events mirror)
--   4007 = Process Activity      (EDR — ocsf_process_events mirror)
--   4008 = File Activity         (FIM)
--   5001 = Inventory Info        (UEM)
--   6001 = Compliance Finding    (GRC / KIC)
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_events
(
    tenant_id     LowCardinality(String),
    event_id      String,
    timestamp     DateTime64(3, 'UTC'),
    class_uid     UInt32,
    severity_id   UInt8,
    severity      LowCardinality(String),
    event_class   LowCardinality(String),
    payload       String,              -- raw JSON payload
    blake3_hash   FixedString(32),     -- content-addressable integrity hash
    agent_id      String,
    _inserted_at  DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, toStartOfHour(timestamp), class_uid, event_id)
TTL timestamp + INTERVAL 90 DAY;

-- =============================================================================
-- TABLE 2 — kubric.ocsf_process_events
-- OCSF class 4007 — Process Activity (EDR / CoreSec).
-- Receives batches from cmd/nats-clickhouse-bridge (subject kubric.*.edr.*).
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_process_events
(
    tenant_id    LowCardinality(String),
    event_id     String,
    time         DateTime64(3, 'UTC'),
    activity_id  UInt8,               -- 1=Launch 2=Terminate 3=Open 4=Inject 5=SetUID
    pid          UInt32,
    ppid         UInt32,
    process_name LowCardinality(String),
    executable   String,
    cmdline      String,
    user         String,
    integrity    LowCardinality(String), -- trusted/untrusted/unknown
    actor        String,              -- JSON: parent process context
    file_hash    String,              -- SHA256 / BLAKE3 of the binary
    metadata     String,              -- JSON: additional OCSF metadata
    agent_id     String,
    blake3_hash  FixedString(32),
    _inserted_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, toStartOfHour(time), activity_id, pid)
TTL time + INTERVAL 90 DAY;

-- =============================================================================
-- TABLE 3 — kubric.ocsf_network_events
-- OCSF class 4001 — Network Activity (NDR / NetGuard).
-- Receives DPI-enriched flow records from NetGuard agent.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_network_events
(
    tenant_id    LowCardinality(String),
    event_id     String,
    time         DateTime64(3, 'UTC'),
    activity_id  UInt8,               -- 1=Open 2=Close 3=Reset 4=Fail 5=Refuse 6=Traffic
    src_ip       IPv6,                -- IPv4-mapped-IPv6 for uniform storage
    src_port     UInt16,
    dst_ip       IPv6,
    dst_port     UInt16,
    protocol     LowCardinality(String), -- TCP/UDP/ICMP/DNS/TLS/HTTP
    direction    LowCardinality(String), -- inbound/outbound/lateral
    bytes_sent   UInt64,
    bytes_recv   UInt64,
    packets_sent UInt32,
    packets_recv UInt32,
    duration_ms  UInt32,
    app_proto    LowCardinality(String), -- nDPI L7 classification
    tls_sni      String,             -- TLS SNI hostname
    dns_query    String,             -- DNS question if DNS event
    http_host    String,
    http_uri     String,
    http_method  LowCardinality(String),
    http_status  UInt16,
    geo_src_cc   LowCardinality(String), -- ISO 3166 country code
    geo_dst_cc   LowCardinality(String),
    is_encrypted UInt8,              -- 0/1
    agent_id     String,
    blake3_hash  FixedString(32),
    _inserted_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, toStartOfHour(time), src_ip, dst_ip)
TTL time + INTERVAL 90 DAY;

-- =============================================================================
-- TABLE 4 — kubric.ocsf_auth_events
-- OCSF class 3002 — Authentication Activity (ITDR).
-- Covers login, MFA, SSO, account lockout, token issue/revocation.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_auth_events
(
    tenant_id     LowCardinality(String),
    event_id      String,
    time          DateTime64(3, 'UTC'),
    activity_id   UInt8,             -- 1=Logon 2=Logoff 3=Auth 4=MFA 5=ImpersonateUser
    user_name     String,
    user_uid      String,            -- directory UUID / SID / UPN
    user_domain   LowCardinality(String),
    src_ip        IPv6,
    src_device    String,            -- hostname or agent_id
    auth_protocol LowCardinality(String), -- NTLM/Kerberos/SAML/OAuth2/OIDC/local
    status        LowCardinality(String), -- success/failure/unknown
    is_mfa        UInt8,             -- 0/1
    failure_reason LowCardinality(String),
    session_uid   String,
    agent_id      String,
    blake3_hash   FixedString(32),
    _inserted_at  DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, toStartOfHour(time), user_name, status)
TTL time + INTERVAL 180 DAY;

-- =============================================================================
-- TABLE 5 — kubric.ocsf_file_events
-- OCSF class 4008 — File Activity (FIM / CoreSec eBPF inotify).
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_file_events
(
    tenant_id    LowCardinality(String),
    event_id     String,
    time         DateTime64(3, 'UTC'),
    activity_id  UInt8,              -- 1=Create 2=Read 3=Update 4=Delete 5=Rename
    path         String,
    file_name    LowCardinality(String),
    extension    LowCardinality(String),
    size_bytes   UInt64,
    sha256       FixedString(64),
    owner_user   String,
    actor_pid    UInt32,
    actor_name   String,
    is_sensitive UInt8,              -- 0/1 (matched PII/secret path patterns)
    agent_id     String,
    blake3_hash  FixedString(32),
    _inserted_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, toStartOfHour(time), path)
TTL time + INTERVAL 90 DAY;

-- =============================================================================
-- TABLE 6 — kubric.ocsf_vuln_findings
-- OCSF class 2002 — Vulnerability Finding (VDR: Nuclei + Trivy + Grype + Syft).
-- One row per CVE/finding per asset per scan run.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_vuln_findings
(
    tenant_id      LowCardinality(String),
    event_id       String,
    time           DateTime64(3, 'UTC'),
    scan_run_id    String,           -- correlates all findings from one scan
    source         LowCardinality(String), -- nuclei/trivy/grype/syft/osquery
    cve_id         LowCardinality(String),
    title          String,
    severity       LowCardinality(String), -- CRITICAL/HIGH/MEDIUM/LOW/INFO
    severity_id    UInt8,
    cvss_score     Float32,
    epss_score     Float32,
    asset          String,           -- hostname/image/package PURL
    asset_type     LowCardinality(String), -- host/container/package/iac
    status         LowCardinality(String), -- open/in_remediation/remediated
    remediated_at  Nullable(DateTime64(3, 'UTC')),
    raw_payload    String,           -- full scanner JSON output
    agent_id       String,
    _inserted_at   DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, time, severity_id DESC, cve_id)
TTL time + INTERVAL 365 DAY;

-- =============================================================================
-- TABLE 7 — kubric.ocsf_compliance_findings
-- OCSF class 6001 — Compliance Finding (KIC: OPA + OpenSCAP + Checkov).
-- One row per control check per agent per assessment run.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_compliance_findings
(
    tenant_id    LowCardinality(String),
    event_id     String,
    time         DateTime64(3, 'UTC'),
    assessment_id String,
    framework    LowCardinality(String), -- CIS_v8/NIST_CSF/SOC2/ISO27001/PCI_DSS
    control_id   String,             -- e.g. CIS-1.1.1
    control_name String,
    status       LowCardinality(String), -- pass/fail/not_applicable
    asset        String,
    remediation  String,
    evidence     String,             -- JSON snippet from scanner
    agent_id     String,
    _inserted_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, time, framework, status)
TTL time + INTERVAL 365 DAY;

-- =============================================================================
-- TABLE 8 — kubric.ocsf_threat_findings
-- OCSF class 1001 — Threat Finding (KAI-TRIAGE + TI layer).
-- Enriched alert with MITRE ATT&CK mappings and LLM triage summary.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ocsf_threat_findings
(
    tenant_id        LowCardinality(String),
    event_id         String,
    time             DateTime64(3, 'UTC'),
    triage_id        String,
    severity         LowCardinality(String),
    severity_id      UInt8,
    confidence       Float32,
    mitre_tactics    Array(LowCardinality(String)),
    mitre_techniques Array(LowCardinality(String)),
    kill_chain_phase LowCardinality(String),
    summary          String,         -- LLM-generated summary
    recommended_action String,
    source_event_ids Array(String),  -- linked ocsf_events UUIDs
    ioc_matches      Array(String),  -- matched IOCs from ti_ioc_cache
    model_used       LowCardinality(String),
    agent_id         String,
    _inserted_at     DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, time, severity_id DESC)
TTL time + INTERVAL 365 DAY;

-- =============================================================================
-- TABLE 9 — kubric.kai_triage_results
-- Written by KAI-TRIAGE after Ollama/LLM enrichment.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.kai_triage_results
(
    tenant_id          LowCardinality(String),
    triage_id          String,
    source_event_id    String,
    timestamp          DateTime64(3, 'UTC'),
    severity           LowCardinality(String),
    mitre_techniques   Array(String),
    summary            String,
    recommended_action String,
    confidence         Float32,
    model_used         LowCardinality(String),
    _inserted_at       DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, timestamp, severity);

-- =============================================================================
-- TABLE 10 — kubric.kai_health_scores
-- KiSS Score time-series written by KAI-SENTINEL (every 5 min per tenant).
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.kai_health_scores
(
    tenant_id        LowCardinality(String),
    computed_at      DateTime64(3, 'UTC'),
    kiss_score       Float32,        -- 0–100 composite
    vuln_score       Float32,        -- VDR component  (30%)
    compliance_score Float32,        -- KIC component  (25%)
    detection_score  Float32,        -- EDR/NDR hits   (25%)
    response_score   Float32,        -- MTTR component (20%)
    open_criticals   UInt32,
    open_highs       UInt32,
    active_incidents UInt32,
    _inserted_at     DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(computed_at))
ORDER BY (tenant_id, computed_at)
TTL computed_at + INTERVAL 365 DAY;

-- =============================================================================
-- TABLE 11 — kubric.kai_foresight_risks
-- KAI-FORESIGHT predictive risk scores (30-min cadence per tenant).
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.kai_foresight_risks
(
    tenant_id          LowCardinality(String),
    computed_at        DateTime64(3, 'UTC'),
    risk_score         Float32,      -- 0–100 predicted risk
    alert_velocity     Float32,      -- alerts/hour over last 24h
    vuln_density       Float32,      -- criticals per 100 assets
    lateral_movement   UInt8,        -- 0/1 detected
    data_exfil_signal  UInt8,        -- 0/1 detected
    top_risk_factors   Array(String),
    recommendations    Array(String),
    _inserted_at       DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(computed_at))
ORDER BY (tenant_id, computed_at)
TTL computed_at + INTERVAL 90 DAY;

-- =============================================================================
-- TABLE 12 — kubric.ti_ioc_cache
-- Threat intelligence IOC dedup cache (multi-source: MISP, CISA KEV, OTX).
-- ReplacingMergeTree deduplicates by (ioc_type, ioc_value, source) on merge.
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.ti_ioc_cache
(
    ioc_type     LowCardinality(String),  -- ip/domain/hash/url/email/asn
    ioc_value    String,
    source       LowCardinality(String),  -- misp/otx/cisa_kev/abuseipdb/spamhaus
    threat_type  LowCardinality(String),  -- malware/c2/phishing/scanner/tor
    malware_family String,
    tags         Array(LowCardinality(String)),
    confidence   Float32,
    tlp          LowCardinality(String),  -- white/green/amber/red
    first_seen   DateTime64(3, 'UTC'),
    last_seen    DateTime64(3, 'UTC'),
    expiry       Nullable(DateTime64(3, 'UTC')),
    _inserted_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (ioc_type, ioc_value, source)
TTL last_seen + INTERVAL 30 DAY;

-- =============================================================================
-- TABLE 13 — kubric.billing_usage_events
-- Per-event usage metering consumed by KAI-CLERK BillingWorkflow.
-- One row per NATS message processed by the bridge (sampled 1:1000 for cost).
-- =============================================================================

CREATE TABLE IF NOT EXISTS kubric.billing_usage_events
(
    tenant_id   LowCardinality(String),
    time        DateTime64(3, 'UTC'),
    event_type  LowCardinality(String), -- process/network/vuln/auth/compliance
    class_uid   UInt32,
    count       UInt64 DEFAULT 1,
    _inserted_at DateTime DEFAULT now()
)
ENGINE = SummingMergeTree(count)
PARTITION BY (tenant_id, toYYYYMM(time))
ORDER BY (tenant_id, toStartOfDay(time), event_type)
TTL time + INTERVAL 365 DAY;

-- =============================================================================
-- MATERIALIZED VIEW — mv_daily_alert_counts
-- Pre-aggregated daily alert counts per tenant (powers KiSS velocity metric).
-- =============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.mv_daily_alert_counts
ENGINE = SummingMergeTree()
PARTITION BY (tenant_id, toYYYYMM(day))
ORDER BY (tenant_id, day, severity)
AS SELECT
    tenant_id,
    toDate(timestamp)                AS day,
    severity,
    countMerge(1)                    AS alert_count   -- compat with SummingMT
FROM kubric.ocsf_events
GROUP BY tenant_id, day, severity;

-- =============================================================================
-- MATERIALIZED VIEW — mv_hourly_network_totals
-- Pre-aggregated hourly bandwidth per tenant (powers NDR anomaly detection).
-- =============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.mv_hourly_network_totals
ENGINE = SummingMergeTree()
PARTITION BY (tenant_id, toYYYYMM(hour))
ORDER BY (tenant_id, hour, app_proto)
AS SELECT
    tenant_id,
    toStartOfHour(time)              AS hour,
    app_proto,
    sum(bytes_sent + bytes_recv)     AS total_bytes,
    sum(packets_sent + packets_recv) AS total_packets,
    count()                          AS flow_count
FROM kubric.ocsf_network_events
GROUP BY tenant_id, hour, app_proto;

-- =============================================================================
-- MATERIALIZED VIEW — mv_tenant_vuln_summary
-- Live vulnerability severity counts per tenant (KiSS vuln_score input).
-- =============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.mv_tenant_vuln_summary
ENGINE = SummingMergeTree()
PARTITION BY (tenant_id, toYYYYMM(day))
ORDER BY (tenant_id, day, severity)
AS SELECT
    tenant_id,
    toDate(time)   AS day,
    severity,
    count()        AS finding_count
FROM kubric.ocsf_vuln_findings
WHERE status != 'remediated'
GROUP BY tenant_id, day, severity;
