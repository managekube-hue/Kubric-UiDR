"""
K-SOC-DET-002 — Sigma rule sync worker.

Pulls the full Sigma rule tree from SigmaHQ GitHub, downloads each YAML rule,
parses it with PyYAML, and upserts into the sigma_rules PostgreSQL table.

Schema (PostgreSQL):
    sigma_rules (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        rule_id     TEXT UNIQUE NOT NULL,
        title       TEXT,
        logsource   JSONB,
        detection   JSONB,
        tags        TEXT[],
        raw_yaml    TEXT,
        synced_at   TIMESTAMPTZ NOT NULL DEFAULT now()
    )

Env vars:
    SIGMA_DB_URL            asyncpg DSN (required)
    GITHUB_TOKEN            optional — raises rate limit from 60 to 5000 req/hr
    SYNC_INTERVAL_SECONDS   seconds between syncs (default: 3600)
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
import structlog
import yaml

log = structlog.get_logger(__name__)

GITHUB_TREE_URL = (
    "https://api.github.com/repos/SigmaHQ/sigma/git/trees/master?recursive=1"
)
GITHUB_RAW_BASE = "https://raw.githubusercontent.com/SigmaHQ/sigma/master/"
SIGMA_DB_URL = os.getenv("SIGMA_DB_URL", "")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN", "")
SYNC_INTERVAL = int(os.getenv("SYNC_INTERVAL_SECONDS", "3600"))

CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS sigma_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id     TEXT UNIQUE NOT NULL,
    title       TEXT,
    logsource   JSONB,
    detection   JSONB,
    tags        TEXT[],
    raw_yaml    TEXT,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_SQL = """
INSERT INTO sigma_rules (rule_id, title, logsource, detection, tags, raw_yaml, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (rule_id) DO UPDATE
    SET title     = EXCLUDED.title,
        logsource = EXCLUDED.logsource,
        detection = EXCLUDED.detection,
        tags      = EXCLUDED.tags,
        raw_yaml  = EXCLUDED.raw_yaml,
        synced_at = now();
"""


def _build_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Accept": "application/vnd.github.v3+json"}
    if GITHUB_TOKEN:
        headers["Authorization"] = f"token {GITHUB_TOKEN}"
    return headers


async def fetch_rule_paths(client: httpx.AsyncClient) -> list[str]:
    """Return all rules/**/*.yml paths from the SigmaHQ master tree."""
    resp = await client.get(GITHUB_TREE_URL, headers=_build_headers())
    resp.raise_for_status()
    tree = resp.json().get("tree", [])
    return [
        item["path"]
        for item in tree
        if item["type"] == "blob"
        and item["path"].startswith("rules/")
        and item["path"].endswith(".yml")
    ]


async def download_rule(client: httpx.AsyncClient, path: str) -> str | None:
    """Download raw YAML for a single rule. Returns None on HTTP error."""
    url = GITHUB_RAW_BASE + path
    try:
        resp = await client.get(url, headers=_build_headers())
        resp.raise_for_status()
        return resp.text
    except httpx.HTTPStatusError as exc:
        log.warning("sigma_sync.download_failed", path=path, status=exc.response.status_code)
        return None


def parse_sigma_yaml(raw: str) -> dict[str, Any] | None:
    """Parse a Sigma YAML string. Returns None if the file is not a valid rule."""
    try:
        doc = yaml.safe_load(raw)
    except yaml.YAMLError as exc:
        log.warning("sigma_sync.yaml_parse_error", error=str(exc))
        return None
    if not isinstance(doc, dict):
        return None
    return doc


async def upsert_rule(conn: asyncpg.Connection, raw: str, doc: dict[str, Any]) -> None:
    """Upsert one parsed Sigma rule into the sigma_rules table."""
    rule_id: str = str(doc.get("id", ""))
    title: str = str(doc.get("title", ""))
    logsource = doc.get("logsource") or {}
    detection = doc.get("detection") or {}
    tags: list[str] = [str(t) for t in (doc.get("tags") or [])]

    await conn.execute(
        UPSERT_SQL,
        rule_id or title,  # fallback to title if no uuid
        title,
        json.dumps(logsource),
        json.dumps(detection),
        tags,
        raw,
    )


async def sync_once(db_pool: asyncpg.Pool) -> int:
    """
    Run one full sync cycle.

    Returns the count of rules successfully upserted.
    """
    log.info("sigma_sync.start")

    limits = httpx.Limits(max_connections=20, max_keepalive_connections=10)
    timeout = httpx.Timeout(30.0)
    upserted = 0

    async with httpx.AsyncClient(limits=limits, timeout=timeout) as client:
        paths = await fetch_rule_paths(client)
        log.info("sigma_sync.paths_found", count=len(paths))

        # Download all rules concurrently in batches of 30 to avoid flooding
        # GitHub's API.
        batch_size = 30
        for batch_start in range(0, len(paths), batch_size):
            batch = paths[batch_start : batch_start + batch_size]
            raw_results = await asyncio.gather(
                *[download_rule(client, p) for p in batch], return_exceptions=False
            )

            async with db_pool.acquire() as conn:
                for raw in raw_results:
                    if raw is None:
                        continue
                    doc = parse_sigma_yaml(raw)
                    if doc is None:
                        continue
                    try:
                        await upsert_rule(conn, raw, doc)
                        upserted += 1
                    except Exception as exc:  # noqa: BLE001
                        log.warning("sigma_sync.upsert_error", error=str(exc))

            # Polite delay between batches to respect GitHub rate limits.
            await asyncio.sleep(0.5)

    log.info("sigma_sync.complete", upserted=upserted)
    return upserted


async def main() -> None:
    """Run the sync loop until interrupted."""
    if not SIGMA_DB_URL:
        log.error("sigma_sync.missing_env", var="SIGMA_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(SIGMA_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_TABLE_SQL)
    log.info("sigma_sync.db_ready")

    loop = asyncio.get_running_loop()
    shutdown = asyncio.Event()

    def _handle_signal() -> None:
        log.info("sigma_sync.shutdown_requested")
        shutdown.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _handle_signal)

    while not shutdown.is_set():
        try:
            await sync_once(pool)
        except Exception as exc:  # noqa: BLE001
            log.error("sigma_sync.cycle_error", error=str(exc))

        try:
            await asyncio.wait_for(shutdown.wait(), timeout=float(SYNC_INTERVAL))
        except asyncio.TimeoutError:
            pass  # Normal — interval elapsed, run next cycle.

    await pool.close()
    log.info("sigma_sync.stopped")


if __name__ == "__main__":
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )
    asyncio.run(main())
