"""
K-SOC-TI-007 — STIX2 bundle parser.

Parses STIX2 JSON bundles (including MISP exports) into normalized IOCRecord
objects and bulk-upserts them to the ioc_feeds PostgreSQL table.

Env vars:
    IOC_DB_URL  asyncpg DSN (required)
"""

from __future__ import annotations

import asyncio
import json
import os
import re
import sys
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

IOC_DB_URL = os.getenv("IOC_DB_URL", "")

try:
    from stix2 import Bundle, Indicator, parse as stix2_parse  # type: ignore[import]
    _HAS_STIX2 = True
except ImportError:
    _HAS_STIX2 = False
    Bundle = None  # type: ignore[assignment,misc]
    Indicator = None  # type: ignore[assignment,misc]
    stix2_parse = None  # type: ignore[assignment]

# Regex patterns for STIX2 indicator patterns.
_PATTERN_RE: list[tuple[str, str]] = [
    (r"\[ipv4-addr:value\s*=\s*'([^']+)'\]", "ipv4"),
    (r"\[ipv6-addr:value\s*=\s*'([^']+)'\]", "ipv6"),
    (r"\[domain-name:value\s*=\s*'([^']+)'\]", "domain"),
    (r"\[url:value\s*=\s*'([^']+)'\]", "url"),
    (r"\[file:hashes\.SHA-256\s*=\s*'([^']+)'\]", "sha256"),
    (r"\[file:hashes\.MD5\s*=\s*'([^']+)'\]", "md5"),
    (r"\[email-addr:value\s*=\s*'([^']+)'\]", "email"),
]

UPSERT_SQL = """
INSERT INTO ioc_feeds
    (tenant_id, ioc_type, value, confidence, source, tlp, tags, first_seen, last_seen)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (tenant_id, ioc_type, value, source) DO UPDATE
    SET confidence = EXCLUDED.confidence,
        tags       = EXCLUDED.tags,
        last_seen  = now();
"""


@dataclass
class IOCRecord:
    ioc_type: str
    value: str
    confidence: int
    tlp: str
    labels: list[str]
    source: str
    valid_from: datetime | None
    valid_until: datetime | None
    stix_id: str
    tenant_id: str = "default"
    tags: list[str] = field(default_factory=list)


class STIXParser:
    """Parse STIX2 bundles into normalized IOCRecord objects."""

    def parse_bundle(self, json_str: str) -> list[IOCRecord]:
        """Parse a STIX2 bundle JSON string and return IOCRecord list."""
        if not _HAS_STIX2:
            raise RuntimeError("stix2 package not installed: pip install stix2")

        try:
            bundle = stix2_parse(json_str, allow_custom=True)
        except Exception as exc:
            log.warning("stix2_parser.parse_error", error=str(exc))
            return []

        records: list[IOCRecord] = []
        objects = getattr(bundle, "objects", []) or []
        for obj in objects:
            if obj.get("type") != "indicator":
                continue
            ioc_type, value = self.parse_indicator_pattern(obj.get("pattern", ""))
            if not ioc_type or not value:
                continue

            confidence = int(obj.get("confidence") or 50)
            tlp = self._extract_tlp(obj)
            labels: list[str] = list(obj.get("labels") or [])
            valid_from = self._parse_dt(obj.get("valid_from"))
            valid_until = self._parse_dt(obj.get("valid_until"))
            stix_id = str(obj.get("id", ""))
            created_by = str(obj.get("created_by_ref", ""))

            records.append(IOCRecord(
                ioc_type=ioc_type,
                value=value,
                confidence=confidence,
                tlp=tlp,
                labels=labels,
                source=created_by or "stix2",
                valid_from=valid_from,
                valid_until=valid_until,
                stix_id=stix_id,
                tags=labels,
            ))
        return records

    @staticmethod
    def parse_indicator_pattern(pattern: str) -> tuple[str, str]:
        """
        Parse a STIX2 indicator pattern like [ipv4-addr:value = '1.2.3.4'].
        Returns (ioc_type, value) or ('', '') if unrecognised.
        """
        for regex, ioc_type in _PATTERN_RE:
            m = re.search(regex, pattern, re.IGNORECASE)
            if m:
                return ioc_type, m.group(1)
        return "", ""

    async def save_to_db(self, records: list[IOCRecord], db_pool: asyncpg.Pool) -> int:
        """Bulk-upsert IOCRecords into ioc_feeds table. Returns count."""
        if not records:
            return 0
        saved = 0
        async with db_pool.acquire() as conn:
            for r in records:
                first_seen = r.valid_from.isoformat() if r.valid_from else None
                last_seen = r.valid_until.isoformat() if r.valid_until else None
                try:
                    await conn.execute(
                        UPSERT_SQL,
                        r.tenant_id,
                        r.ioc_type,
                        r.value,
                        r.confidence,
                        r.source,
                        r.tlp,
                        r.tags,
                        first_seen,
                        last_seen,
                    )
                    saved += 1
                except Exception as exc:  # noqa: BLE001
                    log.warning("stix2_parser.upsert_error", error=str(exc))
        return saved

    async def process_misp_export(self, file_path: str, db_pool: asyncpg.Pool) -> int:
        """Load a MISP STIX2 export JSON file and process all IOCs."""
        with open(file_path, "r", encoding="utf-8") as f:
            json_str = f.read()
        records = self.parse_bundle(json_str)
        count = await self.save_to_db(records, db_pool)
        log.info("stix2_parser.misp_export_processed", file=file_path, count=count)
        return count

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _extract_tlp(obj: Any) -> str:
        for ref in obj.get("object_marking_refs") or []:
            ref_str = str(ref)
            if "tlp-white" in ref_str:
                return "white"
            if "tlp-green" in ref_str:
                return "green"
            if "tlp-amber" in ref_str:
                return "amber"
            if "tlp-red" in ref_str:
                return "red"
        return "amber"

    @staticmethod
    def _parse_dt(value: Any) -> datetime | None:
        if not value:
            return None
        try:
            if isinstance(value, datetime):
                return value
            return datetime.fromisoformat(str(value).replace("Z", "+00:00"))
        except (ValueError, TypeError):
            return None


if __name__ == "__main__":
    import argparse

    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )

    parser = argparse.ArgumentParser(description="Parse a STIX2 bundle file")
    parser.add_argument("--file", required=True, help="Path to STIX2 JSON bundle")
    args = parser.parse_args()

    with open(args.file, "r", encoding="utf-8") as fh:
        raw = fh.read()

    stix_parser = STIXParser()
    records = stix_parser.parse_bundle(raw)
    print(f"Parsed {len(records)} IOC records")
    for r in records[:5]:
        print(f"  [{r.ioc_type}] {r.value}  tlp={r.tlp}  conf={r.confidence}")
