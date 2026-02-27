"""
K-KAI Foresight: EPSS Enrichment
Fetches EPSS scores from the FIRST.org EPSS API and enriches
CVE records stored in the platform's PostgreSQL database.
"""
from __future__ import annotations
import asyncio, logging, os
from dataclasses import dataclass, field
from datetime import date, datetime, timezone
from typing import List, Optional, Dict
import httpx
import asyncpg

logger = logging.getLogger(__name__)

EPSS_API = "https://api.first.org/data/v1/epss"
PG_DSN   = os.getenv("PG_DSN", "postgresql://kubric:kubric@localhost:5432/kubric")
HIGH_EPSS_THRESHOLD = 0.10   # ≥10 % exploitation probability flagged as high

@dataclass
class EpssRecord:
    cve:        str
    epss:       float
    percentile: float
    date:       date
    high_risk:  bool = field(init=False)

    def __post_init__(self) -> None:
        self.high_risk = self.epss >= HIGH_EPSS_THRESHOLD

class EpssEnricher:
    def __init__(self, pg_dsn: str = PG_DSN, batch_size: int = 100):
        self.pg_dsn     = pg_dsn
        self.batch_size = batch_size
        self._pool: Optional[asyncpg.Pool] = None

    # ── lifecycle ──────────────────────────────────────────────────
    async def __aenter__(self) -> "EpssEnricher":
        self._pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=8)
        return self

    async def __aexit__(self, *_) -> None:
        if self._pool:
            await self._pool.close()

    # ── EPSS fetch ─────────────────────────────────────────────────
    async def fetch_scores(self, cves: List[str]) -> List[EpssRecord]:
        records: List[EpssRecord] = []
        async with httpx.AsyncClient(timeout=30) as client:
            for i in range(0, len(cves), self.batch_size):
                chunk = cves[i : i + self.batch_size]
                params = {"cve": ",".join(chunk)}
                resp   = await client.get(EPSS_API, params=params)
                resp.raise_for_status()
                data   = resp.json().get("data", [])
                for item in data:
                    records.append(EpssRecord(
                        cve        = item["cve"],
                        epss       = float(item.get("epss", 0)),
                        percentile = float(item.get("percentile", 0)),
                        date       = date.fromisoformat(item.get("date", date.today().isoformat())),
                    ))
        return records

    # ── DB operations ──────────────────────────────────────────────
    async def upsert_scores(self, records: List[EpssRecord]) -> int:
        if not records or self._pool is None:
            return 0
        sql = """
            INSERT INTO kai_epss_scores (cve, epss, percentile, score_date, high_risk, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (cve) DO UPDATE
              SET epss       = EXCLUDED.epss,
                  percentile = EXCLUDED.percentile,
                  score_date = EXCLUDED.score_date,
                  high_risk  = EXCLUDED.high_risk,
                  updated_at = EXCLUDED.updated_at
        """
        now = datetime.now(timezone.utc)
        async with self._pool.acquire() as conn:
            await conn.executemany(sql, [
                (r.cve, r.epss, r.percentile, r.date, r.high_risk, now)
                for r in records
            ])
        return len(records)

    async def get_pending_cves(self) -> List[str]:
        """Return CVEs not scored within the last 24 hours."""
        if self._pool is None:
            return []
        sql = """
            SELECT DISTINCT cve_id FROM kai_vulnerabilities
            WHERE cve_id IS NOT NULL
              AND (epss_updated_at IS NULL OR epss_updated_at < NOW() - INTERVAL '24 hours')
            LIMIT 1000
        """
        rows = await self._pool.fetch(sql)
        return [r["cve_id"] for r in rows]

    # ── full enrichment run ────────────────────────────────────────
    async def run_enrichment(self) -> Dict:
        cves = await self.get_pending_cves()
        if not cves:
            logger.info("No CVEs pending EPSS enrichment")
            return {"enriched": 0, "high_risk": 0}

        logger.info("Fetching EPSS scores for %d CVEs", len(cves))
        records = await self.fetch_scores(cves)
        upserted = await self.upsert_scores(records)
        high_risk = sum(1 for r in records if r.high_risk)
        logger.info("Upserted %d EPSS records, %d high-risk", upserted, high_risk)
        return {"enriched": upserted, "high_risk": high_risk, "cves_queried": len(cves)}

# ── entrypoint ────────────────────────────────────────────────────
async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    async with EpssEnricher() as enricher:
        result = await enricher.run_enrichment()
    print(result)

if __name__ == "__main__":
    asyncio.run(main())
