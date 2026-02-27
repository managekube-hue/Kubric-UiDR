"""
Click CLI for Kubric platform management.
Module: K-DEV-TOOLS-001

Install: pip install click httpx rich
Usage:
    kubric tenants list
    kubric alerts list --tenant-id <UUID> --severity critical --limit 25
    kubric health check
    kubric version
"""
from __future__ import annotations

import json
import os
import sys
from typing import Optional

import click
import httpx
from rich.console import Console
from rich.table import Table

console = Console()

API_URL = os.environ.get("KUBRIC_API_URL", "http://localhost:8080")
API_TOKEN = os.environ.get("KUBRIC_API_TOKEN", "")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _client() -> httpx.Client:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if API_TOKEN:
        headers["Authorization"] = f"Bearer {API_TOKEN}"
    return httpx.Client(base_url=API_URL, headers=headers, timeout=30.0)


def _tenant_header(tenant_id: Optional[str]) -> dict[str, str]:
    return {"X-Tenant-ID": tenant_id} if tenant_id else {}


def _die(msg: str, code: int = 1) -> None:
    console.print(f"[bold red]ERROR:[/] {msg}")
    sys.exit(code)


# ---------------------------------------------------------------------------
# Root group
# ---------------------------------------------------------------------------
@click.group()
@click.version_option("0.1.0", prog_name="kubric")
def kubric() -> None:
    """Kubric platform management CLI."""


# ---------------------------------------------------------------------------
# tenants
# ---------------------------------------------------------------------------
@kubric.group()
def tenants() -> None:
    """Manage tenants."""


@tenants.command("list")
@click.option("--output", "-o", default="table", type=click.Choice(["table", "json"]))
def tenants_list(output: str) -> None:
    """List all tenants."""
    with _client() as c:
        resp = c.get("/api/v1/admin/tenants")
    if resp.status_code != 200:
        _die(f"API returned {resp.status_code}: {resp.text}")
    data = resp.json().get("tenants", [])
    if output == "json":
        click.echo(json.dumps(data, indent=2))
        return
    t = Table(title="Tenants", highlight=True)
    for col in ("ID", "Name", "Plan", "Status", "Created"):
        t.add_column(col)
    for row in data:
        t.add_row(
            row.get("id", ""),
            row.get("name", ""),
            row.get("plan", ""),
            row.get("status", ""),
            row.get("created_at", "")[:10],
        )
    console.print(t)


@tenants.command("create")
@click.option("--name", required=True, help="Company name")
@click.option("--email", required=True, help="Admin email")
@click.option("--plan", default="professional", help="Tier: starter|growth|professional|enterprise")
def tenants_create(name: str, email: str, plan: str) -> None:
    """Create a new tenant."""
    payload = {"name": name, "email": email, "plan": plan}
    with _client() as c:
        resp = c.post("/api/v1/admin/tenants", json=payload)
    if resp.status_code not in (200, 201):
        _die(f"API returned {resp.status_code}: {resp.text}")
    tenant = resp.json()
    console.print(f"[green]Created tenant[/] {tenant.get('id')}")


@tenants.command("delete")
@click.argument("tenant_id")
@click.option("--yes", is_flag=True, help="Skip confirmation prompt")
def tenants_delete(tenant_id: str, yes: bool) -> None:
    """Delete a tenant by ID."""
    if not yes:
        click.confirm(f"Delete tenant {tenant_id}?", abort=True)
    with _client() as c:
        resp = c.delete(f"/api/v1/admin/tenants/{tenant_id}")
    if resp.status_code not in (200, 204):
        _die(f"API returned {resp.status_code}: {resp.text}")
    console.print(f"[green]Deleted tenant[/] {tenant_id}")


# ---------------------------------------------------------------------------
# alerts
# ---------------------------------------------------------------------------
@kubric.group()
def alerts() -> None:
    """Manage alerts."""


@alerts.command("list")
@click.option("--tenant-id", "-t", default=os.environ.get("KUBRIC_TENANT_ID"), help="Tenant UUID")
@click.option("--severity", help="Filter: critical|high|medium|low|info")
@click.option("--status", help="Filter: open|in_progress|resolved|closed")
@click.option("--limit", default=50, show_default=True, help="Max results")
@click.option("--output", "-o", default="table", type=click.Choice(["table", "json"]))
def alerts_list(
    tenant_id: Optional[str],
    severity: Optional[str],
    status: Optional[str],
    limit: int,
    output: str,
) -> None:
    """List alerts for a tenant."""
    if not tenant_id:
        _die("--tenant-id or KUBRIC_TENANT_ID required")
    params: dict[str, str | int] = {"limit": limit}
    if severity:
        params["severity"] = severity
    if status:
        params["status"] = status
    with _client() as c:
        resp = c.get(
            f"/api/v1/tenants/{tenant_id}/alerts",
            params=params,
            headers=_tenant_header(tenant_id),
        )
    if resp.status_code != 200:
        _die(f"API returned {resp.status_code}: {resp.text}")
    data = resp.json().get("alerts", [])
    if output == "json":
        click.echo(json.dumps(data, indent=2))
        return

    SEVERITY_COLORS = {
        "critical": "bold red",
        "high": "red",
        "medium": "yellow",
        "low": "blue",
        "info": "dim",
    }
    t = Table(title=f"Alerts — tenant {tenant_id}", highlight=True)
    for col in ("ID", "Severity", "Title", "Status", "Created"):
        t.add_column(col)
    for row in data:
        sev = row.get("severity", "")
        color = SEVERITY_COLORS.get(sev, "white")
        t.add_row(
            row.get("id", "")[:8] + "...",
            f"[{color}]{sev}[/]",
            (row.get("title", "") or "")[:60],
            row.get("status", ""),
            (row.get("created_at", "") or "")[:16],
        )
    console.print(t)


# ---------------------------------------------------------------------------
# health
# ---------------------------------------------------------------------------
@kubric.group()
def health() -> None:
    """Health checks."""


@health.command("check")
@click.option("--api-url", default=API_URL, help="Base API URL")
def health_check(api_url: str) -> None:
    """Check health of all Kubric services."""
    endpoints = [
        ("api-gateway", f"{api_url}/healthz"),
        ("kai-core",    f"{api_url}/kai/healthz"),
        ("noc-api",     f"{api_url}/noc/healthz"),
    ]
    t = Table(title="Service Health", highlight=True)
    t.add_column("Service")
    t.add_column("Status")
    t.add_column("Latency")

    import time
    for name, url in endpoints:
        try:
            start = time.monotonic()
            r = httpx.get(url, timeout=5.0)
            latency = (time.monotonic() - start) * 1000
            status_str = "[green]OK[/]" if r.status_code == 200 else f"[red]HTTP {r.status_code}[/]"
            t.add_row(name, status_str, f"{latency:.0f} ms")
        except Exception as exc:
            t.add_row(name, f"[red]DOWN[/]", f"err: {exc}")
    console.print(t)


if __name__ == "__main__":
    kubric()
