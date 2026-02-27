"""
Rich-based output utilities for Kubric CLI tools.
Module: K-DEV-TOOLS-003

Install: pip install rich
"""
from __future__ import annotations

import csv
import os
from datetime import datetime
from typing import Any

from rich.console import Console
from rich.panel import Panel
from rich.progress import BarColumn, Progress, SpinnerColumn, TaskProgressColumn, TextColumn, TimeElapsedColumn
from rich.table import Table
from rich.text import Text

console = Console()

# ---------------------------------------------------------------------------
# Severity helpers
# ---------------------------------------------------------------------------
SEVERITY_STYLE: dict[str, str] = {
    "critical": "bold white on red",
    "high":     "bold red",
    "medium":   "bold yellow",
    "low":      "bold blue",
    "info":     "dim",
}


def format_severity(sev: str) -> Text:
    """Return a Rich Text object with color-coded severity badge."""
    style = SEVERITY_STYLE.get(sev.lower(), "white")
    return Text(f" {sev.upper()} ", style=style)


# ---------------------------------------------------------------------------
# Alert table
# ---------------------------------------------------------------------------
def print_alerts_table(alerts: list[dict[str, Any]], title: str = "Alerts") -> None:
    """Print a colored alert table to the console."""
    t = Table(title=title, highlight=True, show_lines=False)
    t.add_column("ID",        style="dim",    no_wrap=True, width=12)
    t.add_column("Severity",  no_wrap=True)
    t.add_column("Title",     max_width=64)
    t.add_column("Status",    style="cyan",   no_wrap=True)
    t.add_column("Asset",     style="dim",    no_wrap=True, width=14)
    t.add_column("Created",   style="dim",    no_wrap=True, width=16)

    for a in alerts:
        aid = str(a.get("id", ""))[:8]
        sev_text = format_severity(a.get("severity", "info"))
        title_val = (a.get("title") or "")[:64]
        status = a.get("status", "")
        asset = str(a.get("asset_id", "") or "")[:8]
        created = str(a.get("created_at", "") or "")[:16]
        t.add_row(aid, sev_text, title_val, status, asset, created)

    console.print(t)


# ---------------------------------------------------------------------------
# Tenant summary panel
# ---------------------------------------------------------------------------
def print_tenant_summary(tenant: dict[str, Any]) -> None:
    """Print a rich panel summarizing a tenant's key metrics."""
    name = tenant.get("name", "Unknown")
    tid = tenant.get("id", "")
    plan = tenant.get("plan", "")
    status = tenant.get("status", "")
    hle = tenant.get("hle_units", "N/A")
    agents = tenant.get("agent_count", "N/A")
    open_alerts = tenant.get("open_alerts", "N/A")
    created = (tenant.get("created_at") or "")[:10]

    lines = [
        f"[bold]ID:[/]          {tid}",
        f"[bold]Plan:[/]        [cyan]{plan}[/]",
        f"[bold]Status:[/]      {'[green]active[/]' if status == 'active' else '[red]' + status + '[/]'}",
        f"[bold]HLE Units:[/]   {hle}",
        f"[bold]Agents:[/]      {agents}",
        f"[bold]Open Alerts:[/] {open_alerts}",
        f"[bold]Created:[/]     {created}",
    ]
    panel = Panel("\n".join(lines), title=f"[bold blue]{name}[/]", expand=False)
    console.print(panel)


# ---------------------------------------------------------------------------
# Deployment progress
# ---------------------------------------------------------------------------
def print_deployment_progress(steps: list[str]) -> None:
    """Show a progress bar while simulating deployment steps."""
    import time
    with Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        BarColumn(),
        TaskProgressColumn(),
        TimeElapsedColumn(),
        console=console,
    ) as progress:
        task = progress.add_task("[cyan]Deploying...", total=len(steps))
        for step in steps:
            progress.update(task, description=f"[cyan]{step}")
            time.sleep(0.05)  # simulate work
            progress.advance(task)
    console.print("[bold green]Deployment complete.[/]")


