"""
K-SOC-DET-003 — MITRE ATT&CK alert enricher.

Maps Sigma tags and raw alert dicts to MITRE ATT&CK tactics / techniques
using the mitreattack-python library (STIX 2.0 back-end).

Env vars:
    MITRE_STIX_PATH   path to enterprise-attack.json (default: /data/enterprise-attack.json)
    SIGMA_DB_URL      asyncpg DSN — for batch_enrich_alerts (optional)

CLI usage:
    python K-SOC-DET-003_mitre_mapper.py --alert '{"tags":["attack.t1059.001"]}'
"""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import re
import sys
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

MITRE_STIX_PATH = os.getenv("MITRE_STIX_PATH", "/data/enterprise-attack.json")
SIGMA_DB_URL = os.getenv("SIGMA_DB_URL", "")

try:
    from mitreattack.stix20 import MitreAttackData  # type: ignore[import]
    _HAS_MITREATTACK = True
except ImportError:
    _HAS_MITREATTACK = False
    MitreAttackData = None  # type: ignore[assignment,misc]


@dataclass
class AttackTechnique:
    technique_id: str
    name: str
    tactic: str
    url: str
    subtechniques: list[str] = field(default_factory=list)


class MitreMapper:
    """
    Maps alert tags and raw alert dicts to MITRE ATT&CK context.

    Builds an O(1) technique-ID -> AttackTechnique lookup table from the
    local STIX 2.0 enterprise attack bundle at init time.
    """

    def __init__(self, stix_path: str = MITRE_STIX_PATH) -> None:
        if not _HAS_MITREATTACK:
            raise RuntimeError("mitreattack-python not installed: pip install mitreattack-python")
        self._data: Any = MitreAttackData(stix_file=stix_path)
        self._lookup: dict[str, AttackTechnique] = {}
        self.build_technique_lookup()

    def build_technique_lookup(self) -> None:
        """Populate an O(1) technique_id -> AttackTechnique map."""
        all_techniques = self._data.get_techniques(remove_revoked_deprecated=True)
        for obj in all_techniques:
            tech_id = self._external_id(obj)
            if not tech_id:
                continue
            tactic = self._primary_tactic(obj)
            self._lookup[tech_id.upper()] = AttackTechnique(
                technique_id=tech_id,
                name=obj.get("name", ""),
                tactic=tactic,
                url=f"https://attack.mitre.org/techniques/{tech_id.replace('.', '/')}/",
            )
        # Link sub-techniques to parent subtechniques list.
        for tech_id, tech in self._lookup.items():
            if "." in tech_id:
                parent = tech_id.split(".")[0]
                if parent in self._lookup:
                    self._lookup[parent].subtechniques.append(tech_id)
        log.info("mitre_mapper.lookup_built", count=len(self._lookup))

    def map_sigma_tags(self, tags: list[str]) -> list[AttackTechnique]:
        """Parse attack.tXXXX or attack.tXXXX.YYY tags -> AttackTechnique list."""
        results: list[AttackTechnique] = []
        for tag in tags:
            lower = tag.lower()
            if not lower.startswith("attack."):
                continue
            suffix = lower[len("attack."):]
            m = re.fullmatch(r"(t\d{4})(?:\.(\d{3}))?", suffix)
            if not m:
                continue
            tech_id = m.group(1).upper()
            if m.group(2):
                tech_id = f"{tech_id}.{m.group(2)}"
            tech = self._lookup.get(tech_id)
            if tech:
                results.append(tech)
        return results

    def get_tactic_for_technique(self, technique_id: str) -> str:
        """Return primary tactic for a technique ID, or empty string."""
        tech = self._lookup.get(technique_id.upper())
        return tech.tactic if tech else ""

    def enrich_alert(self, alert_dict: dict[str, Any]) -> dict[str, Any]:
        """
        Add mitre_tactic and mitre_technique fields to an alert dict.

        Returns a new dict (does not mutate the input).
        """
        tags: list[str] = alert_dict.get("tags") or []
        techniques = self.map_sigma_tags(tags)
        tech_ids = [t.technique_id for t in techniques]
        tactics = list({t.tactic for t in techniques if t.tactic})
        enriched = dict(alert_dict)
        enriched["mitre_technique"] = tech_ids[0] if tech_ids else None
        enriched["mitre_tactic"] = tactics[0] if tactics else None
        enriched["mitre_techniques_all"] = tech_ids
        enriched["mitre_tactics_all"] = tactics
        return enriched

    async def batch_enrich_alerts(self, tenant_id: str, db_pool: asyncpg.Pool) -> int:
        """
        Pull unenriched kai_alerts for tenant_id, enrich with MITRE data,
        and write mitre_technique / mitre_tactic back to the table.
        Returns count of rows updated.
        """
        select_sql = """
            SELECT id, tags FROM kai_alerts
            WHERE tenant_id = $1
              AND (mitre_technique IS NULL OR mitre_technique = '')
            LIMIT 500
        """
        update_sql = """
            UPDATE kai_alerts SET mitre_technique = $1, mitre_tactic = $2 WHERE id = $3
        """
        updated = 0
        async with db_pool.acquire() as conn:
            rows = await conn.fetch(select_sql, tenant_id)
            for row in rows:
                techs = self.map_sigma_tags(list(row["tags"] or []))
                if not techs:
                    continue
                await conn.execute(update_sql, techs[0].technique_id, techs[0].tactic, row["id"])
                updated += 1
        log.info("mitre_mapper.batch_enriched", tenant_id=tenant_id, count=updated)
        return updated

    @staticmethod
    def _external_id(obj: dict[str, Any]) -> str:
        for ref in obj.get("external_references", []):
            if ref.get("source_name") in ("mitre-attack", "mitre-mobile-attack"):
                return ref.get("external_id", "")
        return ""

    @staticmethod
    def _primary_tactic(obj: dict[str, Any]) -> str:
        phases = obj.get("kill_chain_phases", [])
        return phases[0].get("phase_name", "") if phases else ""


def main() -> None:
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )
    parser = argparse.ArgumentParser(description="Enrich alert with MITRE ATT&CK context")
    parser.add_argument("--alert", help='JSON alert string e.g. \'{"tags":["attack.t1059.001"]}\'')
    parser.add_argument("--stix-path", default=MITRE_STIX_PATH)
    args = parser.parse_args()

    if not _HAS_MITREATTACK:
        print("ERROR: mitreattack-python not installed", file=sys.stderr)
        sys.exit(1)

    mapper = MitreMapper(stix_path=args.stix_path)
    if args.alert:
        alert = json.loads(args.alert)
        print(json.dumps(mapper.enrich_alert(alert), indent=2))
    else:
        print(f"Technique lookup: {len(mapper._lookup)} entries loaded")


if __name__ == "__main__":
    main()
