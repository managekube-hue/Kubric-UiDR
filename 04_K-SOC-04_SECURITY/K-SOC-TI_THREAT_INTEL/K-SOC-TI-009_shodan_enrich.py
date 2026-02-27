"""
K-SOC-TI-009 — Shodan IP enrichment.

Enriches IP addresses with open-port, service, CVE, and OSINT data from the
Shodan API, caches results in PostgreSQL for 24 hours, and publishes CVE
findings to NATS.

Env vars:
    SHODAN_API_KEY  Shodan API key (required)
    IOC_DB_URL      asyncpg DSN (required)
    NATS_URL        nats:// URL (default: nats://localhost:4222)
    TENANT_ID       tenant identifier (default: default)

Schema:
    shodan_cache (
        ip          TEXT PRIMARY KEY,
        result_json JSONB,
        queried_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import time
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

SHODAN_API_KEY = os.getenv("SHODAN_API_KEY", "")
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
TENANT_ID = os.getenv("TENANT_ID", "default")

try:
    import shodan as shodan_sdk  # type: ignore[import]
    _HAS_SHODAN = True
except ImportError:
    _HAS_SHODAN = False
    shodan_sdk = None  # type: ignore[assignment]

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS shodan_cache (
    ip          TEXT PRIMARY KEY,
    result_json JSONB,
    queried_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_SQL = """
INSERT INTO shodan_cache (ip, result_json, queried_at)
VALUES ($1, $2, now())
ON CONFLICT (ip) DO UPDATE
    SET result_json = EXCLUDED.result_json,
        queried_at  = now();
"""

GET_SQL = """
SELECT result_json
FROM shodan_cache
WHERE ip = $1 AND queried_at > now() - INTERVAL '24 hours';
"""


@dataclass
class ShodanInfo:
    ip: str
    hostnames: list[str] = field(default_factory=list)
    ports: list[int] = field(default_factory=list)
    vulns: list[str] = field(default_factory=list)
    org: str = ""
    isp: str = ""
    country_code: str = ""
    tags: list[str] = field(default_factory=list)


class ShodanEnricher:
    """Async Shodan enricher with 24-hour PostgreSQL cache."""

    def __init__(self, db_pool: asyncpg.Pool) -> None:
        self._pool = db_pool
        self._rate_sem = asyncio.Semaphore(1)  # 1 req/s
        if not _HAS_SHODAN:
            log.warning("shodan_enrich.library_missing", hint="pip install shodan")
            self._api = None
        else:
            self._api = shodan_sdk.Shodan(SHODAN_API_KEY)

    async def enrich_ip(self, ip: str) -> ShodanInfo:
        """Enrich a single IP. Returns cached result if < 24 hours old."""
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(GET_SQL, ip)
        if row:
            return self._from_json(ip, json.loads(row["result_json"]))

        if not _HAS_SHODAN or not self._api:
            raise RuntimeError("shodan SDK not available")

        # Rate-limit: 1 req/s.
        async with self._rate_sem:
            loop = asyncio.get_event_loop()
            host_data = await loop.run_in_executor(None, self._api.host, ip)
            await asyncio.sleep(1.0)

        async with self._pool.acquire() as conn:
            await conn.execute(UPSERT_SQL, ip, json.dumps(host_data))

        info = self._from_json(ip, host_data)

        # Publish CVEs to NATS if found.
        if info.vulns:
            try:
                import nats as nats_client  # type: ignore[import]
                nc = await nats_client.connect(NATS_URL)
                for cve_id in info.vulns:
                    subject = f"kubric.{TENANT_ID}.ti.ioc.v1"
                    payload = json.dumps({
                        "tenant_id": TENANT_ID,
                        "ioc_type": "cve",
                        "value": cve_id,
                        "source": "shodan",
                        "ip": ip,
                    }).encode()
                    await nc.publish(subject, payload)
                await nc.close()
            except Exception as exc:  # noqa: BLE001
                log.warning("shodan_enrich.nats_error", error=str(exc))

        return info

    async def batch_enrich(self, ips: list[str]) -> dict[str, ShodanInfo]:
        """Enrich multiple IPs sequentially (Shodan rate limit)."""
        results: dict[str, ShodanInfo] = {}
        for ip in ips:
            try:
                results[ip] = await self.enrich_ip(ip)
            except Exception as exc:  # noqa: BLE001
                log.warning("shodan_enrich.ip_error", ip=ip, error=str(exc))
        return results

    async def scan_subnet(self, cidr: str) -> dict[str, Any]:
        """
        Count hosts matching a CIDR query then initiate a Shodan on-demand scan.
        Requires a paid Shodan plan.
        """
        if not _HAS_SHODAN or not self._api:
            raise RuntimeError("shodan SDK not available")
        loop = asyncio.get_event_loop()
        count = await loop.run_in_executor(None, self._api.count, f"net:{cidr}")
        scan = await loop.run_in_executor(None, self._api.scan, cidr)
        return {"cidr": cidr, "estimated_hosts": count, "scan": scan}

    @staticmethod
    def _from_json(ip: str, data: dict[str, Any]) -> ShodanInfo:
        vulns: list[str] = []
        for item in (data.get("vulns") or {}).keys():
            vulns.append(str(item))
        return ShodanInfo(
            ip=ip,
            hostnames=list(data.get("hostnames") or []),
            ports=list(data.get("ports") or []),
            vulns=vulns,
            org=data.get("org", "") or "",
            isp=data.get("isp", "") or "",
            country_code=data.get("country_code", "") or "",
            tags=list(data.get("tags") or []),
        )


async def main() -> None:
    if not SHODAN_API_KEY:
        log.error("shodan_enrich.missing_env", var="SHODAN_API_KEY")
        sys.exit(1)
    if not IOC_DB_URL:
        log.error("shodan_enrich.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_SQL)

    enricher = ShodanEnricher(pool)
    # Example: enrich a single IP for smoke-testing.
    test_ip = os.getenv("TEST_IP", "8.8.8.8")
    info = await enricher.enrich_ip(test_ip)
    log.info("shodan_enrich.result", ip=info.ip, ports=info.ports[:5], vulns=info.vulns[:5])
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
