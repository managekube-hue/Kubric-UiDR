"""
K-GRC-SCS-008_cyclonedx_sbom.py

CycloneDX SBOM processor using cyclonedx-python-lib.
Generates SBOMs from requirements.txt, parses existing BOM JSON,
compares two BOMs, and uploads to Dependency-Track.
"""
from __future__ import annotations

import asyncio
import json
import os
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import httpx
import structlog

log = structlog.get_logger(__name__)

_DEPTRACK_URL = os.getenv("DEPTRACK_URL", "http://localhost:8081")
_DEPTRACK_API_KEY = os.getenv("DEPTRACK_API_KEY", "")


@dataclass
class BOMComponent:
    name: str
    version: str
    purl: str
    bom_ref: str = ""
    component_type: str = "library"
    licenses: list[str] = field(default_factory=list)
    hashes: dict[str, str] = field(default_factory=dict)


@dataclass
class BOMSummary:
    serial_number: str
    bom_version: int
    component_count: int
    components: list[BOMComponent]
    tool_name: str = ""
    tool_version: str = ""
    target_name: str = ""


@dataclass
class BOMDiff:
    added: list[BOMComponent] = field(default_factory=list)
    removed: list[BOMComponent] = field(default_factory=list)
    version_changed: list[tuple[BOMComponent, BOMComponent]] = field(default_factory=list)  # (old, new)


