-- K-DL-CH-005 — KAI Triage Results
-- Stores every AI-enriched triage result produced by the KAI-TRIAGE agent.
-- Queried by CrewAI security_tools.query_recent_alerts() to give agents
-- real-time situational awareness of alert velocity and severity distribution.

CREATE TABLE IF NOT EXISTS kubric.kai_triage_results
(
    triage_id         UUID DEFAULT generateUUIDv4(),
    tenant_id         String,
    source_subject    String,                           -- originating NATS subject
    source_event_id   String,                           -- upstream OCSF/alert event ID
    timestamp         DateTime64(3) DEFAULT now64(3),   -- ms epoch from KAI agent
    severity          LowCardinality(String),            -- CRITICAL / HIGH / MEDIUM / LOW / INFO
    summary           String,                           -- AI-generated English summary
    recommended_action String,                          -- top remediation step
    mitre_techniques  Array(String),                    -- e.g. ['T1059.001', 'T1078']
    confidence        Float32,                          -- 0.0–1.0 model confidence score
    model_used        LowCardinality(String),            -- e.g. crewai/ollama/llama3.2
    analyst_reviewed  UInt8 DEFAULT 0,                  -- 1 after human analyst sign-off
    analyst_email     Nullable(String),
    escalated         UInt8 DEFAULT 0,                  -- 1 if escalated to KAI-ANALYST
    ticket_id         Nullable(String),                 -- Zammad ticket ID if raised
    event_date        Date DEFAULT toDate(timestamp)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(event_date))
ORDER BY (tenant_id, severity, timestamp)
TTL event_date + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- Materialized view: alert velocity per hour by severity (feeds the KiSS score dashboard)
CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.triage_severity_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, severity, hour)
AS SELECT
    tenant_id,
    severity,
    toStartOfHour(timestamp)  AS hour,
    count()                   AS alert_count,
    avg(confidence)           AS avg_confidence,
    countIf(escalated = 1)    AS escalated_count,
    countIf(analyst_reviewed = 1) AS reviewed_count
FROM kubric.kai_triage_results
GROUP BY tenant_id, severity, hour;

-- Bloom filter index for fast event-ID lookups
ALTER TABLE kubric.kai_triage_results
    ADD INDEX idx_source_event source_event_id TYPE bloom_filter GRANULARITY 4;

-- Set index for severity filtering (only 5 values)
ALTER TABLE kubric.kai_triage_results
    ADD INDEX idx_severity severity TYPE set(5) GRANULARITY 1;

-- Projection: tenant-level summary — pre-aggregated for the "query_recent_alerts" tool
ALTER TABLE kubric.kai_triage_results
    ADD PROJECTION prj_tenant_severity
    (
        SELECT tenant_id, severity, count() AS cnt, max(timestamp) AS last_seen
        FROM kubric.kai_triage_results
        GROUP BY tenant_id, severity
    );
