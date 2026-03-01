"""
K-GRC-SCS-006_osv_api_check.py

OSV (Open Source Vulnerabilities) API client.
Checks packages against api.osv.dev for known vulnerabilities.
Supports requirements.txt and go.mod scanning.
"""
from __future__ import annotations

import asyncio
import json
import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import asyncpg
import httpx
import structlog

log = structlog.get_logger(__name__)

_OSV_QUERY_URL = "https://api.osv.dev/v1/query"
_OSV_QUERYBATCH_URL = "https://api.osv.dev/v1/querybatch"
_CONCURRENCY = int(os.getenv("OSV_CONCURRENCY", "10"))
_TIMEOUT = float(os.getenv("OSV_TIMEOUT_SECS", "30"))


@dataclass
class PackageRef:
    name: str
    version: str
    ecosystem: str   # PyPI, Go, npm, Maven, ...


@dataclass
class OSVVuln:
    vuln_id: str            # e.g. GHSA-xxxx | CVE-xxxx
    summary: str
    severity: str           # CRITICAL | HIGH | MEDIUM | LOW | UNKNOWN
    affected_versions: list[str] = field(default_factory=list)
    fixed_in: str = ""
    reference_urls: list[str] = field(default_factory=list)


@dataclass
class OSVFinding:
    package: PackageRef
    vulns: list[OSVVuln] = field(default_factory=list)


