"""
K-SOC-TI-002 — AbuseIPDB IP reputation checker.

Checks IP addresses against the AbuseIPDB v2 API and caches results in
PostgreSQL.  Subscribes to NATS flow events and publishes high-confidence
IPs as security alerts.

Env vars:
    ABUSEIPDB_API_KEY  AbuseIPDB API key (required)
    IOC_DB_URL         asyncpg DSN (required)
    NATS_URL           nats:// URL (default: nats://localhost:4222)
    TENANT_ID          tenant identifier (default: default)

Schema:
    ioc_abuse_cache (
        ip               TEXT PRIMARY KEY,
        abuse_score      INT,
        is_public        BOOLEAN,
        usage_type       TEXT,
        isp              TEXT,
        domain           TEXT,
        country_code     TEXT,
        total_reports    INT,
        queried_at       TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import time
from dataclasses import dataclass
from typing import Any

import asyncpg
import httpx
import nats as nats_client
import structlog

log = structlog.get_logger(__name__)

ABUSEIPDB_API_KEY = os.getenv("ABUSEIPDB_API_KEY", "")
ABUSEIPDB_BASE = "https://api.abuseipdb.com/api/v2"
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
TENANT_ID = os.getenv("TENANT_ID", "default")
ALERT_THRESHOLD = 75  # abuse score >= this triggers an alert

CREATE_CACHE_SQL = """
CREATE TABLE IF NOT EXISTS ioc_abuse_cache (
    ip            TEXT PRIMARY KEY,
    abuse_score   INT,
    is_public     BOOLEAN,
    usage_type    TEXT,
    isp           TEXT,
    domain        TEXT,
    country_code  TEXT,
    total_reports INT,
    queried_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_CACHE_SQL = """
INSERT INTO ioc_abuse_cache
    (ip, abuse_score, is_public, usage_type, isp, domain, country_code, total_reports, queried_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (ip) DO UPDATE
    SET abuse_score   = EXCLUDED.abuse_score,
        is_public     = EXCLUDED.is_public,
        usage_type    = EXCLUDED.usage_type,
        isp           = EXCLUDED.isp,
        domain        = EXCLUDED.domain,
        country_code  = EXCLUDED.country_code,
        total_reports = EXCLUDED.total_reports,
        queried_at    = now();
"""

GET_CACHE_SQL = """
SELECT ip, abuse_score, is_public, usage_type, isp, domain, country_code, total_reports, queried_at
FROM ioc_abuse_cache
WHERE ip = $1 AND queried_at > now() - INTERVAL '1 hour';
"""


@dataclass
class AbuseResult:
    ip: str
    abuse_confidence_score: int
    is_public: bool
    usage_type: str
    isp: str
    domain: str
    country_code: str
    total_reports: int


class AbuseIPDBClient:
    """Async client for AbuseIPDB v2, with PostgreSQL result cache."""

    def __init__(self, db_pool: asyncpg.Pool) -> None:
        self._pool = db_pool
        self._http = httpx.AsyncClient(
            timeout=httpx.Timeout(15.0),
            headers={
                "Key": ABUSEIPDB_API_KEY,
                "Accept": "application/json",
            },
        )

    async def close(self) -> None:
        await self._http.aclose()

    async def check_ip(self, ip: str) -> AbuseResult:
        """Check a single IP. Returns cached result if under 1 hour old."""
        async with self._pool.acquire() as conn:
            cached = await conn.fetchrow(GET_CACHE_SQL, ip)
        if cached:
            return AbuseResult(
                ip=cached["ip"],
                abuse_confidence_score=cached["abuse_score"],
                is_public=cached["is_public"],
                usage_type=cached["usage_type"] or "",
                isp=cached["isp"] or "",
                domain=cached["domain"] or "",
                country_code=cached["country_code"] or "",
                total_reports=cached["total_reports"] or 0,
            )
        resp = await self._http.get(
            f"{ABUSEIPDB_BASE}/check",
            params={"ipAddress": ip, "maxAgeInDays": 90, "verbose": ""},
        )
        resp.raise_for_status()
        d = resp.json().get("data", {})
        result = AbuseResult(
            ip=ip,
            abuse_confidence_score=d.get("abuseConfidenceScore", 0),
            is_public=d.get("isPublic", False),
            usage_type=d.get("usageType", "") or "",
            isp=d.get("isp", "") or "",
            domain=d.get("domain", "") or "",
            country_code=d.get("countryCode", "") or "",
            total_reports=d.get("totalReports", 0),
        )
        async with self._pool.acquire() as conn:
            await conn.execute(
                UPSERT_CACHE_SQL,
                result.ip,
                result.abuse_confidence_score,
                result.is_public,
                result.usage_type,
                result.isp,
                result.domain,
                result.country_code,
                result.total_reports,
            )
        return result

    async def batch_check(self, ips: list[str]) -> list[AbuseResult]:
        """Concurrently check multiple IPs."""
        return list(await asyncio.gather(*[self.check_ip(ip) for ip in ips]))

    async def report_ip(self, ip: str, categories: list[int], comment: str) -> dict[str, Any]:
        """Report an abusive IP to AbuseIPDB."""
        resp = await self._http.post(
            f"{ABUSEIPDB_BASE}/report",
            data={
                "ip": ip,
                "categories": ",".join(str(c) for c in categories),
                "comment": comment,
            },
        )
        resp.raise_for_status()
        return resp.json()


async def enrich_nats_flow_events(db_pool: asyncpg.Pool) -> None:
    """
    Subscribe to kubric.*.ndr.flow.v1 and check the src_ip in each message.
    Publish an alert to kubric.*.security.alert.v1 if abuse_score >= ALERT_THRESHOLD.
    """
    nc = await nats_client.connect(NATS_URL)
    client = AbuseIPDBClient(db_pool)
    log.info("abuseipdb.subscriber_started")

    async def _handler(msg: Any) -> None:
        try:
            event = json.loads(msg.data.decode())
            src_ip = event.get("src_ip", "")
            if not src_ip:
                return
            result = await client.check_ip(src_ip)
            if result.abuse_confidence_score >= ALERT_THRESHOLD:
                alert_subject = f"kubric.{TENANT_ID}.security.alert.v1"
                alert = {
                    "tenant_id": TENANT_ID,
                    "alert_type": "malicious_ip",
                    "src_ip": src_ip,
                    "abuse_score": result.abuse_confidence_score,
                    "isp": result.isp,
                    "country_code": result.country_code,
                    "source": "abuseipdb",
                }
                await nc.publish(alert_subject, json.dumps(alert).encode())
                log.info("abuseipdb.alert_published", ip=src_ip, score=result.abuse_confidence_score)
        except Exception as exc:  # noqa: BLE001
            log.warning("abuseipdb.handler_error", error=str(exc))

    sub_subject = f"kubric.{TENANT_ID}.ndr.flow.v1"
    await nc.subscribe(sub_subject, cb=_handler)
    log.info("abuseipdb.subscribed", subject=sub_subject)

    try:
        while True:
            await asyncio.sleep(60)
    except (asyncio.CancelledError, KeyboardInterrupt):
        pass
    finally:
        await client.close()
        await nc.close()


async def main() -> None:
    if not ABUSEIPDB_API_KEY:
        log.error("abuseipdb.missing_env", var="ABUSEIPDB_API_KEY")
        sys.exit(1)
    if not IOC_DB_URL:
        log.error("abuseipdb.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_CACHE_SQL)

    await enrich_nats_flow_events(pool)
    await pool.close()


if __name__ == "__main__":
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )
    asyncio.run(main())
