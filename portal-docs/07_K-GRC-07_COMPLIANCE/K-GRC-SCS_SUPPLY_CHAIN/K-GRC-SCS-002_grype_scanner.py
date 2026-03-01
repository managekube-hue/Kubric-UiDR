"""
K-GRC-SCS-002_grype_scanner.py

Grype SBOM vulnerability scanner — executes grype, parses JSON output,
and persists findings to PostgreSQL.
"""
from __future__ import annotations

import asyncio
import json
import os
import subprocess
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

_GRYPE_BIN = os.getenv("GRYPE_BIN", "grype")


@dataclass
class GrypeFinding:
    vulnerability_id: str   # CVE-2024-xxxx
    severity: str           # Critical | High | Medium | Low | Negligible
    package_name: str
    package_version: str
    package_type: str
    fixed_in_version: str
    description: str
    urls: list[str] = field(default_factory=list)


@dataclass
class GrypeScanResult:
    image_ref: str
    schema_version: str
    vulnerabilities: list[GrypeFinding]
    critical_count: int = 0
    high_count: int = 0
    medium_count: int = 0
    low_count: int = 0
    negligible_count: int = 0

    def __post_init__(self) -> None:
        for v in self.vulnerabilities:
            sev = v.severity.lower()
            if sev == "critical":
                self.critical_count += 1
            elif sev == "high":
                self.high_count += 1
            elif sev == "medium":
                self.medium_count += 1
            elif sev == "low":
                self.low_count += 1
            else:
                self.negligible_count += 1


class GrypeSBOMScanner:
    """
    Runs grype against a container image or SBOM file and parses results.
    """

    def __init__(self, grype_bin: str = _GRYPE_BIN) -> None:
        self.grype_bin = grype_bin

    def scan(self, image_or_sbom: str, sbom_format: bool = False) -> GrypeScanResult:
        """
        Execute grype on an image reference or SBOM file.
        If sbom_format=True, passes sbom: prefix to grype.
        Returns a GrypeScanResult.
        """
        source = f"sbom:{image_or_sbom}" if sbom_format else image_or_sbom

        log.info("grype_scan_start", source=source)
        try:
            proc = subprocess.run(
                [self.grype_bin, source, "-o", "json", "--quiet"],
                capture_output=True,
                text=True,
                timeout=300,
            )
        except FileNotFoundError:
            raise RuntimeError(
                f"grype not found at '{self.grype_bin}'. "
                "Install via: curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh"
            )

        if proc.returncode not in (0, 1):
            raise RuntimeError(
                f"grype exited {proc.returncode}: {proc.stderr[:400]}"
            )

        return self._parse(image_or_sbom, proc.stdout)

    def _parse(self, image_ref: str, json_output: str) -> GrypeScanResult:
        """Parse grype JSON output into a GrypeScanResult."""
        try:
            data: dict[str, Any] = json.loads(json_output)
        except json.JSONDecodeError as exc:
            raise ValueError(f"grype output is not valid JSON: {exc}") from exc

        schema_version = (data.get("descriptor", {}) or {}).get("version", "unknown")
        raw_matches: list[dict] = data.get("matches", [])

        findings: list[GrypeFinding] = []
        for match in raw_matches:
            vuln = match.get("vulnerability", {}) or {}
            artifact = match.get("artifact", {}) or {}
            related = match.get("relatedVulnerabilities", []) or []

            fix_versions = vuln.get("fix", {}).get("versions", [])
            fixed_in = fix_versions[0] if fix_versions else ""

            urls = list(vuln.get("urls", []))
            for rv in related:
                urls.extend(rv.get("urls", []))

            findings.append(GrypeFinding(
                vulnerability_id=vuln.get("id", "UNKNOWN"),
                severity=vuln.get("severity", "Unknown"),
                package_name=artifact.get("name", ""),
                package_version=artifact.get("version", ""),
                package_type=artifact.get("type", ""),
                fixed_in_version=fixed_in,
                description=vuln.get("description", ""),
                urls=list(dict.fromkeys(urls)),  # deduplicate while preserving order
            ))

        result = GrypeScanResult(
            image_ref=image_ref,
            schema_version=schema_version,
            vulnerabilities=findings,
        )
        log.info(
            "grype_scan_complete",
            source=image_ref,
            total=len(findings),
            critical=result.critical_count,
            high=result.high_count,
            medium=result.medium_count,
            low=result.low_count,
        )
        return result

    async def save_results(
        self,
        result: GrypeScanResult,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> int:
        """
        Upsert all findings from a GrypeScanResult into the grype_findings table.
        Returns the number of rows inserted/updated.
        """
        count = 0
        for finding in result.vulnerabilities:
            await db_pool.execute(
                """
                INSERT INTO grype_findings
                    (tenant_id, image_ref, vulnerability_id, severity,
                     package_name, package_version, package_type,
                     fixed_in_version, description, urls, scanned_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,NOW())
                ON CONFLICT (tenant_id, image_ref, vulnerability_id, package_name, package_version)
                DO UPDATE SET
                    severity         = EXCLUDED.severity,
                    fixed_in_version = EXCLUDED.fixed_in_version,
                    description      = EXCLUDED.description,
                    scanned_at       = NOW()
                """,
                tenant_id,
                result.image_ref,
                finding.vulnerability_id,
                finding.severity,
                finding.package_name,
                finding.package_version,
                finding.package_type,
                finding.fixed_in_version,
                finding.description,
                json.dumps(finding.urls),
            )
            count += 1

        log.info(
            "grype_findings_saved",
            tenant_id=tenant_id,
            image_ref=result.image_ref,
            count=count,
        )
        return count

    async def scan_and_save(
        self,
        image_ref: str,
        tenant_id: str,
        db_pool: asyncpg.Pool,
        sbom_format: bool = False,
    ) -> GrypeScanResult:
        """Convenience pipeline: scan then persist results."""
        result = self.scan(image_ref, sbom_format=sbom_format)
        await self.save_results(result, tenant_id, db_pool)
        return result


async def main() -> None:
    import sys

    image_ref = os.getenv("GRYPE_TARGET", "alpine:3.18")
    tenant_id = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")

    scanner = GrypeSBOMScanner()

    print(f"Scanning {image_ref}...")
    try:
        result = scanner.scan(image_ref)
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)

    print(f"Found {len(result.vulnerabilities)} vulnerabilities:")
    print(f"  Critical: {result.critical_count}")
    print(f"  High:     {result.high_count}")
    print(f"  Medium:   {result.medium_count}")
    print(f"  Low:      {result.low_count}")

    db_pool = await asyncpg.create_pool(db_url)
    try:
        await scanner.save_results(result, tenant_id, db_pool)
        print("Findings saved to PostgreSQL.")
    finally:
        await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