class CycloneDXProcessor:
    """
    Generates, parses, and compares CycloneDX SBOMs.
    Falls back to a manual JSON builder if cyclonedx-python-lib is not installed.
    """

    # ------------------------------------------------------------------ #
    #  Generation
    # ------------------------------------------------------------------ #

    def generate_from_requirements(self, req_path: str) -> str:
        """
        Generate a CycloneDX BOM from a requirements.txt file.
        Returns the BOM as a JSON string.

        Uses cyclonedx-python-lib if available; otherwise builds a
        minimal CycloneDX 1.4 document manually.
        """
        components: list[dict] = self._parse_requirements(req_path)

        try:
            return self._generate_with_lib(req_path, components)
        except ImportError:
            pass

        # Fallback: manually assemble CycloneDX 1.4 JSON
        return self._generate_manual(req_path, components)

    def _parse_requirements(self, req_path: str) -> list[dict]:
        """Parse requirements.txt into a list of {name, version} dicts."""
        import re
        components = []
        with open(req_path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.split("#", 1)[0].strip()
                if not line or line.startswith("-"):
                    continue
                m = re.match(r"^([A-Za-z0-9_.\-]+)==([^\s;]+)", line)
                if m:
                    components.append({"name": m.group(1), "version": m.group(2)})
        return components

    def _generate_with_lib(self, req_path: str, components_raw: list[dict]) -> str:
        """Generate CycloneDX JSON using cyclonedx-python-lib."""
        from cyclonedx.model.bom import Bom                       # type: ignore
        from cyclonedx.model.component import Component, ComponentType  # type: ignore
        from cyclonedx.output import get_instance, OutputFormat    # type: ignore
        from cyclonedx.output.json import JsonV1Dot4               # type: ignore
        from packageurl import PackageURL                          # type: ignore

        bom = Bom()
        for comp in components_raw:
            purl = PackageURL(type="pypi", name=comp["name"], version=comp["version"])
            bom.components.add(Component(
                name=comp["name"],
                version=comp["version"],
                component_type=ComponentType.LIBRARY,
                purl=purl,
            ))

        outputter = JsonV1Dot4(bom)
        return outputter.output_as_string()

    def _generate_manual(self, source_path: str, components: list[dict]) -> str:
        """Build a minimal CycloneDX 1.4 JSON BOM without external libraries."""
        bom_components = []
        for comp in components:
            name = comp["name"]
            version = comp["version"]
            purl = f"pkg:pypi/{name.lower()}@{version}"
            bom_components.append({
                "type": "library",
                "bom-ref": str(uuid.uuid4()),
                "name": name,
                "version": version,
                "purl": purl,
            })

        bom = {
            "bomFormat": "CycloneDX",
            "specVersion": "1.4",
            "serialNumber": f"urn:uuid:{uuid.uuid4()}",
            "version": 1,
            "metadata": {
                "timestamp": __import__("datetime").datetime.utcnow().isoformat() + "Z",
                "tools": [{"vendor": "Kubric Security", "name": "CycloneDXProcessor", "version": "1.0"}],
                "component": {
                    "type": "application",
                    "name": Path(source_path).stem,
                    "version": "unknown",
                },
            },
            "components": bom_components,
        }
        return json.dumps(bom, indent=2)

    # ------------------------------------------------------------------ #
    #  Parsing
    # ------------------------------------------------------------------ #

    def parse_cyclonedx(self, json_str: str) -> BOMSummary:
        """Parse a CycloneDX JSON BOM string into a BOMSummary."""
        data: dict[str, Any] = json.loads(json_str)

        serial_number = data.get("serialNumber", f"urn:uuid:{uuid.uuid4()}")
        bom_version = int(data.get("version", 1))

        meta = data.get("metadata", {})
        target_name = meta.get("component", {}).get("name", "")
        tools = meta.get("tools", [])
        tool_name = tools[0].get("name", "") if tools else ""
        tool_version = tools[0].get("version", "") if tools else ""

        raw_comps = data.get("components", [])
        components: list[BOMComponent] = []
        for c in raw_comps:
            licenses: list[str] = []
            for lic_entry in c.get("licenses", []):
                if isinstance(lic_entry, dict):
                    lic = lic_entry.get("license", {})
                    lic_id = lic.get("id") or lic.get("name", "")
                    if lic_id:
                        licenses.append(lic_id)

            hashes: dict[str, str] = {}
            for h in c.get("hashes", []):
                hashes[h.get("alg", "")] = h.get("content", "")

            components.append(BOMComponent(
                name=c.get("name", ""),
                version=c.get("version", ""),
                purl=c.get("purl", ""),
                bom_ref=c.get("bom-ref", ""),
                component_type=c.get("type", "library"),
                licenses=licenses,
                hashes=hashes,
            ))

        return BOMSummary(
            serial_number=serial_number,
            bom_version=bom_version,
            component_count=len(components),
            components=components,
            tool_name=tool_name,
            tool_version=tool_version,
            target_name=target_name,
        )

    # ------------------------------------------------------------------ #
    #  Comparison
    # ------------------------------------------------------------------ #

    def compare_boms(self, bom1_json: str, bom2_json: str) -> BOMDiff:
        """
        Compare two CycloneDX BOMs.
        Returns added/removed/version-changed component lists.
        """
        summary1 = self.parse_cyclonedx(bom1_json)
        summary2 = self.parse_cyclonedx(bom2_json)

        # Build name→component maps using purl as the key (fall back to name+version)
        def key(c: BOMComponent) -> str:
            return c.purl or f"{c.name}@{c.version}"

        def name_key(c: BOMComponent) -> str:
            return c.name.lower()

        map1: dict[str, BOMComponent] = {name_key(c): c for c in summary1.components}
        map2: dict[str, BOMComponent] = {name_key(c): c for c in summary2.components}

        diff = BOMDiff()
        for k, comp2 in map2.items():
            if k not in map1:
                diff.added.append(comp2)
            elif map1[k].version != comp2.version:
                diff.version_changed.append((map1[k], comp2))

        for k, comp1 in map1.items():
            if k not in map2:
                diff.removed.append(comp1)

        log.info(
            "bom_diff_complete",
            added=len(diff.added),
            removed=len(diff.removed),
            version_changed=len(diff.version_changed),
        )
        return diff

    # ------------------------------------------------------------------ #
    #  Upload to Dependency-Track
    # ------------------------------------------------------------------ #

    async def upload_to_dependency_track(
        self,
        bom_json: str,
        project_uuid: str,
        base_url: str = _DEPTRACK_URL,
        api_key: str = _DEPTRACK_API_KEY,
    ) -> str:
        """
        Upload a CycloneDX BOM JSON to Dependency-Track.
        Returns the processing token.
        """
        import base64

        encoded = base64.b64encode(bom_json.encode()).decode()
        body = {
            "project": project_uuid,
            "bom": encoded,
        }

        async with httpx.AsyncClient(timeout=60.0) as client:
            resp = await client.put(
                f"{base_url}/api/v1/bom",
                json=body,
                headers={
                    "X-Api-Key": api_key,
                    "Content-Type": "application/json",
                },
            )
            resp.raise_for_status()
            token = resp.json().get("token", "")

        log.info("bom_uploaded_to_deptrack", project_uuid=project_uuid, token=token)
        return token


async def main() -> None:
    import sys

    req_path = os.getenv("REQUIREMENTS_PATH", "requirements.txt")
    project_uuid = os.getenv("DEPTRACK_PROJECT_UUID", "")

    processor = CycloneDXProcessor()

    if not os.path.exists(req_path):
        print(f"File not found: {req_path}", file=sys.stderr)
        sys.exit(1)

    print(f"Generating CycloneDX BOM from {req_path}...")
    bom_json = processor.generate_from_requirements(req_path)

    summary = processor.parse_cyclonedx(bom_json)
    print(f"BOM generated: {summary.component_count} components")
    print(f"Serial: {summary.serial_number}")

    # Save BOM to disk
    out_path = req_path.replace(".txt", "-bom.json")
    with open(out_path, "w", encoding="utf-8") as fh:
        fh.write(bom_json)
    print(f"BOM saved to {out_path}")

    if project_uuid:
        print(f"Uploading to Dependency-Track project {project_uuid}...")
        token = await processor.upload_to_dependency_track(bom_json, project_uuid)
        print(f"Upload token: {token}")


if __name__ == "__main__":
    asyncio.run(main())
