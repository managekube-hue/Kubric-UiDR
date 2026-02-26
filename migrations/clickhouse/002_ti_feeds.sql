-- =============================================================================
-- ClickHouse Migration 002 — TI Feed tables (L3-4)
-- =============================================================================
-- Adds three tables consumed by kai/intel/ti_feeds.py:
--   kubric.nvd_cve          — raw NVD CVE dataset (hourly)
--   kubric.epss_scores      — FIRST EPSS probability scores (daily)
--   kubric.ti_indicators    — flat IOC table (OTX, AbuseIPDB, IPSum, MISP)
-- =============================================================================

-- ── NVD CVE (raw dataset) ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS kubric.nvd_cve
(
    cve_id       String                  NOT NULL,
    cvss_score   Float32                 NOT NULL DEFAULT 0.0,
    published_at DateTime64(0, 'UTC')    NOT NULL,
    description  String                  NOT NULL DEFAULT '',
    source       LowCardinality(String)  NOT NULL DEFAULT 'nvd',
    in_kev       UInt8                   NOT NULL DEFAULT 0,
    tenant_id    LowCardinality(String)  NOT NULL,
    fetched_at   DateTime64(0, 'UTC')    NOT NULL
)
ENGINE = ReplacingMergeTree(fetched_at)
PARTITION BY toYYYYMM(published_at)
ORDER BY (cve_id, tenant_id)
SETTINGS index_granularity = 8192;

-- ── FIRST EPSS scores ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS kubric.epss_scores
(
    cve_id      String                  NOT NULL,
    epss_score  Float32                 NOT NULL DEFAULT 0.0,
    percentile  Float32                 NOT NULL DEFAULT 0.0,
    fetched_at  DateTime64(0, 'UTC')    NOT NULL,
    tenant_id   LowCardinality(String)  NOT NULL
)
ENGINE = ReplacingMergeTree(fetched_at)
ORDER BY (cve_id, tenant_id)
SETTINGS index_granularity = 8192;

-- ── TI Indicators (flat IOC table) ─────────────────────────────────────────
-- Designed for high-throughput bulk inserts from OTX, AbuseIPDB, IPSum, MISP.
-- Distinct from kubric.threat_intel which is used for structured STIX/TAXII.
CREATE TABLE IF NOT EXISTS kubric.ti_indicators
(
    ioc_type    LowCardinality(String)  NOT NULL,  -- ip, domain, hash, url, email, etc.
    ioc_value   String                  NOT NULL,
    source      LowCardinality(String)  NOT NULL,  -- otx, abuseipdb, ipsum, misp
    confidence  UInt8                   NOT NULL DEFAULT 50,  -- 0-100
    tenant_id   LowCardinality(String)  NOT NULL,
    fetched_at  DateTime64(0, 'UTC')    NOT NULL,
    tags        String                  NOT NULL DEFAULT ''   -- comma-separated
)
ENGINE = MergeTree()
PARTITION BY (source, toYYYYMM(fetched_at))
ORDER BY (ioc_type, ioc_value, source, fetched_at)
TTL toDateTime(fetched_at) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
