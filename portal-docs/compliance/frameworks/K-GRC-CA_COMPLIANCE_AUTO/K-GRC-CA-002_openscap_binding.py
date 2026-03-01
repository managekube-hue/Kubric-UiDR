"""
K-GRC-CA-002_openscap_binding.py

OpenSCAP XCCDF compliance scanner — runs oscap xccdf eval via subprocess,
parses XML results, saves to PostgreSQL.
"""
from __future__ import annotations

import asyncio
import os
import subprocess
import tempfile
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

# XCCDF result namespace
_NS = {
    "xccdf": "http://checklists.nist.gov/xccdf/1.2",
    "arf":   "http://scap.nist.gov/schema/asset-reporting-format/1.1",
}


@dataclass
class RuleResult:
    rule_id:  str
    title:    str
    result:   str    # pass | fail | notapplicable | notchecked | error
    severity: str    # high | medium | low | unknown


@dataclass
class SCAResult:
    hostname:    str
    profile:     str
    pass_count:  int = 0
    fail_count:  int = 0
    error_count: int = 0
    rules:       list[RuleResult] = field(default_factory=list)


@dataclass
class RemediationItem:
    rule_id:     str
    title:       str
    severity:    str
    description: str


class OpenSCAPScanner:
    """Run OpenSCAP XCCDF evaluations and parse results."""

    def scan_host(self, hostname: str, profile: str, benchmark_path: str) -> SCAResult:
        """
        Execute oscap xccdf eval and return a parsed SCAResult.
        Runs on the local machine (or via SSH if hostname != localhost).
        """
        with tempfile.TemporaryDirectory() as tmpdir:
            results_xml = os.path.join(tmpdir, "results-arf.xml")
            report_html = os.path.join(tmpdir, "report.html")

            cmd = [
                "oscap", "xccdf", "eval",
                "--profile", profile,
                "--results-arf", results_xml,
                "--report", report_html,
                benchmark_path,
            ]

            log.info("openscap_scan_start", hostname=hostname, profile=profile, benchmark=benchmark_path)
            proc = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300,
            )
            # oscap exits with 2 if there are failures, which is expected
            if proc.returncode not in (0, 1, 2):
                raise RuntimeError(
                    f"oscap exited with code {proc.returncode}: {proc.stderr[:500]}"
                )

            return self._parse_arf(results_xml, hostname, profile)

    def _parse_arf(self, arf_path: str, hostname: str, profile: str) -> SCAResult:
        """Parse SCAP ARF results XML and extract per-rule outcomes."""
        result = SCAResult(hostname=hostname, profile=profile)

        if not os.path.exists(arf_path):
            log.warning("openscap_arf_not_found", path=arf_path)
            return result

        tree = ET.parse(arf_path)
        root = tree.getroot()

        # Find all rule-result elements (works for both ARF and plain XCCDF)
        for rr in root.iter():
            if rr.tag.endswith("}rule-result") or rr.tag == "rule-result":
                rule_id = rr.get("idref", "unknown")
                severity = rr.get("severity", "unknown")

                # Result value is in the <result> child element
                result_text = "notchecked"
                title_text = rule_id
                for child in rr:
                    if child.tag.endswith("}result") or child.tag == "result":
                        result_text = (child.text or "notchecked").strip()
                    if child.tag.endswith("}title") or child.tag == "title":
                        title_text = (child.text or rule_id).strip()

                rr_obj = RuleResult(
                    rule_id=rule_id,
                    title=title_text,
                    result=result_text,
                    severity=severity,
                )
                result.rules.append(rr_obj)

                if result_text == "pass":
                    result.pass_count += 1
                elif result_text == "fail":
                    result.fail_count += 1
                elif result_text == "error":
                    result.error_count += 1

        log.info(
            "openscap_scan_complete",
            hostname=hostname,
            passed=result.pass_count,
            failed=result.fail_count,
            errors=result.error_count,
        )
        return result

    def get_available_profiles(self, benchmark_path: str) -> list[str]:
        """Return the list of profile IDs from an XCCDF benchmark."""
        proc = subprocess.run(
            ["oscap", "info", benchmark_path],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode != 0:
            raise RuntimeError(f"oscap info failed: {proc.stderr[:300]}")

        profiles: list[str] = []
        in_profiles = False
        for line in proc.stdout.splitlines():
            stripped = line.strip()
            if stripped.startswith("Profiles:"):
                in_profiles = True
                continue
            if in_profiles:
                if stripped.startswith("Found") or stripped == "" or not stripped.startswith("Id:"):
                    if profiles:
                        break
                    continue
                if stripped.startswith("Id:"):
                    profiles.append(stripped.split("Id:", 1)[1].strip())
        return profiles

    async def save_results(
        self,
        results: SCAResult,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> None:
        """Upsert scan results to openscap_results table."""
        for rule in results.rules:
            await db_pool.execute(
                """
                INSERT INTO openscap_results
                    (tenant_id, hostname, profile, rule_id, title, result, severity, scanned_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
                ON CONFLICT (tenant_id, hostname, rule_id) DO UPDATE SET
                    result     = EXCLUDED.result,
                    severity   = EXCLUDED.severity,
                    scanned_at = NOW()
                """,
                tenant_id,
                results.hostname,
                results.profile,
                rule.rule_id,
                rule.title,
                rule.result,
                rule.severity,
            )
        log.info(
            "openscap_results_saved",
            tenant_id=tenant_id,
            hostname=results.hostname,
            rule_count=len(results.rules),
        )

    def generate_remediation_plan(self, results: SCAResult) -> list[RemediationItem]:
        """Return a list of remediation items for all failed rules."""
        items: list[RemediationItem] = []
        for rule in results.rules:
            if rule.result != "fail":
                continue
            items.append(
                RemediationItem(
                    rule_id=rule.rule_id,
                    title=rule.title,
                    severity=rule.severity,
                    description=(
                        f"Rule '{rule.rule_id}' failed on host '{results.hostname}'. "
                        f"Review the benchmark profile '{results.profile}' and apply "
                        f"the recommended configuration fix."
                    ),
                )
            )
        # Sort by severity: high first
        severity_order = {"high": 0, "medium": 1, "low": 2, "unknown": 3}
        items.sort(key=lambda x: severity_order.get(x.severity, 3))
        return items


if __name__ == "__main__":
    import sys

    scanner = OpenSCAPScanner()
    benchmark = os.getenv(
        "OSCAP_BENCHMARK",
        "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
    )
    profile = os.getenv("OSCAP_PROFILE", "xccdf_org.ssgproject.content_profile_cis")

    print(f"Available profiles in {benchmark}:")
    try:
        for p in scanner.get_available_profiles(benchmark):
            print(f"  {p}")
    except Exception as e:
        print(f"  (could not list profiles: {e})")

    print(f"\nRunning scan with profile {profile}...")
    try:
        r = scanner.scan_host("localhost", profile, benchmark)
        print(f"Pass: {r.pass_count}, Fail: {r.fail_count}, Error: {r.error_count}")
        plan = scanner.generate_remediation_plan(r)
        print(f"Remediation items: {len(plan)}")
        for item in plan[:5]:
            print(f"  [{item.severity.upper()}] {item.rule_id}: {item.title}")
    except Exception as e:
        print(f"Scan error: {e}", file=sys.stderr)
