"""
K-SOC-TI-011 — GreyNoise IP noise filter.

Filters out known internet noise/scanner IPs from security alerts using the
GreyNoise API.  Applies noise classification to reduce alert fatigue.

Env vars:
    GREYNOISE_API_KEY  GreyNoise API key (required)
    IOC_DB_URL         asyncpg DSN (required)
    NATS_URL           nats:// URL (default: nats://localhost:4222)
    TENANT_ID          tenant identifier (default: default)

Schema:
    greynoise_cache (
        ip           TEXT PRIMARY KEY,
        noise        BOOLEAN,
        riot         BOOLEAN,
        classification TEXT,
        name         TEXT,
        link         TEXT,
        last_seen    TEXT,
        tags         TEXT[],
        queried_at   TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import httpx
import structlog

log = structlog.get_logger(__name__)

GREYNOISE_API_KEY = os.getenv("GREYNOISE_API_KEY", "")
GREYNOISE_BASE = "https://api.greynoise.io"
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
TENANT_ID = os.getenv("TENANT_ID", "default")

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS greynoise_cache (
    ip             TEXT PRIMARY KEY,
    noise          BOOLEAN,
    riot           BOOLEAN,
    classification TEXT,
    name           TEXT,
    link           TEXT,
    last_seen      TEXT,
    tags           TEXT[],
    queried_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_SQL = """
INSERT INTO greynoise_cache
    (ip, noise, riot, classification, name, link, last_seen, tags, queried_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (ip) DO UPDATE
    SET noise          = EXCLUDED.noise,
        riot           = EXCLUDED.riot,
        classification = EXCLUDED.classification,
        name           = EXCLUDED.name,
        link           = EXCLUDED.link,
        last_seen      = EXCLUDED.last_seen,
        tags           = EXCLUDED.tags,
        queried_at     = now();
"""

GET_SQL = """
SELECT noise, riot, classification, name, link, last_seen, tags
FROM greynoise_cache
WHERE ip = $1 AND queried_at > now() - INTERVAL '1 hour';
"""


@dataclass
class GreyNoiseInfo:
    ip: str
    noise: bool
    riot: bool
    classification: str
    name: str
    link: str
    last_seen: str
    tags: list[str] = field(default_factory=list)


class GreyNoiseFilter:
    """Async GreyNoise IP classification with PostgreSQL cache."""

    def __init__(self, db_pool: asyncpg.Pool) -> None:
        self._pool = db_pool
        self._http = httpx.AsyncClient(
            timeout=httpx.Timeout(15.0),
            headers={"key": GREYNOISE_API_KEY, "Accept": "application/json"},
        )

    async def close(self) -> None:
        await self._http.aclose()

    async def is_noise(self, ip: str) -> bool:
        """Return True if the IP is classified as noise or riot by GreyNoise."""
        info = await self.enrich_ip(ip)
        return info.noise or info.riot

    async def bulk_quick(self, ips: list[str]) -> dict[str, bool]:
        """
        Bulk quick-check multiple IPs via POST /v2/noise/quick.
        Returns {ip: is_noise_bool}.
        """
        if not ips:
            return {}
        resp = await self._http.post(
            f"{GREYNOISE_BASE}/v2/noise/quick",
            json={"ips": ips},
        )
        resp.raise_for_status()
        data = resp.json()
        result: dict[str, bool] = {}
        for entry in data:
            ip_addr = entry.get("ip", "")
            result[ip_addr] = bool(entry.get("noise") or entry.get("riot"))
        return result

    async def enrich_ip(self, ip: str) -> GreyNoiseInfo:
        """Full context lookup for an IP. Returns cached result if < 1 hour old."""
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(GET_SQL, ip)
        if row:
            return GreyNoiseInfo(
                ip=ip,
                noise=bool(row["noise"]),
                riot=bool(row["riot"]),
                classification=row["classification"] or "",
                name=row["name"] or "",
                link=row["link"] or "",
                last_seen=row["last_seen"] or "",
                tags=list(row["tags"] or []),
            )

        # Community endpoint works without a paid key.
        resp = await self._http.get(f"{GREYNOISE_BASE}/v3/community/{ip}")
        if resp.status_code == 404:
            info = GreyNoiseInfo(ip=ip, noise=False, riot=False,
                                 classification="unknown", name="", link="", last_seen="")
        elif resp.status_code == 200:
            d = resp.json()
            info = GreyNoiseInfo(
                ip=ip,
                noise=bool(d.get("noise")),
                riot=bool(d.get("riot")),
                classification=d.get("classification", "") or "",
                name=d.get("name", "") or "",
                link=d.get("link", "") or "",
                last_seen=d.get("last_seen", "") or "",
                tags=list(d.get("tags") or []),
            )
        else:
            resp.raise_for_status()
            raise RuntimeError("unreachable")

        async with self._pool.acquire() as conn:
            await conn.execute(
                UPSERT_SQL,
                ip, info.noise, info.riot, info.classification,
                info.name, info.link, info.last_seen, info.tags,
            )
        return info

    def filter_alert(self, alert: dict[str, Any], info: GreyNoiseInfo) -> dict[str, Any]:
        """
        If the alert source IP is classified as noise or riot, downgrade
        its priority to 'low'.  Returns modified copy.
        """
        if info.noise or info.riot:
            updated = dict(alert)
            updated["priority"] = "low"
            updated["greynoise_noise"] = True
            updated["greynoise_classification"] = info.classification
            return updated
        return alert

    async def filter_alert_stream(self, tenant_id: str) -> None:
        """
        Poll new kai_alerts for tenant_id, apply GreyNoise noise filter,
        and update priority in-place.
        """
        select_sql = """
            SELECT id, src_ip, priority
            FROM kai_alerts
            WHERE tenant_id = $1
              AND greynoise_checked IS NULL
              AND src_ip IS NOT NULL
            LIMIT 200
        """
        update_sql = """
            UPDATE kai_alerts
            SET priority = $1, greynoise_checked = now()
            WHERE id = $2
        """
        log.info("greynoise.stream_filter_start", tenant_id=tenant_id)
        while True:
            try:
                async with self._pool.acquire() as conn:
                    rows = await conn.fetch(select_sql, tenant_id)
                for row in rows:
                    info = await self.enrich_ip(row["src_ip"])
                    new_priority = "low" if (info.noise or info.riot) else row["priority"]
                    async with self._pool.acquire() as conn:
                        await conn.execute(update_sql, new_priority, row["id"])
                    if info.noise or info.riot:
                        log.info("greynoise.noise_filtered", ip=row["src_ip"],
                                 classification=info.classification)
            except Exception as exc:  # noqa: BLE001
                log.error("greynoise.stream_error", error=str(exc))
            await asyncio.sleep(30)


async def main() -> None:
    if not GREYNOISE_API_KEY:
        log.error("greynoise.missing_env", var="GREYNOISE_API_KEY")
        sys.exit(1)
    if not IOC_DB_URL:
        log.error("greynoise.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_SQL)

    gn = GreyNoiseFilter(pool)
    await gn.filter_alert_stream(TENANT_ID)
    await gn.close()
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
