"""
K-KAI Foresight: Hikari Feature Trainer
Uses a Hikari-style asyncpg connection pool to pull raw alert telemetry,
engineers time-window features, and persists the feature matrix to Parquet
for downstream LSTM / XGBoost training.
"""
from __future__ import annotations
import asyncio, logging, os
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Dict, Any
import asyncpg
import polars as pl

logger = logging.getLogger(__name__)

PG_DSN         = os.getenv("PG_DSN", "postgresql://kubric:kubric@localhost:5432/kubric")
OUTPUT_DIR     = Path(os.getenv("FEATURE_OUTPUT_DIR", "/tmp/kai_features"))
WINDOW_HOURS   = [1, 6, 24]   # rolling count windows
TENANT_CAP     = 500           # max tenants per batch

FEATURE_QUERY = """
SELECT
    DATE_TRUNC('hour', a.created_at)   AS hour_bucket,
    a.tenant_id,
    a.severity_id,
    a.class_uid,
    COUNT(*)                           AS event_count,
    COUNT(DISTINCT a.src_ip)           AS unique_src_ips,
    AVG(a.risk_score)                  AS avg_risk,
    MAX(a.risk_score)                  AS max_risk
FROM kai_alerts a
WHERE a.created_at >= $1
  AND a.created_at <  $2
GROUP BY 1, 2, 3, 4
ORDER BY 1
"""

class HikariFeatureTrainer:
    def __init__(self, pg_dsn: str = PG_DSN, output_dir: Path = OUTPUT_DIR):
        self.pg_dsn     = pg_dsn
        self.output_dir = output_dir
        self._pool: asyncpg.Pool | None = None

    # ── pool ───────────────────────────────────────────────────────
    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(
            self.pg_dsn, min_size=4, max_size=16,
            max_inactive_connection_lifetime=300,
            command_timeout=120,
        )
        logger.info("asyncpg pool created (Hikari-style)")

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()

    # ── data pull ─────────────────────────────────────────────────
    async def pull_raw(self, lookback_days: int = 30) -> pl.DataFrame:
        end_ts   = datetime.now(timezone.utc)
        start_ts = end_ts - timedelta(days=lookback_days)
        assert self._pool, "Call connect() first"
        rows = await self._pool.fetch(FEATURE_QUERY, start_ts, end_ts)
        if not rows:
            return pl.DataFrame()
        return pl.from_records(
            [dict(r) for r in rows],
            schema={
                "hour_bucket":    pl.Datetime,
                "tenant_id":      pl.Utf8,
                "severity_id":    pl.Int32,
                "class_uid":      pl.Int32,
                "event_count":    pl.Int64,
                "unique_src_ips": pl.Int64,
                "avg_risk":       pl.Float64,
                "max_risk":       pl.Float64,
            },
        )

    # ── feature engineering ────────────────────────────────────────
    @staticmethod
    def engineer(df: pl.DataFrame) -> pl.DataFrame:
        if df.is_empty():
            return df
        # Total counts per tenant-hour (flatten class/severity)
        base = (
            df.group_by(["hour_bucket", "tenant_id"])
            .agg([
                pl.sum("event_count").alias("total_events"),
                pl.sum("unique_src_ips").alias("total_src_ips"),
                pl.max("max_risk").alias("max_risk"),
                pl.mean("avg_risk").alias("avg_risk"),
            ])
            .sort("hour_bucket")
        )
        # Rolling windows per tenant
        dfs: List[pl.DataFrame] = []
        for tenant, grp in base.partition_by("tenant_id", as_dict=True).items():
            grp = grp.sort("hour_bucket")
            for w in WINDOW_HOURS:
                grp = grp.with_columns([
                    pl.col("total_events")
                      .rolling_sum(window_size=w)
                      .alias(f"events_{w}h"),
                    pl.col("total_src_ips")
                      .rolling_sum(window_size=w)
                      .alias(f"src_ips_{w}h"),
                ])
            dfs.append(grp)
        return pl.concat(dfs) if dfs else base

    # ── persist ───────────────────────────────────────────────────
    def save(self, df: pl.DataFrame, tag: str = "features") -> Path:
        self.output_dir.mkdir(parents=True, exist_ok=True)
        ts   = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S")
        dest = self.output_dir / f"{tag}_{ts}.parquet"
        df.write_parquet(dest, compression="zstd")
        logger.info("Saved %d rows → %s", len(df), dest)
        return dest

    # ── pipeline ──────────────────────────────────────────────────
    async def run(self, lookback_days: int = 30) -> Dict[str, Any]:
        raw = await self.pull_raw(lookback_days)
        if raw.is_empty():
            logger.warning("No data pulled from DB")
            return {"rows": 0, "output": None}
        features = self.engineer(raw)
        path     = self.save(features)
        return {"rows": len(features), "output": str(path), "columns": features.columns}

# ── entrypoint ────────────────────────────────────────────────────
async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    trainer = HikariFeatureTrainer()
    await trainer.connect()
    try:
        result = await trainer.run()
        print(result)
    finally:
        await trainer.close()

if __name__ == "__main__":
    asyncio.run(main())
