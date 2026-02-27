"""
K-KAI Workflow: Huey Scheduler
Defines periodic maintenance tasks using Huey (Redis-backed task queue):
- EPSS score refresh, health score snapshots, stale session cleanup,
  drift detection sweeps, and certificate expiry checks.
"""
from __future__ import annotations
import json, logging, os
from datetime import datetime, timezone, timedelta
from typing import Optional

from huey import RedisHuey, crontab

logger = logging.getLogger(__name__)
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/2")
PG_DSN    = os.getenv("PG_DSN",   "postgresql://kubric:kubric@localhost:5432/kubric")

huey = RedisHuey("kubric-kai", url=REDIS_URL, store_none=False)

# ── periodic tasks ────────────────────────────────────────────────

@huey.periodic_task(crontab(hour="*/6", minute="0"))
def refresh_epss_scores() -> None:
    """Refresh EPSS exploitation scores every 6 hours."""
    import asyncio, sys, os
    sys.path.insert(0, os.path.dirname(__file__))
    try:
        from K_KAI_FS_002_epss_enrichment import EpssEnricher  # type: ignore
        async def _run():
            async with EpssEnricher(pg_dsn=PG_DSN) as enricher:
                return await enricher.run_enrichment()
        result = asyncio.run(_run())
        logger.info("EPSS refresh complete: %s", result)
    except ImportError:
        logger.warning("EpssEnricher not importable, skipping EPSS refresh")

@huey.periodic_task(crontab(minute="*/5"))
def snapshot_health_scores() -> None:
    """Snapshot current tenant health scores to history table every 5 minutes."""
    import psycopg2
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO kai_health_history (tenant_id, overall_score, security_score,
                    compliance_score, availability_score, snapped_at)
                SELECT tenant_id, overall_score, security_score,
                    compliance_score, availability_score, NOW()
                FROM kai_current_health
            """)
        conn.commit()
        logger.info("Health score snapshot taken at %s", datetime.now(timezone.utc).isoformat())
    except Exception as exc:
        logger.error("Health snapshot failed: %s", exc)
        conn.rollback()
    finally:
        conn.close()

@huey.periodic_task(crontab(hour="2", minute="0"))
def cleanup_stale_sessions() -> None:
    """Remove expired user sessions older than 30 days every night at 02:00."""
    import psycopg2
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor() as cur:
            cur.execute("""
                DELETE FROM kai_user_sessions
                WHERE last_active < NOW() - INTERVAL '30 days'
            """)
            deleted = cur.rowcount
        conn.commit()
        logger.info("Cleaned up %d stale sessions", deleted)
    except Exception as exc:
        logger.error("Session cleanup failed: %s", exc)
        conn.rollback()
    finally:
        conn.close()

@huey.periodic_task(crontab(minute="*/15"))
def drift_detection_sweep() -> None:
    """Sweep for configuration drift events every 15 minutes."""
    import psycopg2, psycopg2.extras
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
            # Identify agents that have drifted from baseline
            cur.execute("""
                SELECT a.id AS agent_id, a.tenant_id, a.hostname,
                       b.config_hash AS baseline_hash, a.last_config_hash AS current_hash
                FROM kai_agents a
                JOIN kai_baselines b ON b.agent_id = a.id
                WHERE a.last_config_hash != b.config_hash
                  AND a.is_online = TRUE
                LIMIT 200
            """)
            drifted = cur.fetchall()

            for row in drifted:
                cur.execute("""
                    INSERT INTO kai_drift_events (agent_id, tenant_id, drift_type,
                        config_hash_expected, config_hash_actual, detected_at)
                    VALUES (%s, %s, 'config_hash_mismatch', %s, %s, NOW())
                    ON CONFLICT DO NOTHING
                """, (row["agent_id"], row["tenant_id"],
                      row["baseline_hash"], row["current_hash"]))
        conn.commit()
        if drifted:
            logger.warning("Detected drift on %d agents", len(drifted))
    except Exception as exc:
        logger.error("Drift sweep failed: %s", exc)
        conn.rollback()
    finally:
        conn.close()

@huey.periodic_task(crontab(hour="1", minute="0"))
def check_certificate_expiry() -> None:
    """Check TLS certificate expiry for monitored domains daily at 01:00."""
    import ssl, socket, psycopg2, psycopg2.extras
    conn = psycopg2.connect(PG_DSN)
    try:
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
            cur.execute("SELECT id, tenant_id, hostname FROM kai_monitored_domains LIMIT 500")
            domains = cur.fetchall()

        for domain in domains:
            hostname = domain["hostname"]
            try:
                ctx   = ssl.create_default_context()
                with socket.create_connection((hostname, 443), timeout=5) as sock:
                    with ctx.wrap_socket(sock, server_hostname=hostname) as tls:
                        cert = tls.getpeercert()
                        not_after = datetime.strptime(cert["notAfter"], "%b %d %H:%M:%S %Y %Z")
                        days_left = (not_after - datetime.utcnow()).days
                        if days_left <= 30:
                            logger.warning("Cert for %s expires in %d days", hostname, days_left)
                            with conn.cursor() as cur2:
                                cur2.execute("""
                                    INSERT INTO kai_cert_alerts (tenant_id, hostname, expires_at, days_left, created_at)
                                    VALUES (%s, %s, %s, %s, NOW())
                                    ON CONFLICT (hostname) DO UPDATE
                                      SET expires_at=%s, days_left=%s
                                """, (domain["tenant_id"], hostname, not_after, days_left, not_after, days_left))
                            conn.commit()
            except Exception as exc:
                logger.debug("Cert check failed for %s: %s", hostname, exc)
    finally:
        conn.close()

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    logger.info("Huey scheduler starting -- Redis: %s", REDIS_URL)
    # Run consumer: python -m huey.bin.huey_consumer K_KAI_WF_TEMP_006_huey_scheduler.huey
