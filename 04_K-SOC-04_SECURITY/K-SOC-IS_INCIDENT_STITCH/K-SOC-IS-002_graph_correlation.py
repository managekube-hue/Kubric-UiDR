"""
K-SOC-IS-002 — Graph-based incident correlation.

Correlates security alerts into incidents using NetworkX directed graph
analysis.  Builds an event graph per tenant over a rolling 30-minute window,
finds weakly-connected components, scores each component, and creates
incident records in the kai_incidents table.

Env vars:
    SIGMA_DB_URL  asyncpg DSN (required)
    TENANT_ID     tenant identifier (default: default)

Schema (PostgreSQL):
    kai_incidents (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        tenant_id   TEXT NOT NULL,
        alert_ids   TEXT[],
        severity    TEXT,
        status      TEXT DEFAULT 'open',
        score       FLOAT,
        created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    )
"""

from __future__ import annotations

import asyncio
import os
import sys
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

SIGMA_DB_URL = os.getenv("SIGMA_DB_URL", "")
TENANT_ID = os.getenv("TENANT_ID", "default")

try:
    import networkx as nx  # type: ignore[import]
    _HAS_NX = True
except ImportError:
    _HAS_NX = False
    nx = None  # type: ignore[assignment]

CREATE_INCIDENTS_SQL = """
CREATE TABLE IF NOT EXISTS kai_incidents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    alert_ids   TEXT[],
    severity    TEXT,
    status      TEXT DEFAULT 'open',
    score       FLOAT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

SELECT_ALERTS_SQL = """
SELECT id, asset_id, src_ip, parent_process_id, severity, created_at
FROM kai_alerts
WHERE tenant_id = $1
  AND created_at > now() - ($2::int || ' minutes')::INTERVAL
  AND incident_id IS NULL
