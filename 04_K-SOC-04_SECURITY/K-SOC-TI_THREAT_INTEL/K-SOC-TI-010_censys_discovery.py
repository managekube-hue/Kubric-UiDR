"""
K-SOC-TI-010 — Censys attack surface discovery.

Discovers exposed assets, CVE-affected hosts, and certificate details via
the Censys Internet Search API.

Env vars:
    CENSYS_API_ID      Censys API ID (required)
    CENSYS_API_SECRET  Censys API Secret (required)
    IOC_DB_URL         asyncpg DSN (required)
    TENANT_ID          tenant identifier (default: default)

Schema:
    exposed_assets (
        id           BIGSERIAL PRIMARY KEY,
        tenant_id    TEXT NOT NULL,
        ip           TEXT,
        port         INT,
        protocol     TEXT,
        service_name TEXT,
        http_title   TEXT,
        last_seen    TEXT,
        certs        JSONB,
        UNIQUE (tenant_id, ip, port, protocol)
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
import structlog

log = structlog.get_logger(__name__)

CENSYS_API_ID = os.getenv("CENSYS_API_ID", "")
CENSYS_API_SECRET = os.getenv("CENSYS_API_SECRET", "")
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
TENANT_ID = os.getenv("TENANT_ID", "default")

# Censys rate limit: 120 queries/min = 2/s; we stay conservative at 1/s.
_RATE_LIMIT_SEC = 1.0

try:
    from censys.search import CensysHosts  # type: ignore[import]
    _HAS_CENSYS = True
except ImportError:
    _HAS_CENSYS = False
    CensysHosts = None  # type: ignore[assignment]

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS exposed_assets (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    ip           TEXT,
    port         INT,
    protocol     TEXT,
    service_name TEXT,
    http_title   TEXT,
    last_seen    TEXT,
    certs        JSONB,
    UNIQUE (tenant_id, ip, port, protocol)
);
"""

UPSERT_SQL = """
INSERT INTO exposed_assets (tenant_id, ip, port, protocol, service_name, http_title, last_seen, certs)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, ip, port, protocol) DO UPDATE
    SET service_name = EXCLUDED.service_name,
        http_title   = EXCLUDED.http_title,
        last_seen    = EXCLUDED.last_seen,
        certs        = EXCLUDED.certs;
"""


@dataclass
class ExposedAsset:
    ip: str
    port: int
    protocol: str
    service_name: str
    certs: list[dict[str, Any]] = field(default_factory=list)
    http_title: str = ""
    last_seen: str = ""


@dataclass
class CertInfo:
    domain: str
    subject_cn: str
    issuer: str
    not_after: str
    fingerprint: str


class CensysDiscovery:
    """Discovers internet-exposed assets via the Censys Hosts search API."""

    def __init__(self, db_pool: asyncpg.Pool) -> None:
        self._pool = db_pool
        if not _HAS_CENSYS:
            log.warning("censys.library_missing", hint="pip install censys")
            self._hosts = None
        else:
            self._hosts = CensysHosts(api_id=CENSYS_API_ID, api_secret=CENSYS_API_SECRET)

    async def find_exposed_assets(
        self, org_name: str, ip_ranges: list[str]
    ) -> list[ExposedAsset]:
        """Search Censys for exposed assets matching org name or IP ranges."""
        if not _HAS_CENSYS or not self._hosts:
            raise RuntimeError("censys package not installed")

        query_parts = []
        if org_name:
            query_parts.append(f'autonomous_system.organization: "{org_name}"')
        for r in ip_ranges:
            query_parts.append(f"ip: {r}")
        query = " OR ".join(query_parts) if query_parts else "*"

        assets: list[ExposedAsset] = []
        loop = asyncio.get_event_loop()

        def _search() -> Any:
            return list(self._hosts.search(query, pages=2, per_page=100))

        results = await loop.run_in_executor(None, _search)
        await asyncio.sleep(_RATE_LIMIT_SEC)

        for host in results:
            ip = host.get("ip", "")
            for svc in host.get("services", []) or []:
                asset = ExposedAsset(
                    ip=ip,
                    port=int(svc.get("port", 0)),
                    protocol=svc.get("transport_protocol", ""),
                    service_name=svc.get("service_name", ""),
                    http_title=svc.get("http", {}).get("response", {}).get("html_title", "") or "",
                    last_seen=host.get("last_updated_at", ""),
                )
                assets.append(asset)

        await self._upsert_assets(assets)
        return assets

    async def search_cve(self, cve_id: str) -> list[ExposedAsset]:
        """Find hosts exposed with a specific CVE."""
        if not _HAS_CENSYS or not self._hosts:
            raise RuntimeError("censys package not installed")

        loop = asyncio.get_event_loop()
        query = f"vulnerabilities.cve_id: {cve_id}"

        def _search() -> Any:
            return list(self._hosts.search(query, pages=1, per_page=100))

        results = await loop.run_in_executor(None, _search)
        await asyncio.sleep(_RATE_LIMIT_SEC)

        assets: list[ExposedAsset] = []
        for host in results:
            ip = host.get("ip", "")
            for svc in host.get("services", []) or []:
                assets.append(ExposedAsset(
                    ip=ip,
                    port=int(svc.get("port", 0)),
                    protocol=svc.get("transport_protocol", ""),
                    service_name=svc.get("service_name", ""),
                    last_seen=host.get("last_updated_at", ""),
                ))
        await self._upsert_assets(assets)
        return assets

    async def check_certificate_expiry(self, domain: str) -> list[CertInfo]:
        """Find certificates for a domain and return their expiry info."""
        if not _HAS_CENSYS or not self._hosts:
            raise RuntimeError("censys package not installed")

        loop = asyncio.get_event_loop()
        query = f"services.tls.certificates.leaf_data.names: {domain}"

        def _search() -> Any:
            return list(self._hosts.search(query, pages=1, per_page=50))

        results = await loop.run_in_executor(None, _search)
        certs: list[CertInfo] = []
        for host in results:
            for svc in host.get("services", []) or []:
                tls = svc.get("tls", {}) or {}
                cert = tls.get("certificates", {}).get("leaf_data", {}) or {}
                if not cert:
                    continue
                certs.append(CertInfo(
                    domain=domain,
                    subject_cn=cert.get("subject", {}).get("common_name", "") or "",
                    issuer=cert.get("issuer", {}).get("common_name", "") or "",
                    not_after=cert.get("not_after", ""),
                    fingerprint=cert.get("fingerprint", ""),
                ))
        return certs

    async def _upsert_assets(self, assets: list[ExposedAsset]) -> None:
        async with self._pool.acquire() as conn:
            for a in assets:
                await conn.execute(
                    UPSERT_SQL,
                    TENANT_ID, a.ip, a.port, a.protocol,
                    a.service_name, a.http_title, a.last_seen,
                    json.dumps(a.certs),
                )


async def main() -> None:
    if not CENSYS_API_ID or not CENSYS_API_SECRET:
        log.error("censys.missing_env", vars="CENSYS_API_ID, CENSYS_API_SECRET")
        sys.exit(1)
    if not IOC_DB_URL:
        log.error("censys.missing_env", var="IOC_DB_URL")
        sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_SQL)

    discovery = CensysDiscovery(pool)
    org_name = os.getenv("TARGET_ORG", "")
    if org_name:
        assets = await discovery.find_exposed_assets(org_name, [])
        log.info("censys.discovery_done", org=org_name, count=len(assets))
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
