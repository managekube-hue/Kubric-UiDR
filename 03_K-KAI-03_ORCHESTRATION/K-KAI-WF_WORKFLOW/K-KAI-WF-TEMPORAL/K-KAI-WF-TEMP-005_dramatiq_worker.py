"""
K-KAI Workflow: Dramatiq Worker
Defines Dramatiq actors for asynchronous security automation tasks:
- alert ingestion, IOC enrichment, patch scheduling, report generation.
Uses Redis broker and PostgreSQL result backend.
"""
from __future__ import annotations
import json, logging, os
from datetime import datetime, timezone
from typing import Any, Dict

import dramatiq
from dramatiq.brokers.redis import RedisBroker
from dramatiq.middleware import AgeLimit, Retries, TimeLimit

logger = logging.getLogger(__name__)

REDIS_URL = os.getenv("REDIS_URL",  "redis://localhost:6379/1")
PG_DSN    = os.getenv("PG_DSN",    "postgresql://kubric:kubric@localhost:5432/kubric")

# ── broker setup ─────────────────────────────────────────────────
broker = RedisBroker(url=REDIS_URL, middleware=[
    AgeLimit(),
    TimeLimit(),
    Retries(max_retries=3, min_backoff=1_000, max_backoff=60_000),
])
dramatiq.set_broker(broker)

# ── actors ────────────────────────────────────────────────────────

@dramatiq.actor(queue_name="alerts", max_retries=3, time_limit=60_000)
def ingest_alert(alert_json: str) -> None:
    """Persist a raw OCSF alert JSON to the database."""
    import psycopg2, psycopg2.extras
    alert = json.loads(alert_json)
    conn  = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """INSERT INTO kai_alerts (tenant_id, class_uid, severity_id, src_ip, raw, created_at)
                   VALUES (%s, %s, %s, %s, %s, %s)
                   ON CONFLICT DO NOTHING""",
                (
                    alert.get("metadata", {}).get("tenant_id"),
                    alert.get("class_uid"),
                    alert.get("severity_id"),
                    alert.get("src_endpoint", {}).get("ip"),
                    json.dumps(alert),
                    datetime.now(timezone.utc),
                ),
            )
        conn.commit()
    finally:
        conn.close()
    logger.info("Ingested alert class_uid=%s", alert.get("class_uid"))

@dramatiq.actor(queue_name="enrichment", max_retries=5, time_limit=120_000)
def enrich_ioc(ioc: str, ioc_type: str, tenant_id: str) -> None:
    """Enrich an IOC (IP, domain, hash) via VirusTotal and store results."""
    import httpx
    vt_key = os.getenv("VIRUSTOTAL_API_KEY", "")
    if not vt_key:
        logger.warning("VIRUSTOTAL_API_KEY not set, skipping enrichment for %s", ioc)
        return

    endpoint_map = {
        "ip":     f"https://www.virustotal.com/api/v3/ip_addresses/{ioc}",
        "domain": f"https://www.virustotal.com/api/v3/domains/{ioc}",
        "hash":   f"https://www.virustotal.com/api/v3/files/{ioc}",
    }
    url = endpoint_map.get(ioc_type)
    if not url:
        logger.error("Unknown IOC type: %s", ioc_type)
        return

    resp = httpx.get(url, headers={"x-apikey": vt_key}, timeout=30)
    if resp.status_code == 404:
        logger.info("IOC %s not found in VT", ioc)
        return
    resp.raise_for_status()
    data = resp.json().get("data", {}).get("attributes", {})

    import psycopg2
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """INSERT INTO kai_ioc_enrichments
                   (ioc, ioc_type, tenant_id, malicious, suspicious, harmless, source, raw, updated_at)
                   VALUES (%s, %s, %s, %s, %s, %s, 'virustotal', %s, %s)
                   ON CONFLICT (ioc, ioc_type) DO UPDATE
                     SET malicious=%s, suspicious=%s, updated_at=%s""",
                (
                    ioc, ioc_type, tenant_id,
                    data.get("last_analysis_stats", {}).get("malicious", 0),
                    data.get("last_analysis_stats", {}).get("suspicious", 0),
                    data.get("last_analysis_stats", {}).get("harmless", 0),
                    json.dumps(data), datetime.now(timezone.utc),
                    data.get("last_analysis_stats", {}).get("malicious", 0),
                    data.get("last_analysis_stats", {}).get("suspicious", 0),
                    datetime.now(timezone.utc),
                ),
            )
        conn.commit()
    finally:
        conn.close()

@dramatiq.actor(queue_name="patching", max_retries=2, time_limit=300_000)
def schedule_patch(tenant_id: str, cve: str, agent_ids: str) -> None:
    """Schedule a patch job for a CVE across the specified agents."""
    import psycopg2
    agents = json.loads(agent_ids)
    conn   = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            for agent_id in agents:
                cur.execute(
                    """INSERT INTO kai_patch_jobs (tenant_id, cve_id, agent_id, status, created_at)
                       VALUES (%s, %s, %s, 'pending', %s)
                       ON CONFLICT DO NOTHING""",
                    (tenant_id, cve, agent_id, datetime.now(timezone.utc)),
                )
        conn.commit()
    finally:
        conn.close()
    logger.info("Scheduled patch for %s on %d agents", cve, len(agents))

@dramatiq.actor(queue_name="reports", max_retries=1, time_limit=600_000)
def generate_report(tenant_id: str, report_type: str, params_json: str) -> None:
    """Generate a security report (PDF/JSON) and store the result."""
    import psycopg2
    params = json.loads(params_json)
    report = {
        "tenant_id":   tenant_id,
        "report_type": report_type,
        "params":      params,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "status":      "completed",
        "data":        {"placeholder": "report data would be here"},
    }
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """INSERT INTO kai_reports (tenant_id, report_type, data, created_at)
                   VALUES (%s, %s, %s, %s)""",
                (tenant_id, report_type, json.dumps(report), datetime.now(timezone.utc)),
            )
        conn.commit()
    finally:
        conn.close()
    logger.info("Report generated: %s for tenant %s", report_type, tenant_id)

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    logger.info("Starting Dramatiq worker -- queues: alerts, enrichment, patching, reports")
    broker.emit_after("process_boot")
