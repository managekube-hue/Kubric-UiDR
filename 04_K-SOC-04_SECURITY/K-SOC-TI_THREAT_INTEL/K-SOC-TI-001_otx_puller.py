"""
K-SOC-TI-001 — AlienVault OTX threat intelligence puller.

Pulls IOC pulses from OTX API and upserts them into the ioc_feeds table,
then publishes each IOC to NATS for downstream enforcement.

Env vars:
    OTX_API_KEY            AlienVault OTX API key (required)
    IOC_DB_URL             asyncpg DSN (required)
    NATS_URL               nats:// connection string (default: nats://localhost:4222)
    PULL_INTERVAL_SECONDS  seconds between runs (default: 900)

Schema (PostgreSQL):
    ioc_feeds (
        id          BIGSERIAL PRIMARY KEY,
        tenant_id   TEXT NOT NULL,
        ioc_type    TEXT NOT NULL,
        value       TEXT NOT NULL,
        confidence  INT,
        source      TEXT,
        tlp         TEXT,
        tags        TEXT[],
        first_seen  TIMESTAMPTZ,
        last_seen   TIMESTAMPTZ,
        UNIQUE (tenant_id, ioc_type, value, source)
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import signal
import sys
import time
from typing import Any

import asyncpg
import httpx
import nats as nats_client
import structlog

log = structlog.get_logger(__name__)

OTX_API_KEY = os.getenv("OTX_API_KEY", "")
OTX_BASE = "https://otx.alienvault.com/api/v1"
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
PULL_INTERVAL = int(os.getenv("PULL_INTERVAL_SECONDS", "900"))
TENANT_ID = os.getenv("TENANT_ID", "default")

# IOC types we extract from OTX.
WANTED_TYPES = {"IPv4", "domain", "URL", "FileHash-SHA256", "email"}

CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS ioc_feeds (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    ioc_type    TEXT NOT NULL,
    value       TEXT NOT NULL,
    confidence  INT,
    source      TEXT,
    tlp         TEXT,
    tags        TEXT[],
    first_seen  TIMESTAMPTZ,
    last_seen   TIMESTAMPTZ,
    UNIQUE (tenant_id, ioc_type, value, source)
);
"""

UPSERT_SQL = """
INSERT INTO ioc_feeds (tenant_id, ioc_type, value, confidence, source, tlp, tags, first_seen, last_seen)
VALUES ($1, $2, $3, $4, 'otx', 'amber', $5, $6, now())
ON CONFLICT (tenant_id, ioc_type, value, source) DO UPDATE
    SET confidence = EXCLUDED.confidence,
        tags       = EXCLUDED.tags,
        last_seen  = now();
"""


async def _fetch_pulses(client: httpx.AsyncClient, page: int) -> dict[str, Any]:
    """Fetch one page of subscribed OTX pulses."""
    headers = {"X-OTX-API-KEY": OTX_API_KEY}
    resp = await client.get(
        f"{OTX_BASE}/pulses/subscribed",
        headers=headers,
        params={"limit": 50, "page": page},
    )
    resp.raise_for_status()
    return resp.json()


async def _pull_with_backoff(
    client: httpx.AsyncClient, page: int, attempt: int = 0
) -> dict[str, Any]:
    """Retry with exponential backoff on rate-limit (429) or 503."""
    try:
        return await _fetch_pulses(client, page)
    except httpx.HTTPStatusError as exc:
        if exc.response.status_code in (429, 503) and attempt < 5:
            backoff = min(2 ** attempt * 2, 60)
            log.warning("otx_puller.rate_limited", page=page, retry_in=backoff)
            await asyncio.sleep(backoff)
            return await _pull_with_backoff(client, page, attempt + 1)
        raise


def _extract_indicators(pulse: dict[str, Any]) -> list[dict[str, Any]]:
    """Return a flat list of IOC dicts for wanted indicator types."""
    results = []
    created = pulse.get("created", "")
    tags = [t for t in (pulse.get("tags") or []) if isinstance(t, str)]
    for ind in pulse.get("indicators") or []:
        ind_type = ind.get("type", "")
        if ind_type not in WANTED_TYPES:
            continue
        results.append({
            "ioc_type": ind_type,
            "value": ind.get("indicator", ""),
            "confidence": int(ind.get("pulse_key_confidence") or 50),
            "tags": tags,
            "first_seen": created or None,
        })
    return results


async def pull_once(db_pool: asyncpg.Pool, nc: Any) -> int:
    """One full pull cycle. Returns total IOCs upserted."""
    if not OTX_API_KEY:
        log.error("otx_puller.missing_api_key")
        return 0

    total = 0
    limits = httpx.Limits(max_connections=5)
    timeout = httpx.Timeout(30.0)

    async with httpx.AsyncClient(limits=limits, timeout=timeout) as client:
        page = 1
        while True:
            data = await _pull_with_backoff(client, page)
            results = data.get("results", [])
            if not results:
                break

            indicators: list[dict[str, Any]] = []
            for pulse in results:
                indicators.extend(_extract_indicators(pulse))

            async with db_pool.acquire() as conn:
                for ioc in indicators:
                    if not ioc["value"]:
                        continue
                    await conn.execute(
                        UPSERT_SQL,
                        TENANT_ID,
                        ioc["ioc_type"],
                        ioc["value"],
                        ioc["confidence"],
                        ioc["tags"],
                        ioc["first_seen"],
                    )
                    # Publish to NATS.
                    subject = f"kubric.{TENANT_ID}.ti.ioc.v1"
                    payload = json.dumps({
                        "tenant_id": TENANT_ID,
                        "ioc_type": ioc["ioc_type"],
                        "value": ioc["value"],
                        "confidence": ioc["confidence"],
                        "source": "otx",
                        "tlp": "amber",
                        "tags": ioc["tags"],
                    }).encode()
                    await nc.publish(subject, payload)
                    total += 1

            next_page = data.get("next")
            if not next_page:
                break
            page += 1
            await asyncio.sleep(1.0)  # 1 req/s rate limit

    log.info("otx_puller.cycle_done", iocs_upserted=total)
    return total


async def main() -> None:
    if not OTX_API_KEY:
        log.error("otx_puller.missing_env", var="OTX_API_KEY")
        sys.exit(1)
    if not IOC_DB_URL:
        log.error("otx_puller.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_TABLE_SQL)

    nc = await nats_client.connect(NATS_URL)
    log.info("otx_puller.started", interval=PULL_INTERVAL)

    shutdown = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, shutdown.set)

    while not shutdown.is_set():
        try:
            await pull_once(pool, nc)
        except Exception as exc:  # noqa: BLE001
            log.error("otx_puller.error", error=str(exc))
        try:
            await asyncio.wait_for(shutdown.wait(), timeout=float(PULL_INTERVAL))
        except asyncio.TimeoutError:
            pass

    await nc.close()
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