ORDER BY created_at;
"""

UPDATE_ALERT_INCIDENT_SQL = """
UPDATE kai_alerts SET incident_id = $1 WHERE id = ANY($2) AND tenant_id = $3;
"""

INSERT_INCIDENT_SQL = """
INSERT INTO kai_incidents (tenant_id, alert_ids, severity, score)
VALUES ($1, $2, $3, $4)
RETURNING id;
"""


def _severity_weight(severity: str) -> float:
    """Map severity string to numeric weight for incident scoring."""
    return {
        "critical": 10.0,
        "high": 7.0,
        "medium": 4.0,
        "low": 2.0,
        "info": 1.0,
    }.get((severity or "").lower(), 1.0)


class IncidentCorrelator:
    """Correlates kai_alerts into incidents via NetworkX graph analysis."""

    def __init__(self, db_pool: asyncpg.Pool) -> None:
        self._pool = db_pool
        if not _HAS_NX:
            raise RuntimeError("networkx not installed: pip install networkx")

    async def build_event_graph(
        self, tenant_id: str, window_minutes: int = 30
    ) -> "nx.DiGraph":
        """
        Build a directed graph from recent alerts.

        Nodes: alert IDs.
        Edges: connect alerts that share asset_id, src_ip, or parent_process_id.
        """
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(SELECT_ALERTS_SQL, tenant_id, window_minutes)

        G: nx.DiGraph = nx.DiGraph()

        # Index alerts by shared attributes for efficient edge creation.
        asset_map: dict[str, list[str]] = {}   # asset_id -> [alert_id]
        ip_map: dict[str, list[str]] = {}      # src_ip -> [alert_id]
        proc_map: dict[str, list[str]] = {}    # parent_process_id -> [alert_id]

        for row in rows:
            alert_id = str(row["id"])
            G.add_node(alert_id, severity=row["severity"] or "info", row=dict(row))

            asset_id = str(row["asset_id"] or "")
            if asset_id:
                asset_map.setdefault(asset_id, []).append(alert_id)

            src_ip = str(row["src_ip"] or "")
            if src_ip:
                ip_map.setdefault(src_ip, []).append(alert_id)

            ppid = str(row["parent_process_id"] or "")
            if ppid:
                proc_map.setdefault(ppid, []).append(alert_id)

        # Add edges for correlated alerts.
        for bucket in (asset_map, ip_map, proc_map):
            for group in bucket.values():
                for i in range(len(group)):
                    for j in range(i + 1, len(group)):
                        G.add_edge(group[i], group[j])

        log.info(
            "incident_correlator.graph_built",
            tenant_id=tenant_id,
            nodes=G.number_of_nodes(),
            edges=G.number_of_edges(),
        )
        return G

    def find_connected_components(self, G: "nx.DiGraph") -> list[list[str]]:
        """Return weakly-connected components as lists of alert IDs."""
        components = list(nx.weakly_connected_components(G))
        return [list(comp) for comp in components if len(comp) > 1]

    def score_incident(self, alert_ids: list[str], G: "nx.DiGraph") -> float:
        """
        Calculate incident score as the weighted sum of member alert severities.

        Score = sum(severity_weight * degree_centrality_boost) for each node.
        """
        total = 0.0
        for alert_id in alert_ids:
            severity = G.nodes[alert_id].get("severity", "info")
            weight = _severity_weight(severity)
            # Boost for highly connected nodes (potential pivot point).
            degree = G.degree(alert_id)
            boost = 1.0 + (degree * 0.1)
            total += weight * boost
        return round(total, 2)

    def _severity_for_incident(self, alert_ids: list[str], G: "nx.DiGraph") -> str:
        """Return the highest severity among component alert nodes."""
        order = ["critical", "high", "medium", "low", "info"]
        best = "info"
        for alert_id in alert_ids:
            sev = (G.nodes[alert_id].get("severity") or "info").lower()
            if order.index(sev) < order.index(best):
                best = sev
        return best

    async def create_incidents(self, tenant_id: str) -> int:
        """
        Run the full correlation cycle for tenant_id.

        1. Build event graph.
        2. Find components with >= 2 alerts.
        3. Create kai_incidents rows.
        4. Mark alert.incident_id.
        Returns count of incidents created.
        """
        G = await self.build_event_graph(tenant_id)
        components = self.find_connected_components(G)

        count = 0
        for component in components:
            score = self.score_incident(component, G)
            severity = self._severity_for_incident(component, G)

            async with self._pool.acquire() as conn:
                row = await conn.fetchrow(
                    INSERT_INCIDENT_SQL,
                    tenant_id,
                    component,
                    severity,
                    score,
                )
                incident_id = row["id"]
                await conn.execute(
                    UPDATE_ALERT_INCIDENT_SQL,
                    incident_id,
                    component,
                    tenant_id,
                )
            count += 1
            log.info(
                "incident_correlator.incident_created",
                incident_id=str(incident_id),
                alert_count=len(component),
                score=score,
                severity=severity,
            )

        log.info("incident_correlator.cycle_done", tenant_id=tenant_id, incidents=count)
        return count

    async def run_forever(self, tenant_id: str, interval_seconds: int = 60) -> None:
        """Run the correlation cycle indefinitely with the given interval."""
        async with self._pool.acquire() as conn:
            await conn.execute(CREATE_INCIDENTS_SQL)
        log.info("incident_correlator.started", tenant_id=tenant_id, interval=interval_seconds)
        while True:
            try:
                await self.create_incidents(tenant_id)
            except Exception as exc:  # noqa: BLE001
                log.error("incident_correlator.error", error=str(exc))
            await asyncio.sleep(interval_seconds)


async def main() -> None:
    if not SIGMA_DB_URL:
        log.error("incident_correlator.missing_env", var="SIGMA_DB_URL")
        sys.exit(1)
    if not _HAS_NX:
        log.error("incident_correlator.networkx_missing")
        sys.exit(1)

    pool = await asyncpg.create_pool(SIGMA_DB_URL, min_size=2, max_size=5)
    correlator = IncidentCorrelator(pool)
    await correlator.run_forever(TENANT_ID, interval_seconds=60)
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
