"""
K-KAI Sentinel: Health Score Publisher
Computes a composite health score for each tenant (0-100) from
security posture, compliance drift, agent availability, and open incidents.
Publishes scores to NATS subject kai.health.<tenant_id>.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from typing import Dict, List, Optional
import asyncpg
import nats
from nats.aio.client import Client as NATS

logger = logging.getLogger(__name__)
PG_DSN   = os.getenv("PG_DSN",  "postgresql://kubric:kubric@localhost:5432/kubric")
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
INTERVAL = int(os.getenv("HEALTH_INTERVAL_SECS", "300"))   # 5 min default

@dataclass
class TenantHealth:
    tenant_id:         str
    overall_score:     float        # 0-100
    security_score:    float
    compliance_score:  float
    availability_score: float
    open_incidents:    int
    critical_vulns:    int
    agent_count:       int
    agents_online:     int
    computed_at:       str          # ISO timestamp

    @property
    def status(self) -> str:
        if self.overall_score >= 80: return "healthy"
        if self.overall_score >= 60: return "degraded"
        if self.overall_score >= 40: return "at_risk"
        return "critical"

def _score(healthy: float, total: float, weight: float = 100.0) -> float:
    if total == 0:
        return weight
    return round((healthy / total) * weight, 2)

async def compute_health(pool: asyncpg.Pool, tenant_id: str) -> TenantHealth:
    async with pool.acquire() as conn:
        # Open incidents (severity 4-5)
        open_inc = await conn.fetchval(
            "SELECT COUNT(*) FROM kai_incidents WHERE tenant_id=$1 AND status='open' AND severity>=4",
            tenant_id) or 0

        # Critical vulnerabilities
        crit_vulns = await conn.fetchval(
            "SELECT COUNT(*) FROM kai_vulnerabilities WHERE tenant_id=$1 AND cvss>=9.0 AND status!='patched'",
            tenant_id) or 0

        # Agent availability
        agents = await conn.fetch(
            "SELECT is_online FROM kai_agents WHERE tenant_id=$1", tenant_id)
        agent_count   = len(agents)
        agents_online = sum(1 for a in agents if a["is_online"])

        # Compliance drift events in last 24h
        drift_events = await conn.fetchval(
            "SELECT COUNT(*) FROM kai_drift_events WHERE tenant_id=$1 AND created_at > NOW()-INTERVAL '24 hours'",
            tenant_id) or 0

    # Sub-scores
    security_score    = max(0, 100 - (open_inc * 10) - (crit_vulns * 5))
    security_score    = min(100.0, float(security_score))
    compliance_score  = max(0.0, 100.0 - drift_events * 2.0)
    availability_score = _score(agents_online, agent_count)

    # Weighted composite
    overall = round(
        0.40 * security_score +
        0.30 * compliance_score +
        0.30 * availability_score, 2)

    return TenantHealth(
        tenant_id=tenant_id,
        overall_score=overall,
        security_score=security_score,
        compliance_score=compliance_score,
        availability_score=availability_score,
        open_incidents=open_inc,
        critical_vulns=crit_vulns,
        agent_count=agent_count,
        agents_online=agents_online,
        computed_at=datetime.now(timezone.utc).isoformat(),
    )

class HealthScorePublisher:
    def __init__(self, pg_dsn: str = PG_DSN, nats_url: str = NATS_URL):
        self.pg_dsn   = pg_dsn
        self.nats_url = nats_url
        self._pool: Optional[asyncpg.Pool] = None
        self._nc: Optional[NATS] = None

    async def start(self) -> None:
        self._pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=8)
        self._nc   = await nats.connect(self.nats_url)
        logger.info("HealthScorePublisher connected (PG + NATS)")

    async def stop(self) -> None:
        if self._nc:  await self._nc.close()
        if self._pool: await self._pool.close()

    async def _get_tenants(self) -> List[str]:
        assert self._pool
        rows = await self._pool.fetch("SELECT id FROM kai_tenants WHERE status='active'")
        return [r["id"] for r in rows]

    async def publish_once(self) -> int:
        tenants = await self._get_tenants()
        if not tenants:
            return 0
        published = 0
        assert self._pool and self._nc
        for tid in tenants:
            try:
                health  = await compute_health(self._pool, tid)
                subject = f"kai.health.{tid}"
                payload = json.dumps(asdict(health)).encode()
                await self._nc.publish(subject, payload)
                published += 1
            except Exception as exc:
                logger.error("Health score failed for %s: %s", tid, exc)
        logger.info("Published health scores for %d tenants", published)
        return published

    async def run_forever(self, interval: int = INTERVAL) -> None:
        await self.start()
        try:
            while True:
                await self.publish_once()
                await asyncio.sleep(interval)
        except asyncio.CancelledError:
            pass
        finally:
            await self.stop()

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    asyncio.run(HealthScorePublisher().run_forever())
