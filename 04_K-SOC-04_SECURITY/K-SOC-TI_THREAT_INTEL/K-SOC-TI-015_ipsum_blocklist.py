"""
K-SOC-TI-015 — IPsum blocklist publisher.

Downloads the IPsum tiered blocklist from GitHub, filters by configured
minimum threat level, upserts to PostgreSQL, and publishes batches to NATS
for enforcement by downstream firewalls / EDR agents.

Env vars:
    IOC_DB_URL           asyncpg DSN (required)
    NATS_URL             nats:// URL (default: nats://localhost:4222)
    TENANT_ID            tenant identifier (default: default)
    IPSUM_MIN_LEVEL      minimum threat level to block 1-10 (default: 5)

Schema:
    ip_blocklist (
        ip        TEXT NOT NULL,
        source    TEXT NOT NULL,
        level     INT,
        added_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
        PRIMARY KEY (ip, source)
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import signal
import sys
from typing import Any

import asyncpg
import httpx
import nats as nats_client
import structlog

log = structlog.get_logger(__name__)

IPSUM_URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt"
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
TENANT_ID = os.getenv("TENANT_ID", "default")
IPSUM_MIN_LEVEL = int(os.getenv("IPSUM_MIN_LEVEL", "5"))
REFRESH_INTERVAL = 3600  # 1 hour

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS ip_blocklist (
    ip        TEXT NOT NULL,
    source    TEXT NOT NULL,
    level     INT,
    added_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ip, source)
);
"""

UPSERT_SQL = """
INSERT INTO ip_blocklist (ip, source, level, added_at)
VALUES ($1, 'ipsum', $2, now())
ON CONFLICT (ip, source) DO UPDATE
    SET level    = EXCLUDED.level,
        added_at = now();
"""


def parse_ipsum(text: str, min_level: int) -> list[dict[str, Any]]:
    """
    Parse IPsum blocklist text.

    Each non-comment line is: <ip>\\t<level>
    Returns list of {"ip": str, "level": int} for entries meeting min_level.
    """
    entries: list[dict[str, Any]] = []
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        if len(parts) < 2:
            continue
        ip = parts[0].strip()
        try:
            level = int(parts[1].strip())
        except ValueError:
            continue
        if level >= min_level:
            entries.append({"ip": ip, "level": level})
    return entries


class IPsumFeed:
    """Downloads IPsum, upserts to PostgreSQL, and publishes to NATS."""

    def __init__(self, db_pool: asyncpg.Pool, nc: Any) -> None:
        self._pool = db_pool
        self._nc = nc

    async def refresh_once(self) -> int:
        """Download, parse, upsert, and publish one full cycle."""
        async with httpx.AsyncClient(timeout=httpx.Timeout(60.0)) as client:
            resp = await client.get(IPSUM_URL)
            resp.raise_for_status()
            text = resp.text

        entries = parse_ipsum(text, IPSUM_MIN_LEVEL)
        log.info("ipsum.parsed", total_blocking=len(entries), min_level=IPSUM_MIN_LEVEL)

        # Upsert to DB.
        async with self._pool.acquire() as conn:
            for entry in entries:
                await conn.execute(UPSERT_SQL, entry["ip"], entry["level"])

        # Publish batches of 500 to NATS.
        subject = f"kubric.{TENANT_ID}.blocklist.ip.v1"
        batch_size = 500
        for i in range(0, len(entries), batch_size):
            batch = entries[i : i + batch_size]
            payload = json.dumps(batch).encode()
            await self._nc.publish(subject, payload)

        log.info("ipsum.published", subject=subject, batches=(len(entries) + batch_size - 1) // batch_size)
        return len(entries)

    async def run_forever(self) -> None:
        """Run the refresh loop until SIGINT/SIGTERM."""
        shutdown = asyncio.Event()
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, shutdown.set)

        while not shutdown.is_set():
            try:
                count = await self.refresh_once()
                log.info("ipsum.cycle_complete", ips_published=count)
            except Exception as exc:  # noqa: BLE001
                log.error("ipsum.cycle_error", error=str(exc))
            try:
                await asyncio.wait_for(shutdown.wait(), timeout=float(REFRESH_INTERVAL))
            except asyncio.TimeoutError:
                pass

        log.info("ipsum.stopped")


async def main() -> None:
    if not IOC_DB_URL:
        log.error("ipsum.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_SQL)

    nc = await nats_client.connect(NATS_URL)
    feed = IPsumFeed(pool, nc)
    await feed.run_forever()
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
