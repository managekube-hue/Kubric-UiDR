-- K-DL-CH-004 — Agent Decision History
-- Tracks every automated decision made by KAI agents for audit trail
-- Immutable append-only table with blake3 chain integrity

CREATE TABLE IF NOT EXISTS kubric.agent_decision_history
(
    decision_id       UUID DEFAULT generateUUIDv4(),
    tenant_id         String,
    agent_name        LowCardinality(String),    -- kai-triage, kai-keeper, etc.
    decision_type     LowCardinality(String),    -- alert_escalation, auto_remediation, ticket_creation, etc.
    severity          UInt8,                      -- 1=info, 2=low, 3=medium, 4=high, 5=critical
    input_event_id    String,                     -- OCSF event that triggered the decision
    input_summary     String,                     -- Human-readable summary of input
    decision_output   String,                     -- JSON: the action taken
    confidence_score  Float32,                    -- 0.0 - 1.0 model confidence
    model_name        LowCardinality(String),     -- llama-3.1-70b, phi-3-mini, xgboost-malware, etc.
    model_version     String,
    latency_ms        UInt32,                     -- Inference latency
    was_overridden    UInt8 DEFAULT 0,            -- 1 if analyst overrode the decision
    override_reason   Nullable(String),
    override_by       Nullable(String),           -- Analyst email
    blake3_hash       FixedString(64),            -- blake3(prev_hash || decision_json)
    prev_hash         FixedString(64),            -- Previous row's blake3_hash (chain)
    created_at        DateTime64(3) DEFAULT now64(3),
    event_date        Date DEFAULT toDate(created_at)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(event_date))
ORDER BY (tenant_id, agent_name, created_at)
TTL event_date + INTERVAL 730 DAY
SETTINGS index_granularity = 8192;

-- Materialized view: decision counts per agent per hour
CREATE MATERIALIZED VIEW IF NOT EXISTS kubric.agent_decision_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, agent_name, decision_type, hour)
AS SELECT
    tenant_id,
    agent_name,
    decision_type,
    toStartOfHour(created_at) AS hour,
    count() AS decision_count,
    avg(confidence_score) AS avg_confidence,
    avg(latency_ms) AS avg_latency_ms,
    countIf(was_overridden = 1) AS override_count
FROM kubric.agent_decision_history
GROUP BY tenant_id, agent_name, decision_type, hour;

-- Index for fast lookup by event_id
ALTER TABLE kubric.agent_decision_history
    ADD INDEX idx_input_event input_event_id TYPE bloom_filter GRANULARITY 4;

-- Index for override audit queries
ALTER TABLE kubric.agent_decision_history
    ADD INDEX idx_overridden was_overridden TYPE set(2) GRANULARITY 1;
