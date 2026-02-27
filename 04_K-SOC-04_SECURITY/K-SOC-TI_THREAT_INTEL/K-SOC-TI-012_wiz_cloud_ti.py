"""
K-SOC-TI-012 — Wiz cloud security findings puller.

Pulls security issues and vulnerability findings from the Wiz platform via
its GraphQL API, syncing critical issues to the local DB and publishing
to NATS for downstream remediation workflows.

Env vars:
    WIZ_CLIENT_ID       Wiz OAuth client ID (required)
    WIZ_CLIENT_SECRET   Wiz OAuth client secret (required)
    WIZ_API_URL         Wiz GraphQL API endpoint (required)
    IOC_DB_URL          asyncpg DSN (required)
    NATS_URL            nats:// URL (default: nats://localhost:4222)
    TENANT_ID           tenant identifier (default: default)
    SYNC_INTERVAL_SECONDS  seconds between syncs (default: 900)

Schema:
    cloud_security_findings (
        id            TEXT PRIMARY KEY,
        tenant_id     TEXT NOT NULL,
        title         TEXT,
        severity      TEXT,
        status        TEXT,
        resource_id   TEXT,
        resource_type TEXT,
        created_at    TEXT,
        remediation   TEXT,
        synced_at     TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import json
import os
import signal
import sys
from dataclasses import dataclass
from typing import Any

import asyncpg
import httpx
import structlog

log = structlog.get_logger(__name__)

WIZ_CLIENT_ID = os.getenv("WIZ_CLIENT_ID", "")
WIZ_CLIENT_SECRET = os.getenv("WIZ_CLIENT_SECRET", "")
WIZ_API_URL = os.getenv("WIZ_API_URL", "")
IOC_DB_URL = os.getenv("IOC_DB_URL", "")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
TENANT_ID = os.getenv("TENANT_ID", "default")
SYNC_INTERVAL = int(os.getenv("SYNC_INTERVAL_SECONDS", "900"))

WIZ_AUTH_URL = "https://auth.app.wiz.io/oauth/token"

CREATE_SQL = """
CREATE TABLE IF NOT EXISTS cloud_security_findings (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL,
    title         TEXT,
    severity      TEXT,
    status        TEXT,
    resource_id   TEXT,
    resource_type TEXT,
    created_at    TEXT,
    remediation   TEXT,
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

UPSERT_SQL = """
INSERT INTO cloud_security_findings
    (id, tenant_id, title, severity, status, resource_id, resource_type, created_at, remediation, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (id) DO UPDATE
    SET status     = EXCLUDED.status,
        synced_at  = now();
"""

_ISSUES_QUERY = """
query GetIssues($severity: String, $after: String) {
  issues(
    filterBy: { severity: [$severity] }
    first: 100
    after: $after
  ) {
    nodes {
      id
      title
      severity
      status
      createdAt
      entity { id type }
      remediation
    }
    pageInfo { hasNextPage endCursor }
  }
}
"""

_VULNS_QUERY = """
query GetVulns($assetId: String!) {
  vulnerabilityFindings(
    filterBy: { assetId: [$assetId] }
    first: 50
  ) {
    nodes {
      id
      name
      severity
      cveId
      fixedVersion
      firstSeenAt
    }
  }
}
"""


@dataclass
class WizIssue:
    id: str
    title: str
    severity: str
    status: str
    resource_id: str
    resource_type: str
    created_at: str
    remediation: str


@dataclass
class WizVuln:
    id: str
    name: str
    severity: str
    cve_id: str
    fixed_version: str
    first_seen_at: str


class WizClient:
    """GraphQL client for the Wiz security platform."""

    def __init__(self) -> None:
        self._token: str = ""
        self._http = httpx.AsyncClient(timeout=httpx.Timeout(30.0))

    async def close(self) -> None:
        await self._http.aclose()

    async def get_access_token(self) -> str:
        """Fetch a short-lived OAuth2 bearer token from Wiz auth endpoint."""
        resp = await self._http.post(
            WIZ_AUTH_URL,
            data={
                "grant_type": "client_credentials",
                "client_id": WIZ_CLIENT_ID,
                "client_secret": WIZ_CLIENT_SECRET,
                "audience": "wiz-api",
            },
        )
        resp.raise_for_status()
        self._token = resp.json()["access_token"]
        return self._token

    async def query(self, gql: str, variables: dict[str, Any]) -> dict[str, Any]:
        """Execute a GraphQL query with Bearer auth. Auto-refreshes token if needed."""
        if not self._token:
            await self.get_access_token()
        resp = await self._http.post(
            WIZ_API_URL,
            json={"query": gql, "variables": variables},
            headers={"Authorization": f"Bearer {self._token}",
                     "Content-Type": "application/json"},
        )
        if resp.status_code == 401:
            # Token may have expired; refresh once and retry.
            await self.get_access_token()
            resp = await self._http.post(
                WIZ_API_URL,
                json={"query": gql, "variables": variables},
                headers={"Authorization": f"Bearer {self._token}",
                         "Content-Type": "application/json"},
            )
        resp.raise_for_status()
        data = resp.json()
        if "errors" in data:
            raise RuntimeError(f"Wiz GraphQL errors: {data['errors']}")
        return data.get("data", {})

    async def get_issues(self, severity: str = "CRITICAL") -> list[WizIssue]:
        """Paginate and fetch all issues at the given severity level."""
        issues: list[WizIssue] = []
        cursor: str | None = None
        while True:
            variables: dict[str, Any] = {"severity": severity}
            if cursor:
                variables["after"] = cursor
            data = await self.query(_ISSUES_QUERY, variables)
            nodes = data.get("issues", {}).get("nodes", []) or []
            for n in nodes:
                entity = n.get("entity") or {}
                issues.append(WizIssue(
                    id=n.get("id", ""),
                    title=n.get("title", ""),
                    severity=n.get("severity", ""),
                    status=n.get("status", ""),
                    resource_id=entity.get("id", ""),
                    resource_type=entity.get("type", ""),
                    created_at=n.get("createdAt", ""),
                    remediation=n.get("remediation", "") or "",
                ))
            page_info = data.get("issues", {}).get("pageInfo", {})
            if not page_info.get("hasNextPage"):
                break
            cursor = page_info.get("endCursor")
        return issues

    async def get_vulnerabilities(self, asset_id: str) -> list[WizVuln]:
        """Fetch vulnerabilities for a specific asset ID."""
        data = await self.query(_VULNS_QUERY, {"assetId": asset_id})
        vulns: list[WizVuln] = []
        for n in (data.get("vulnerabilityFindings", {}).get("nodes", []) or []):
            vulns.append(WizVuln(
                id=n.get("id", ""),
                name=n.get("name", ""),
                severity=n.get("severity", ""),
                cve_id=n.get("cveId", "") or "",
                fixed_version=n.get("fixedVersion", "") or "",
                first_seen_at=n.get("firstSeenAt", ""),
            ))
        return vulns


async def sync_loop(db_pool: asyncpg.Pool) -> None:
    import nats as nats_client  # type: ignore[import]
    nc = await nats_client.connect(NATS_URL)
    client = WizClient()
    log.info("wiz_ti.sync_loop_started", interval=SYNC_INTERVAL)

    shutdown = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, shutdown.set)

    while not shutdown.is_set():
        try:
            issues = await client.get_issues("CRITICAL")
            async with db_pool.acquire() as conn:
                for issue in issues:
                    await conn.execute(
                        UPSERT_SQL,
                        issue.id, TENANT_ID, issue.title, issue.severity,
                        issue.status, issue.resource_id, issue.resource_type,
                        issue.created_at, issue.remediation,
                    )
                    subject = f"kubric.{TENANT_ID}.vdr.vuln.v1"
                    payload = json.dumps({
                        "tenant_id": TENANT_ID,
                        "source": "wiz",
                        "issue_id": issue.id,
                        "title": issue.title,
                        "severity": issue.severity,
                        "resource_id": issue.resource_id,
                    }).encode()
                    await nc.publish(subject, payload)
            log.info("wiz_ti.sync_done", count=len(issues))
        except Exception as exc:  # noqa: BLE001
            log.error("wiz_ti.sync_error", error=str(exc))
        try:
            await asyncio.wait_for(shutdown.wait(), timeout=float(SYNC_INTERVAL))
        except asyncio.TimeoutError:
            pass

    await client.close()
    await nc.close()


async def main() -> None:
    for var in ("WIZ_CLIENT_ID", "WIZ_CLIENT_SECRET", "WIZ_API_URL", "IOC_DB_URL"):
        if not os.getenv(var):
            log.error("wiz_ti.missing_env", var=var)
            sys.exit(1)

    pool = await asyncpg.create_pool(IOC_DB_URL, min_size=2, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(CREATE_SQL)
    await sync_loop(pool)
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
