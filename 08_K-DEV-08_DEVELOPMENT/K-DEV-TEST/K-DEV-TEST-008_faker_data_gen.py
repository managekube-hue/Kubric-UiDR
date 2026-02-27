"""
Mass test data generator using Faker and asyncpg.
Module: K-DEV-TEST-008

Install: pip install faker asyncpg structlog
Usage:
    python K-DEV-TEST-008_faker_data_gen.py \\
        --tenants 10 \\
        --users 50 \\
        --alerts 1000 \\
        --vulns 200 \\
        --patch-jobs 100 \\
        --db-url postgresql://kubric:kubric@localhost:5432/kubric
"""
from __future__ import annotations

import argparse
import asyncio
import logging
import random
import sys
from datetime import datetime, timedelta, timezone
from uuid import uuid4

import asyncpg
import structlog

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
structlog.configure(
    processors=[
        structlog.stdlib.add_log_level,
        structlog.dev.ConsoleRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(__name__)

try:
    from faker import Faker
except ImportError:
    print("ERROR: pip install faker", file=sys.stderr)
    sys.exit(1)

_fake = Faker()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


def _past(days: int) -> datetime:
    return _utcnow() - timedelta(days=random.randint(0, days))


SEVERITIES = ["critical", "high", "medium", "low", "info"]
SEVERITY_WEIGHTS = [0.05, 0.15, 0.40, 0.25, 0.15]

OS_FAMILIES = ["linux", "windows", "macos", "cloud_vm", "container"]
PLANS = ["starter", "growth", "professional", "enterprise"]
STATUSES = ["open", "in_progress", "resolved", "closed", "false_positive"]
ASSET_TYPES = ["server", "workstation", "cloud_vm", "container", "network"]


# ---------------------------------------------------------------------------
# Tenant generator
# ---------------------------------------------------------------------------
async def generate_tenants(db_pool: asyncpg.Pool, count: int = 100) -> list[str]:
    """Insert `count` tenants, return list of their UUIDs."""
    tenant_ids: list[str] = []
    records = []
    for _ in range(count):
        tid = str(uuid4())
        tenant_ids.append(tid)
        records.append((
            tid,
            _fake.company(),
            _fake.company_email(),
            random.choice(PLANS),
            "active",
            _past(365),
            _past(30),
        ))

    async with db_pool.acquire() as conn:
        await conn.executemany(
            """
            INSERT INTO tenants (id, name, email, plan, status, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            ON CONFLICT (email) DO NOTHING
            """,
            records,
        )
    log.info("tenants generated", count=len(records))
    return tenant_ids


# ---------------------------------------------------------------------------
# User generator
# ---------------------------------------------------------------------------
async def generate_users(
    db_pool: asyncpg.Pool, tenant_ids: list[str], count: int = 500
) -> list[str]:
    """Insert `count` users distributed across tenants."""
    ROLES = ["kubric:admin", "kubric:analyst", "kubric:readonly", "kubric:responder"]
    user_ids: list[str] = []
    records = []
    for _ in range(count):
        uid = str(uuid4())
        user_ids.append(uid)
        records.append((
            uid,
            random.choice(tenant_ids),
            _fake.email(),
            random.choice(ROLES),
            True,
            _past(365),
        ))

    async with db_pool.acquire() as conn:
        await conn.executemany(
            """
            INSERT INTO users (id, tenant_id, email, role, is_active, created_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT DO NOTHING
            """,
            records,
        )
    log.info("users generated", count=len(records))
    return user_ids


# ---------------------------------------------------------------------------
# Asset generator
# ---------------------------------------------------------------------------
async def generate_assets(
    db_pool: asyncpg.Pool, tenant_ids: list[str], count: int = 500
) -> list[str]:
    """Insert `count` assets, return list of UUIDs."""
    asset_ids: list[str] = []
    records = []
    for _ in range(count):
        aid = str(uuid4())
        asset_ids.append(aid)
        hostname = _fake.hostname()
        records.append((
            aid,
            random.choice(tenant_ids),
            hostname,
            f"{hostname}.internal",
            _fake.ipv4_private(),
            random.choice(OS_FAMILIES),
            random.choice(ASSET_TYPES),
            random.randint(1, 5),
            _past(1),
            True,
            _past(365),
            _past(7),
        ))

    async with db_pool.acquire() as conn:
        await conn.executemany(
            """
            INSERT INTO assets
                (id, tenant_id, hostname, fqdn, ip_address, os_family, asset_type,
                 criticality_tier, last_seen_at, is_active, created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5::inet,$6,$7,$8,$9,$10,$11,$12)
            ON CONFLICT DO NOTHING
            """,
            records,
        )
    log.info("assets generated", count=len(records))
    return asset_ids


# ---------------------------------------------------------------------------
# Alert generator
# ---------------------------------------------------------------------------
async def generate_alerts(
    db_pool: asyncpg.Pool,
    tenant_ids: list[str],
    asset_ids: list[str],
    count: int = 10_000,
) -> list[str]:
    """Insert `count` alerts with weighted severity distribution."""
    SOURCES = ["edr", "siem", "vuln_scanner", "kai_ml", "nids", "firewall"]
    alert_ids: list[str] = []
    records = []

    for _ in range(count):
        aid = str(uuid4())
        alert_ids.append(aid)
        created = _past(90)
        severity = random.choices(SEVERITIES, weights=SEVERITY_WEIGHTS)[0]
        status = random.choice(STATUSES)
        closed_at = created + timedelta(hours=random.uniform(0.5, 120)) if status == "closed" else None
        records.append((
            aid,
            random.choice(tenant_ids),
            severity,
            _fake.sentence(nb_words=8),
            _fake.paragraph(),
            status,
            random.choice(SOURCES),
            random.choice(asset_ids) if asset_ids else None,
            created,
            closed_at,
        ))

    # batch insert in chunks of 1000
    async with db_pool.acquire() as conn:
        for i in range(0, len(records), 1000):
            chunk = records[i : i + 1000]
            await conn.executemany(
                """
                INSERT INTO kai_alerts
                    (id, tenant_id, severity, title, description, status, source,
                     asset_id, created_at, closed_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
                ON CONFLICT DO NOTHING
                """,
                chunk,
            )
    log.info("alerts generated", count=len(records))
    return alert_ids


# ---------------------------------------------------------------------------
# Vulnerability generator
# ---------------------------------------------------------------------------
async def generate_vulnerabilities(
    db_pool: asyncpg.Pool, asset_ids: list[str], count: int = 500
) -> list[str]:
    """Insert `count` vulnerability findings."""
    vuln_ids: list[str] = []
    records = []
    for _ in range(count):
        vid = str(uuid4())
        vuln_ids.append(vid)
        cvss = round(random.uniform(4.0, 10.0), 1)
        records.append((
            vid,
            random.choice(asset_ids) if asset_ids else str(uuid4()),
            f"CVE-{random.randint(2019, 2025)}-{random.randint(1000, 99999)}",
            cvss,
            round(random.uniform(0.001, 0.95), 4),
            cvss >= 9.0,
            random.choice([True, False]),
            _fake.word(),
            f"{random.randint(1,9)}.{random.randint(0,20)}.{random.randint(0,5)}",
            "open",
            _past(180),
        ))

    async with db_pool.acquire() as conn:
        await conn.executemany(
            """
            INSERT INTO kai_vulnerabilities
                (id, asset_id, cve_id, cvss_v3, epss_score, exploit_available,
                 patch_available, affected_package, affected_version, status, first_seen_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
            ON CONFLICT DO NOTHING
            """,
            records,
        )
    log.info("vulnerabilities generated", count=len(records))
    return vuln_ids


# ---------------------------------------------------------------------------
# Patch job generator
# ---------------------------------------------------------------------------
async def generate_patch_jobs(
    db_pool: asyncpg.Pool,
    tenant_ids: list[str],
    asset_ids: list[str],
    count: int = 200,
) -> list[str]:
    """Insert `count` patch jobs."""
    STATUSES_PATCH = ["pending", "scheduled", "running", "completed", "failed"]
    job_ids: list[str] = []
    records = []
    for _ in range(count):
        jid = str(uuid4())
        job_ids.append(jid)
        status = random.choice(STATUSES_PATCH)
        cve_ids_json = [
            f"CVE-{random.randint(2020,2025)}-{random.randint(1000,99999)}"
            for _ in range(random.randint(1, 5))
        ]
        scheduled = _past(30)
        started = scheduled + timedelta(minutes=random.randint(1, 60)) if status in ("running", "completed", "failed") else None
        completed = started + timedelta(minutes=random.randint(5, 120)) if status in ("completed", "failed") and started else None
        records.append((
            jid,
            random.choice(tenant_ids),
            random.choice(asset_ids) if asset_ids else str(uuid4()),
            status,
            cve_ids_json,
            scheduled,
            started,
            completed,
            "kai_agent" if random.random() > 0.3 else "manual",
            _past(30),
        ))

    async with db_pool.acquire() as conn:
        await conn.executemany(
            """
            INSERT INTO kai_patch_jobs
                (id, tenant_id, asset_id, status, cve_ids, scheduled_at,
                 started_at, completed_at, initiated_by, created_at)
            VALUES ($1,$2,$3,$4,$5::text[],$6,$7,$8,$9,$10)
            ON CONFLICT DO NOTHING
            """,
            records,
        )
    log.info("patch_jobs generated", count=len(records))
    return job_ids


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
async def main(args: argparse.Namespace) -> None:
    log.info("connecting to database", db_url=args.db_url)
    pool = await asyncpg.create_pool(args.db_url, min_size=2, max_size=10)

    try:
        tenant_ids = await generate_tenants(pool, args.tenants)
        user_ids = await generate_users(pool, tenant_ids, args.users)  # noqa: F841
        asset_ids = await generate_assets(pool, tenant_ids, args.assets)
        await generate_alerts(pool, tenant_ids, asset_ids, args.alerts)
        await generate_vulnerabilities(pool, asset_ids, args.vulns)
        await generate_patch_jobs(pool, tenant_ids, asset_ids, args.patch_jobs)
        log.info("data generation complete")
    finally:
        await pool.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Kubric test data generator")
    parser.add_argument("--tenants",    type=int, default=100)
    parser.add_argument("--users",      type=int, default=500)
    parser.add_argument("--assets",     type=int, default=500)
    parser.add_argument("--alerts",     type=int, default=10_000)
    parser.add_argument("--vulns",      type=int, default=500)
    parser.add_argument("--patch-jobs", type=int, default=200, dest="patch_jobs")
    parser.add_argument(
        "--db-url",
        default="postgresql://kubric:kubric@localhost:5432/kubric",
        dest="db_url",
    )
    asyncio.run(main(parser.parse_args()))
