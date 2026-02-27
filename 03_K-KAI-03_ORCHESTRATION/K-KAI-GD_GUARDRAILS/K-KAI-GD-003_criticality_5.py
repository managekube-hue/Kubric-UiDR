"""
K-KAI-GD-003: Asset criticality scoring system (1-5 scale per Kubric spec).
Scores assets based on exposure, data sensitivity, and infrastructure role.
Stores results in PostgreSQL table: asset_criticality.
"""

import logging
import os
from typing import Any

import asyncpg

logger = logging.getLogger("kai.gd.criticality")

_DDL = """
CREATE TABLE IF NOT EXISTS asset_criticality (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    asset_id        TEXT NOT NULL,
    asset_name      TEXT,
    asset_type      TEXT,
    score           INT NOT NULL CHECK (score BETWEEN 1 AND 5),
    label           TEXT NOT NULL,
    factors         JSONB NOT NULL DEFAULT '{}',
    scored_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_asset_crit_tenant ON asset_criticality(tenant_id);
CREATE INDEX IF NOT EXISTS idx_asset_crit_score ON asset_criticality(score);
"""

# ---------------------------------------------------------------------------
# Label map
# ---------------------------------------------------------------------------
CRITICALITY_LABELS = {
    1: "Minimal",
    2: "Low",
    3: "Medium",
    4: "High",
    5: "Critical",
}


# ---------------------------------------------------------------------------
# CriticalityAssessor
# ---------------------------------------------------------------------------
class CriticalityAssessor:
    """
    Scores an asset on a 1-5 criticality scale and persists the result.
    Scoring factors (additive, clamped to 1-5):
      - Base score:                         1
      - is_internet_facing:                +2
      - handles_pii:                       +1
      - is_domain_controller:              +2
      - is_database:                       +1
      - cluster_node_count > 5:            +1
    Final score is clamped to [1, 5].
    """

    def __init__(self) -> None:
        self._pool: asyncpg.Pool | None = None

    async def _ensure_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            dsn = os.environ.get("DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai")
            self._pool = await asyncpg.create_pool(dsn, min_size=1, max_size=5)
            async with self._pool.acquire() as conn:
                await conn.execute(_DDL)
        return self._pool

    # ------------------------------------------------------------------
    # Scoring
    # ------------------------------------------------------------------
    def score_asset(self, asset: dict) -> int:
        """
        Compute and return the criticality score (1-5) for *asset*.
        The asset dict may contain any subset of the scoring factor keys.
        """
        score = 1  # base

        factors_applied: dict[str, Any] = {}

        if asset.get("is_internet_facing"):
            score += 2
            factors_applied["is_internet_facing"] = True

        if asset.get("handles_pii"):
            score += 1
            factors_applied["handles_pii"] = True

        if asset.get("is_domain_controller"):
            score += 2
            factors_applied["is_domain_controller"] = True

        if asset.get("is_database"):
            score += 1
            factors_applied["is_database"] = True

        cluster_node_count = asset.get("cluster_node_count", 0)
        if isinstance(cluster_node_count, (int, float)) and cluster_node_count > 5:
            score += 1
            factors_applied["cluster_node_count"] = cluster_node_count

        final = max(1, min(5, score))
        logger.debug(
            "score_asset asset_id=%s score=%d factors=%s",
            asset.get("id", "?"),
            final,
            factors_applied,
        )
        return final

    def get_criticality_label(self, score: int) -> str:
        """Return the human-readable label for a criticality score 1-5."""
        return CRITICALITY_LABELS.get(max(1, min(5, score)), "Unknown")

    # ------------------------------------------------------------------
    # Persistence
    # ------------------------------------------------------------------
    async def assess_and_store(
        self,
        tenant_id: str,
        asset: dict,
    ) -> dict:
        """
        Score *asset* and upsert the result into asset_criticality.
        Returns the full assessment record.
        """
        score = self.score_asset(asset)
        label = self.get_criticality_label(score)
        factors = {
            "is_internet_facing": bool(asset.get("is_internet_facing")),
            "handles_pii": bool(asset.get("handles_pii")),
            "is_domain_controller": bool(asset.get("is_domain_controller")),
            "is_database": bool(asset.get("is_database")),
            "cluster_node_count": asset.get("cluster_node_count", 0),
        }
        import json
        pool = await self._ensure_pool()
        async with pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO asset_criticality
                    (tenant_id, asset_id, asset_name, asset_type, score, label, factors)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
                ON CONFLICT (tenant_id, asset_id)
                DO UPDATE SET score=EXCLUDED.score,
                              label=EXCLUDED.label,
                              factors=EXCLUDED.factors,
                              asset_name=EXCLUDED.asset_name,
                              asset_type=EXCLUDED.asset_type,
                              scored_at=NOW()
                """,
                tenant_id,
                str(asset.get("id", asset.get("asset_id", "unknown"))),
                asset.get("name") or asset.get("hostname"),
                asset.get("type") or asset.get("asset_type"),
                score,
                label,
                json.dumps(factors),
            )
        record = {
            "tenant_id": tenant_id,
            "asset_id": str(asset.get("id", asset.get("asset_id", "unknown"))),
            "score": score,
            "label": label,
            "factors": factors,
        }
        logger.info(
            "assess_and_store tenant=%s asset_id=%s score=%d (%s)",
            tenant_id,
            record["asset_id"],
            score,
            label,
        )
        return record

    async def list_critical_assets(
        self,
        tenant_id: str,
        min_score: int = 4,
    ) -> list[dict]:
        """
        Return all assets for *tenant_id* with score >= *min_score*
        (default: High and Critical).
        """
        pool = await self._ensure_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT * FROM asset_criticality
                WHERE tenant_id = $1 AND score >= $2
                ORDER BY score DESC, scored_at DESC
                """,
                tenant_id,
                min_score,
            )
        return [dict(r) for r in rows]

    async def get_asset_score(
        self,
        tenant_id: str,
        asset_id: str,
    ) -> dict | None:
        """Fetch the stored criticality record for a single asset."""
        pool = await self._ensure_pool()
        async with pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT * FROM asset_criticality WHERE tenant_id=$1 AND asset_id=$2",
                tenant_id,
                asset_id,
            )
        return dict(row) if row else None