class OSVClient:
    """Async httpx-based client for the OSV.dev vulnerability API."""

    def __init__(self) -> None:
        self._http = httpx.AsyncClient(timeout=_TIMEOUT)
        self._sem = asyncio.Semaphore(_CONCURRENCY)

    async def close(self) -> None:
        await self._http.aclose()

    # ------------------------------------------------------------------ #
    #  Core API calls
    # ------------------------------------------------------------------ #

    async def query(self, package: PackageRef) -> list[OSVVuln]:
        """Query OSV for a single package version."""
        body = {
            "version": package.version,
            "package": {
                "name": package.name,
                "ecosystem": package.ecosystem,
            },
        }
        async with self._sem:
            try:
                resp = await self._http.post(_OSV_QUERY_URL, json=body)
                resp.raise_for_status()
            except httpx.HTTPError as exc:
                log.warning("osv_query_failed", package=package.name, error=str(exc))
                return []

        data = resp.json()
        return self._parse_response(data)

    async def querybatch(self, packages: list[PackageRef]) -> list[list[OSVVuln]]:
        """Batch query OSV for multiple packages."""
        queries = [
            {
                "version": p.version,
                "package": {"name": p.name, "ecosystem": p.ecosystem},
            }
            for p in packages
        ]
        body = {"queries": queries}
        async with self._sem:
            try:
                resp = await self._http.post(_OSV_QUERYBATCH_URL, json=body)
                resp.raise_for_status()
            except httpx.HTTPError as exc:
                log.warning("osv_querybatch_failed", count=len(packages), error=str(exc))
                return [[] for _ in packages]

        results = resp.json().get("results", [])
        return [self._parse_response(r) for r in results]

    def _parse_response(self, data: dict) -> list[OSVVuln]:
        """Parse OSV API response into OSVVuln objects."""
        vulns: list[OSVVuln] = []
        for v in data.get("vulns", []):
            severity = "UNKNOWN"
            for sev in v.get("severity", []):
                if sev.get("type") == "CVSS_V3":
                    # Derive label from score
                    score_str = sev.get("score", "0")
                    try:
                        score = float(score_str)
                        if score >= 9.0:
                            severity = "CRITICAL"
                        elif score >= 7.0:
                            severity = "HIGH"
                        elif score >= 4.0:
                            severity = "MEDIUM"
                        else:
                            severity = "LOW"
                    except ValueError:
                        severity = score_str.upper()

            affected_versions: list[str] = []
            fixed_in = ""
            for aff in v.get("affected", []):
                for rng in aff.get("ranges", []):
                    for event in rng.get("events", []):
                        if "introduced" in event:
                            affected_versions.append(event["introduced"])
                        if "fixed" in event:
                            fixed_in = event["fixed"]
                for ver in aff.get("versions", []):
                    if ver not in affected_versions:
                        affected_versions.append(ver)

            ref_urls = [r["url"] for r in v.get("references", []) if "url" in r]

            vulns.append(OSVVuln(
                vuln_id=v.get("id", "UNKNOWN"),
                summary=v.get("summary", ""),
                severity=severity,
                affected_versions=affected_versions[:20],  # cap at 20
                fixed_in=fixed_in,
                reference_urls=ref_urls[:5],
            ))
        return vulns

    # ------------------------------------------------------------------ #
    #  Batch helper: concurrently check a list of packages
    # ------------------------------------------------------------------ #

    async def batch_check(self, packages: list[PackageRef]) -> list[OSVFinding]:
        """Check all packages concurrently using querybatch in chunks of 50."""
        findings: list[OSVFinding] = []
        chunk_size = 50
        for i in range(0, len(packages), chunk_size):
            chunk = packages[i : i + chunk_size]
            results = await self.querybatch(chunk)
            for pkg, vulns in zip(chunk, results):
                findings.append(OSVFinding(package=pkg, vulns=vulns))
        return findings

    # ------------------------------------------------------------------ #
    #  File scanners
    # ------------------------------------------------------------------ #

    def parse_requirements_txt(self, path: str) -> list[PackageRef]:
        """Parse a requirements.txt and return package references."""
        packages: list[PackageRef] = []
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.split("#", 1)[0].strip()
                if not line or line.startswith("-"):
                    continue
                match = re.match(r"^([A-Za-z0-9_.\-]+)==([^\s;]+)", line)
                if match:
                    packages.append(PackageRef(
                        name=match.group(1),
                        version=match.group(2),
                        ecosystem="PyPI",
                    ))
        return packages

    def parse_go_mod(self, path: str) -> list[PackageRef]:
        """Parse a go.mod and return package references."""
        packages: list[PackageRef] = []
        in_require = False
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                stripped = line.strip()
                if stripped.startswith("require ("):
                    in_require = True
                    continue
                if in_require and stripped == ")":
                    in_require = False
                    continue
                # Single-line: require github.com/foo/bar v1.2.3
                m = re.match(r"^require\s+(\S+)\s+(v[\d.][^\s]+)", stripped)
                if m:
                    packages.append(PackageRef(
                        name=m.group(1),
                        version=m.group(2).lstrip("v"),
                        ecosystem="Go",
                    ))
                    continue
                if in_require:
                    m = re.match(r"^(\S+)\s+(v[\d.][^\s]+)", stripped)
                    if m and not stripped.startswith("//"):
                        packages.append(PackageRef(
                            name=m.group(1),
                            version=m.group(2).lstrip("v"),
                            ecosystem="Go",
                        ))
        return packages

    async def scan_requirements_txt(
        self,
        path: str,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> list[OSVFinding]:
        """Scan requirements.txt, save findings, return results."""
        packages = self.parse_requirements_txt(path)
        log.info("osv_scan_requirements", path=path, count=len(packages))
        findings = await self.batch_check(packages)
        await self.save_findings(findings, tenant_id, db_pool, source=path)
        return findings

    async def scan_go_mod(
        self,
        path: str,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> list[OSVFinding]:
        """Scan go.mod, save findings, return results."""
        packages = self.parse_go_mod(path)
        log.info("osv_scan_go_mod", path=path, count=len(packages))
        findings = await self.batch_check(packages)
        await self.save_findings(findings, tenant_id, db_pool, source=path)
        return findings

    # ------------------------------------------------------------------ #
    #  Persistence
    # ------------------------------------------------------------------ #

    async def save_findings(
        self,
        findings: list[OSVFinding],
        tenant_id: str,
        db_pool: asyncpg.Pool,
        source: str = "",
    ) -> int:
        """Upsert OSV findings into the osv_findings table."""
        count = 0
        for finding in findings:
            for vuln in finding.vulns:
                await db_pool.execute(
                    """
                    INSERT INTO osv_findings
                        (tenant_id, source_file, ecosystem, package_name, package_version,
                         vuln_id, severity, summary, fixed_in, reference_urls, scanned_at)
                    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,NOW())
                    ON CONFLICT (tenant_id, ecosystem, package_name, package_version, vuln_id)
                    DO UPDATE SET
                        severity       = EXCLUDED.severity,
                        summary        = EXCLUDED.summary,
                        fixed_in       = EXCLUDED.fixed_in,
                        scanned_at     = NOW()
                    """,
                    tenant_id,
                    source,
                    finding.package.ecosystem,
                    finding.package.name,
                    finding.package.version,
                    vuln.vuln_id,
                    vuln.severity,
                    vuln.summary,
                    vuln.fixed_in,
                    json.dumps(vuln.reference_urls),
                )
                count += 1

        log.info(
            "osv_findings_saved",
            tenant_id=tenant_id,
            source=source,
            findings=count,
        )
        return count


async def main() -> None:
    import sys

    target = os.getenv("OSV_TARGET_FILE", "requirements.txt")
    tenant_id = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")

    client = OSVClient()
    db_pool = await asyncpg.create_pool(db_url)
    try:
        if not os.path.exists(target):
            print(f"File not found: {target}", file=sys.stderr)
            sys.exit(1)

        if target.endswith("go.mod"):
            findings = await client.scan_go_mod(target, tenant_id, db_pool)
        else:
            findings = await client.scan_requirements_txt(target, tenant_id, db_pool)

        vuln_count = sum(len(f.vulns) for f in findings)
        print(f"Scanned {len(findings)} packages, found {vuln_count} vulnerabilities.")
        for f in findings:
            if f.vulns:
                print(f"  {f.package.name}=={f.package.version} "
                      f"[{f.package.ecosystem}]: {len(f.vulns)} vulns")
                for v in f.vulns[:3]:
                    print(f"    {v.vuln_id} [{v.severity}] {v.summary[:80]}")
    finally:
        await client.close()
        await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
