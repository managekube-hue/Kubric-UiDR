"""
K-GRC-OSCAL-001_nist_ingest.py

NIST SP 800-53 OSCAL catalog ingestor.
Loads a NIST OSCAL JSON file and upserts controls into the oscal_controls table.
"""
from __future__ import annotations

import asyncio
import json
import os
import sys
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)


@dataclass
class OSCALControl:
    control_id: str
    catalog: str
    family: str
    title: str
    parameters: dict = field(default_factory=dict)
    guidance: str = ""


class NistCatalogIngestor:
    """
    Parses an OSCAL JSON catalog file (NIST SP 800-53 Rev 5 format)
    and upserts each control into the oscal_controls table.
    """

    def __init__(self, catalog_name: str = "NIST SP 800-53 Rev 5") -> None:
        self.catalog_name = catalog_name

    def load(self, path: str) -> list[OSCALControl]:
        """Load and parse an OSCAL JSON file, returning a flat list of controls."""
        with open(path, "r", encoding="utf-8") as fh:
            data = json.load(fh)
        return self._parse_catalog(data)

    def _parse_catalog(self, data: dict) -> list[OSCALControl]:
        """Recursively extract controls from OSCAL catalog structure."""
        catalog = data.get("catalog", data)
        controls: list[OSCALControl] = []

        groups = catalog.get("groups", [])
        for group in groups:
            family = group.get("title", group.get("id", "unknown"))
            self._extract_controls(group, family, controls)

        # Top-level controls (no group)
        for ctrl in catalog.get("controls", []):
            family = ctrl.get("class", "unknown")
            self._extract_one_control(ctrl, family, controls)

        log.info("nist_catalog_parsed", count=len(controls), catalog=self.catalog_name)
        return controls

    def _extract_controls(self, node: dict, family: str, out: list[OSCALControl]) -> None:
        """Walk groups recursively to extract all controls."""
        for ctrl in node.get("controls", []):
            self._extract_one_control(ctrl, family, out)
            # Enhancement controls nested inside
            for enh in ctrl.get("controls", []):
                self._extract_one_control(enh, family, out)

        for subgroup in node.get("groups", []):
            sub_family = subgroup.get("title", family)
            self._extract_controls(subgroup, sub_family, out)

    def _extract_one_control(self, ctrl: dict, family: str, out: list[OSCALControl]) -> None:
        control_id = ctrl.get("id", "").upper()
        if not control_id:
            return

        title = ""
        parameters: dict[str, Any] = {}
        guidance = ""

        for part in ctrl.get("parts", []):
            if part.get("name") == "statement":
                guidance = self._flatten_parts(part)
            if part.get("name") == "guidance":
                guidance = guidance + "\n" + self._flatten_parts(part) if guidance else self._flatten_parts(part)

        for prop in ctrl.get("props", []):
            if prop.get("name") == "label":
                title = title or prop.get("value", "")

        title = ctrl.get("title", title) or control_id

        for param in ctrl.get("params", []):
            pid = param.get("id", "")
            label = param.get("label", "")
            values = param.get("values", [])
            parameters[pid] = {"label": label, "values": values}

        out.append(OSCALControl(
            control_id=control_id,
            catalog=self.catalog_name,
            family=family,
            title=title,
            parameters=parameters,
            guidance=guidance.strip(),
        ))

    def _flatten_parts(self, part: dict) -> str:
        """Recursively collect prose text from OSCAL parts."""
        texts: list[str] = []
        prose = part.get("prose", "")
        if prose:
            texts.append(prose)
        for child in part.get("parts", []):
            texts.append(self._flatten_parts(child))
        return " ".join(t for t in texts if t)

    async def upsert_control(self, ctrl: OSCALControl, db_pool: asyncpg.Pool) -> None:
        """Upsert a single control into oscal_controls."""
        await db_pool.execute(
            """
            INSERT INTO oscal_controls
                (control_id, catalog, family, title, parameters, guidance, created_at)
            VALUES ($1, $2, $3, $4, $5::jsonb, $6, NOW())
            ON CONFLICT (control_id) DO UPDATE SET
                catalog    = EXCLUDED.catalog,
                family     = EXCLUDED.family,
                title      = EXCLUDED.title,
                parameters = EXCLUDED.parameters,
                guidance   = EXCLUDED.guidance
            """,
            ctrl.control_id,
            ctrl.catalog,
            ctrl.family,
            ctrl.title,
            json.dumps(ctrl.parameters),
            ctrl.guidance,
        )

    async def ingest(self, path: str, db_pool: asyncpg.Pool) -> int:
        """Load the OSCAL file and upsert all controls. Returns the count inserted/updated."""
        controls = self.load(path)
        for ctrl in controls:
            await self.upsert_control(ctrl, db_pool)
        log.info("nist_ingest_complete", count=len(controls), path=path)
        return len(controls)


async def main() -> None:
    oscal_path = os.getenv("OSCAL_CATALOG_PATH", "nist-sp800-53-rev5-catalog.json")
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")

    if not os.path.exists(oscal_path):
        print(f"ERROR: OSCAL file not found: {oscal_path}", file=sys.stderr)
        sys.exit(1)

    db_pool = await asyncpg.create_pool(db_url)
    try:
        ingestor = NistCatalogIngestor()
        count = await ingestor.ingest(oscal_path, db_pool)
        print(f"Ingested {count} NIST controls from {oscal_path}")
    finally:
        await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
