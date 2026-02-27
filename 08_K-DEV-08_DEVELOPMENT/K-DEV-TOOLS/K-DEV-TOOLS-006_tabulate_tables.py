"""
Tabulate-based table formatters for Kubric CLI output.
Module: K-DEV-TOOLS-006

Install: pip install tabulate
"""
from __future__ import annotations

from typing import Any

from tabulate import tabulate

# Default format — can be overridden per call.
DEFAULT_FMT = "grid"

# Supported tabulate formats exposed to callers.
FORMATS = ("grid", "plain", "pipe", "github", "html", "psql", "rst", "tsv")


# ---------------------------------------------------------------------------
# Alerts
# ---------------------------------------------------------------------------
def format_alerts(
    alerts: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of alert dicts as a tabulate table string."""
    headers = ["ID", "Severity", "Title", "Status", "Asset", "Created"]
    rows = []
    for a in alerts:
        rows.append([
            str(a.get("id", ""))[:12],
            a.get("severity", ""),
            (a.get("title") or "")[:60],
            a.get("status", ""),
            str(a.get("asset_id") or "")[:8],
            str(a.get("created_at") or "")[:16],
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# Tenants
# ---------------------------------------------------------------------------
def format_tenants(
    tenants: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of tenant dicts as a tabulate table string."""
    headers = ["ID", "Name", "Email", "Plan", "Status", "HLE Units", "Created"]
    rows = []
    for t in tenants:
        rows.append([
            str(t.get("id", ""))[:36],
            (t.get("name") or "")[:40],
            (t.get("email") or "")[:40],
            t.get("plan", ""),
            t.get("status", ""),
            t.get("hle_units", ""),
            str(t.get("created_at") or "")[:10],
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# Vulnerabilities
# ---------------------------------------------------------------------------
def format_vulnerabilities(
    vulns: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of vulnerability dicts as a tabulate table string."""
    headers = ["CVE ID", "CVSS v3", "EPSS", "Exploitable", "Package", "Version", "Severity"]
    rows = []
    for v in vulns:
        rows.append([
            v.get("cve_id", ""),
            v.get("cvss_v3", ""),
            v.get("epss_score", ""),
            "YES" if v.get("exploit_available") else "no",
            v.get("affected_package", ""),
            v.get("affected_version", ""),
            v.get("severity", ""),
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# Patch jobs
# ---------------------------------------------------------------------------
def format_patch_jobs(
    jobs: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of patch job dicts as a tabulate table string."""
    headers = ["ID", "Asset ID", "Status", "CVEs", "Initiated By", "Scheduled", "Completed"]
    rows = []
    for j in jobs:
        cve_list = j.get("cve_ids", [])
        cve_str = ", ".join(cve_list[:3]) + ("…" if len(cve_list) > 3 else "")
        rows.append([
            str(j.get("id", ""))[:12],
            str(j.get("asset_id", ""))[:8],
            j.get("status", ""),
            cve_str,
            j.get("initiated_by", ""),
            str(j.get("scheduled_at") or "")[:16],
            str(j.get("completed_at") or "")[:16],
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# SLA report
# ---------------------------------------------------------------------------
def format_sla_report(
    report: dict[str, Any],
    fmt: str = DEFAULT_FMT,
) -> str:
    """
    Format an SLA report dict as a key/value tabulate table.
    `report` is expected to have keys: tenant_id, period_start, period_end,
    p1_sla_met, p1_count, p2_sla_met, p2_count, avg_mttr_hours,
    detection_rate_pct, fp_rate_pct.
    """
    rows = [
        ["Tenant ID",           report.get("tenant_id", "")],
        ["Period Start",        report.get("period_start", "")],
        ["Period End",          report.get("period_end", "")],
        ["P1 SLA (4h)",         "MET" if report.get("p1_sla_met") else "BREACHED"],
        ["P1 Ticket Count",     report.get("p1_count", 0)],
        ["P2 SLA (8h)",         "MET" if report.get("p2_sla_met") else "BREACHED"],
        ["P2 Ticket Count",     report.get("p2_count", 0)],
        ["MTTR (avg hours)",    report.get("avg_mttr_hours", "N/A")],
        ["Detection Rate %",    report.get("detection_rate_pct", "N/A")],
        ["False Positive Rate %", report.get("fp_rate_pct", "N/A")],
        ["Alerts Total",        report.get("alerts_total", "N/A")],
        ["Alerts Closed",       report.get("alerts_closed", "N/A")],
    ]
    return tabulate(rows, headers=["Metric", "Value"], tablefmt=fmt)


# ---------------------------------------------------------------------------
# Assets
# ---------------------------------------------------------------------------
def format_assets(
    assets: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of asset dicts as a tabulate table string."""
    headers = ["ID", "Hostname", "IP", "OS", "Type", "Tier", "Last Seen", "Active"]
    rows = []
    for a in assets:
        rows.append([
            str(a.get("id", ""))[:12],
            (a.get("hostname") or "")[:30],
            a.get("ip_address", ""),
            a.get("os_family", ""),
            a.get("asset_type", ""),
            a.get("criticality_tier", ""),
            str(a.get("last_seen_at") or "")[:16],
            "yes" if a.get("is_active") else "no",
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# Agent decisions
# ---------------------------------------------------------------------------
def format_agent_decisions(
    decisions: list[dict[str, Any]],
    fmt: str = DEFAULT_FMT,
) -> str:
    """Format a list of agent decision history rows."""
    headers = ["Decision ID", "Persona", "Type", "SSVC", "Confidence", "Model", "Exec ms", "Created"]
    rows = []
    for d in decisions:
        rows.append([
            str(d.get("decision_id", ""))[:12],
            d.get("agent_persona", ""),
            d.get("decision_type", ""),
            d.get("ssvc_outcome", ""),
            f"{d.get('confidence_score', 0.0):.2f}",
            d.get("model_name", ""),
            d.get("execution_ms", ""),
            str(d.get("created_at") or "")[:16],
        ])
    return tabulate(rows, headers=headers, tablefmt=fmt)


# ---------------------------------------------------------------------------
# CLI demo
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    sample_alerts = [
        {
            "id": "abc-123", "severity": "critical",
            "title": "Ransomware detected on prod-01", "status": "open",
            "asset_id": "asset-1", "created_at": "2025-01-15T08:00:00Z",
        },
        {
            "id": "def-456", "severity": "high",
            "title": "Lateral movement via SMB", "status": "in_progress",
            "asset_id": "asset-2", "created_at": "2025-01-15T09:00:00Z",
        },
    ]
    for fmt in ("grid", "github", "pipe", "html"):
        print(f"\n── Format: {fmt} ──")
        print(format_alerts(sample_alerts, fmt=fmt))

    sample_sla = {
        "tenant_id": "tenant-001", "period_start": "2025-01-01", "period_end": "2025-01-31",
        "p1_sla_met": True, "p1_count": 3, "p2_sla_met": False, "p2_count": 12,
        "avg_mttr_hours": 4.7, "detection_rate_pct": 98.2, "fp_rate_pct": 1.8,
        "alerts_total": 1247, "alerts_closed": 1201,
    }
    print("\n── SLA Report (grid) ──")
    print(format_sla_report(sample_sla))
