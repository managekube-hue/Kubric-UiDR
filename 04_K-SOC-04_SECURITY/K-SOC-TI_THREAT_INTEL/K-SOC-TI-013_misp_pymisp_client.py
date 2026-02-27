"""
K-SOC-TI-013 — PyMISP client for MISP threat sharing.

Full CRUD client for MISP events and attributes.  Syncs recent MISP events
to a local PostgreSQL table and exports event data as STIX2.

Env vars:
    MISP_URL     MISP instance URL (required)
    MISP_KEY     MISP API authentication key (required)
    MISP_DB_URL  asyncpg DSN for local MISP mirror (required)

Schema:
    misp_events (
        id           INT PRIMARY KEY,
        tenant_id    TEXT NOT NULL,
        uuid         TEXT,
        title        TEXT,
        threat_level INT,
        distribution INT,
        tags         TEXT[],
        raw_json     JSONB,
        synced_at    TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
from datetime import datetime, timedelta, timezone
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

MISP_URL = os.getenv("MISP_URL", "")
MISP_KEY = os.getenv("MISP_KEY", "")
MISP_DB_URL = os.getenv("MISP_DB_URL", "")
TENANT_ID = os.getenv("TENANT_ID", "default")

try:
    from pymisp import PyMISP, MISPEvent, MISPAttribute  # type: ignore[import]
    _HAS_PYMISP = True
except ImportError:
    _HAS_PYMISP = False
    PyMISP = None  # type: ignore[assignment]
    MISPEvent = None  # type: ignore[assignment]
    MISPAttribute = None  # type: ignore[assignment]

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS misp_events (
    id           INT PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    uuid         TEXT,
    title        TEXT,
    threat_level INT,
    distribution INT,
    tags         TEXT[],
    raw_json     JSONB,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_SQL = """
INSERT INTO misp_events
    (id, tenant_id, uuid, title, threat_level, distribution, tags, raw_json, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (id) DO UPDATE
    SET title        = EXCLUDED.title,
        threat_level = EXCLUDED.threat_level,
        tags         = EXCLUDED.tags,
        raw_json     = EXCLUDED.raw_json,
        synced_at    = now();
"""


class MISPClient:
    """Full PyMISP wrapper for MISP event and attribute operations."""

    def __init__(self) -> None:
        if not _HAS_PYMISP:
            raise RuntimeError("pymisp not installed: pip install pymisp")
        if not MISP_URL or not MISP_KEY:
            raise RuntimeError("MISP_URL and MISP_KEY env vars required")
        self._misp: Any = PyMISP(MISP_URL, MISP_KEY, ssl=True)

    def search_events(self, tags: list[str], from_date: str) -> list[dict[str, Any]]:
        """Search MISP events by tag list and from_date (YYYY-MM-DD)."""
        results = self._misp.search(
            controller="events",
            tags=tags,
            date_from=from_date,
            with_attachments=False,
            pythonify=False,
        )
        return results if isinstance(results, list) else []

    def get_event(self, event_id: int) -> dict[str, Any]:
        """Fetch a single MISP event by ID."""
        event = self._misp.get_event(event_id, pythonify=False)
        return dict(event)

    def create_event(self, title: str, threat_level: int, distribution: int) -> Any:
        """Create a new MISP event. Returns MISPEvent."""
        event: Any = MISPEvent()
        event.info = title
        event.threat_level_id = threat_level
        event.distribution = distribution
        result = self._misp.add_event(event, pythonify=True)
        log.info("misp.event_created", title=title, id=getattr(result, "id", None))
        return result

    def add_attribute(
        self, event_id: int, attr_type: str, value: str, category: str
    ) -> Any:
        """Add an attribute to an existing event. Returns MISPAttribute."""
        attr: Any = MISPAttribute()
        attr.type = attr_type
        attr.value = value
        attr.category = category
        result = self._misp.add_attribute(event_id, attr, pythonify=True)
        return result

    def add_ioc(self, event_id: int, ioc_type: str, value: str) -> Any:
        """Convenience method: add an IOC attribute with auto-detected category."""
        category_map = {
            "ip-src": "Network activity",
            "ip-dst": "Network activity",
            "domain": "Network activity",
            "url": "Network activity",
            "md5": "Payload delivery",
            "sha256": "Payload delivery",
            "sha1": "Payload delivery",
            "email-src": "Payload delivery",
            "email-dst": "Payload delivery",
        }
        category = category_map.get(ioc_type, "External analysis")
        return self.add_attribute(event_id, ioc_type, value, category)

    def push_sighting(self, value: str, source: str) -> None:
        """Push a sighting for a given IOC value."""
        self._misp.add_sighting({"value": value, "source": source})
        log.info("misp.sighting_pushed", value=value, source=source)

    async def export_to_stix(self, event_id: int) -> str:
        """Export a MISP event as a STIX2 JSON bundle string."""
        loop = asyncio.get_event_loop()
        data = await loop.run_in_executor(
            None,
            lambda: self._misp.get_event(event_id, pythonify=False),
        )
        # Request STIX2 formatted export via the MISP restSearch endpoint.
        stix_data = await loop.run_in_executor(
            None,
            lambda: self._misp.search(
                controller="events",
                eventid=str(event_id),
                return_format="stix2",
            ),
        )
        if isinstance(stix_data, bytes):
            return stix_data.decode()
        if isinstance(stix_data, str):
            return stix_data
        return json.dumps(stix_data)

    async def sync_to_local_db(self, db_pool: asyncpg.Pool) -> int:
        """Pull events from last 24h and upsert into misp_events table."""
        yesterday = (datetime.now(tz=timezone.utc) - timedelta(days=1)).strftime(
            "%Y-%m-%d"
        )
        loop = asyncio.get_event_loop()
        events = await loop.run_in_executor(
            None,
            lambda: self._misp.search(
                controller="events",
                date_from=yesterday,
                with_attachments=False,
                pythonify=False,
            ),
        )
        if not isinstance(events, list):
            return 0

        upserted = 0
        async with db_pool.acquire() as conn:
            await conn.execute(CREATE_SQL)
            for ev in events:
                e = ev.get("Event") or ev
                event_id = int(e.get("id", 0))
                if not event_id:
                    continue
                tags = [t.get("name", "") for t in (e.get("Tag") or [])]
                await conn.execute(
                    UPSERT_SQL,
                    event_id,
                    TENANT_ID,
                    e.get("uuid", ""),
                    e.get("info", ""),
                    int(e.get("threat_level_id", 0)),
                    int(e.get("distribution", 0)),
                    tags,
                    json.dumps(e),
                )
                upserted += 1

        log.info("misp.sync_done", upserted=upserted)
        return upserted


async def main() -> None:
    for var in ("MISP_URL", "MISP_KEY", "MISP_DB_URL"):
        if not os.getenv(var):
            log.error("misp_client.missing_env", var=var)
            sys.exit(1)
    if not _HAS_PYMISP:
        log.error("misp_client.pymisp_missing")
        sys.exit(1)

    pool = await asyncpg.create_pool(MISP_DB_URL, min_size=2, max_size=5)
    client = MISPClient()
    count = await client.sync_to_local_db(pool)
    log.info("misp_client.main_done", synced=count)
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
