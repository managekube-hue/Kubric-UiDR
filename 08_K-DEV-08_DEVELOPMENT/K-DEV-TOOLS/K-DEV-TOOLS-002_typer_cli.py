"""
Typer CLI for Kubric platform management (fully type-annotated).
Module: K-DEV-TOOLS-002

Install: pip install typer httpx rich
Usage:
    python K-DEV-TOOLS-002_typer_cli.py tenants professional
    python K-DEV-TOOLS-002_typer_cli.py alerts <tenant_id> --severity critical --limit 25
    python K-DEV-TOOLS-002_typer_cli.py health
"""
from __future__ import annotations

import json
import os
import sys
import time
from typing import Optional

import httpx
import typer
from rich.console import Console
from rich.table import Table

console = Console()
app = typer.Typer(name="kubric", help="Kubric platform management CLI (Typer edition).")

_API_URL = os.environ.get("KUBRIC_API_URL", "http://localhost:8080")
_API_TOKEN = os.environ.get("KUBRIC_API_TOKEN", "")


# ---------------------------------------------------------------------------
# Shared utilities
# ---------------------------------------------------------------------------
def _client() -> httpx.Client:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if _API_TOKEN:
        headers["Authorization"] = f"Bearer {_API_TOKEN}"
    return httpx.Client(base_url=_API_URL, headers=headers, timeout=30.0)


def _die(msg: str, code: int = 1) -> None:
    console.print(f"[bold red]ERROR:[/] {msg}")
    raise typer.Exit(code=code)


# ---------------------------------------------------------------------------
# tenants command
# ---------------------------------------------------------------------------
@app.command()
def tenants(
    action: str = typer.Argument(..., help="Action: list | create | delete"),
    tenant_id: Optional[str] = typer.Option(None, "--tenant-id", "-t", help="Tenant UUID (for delete)"),
    name: Optional[str] = typer.Option(None, help="Tenant name (for create)"),
    email: Optional[str] = typer.Option(None, help="Admin email (for create)"),
    plan: str = typer.Option("professional", help="Billing plan (for create)"),
    output: str = typer.Option("table", "--output", "-o", help="table|json"),
    yes: bool = typer.Option(False, "--yes", help="Skip confirmation for delete"),
) -> None:
    """Manage tenants: list | create | delete."""
    if action == "list":
        with _client() as c:
            resp = c.get("/api/v1/admin/tenants")
        if resp.status_code != 200:
            _die(f"API {resp.status_code}: {resp.text}")
        data = resp.json().get("tenants", [])
        if output == "json":
            typer.echo(json.dumps(data, indent=2))
            return
        t = Table(title="Tenants")
        for col in ("ID", "Name", "Plan", "Status", "Created"):
            t.add_column(col)
        for row in data:
            t.add_row(
                row.get("id", ""),
                row.get("name", ""),
                row.get("plan", ""),
                row.get("status", ""),
                (row.get("created_at") or "")[:10],
            )
        console.print(t)

    elif action == "create":
        if not name or not email:
            _die("--name and --email are required for create")
        with _client() as c:
            resp = c.post("/api/v1/admin/tenants", json={"name": name, "email": email, "plan": plan})
        if resp.status_code not in (200, 201):
            _die(f"API {resp.status_code}: {resp.text}")
        result = resp.json()
        console.print(f"[green]Created tenant[/] {result.get('id')}")

    elif action == "delete":
        if not tenant_id:
            _die("--tenant-id required for delete")
        if not yes:
            typer.confirm(f"Delete tenant {tenant_id}?", abort=True)
        with _client() as c:
            resp = c.delete(f"/api/v1/admin/tenants/{tenant_id}")
        if resp.status_code not in (200, 204):
            _die(f"API {resp.status_code}: {resp.text}")
        console.print(f"[green]Deleted tenant[/] {tenant_id}")

    else:
        _die(f"Unknown action '{action}'. Valid: list, create, delete")


# ---------------------------------------------------------------------------
# alerts command
# ---------------------------------------------------------------------------
@app.command()
def alerts(
    tenant_id: str = typer.Argument(..., help="Tenant UUID"),
    severity: Optional[str] = typer.Option(None, help="Filter: critical|high|medium|low|info"),
    status: Optional[str] = typer.Option(None, help="Filter: open|in_progress|resolved|closed"),
    limit: int = typer.Option(50, help="Max results"),
    output: str = typer.Option("table", "--output", "-o", help="table|json"),
) -> None:
    """List alerts for a tenant."""
    params: dict[str, str | int] = {"limit": limit}
    if severity:
        params["severity"] = severity
    if status:
        params["status"] = status

    with _client() as c:
        resp = c.get(
            f"/api/v1/tenants/{tenant_id}/alerts",
            params=params,
            headers={"X-Tenant-ID": tenant_id},
        )
    if resp.status_code != 200:
        _die(f"API {resp.status_code}: {resp.text}")
    data = resp.json().get("alerts", [])

    if output == "json":
        typer.echo(json.dumps(data, indent=2))
        return

    SEVERITY_COLORS = {
        "critical": "bold red", "high": "red",
        "medium": "yellow", "low": "blue", "info": "dim",
    }
    t = Table(title=f"Alerts — {tenant_id}")
    for col in ("ID", "Severity", "Title", "Status", "Created"):
        t.add_column(col)
    for row in data:
        sev = row.get("severity", "")
        color = SEVERITY_COLORS.get(sev, "white")
        t.add_row(
            (row.get("id") or "")[:8] + "...",
            f"[{color}]{sev}[/]",
            (row.get("title") or "")[:60],
            row.get("status", ""),
            (row.get("created_at") or "")[:16],
        )
    console.print(t)


# ---------------------------------------------------------------------------
# health command
# ---------------------------------------------------------------------------
@app.command()
def health(
    api_url: str = typer.Option(_API_URL, help="Base API URL"),
) -> None:
    """Check health of all Kubric services."""
    endpoints = [
        ("api-gateway", f"{api_url}/healthz"),
        ("kai-core",    f"{api_url}/kai/healthz"),
        ("noc-api",     f"{api_url}/noc/healthz"),
    ]
    t = Table(title="Service Health")
    t.add_column("Service")
    t.add_column("Status")
    t.add_column("Latency")
    all_ok = True
    for name, url in endpoints:
        try:
            start = time.monotonic()
            r = httpx.get(url, timeout=5.0)
            latency = (time.monotonic() - start) * 1000
            ok = r.status_code == 200
            if not ok:
                all_ok = False
            status_str = "[green]OK[/]" if ok else f"[red]HTTP {r.status_code}[/]"
            t.add_row(name, status_str, f"{latency:.0f} ms")
        except Exception as exc:
            all_ok = False
            t.add_row(name, "[red]DOWN[/]", str(exc)[:50])
    console.print(t)
    if not all_ok:
        raise typer.Exit(code=1)


# ---------------------------------------------------------------------------
# version command
# ---------------------------------------------------------------------------
@app.command()
def version() -> None:
    """Print build version."""
    typer.echo("Kubric CLI (Typer) v0.1.0")


if __name__ == "__main__":
    app()
