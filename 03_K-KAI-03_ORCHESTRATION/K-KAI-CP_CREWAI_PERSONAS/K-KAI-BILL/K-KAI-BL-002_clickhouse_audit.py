"""
K-KAI Billing: ClickHouse Audit
Queries ClickHouse for usage analytics (events/day, storage deltas,
ML call rates) to produce billing-period audit summaries.
"""
from __future__ import annotations
import json, logging, os
from dataclasses import dataclass, field
from datetime import date, datetime, timedelta, timezone
from typing import Any, Dict, List, Optional
import clickhouse_connect

logger = logging.getLogger(__name__)
CH_HOST     = os.getenv("CLICKHOUSE_HOST",     "localhost")
CH_PORT     = int(os.getenv("CLICKHOUSE_PORT", "8123"))
CH_DATABASE = os.getenv("CLICKHOUSE_DB",       "kubric")
CH_USER     = os.getenv("CLICKHOUSE_USER",     "default")
CH_PASSWORD = os.getenv("CLICKHOUSE_PASSWORD", "")

@dataclass
class DailyUsage:
    day:         str
    tenant_id:   str
    events:      int
    alerts_high: int
    ml_calls:    int
    storage_gb:  float

@dataclass
class BillingAuditReport:
    tenant_id:     str
    period_start:  str
    period_end:    str
    total_events:  int
    peak_day:      str
    avg_events_day: float
    total_ml_calls: int
    max_storage_gb: float
    daily_breakdown: List[DailyUsage]
    generated_at:   str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

class ClickHouseAuditClient:
    def __init__(self,
                 host: str = CH_HOST, port: int = CH_PORT,
                 database: str = CH_DATABASE, user: str = CH_USER,
                 password: str = CH_PASSWORD):
        self._client = clickhouse_connect.get_client(
            host=host, port=port, database=database,
            username=user, password=password,
        )

    def daily_event_breakdown(self, tenant_id: str, start: date, end: date) -> List[DailyUsage]:
        query = """
            SELECT
                toDate(event_time)                    AS day,
                tenant_id,
                countIf(class_uid IN (2004,4001,4003,4007)) AS events,
                countIf(severity_id >= 4)             AS alerts_high,
                countIf(class_uid = 9999)             AS ml_calls,
                max(storage_bytes) / 1e9              AS storage_gb
            FROM kubric_events
            WHERE tenant_id = {tenant_id:String}
              AND event_time >= {start:Date}
              AND event_time <  {end:Date}
            GROUP BY day, tenant_id
            ORDER BY day
        """
        result = self._client.query(query, parameters={
            "tenant_id": tenant_id,
            "start":     start.isoformat(),
            "end":       (end + timedelta(days=1)).isoformat(),
        })
        rows: List[DailyUsage] = []
        for row in result.result_rows:
            rows.append(DailyUsage(
                day=str(row[0]),
                tenant_id=str(row[1]),
                events=int(row[2]),
                alerts_high=int(row[3]),
                ml_calls=int(row[4]),
                storage_gb=float(row[5]),
            ))
        return rows

    def generate_audit_report(
        self,
        tenant_id: str,
        billing_month: str,        # YYYY-MM
    ) -> BillingAuditReport:
        start = date.fromisoformat(billing_month + "-01")
        if start.month == 12:
            end = date(start.year + 1, 1, 1)
        else:
            end = date(start.year, start.month + 1, 1)

        daily = self.daily_event_breakdown(tenant_id, start, end)
        total_events   = sum(d.events   for d in daily)
        total_ml_calls = sum(d.ml_calls for d in daily)
        max_storage    = max((d.storage_gb for d in daily), default=0.0)
        avg_events_day = total_events / max(len(daily), 1)
        peak_day = max(daily, key=lambda d: d.events).day if daily else ""

        return BillingAuditReport(
            tenant_id=tenant_id,
            period_start=start.isoformat(),
            period_end=(end - timedelta(days=1)).isoformat(),
            total_events=total_events,
            peak_day=peak_day,
            avg_events_day=round(avg_events_day, 1),
            total_ml_calls=total_ml_calls,
            max_storage_gb=round(max_storage, 3),
            daily_breakdown=daily,
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    client = ClickHouseAuditClient()
    try:
        report = client.generate_audit_report("demo-tenant", "2025-01")
        out = {k: v for k, v in vars(report).items() if k != "daily_breakdown"}
        out["days_reported"] = len(report.daily_breakdown)
        print(json.dumps(out, indent=2))
    except Exception as exc:
        print(json.dumps({"error": str(exc), "note": "ClickHouse not reachable in demo mode"}))