# ---------------------------------------------------------------------------
# Vulnerability table
# ---------------------------------------------------------------------------
def print_vulns_table(vulns: list[dict[str, Any]], title: str = "Vulnerabilities") -> None:
    t = Table(title=title, highlight=True)
    t.add_column("CVE ID",     style="bold cyan", no_wrap=True)
    t.add_column("CVSS",       no_wrap=True, justify="right")
    t.add_column("EPSS",       no_wrap=True, justify="right")
    t.add_column("Exploitable")
    t.add_column("Package",    style="dim")
    t.add_column("Version",    style="dim")

    for v in vulns:
        cvss = v.get("cvss_v3", 0.0)
        cvss_style = "bold red" if float(cvss) >= 9.0 else "red" if float(cvss) >= 7.0 else "yellow"
        t.add_row(
            v.get("cve_id", ""),
            Text(str(cvss), style=cvss_style),
            str(v.get("epss_score", "")),
            "[red]YES[/]" if v.get("exploit_available") else "[green]no[/]",
            v.get("affected_package", ""),
            v.get("affected_version", ""),
        )
    console.print(t)


# ---------------------------------------------------------------------------
# Export to CSV
# ---------------------------------------------------------------------------
def export_to_csv(data: list[dict[str, Any]], path: str) -> None:
    """Write a list of dicts to a CSV file."""
    if not data:
        console.print("[yellow]No data to export.[/]")
        return
    fieldnames = list(data[0].keys())
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(data)
    console.print(f"[green]Exported {len(data)} rows to[/] {path}")


# ---------------------------------------------------------------------------
# SLA report panel
# ---------------------------------------------------------------------------
def print_sla_report(report: dict[str, Any]) -> None:
    """Print an SLA compliance report panel."""
    lines = [
        f"[bold]Tenant:[/]           {report.get('tenant_id', '')}",
        f"[bold]Period:[/]           {report.get('period_start', '')} → {report.get('period_end', '')}",
        f"[bold]P1 SLA (4h):[/]      {'[green]MET[/]' if report.get('p1_sla_met') else '[red]BREACHED[/]'} — {report.get('p1_count', 0)} tickets",
        f"[bold]P2 SLA (8h):[/]      {'[green]MET[/]' if report.get('p2_sla_met') else '[red]BREACHED[/]'} — {report.get('p2_count', 0)} tickets",
        f"[bold]MTTR (avg):[/]        {report.get('avg_mttr_hours', 'N/A')}h",
        f"[bold]Detection Rate:[/]   {report.get('detection_rate_pct', 'N/A')}%",
        f"[bold]False Positive Rate:[/] {report.get('fp_rate_pct', 'N/A')}%",
    ]
    console.print(Panel("\n".join(lines), title="[bold]SLA Report[/]", expand=False))


if __name__ == "__main__":
    # Demo
    sample_alerts = [
        {"id": "aaa-111-bbb", "severity": "critical", "title": "Ransomware activity detected on prod-01",
         "status": "open", "asset_id": "asset-001", "created_at": "2025-01-15T08:23:00Z"},
        {"id": "aaa-222-ccc", "severity": "high", "title": "Lateral movement via SMB",
         "status": "in_progress", "asset_id": "asset-002", "created_at": "2025-01-15T09:10:00Z"},
        {"id": "aaa-333-ddd", "severity": "medium", "title": "Suspicious PowerShell execution",
         "status": "open", "asset_id": "asset-003", "created_at": "2025-01-15T10:00:00Z"},
    ]
    print_alerts_table(sample_alerts)

    sample_tenant = {
        "id": "00000000-0000-0000-0000-000000000001", "name": "Acme Corp",
        "plan": "enterprise", "status": "active",
        "hle_units": 12.5, "agent_count": 847, "open_alerts": 3,
        "created_at": "2024-06-01T00:00:00Z",
    }
    print_tenant_summary(sample_tenant)

    print_deployment_progress([
        "Apply DB migrations", "Deploy API gateway", "Deploy KAI core",
        "Deploy NOC API", "Run smoke tests",
    ])
