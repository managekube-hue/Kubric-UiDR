"""
K-GRC-OSCAL-002_soc2_mapper.py

SOC 2 Trust Services Criteria (TSC) → NIST SP 800-53 control mapping.
Provides cross-walk lookups, reverse lookups, and async compliance checks.
"""
from __future__ import annotations

import asyncio
import os
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)


# ---------------------------------------------------------------------------
# TSC → NIST 800-53 mapping (14 TSC criteria — exactly as specified)
# ---------------------------------------------------------------------------
TSC_TO_NIST: dict[str, list[str]] = {
    # CC6 — Logical and Physical Access Controls
    "CC6.1": ["AC-2", "AC-3", "IA-2", "IA-5"],
    "CC6.2": ["AC-6", "AC-7", "SC-28"],
    "CC6.3": ["IA-8", "CM-7"],

    # CC7 — System Operations
    "CC7.1": ["AU-2", "AU-3", "AU-12", "SI-4"],
    "CC7.2": ["IR-4", "IR-5", "IR-8"],

    # CC8 — Change Management
    "CC8.1": ["CM-2", "CM-3", "CM-4", "CM-6"],

    # CC9 — Risk Mitigation
    "CC9.1": ["RA-3", "RA-5", "PM-9"],

    # A1 — Availability
    "A1.1": ["CP-2", "CP-6", "CP-7", "CP-8"],
    "A1.2": ["CP-9", "CP-10", "MA-6"],

    # PI — Processing Integrity
    "PI1.2": ["SA-8", "SA-11", "SA-15"],

    # P — Privacy
    "P3.1": ["PL-8", "RA-2", "PM-1"],
    "P6.1": ["AC-20", "MP-6"],
    "P7.1": ["AC-21", "PM-20"],
    "P8.1": ["AR-2", "IP-2"],
}

# Reverse lookup: NIST control → list[TSC criteria]
NIST_TO_TSC: dict[str, list[str]] = {}
for tsc, nist_controls in TSC_TO_NIST.items():
    for nc in nist_controls:
        NIST_TO_TSC.setdefault(nc, []).append(tsc)


@dataclass
class SOC2ControlResult:
    tsc_id: str
    nist_controls: list[str]
    assessed: list[str]     # NIST controls that have a recorded assessment
    met: list[str]          # assessed controls with status = pass
    status: str             # compliant | partial | not_compliant | not_assessed


@dataclass
class SOC2Report:
    tenant_id: str
    total_tsc: int
    pass_count: int
    fail_count: int
    not_assessed: int
    criteria_results: list[SOC2ControlResult] = field(default_factory=list)

    @property
    def compliance_pct(self) -> float:
        if self.total_tsc == 0:
            return 0.0
        return round(self.pass_count / self.total_tsc * 100, 1)


class SOC2Mapper:
    """Maps SOC 2 TSC criteria to NIST 800-53 controls and checks compliance."""

    def get_nist_controls(self, tsc_id: str) -> list[str]:
        """Return NIST controls mapped to a TSC criterion."""
        return TSC_TO_NIST.get(tsc_id, [])

    def get_tsc_for_nist(self, nist_control: str) -> list[str]:
        """Reverse lookup: which TSC criteria does a NIST control satisfy?"""
        return NIST_TO_TSC.get(nist_control, [])

    def generate_crosswalk_table(self) -> list[dict[str, Any]]:
        """Return the full crosswalk as a list of dicts, suitable for CSV/JSON export."""
        rows: list[dict[str, Any]] = []
        for tsc_id, nist_list in sorted(TSC_TO_NIST.items()):
            for nist in nist_list:
                rows.append({
                    "tsc_criteria": tsc_id,
                    "nist_control": nist,
                    "mapping_notes": f"SOC2 {tsc_id} is supported by NIST {nist}",
                })
        return rows

    async def check_soc2_compliance(
        self,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> SOC2Report:
        """
        Query the assessments table for each NIST control mapped to a TSC criterion
        and determine per-TSC compliance status.
        """
        # Fetch all NIST control assessments for this tenant in one query
        rows = await db_pool.fetch(
            """
            SELECT control_id, status
            FROM assessments
            WHERE tenant_id = $1
              AND framework IN ('OSCAL', 'nist', 'NIST 800-53')
            """,
            tenant_id,
        )

        # Build a lookup: nist_control → status
        nist_status: dict[str, str] = {str(r["control_id"]): str(r["status"]) for r in rows}

        report = SOC2Report(
            tenant_id=tenant_id,
            total_tsc=len(TSC_TO_NIST),
            pass_count=0,
            fail_count=0,
            not_assessed=0,
        )

        for tsc_id, nist_controls in TSC_TO_NIST.items():
            assessed = [n for n in nist_controls if n in nist_status]
            met = [n for n in assessed if nist_status[n] == "pass"]

            if not assessed:
                status = "not_assessed"
                report.not_assessed += 1
            elif len(met) == len(nist_controls):
                status = "compliant"
                report.pass_count += 1
            elif met:
                status = "partial"
                report.fail_count += 1
            else:
                status = "not_compliant"
                report.fail_count += 1

            report.criteria_results.append(SOC2ControlResult(
                tsc_id=tsc_id,
                nist_controls=nist_controls,
                assessed=assessed,
                met=met,
                status=status,
            ))

        log.info(
            "soc2_compliance_check_complete",
            tenant_id=tenant_id,
            pass_count=report.pass_count,
            fail_count=report.fail_count,
            not_assessed=report.not_assessed,
            pct=report.compliance_pct,
        )
        return report


async def main() -> None:
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")
    tenant_id = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")

    db_pool = await asyncpg.create_pool(db_url)
    try:
        mapper = SOC2Mapper()

        print("=== SOC 2 → NIST 800-53 Crosswalk ===")
        for row in mapper.generate_crosswalk_table()[:5]:
            print(f"  {row['tsc_criteria']} → {row['nist_control']}")
        print(f"  ... ({len(mapper.generate_crosswalk_table())} total mappings)")

        report = await mapper.check_soc2_compliance(tenant_id, db_pool)
        print(f"\n=== SOC 2 Compliance Report for tenant {tenant_id} ===")
        print(f"Total TSC criteria: {report.total_tsc}")
        print(f"Pass:               {report.pass_count}")
        print(f"Fail:               {report.fail_count}")
        print(f"Not assessed:       {report.not_assessed}")
        print(f"Overall:            {report.compliance_pct}%")
    finally:
        await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
