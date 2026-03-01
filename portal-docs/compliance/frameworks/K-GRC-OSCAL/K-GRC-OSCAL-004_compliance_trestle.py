"""
K-GRC-OSCAL-004_compliance_trestle.py

Automation wrapper for NIST Compliance Trestle (compliance-trestle) CLI.
Provides init_workspace, import_catalog, create_ssp, validate_ssp, and
generate_ssp_markdown via subprocess calls.
"""
from __future__ import annotations

import asyncio
import os
import subprocess
import tempfile
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import structlog

log = structlog.get_logger(__name__)


@dataclass
class ValidationResult:
    valid: bool
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    stdout: str = ""


class TrestleAutomation:
    """
    Wraps the `trestle` CLI (compliance-trestle Python package).
    All operations assume a trestle workspace directory.
    Install: pip install compliance-trestle
    """

    def __init__(self, workspace_dir: str | None = None) -> None:
        self.workspace_dir = workspace_dir or os.getenv(
            "TRESTLE_WORKSPACE", os.path.join(tempfile.gettempdir(), "trestle_workspace")
        )
        self._trestle_bin = os.getenv("TRESTLE_BIN", "trestle")

    def _run(self, args: list[str], cwd: str | None = None) -> subprocess.CompletedProcess:
        """Execute a trestle CLI command."""
        cmd = [self._trestle_bin] + args
        log.info("trestle_exec", cmd=" ".join(cmd), cwd=cwd or self.workspace_dir)
        result = subprocess.run(
            cmd,
            cwd=cwd or self.workspace_dir,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            log.warning(
                "trestle_nonzero_exit",
                returncode=result.returncode,
                stderr=result.stderr[:500],
            )
        return result

    def init_workspace(self) -> bool:
        """
        Initialise a new trestle workspace at self.workspace_dir.
        Creates the directory if it does not exist.
        """
        os.makedirs(self.workspace_dir, exist_ok=True)
        result = self._run(["init"], cwd=self.workspace_dir)
        success = result.returncode == 0
        log.info("trestle_init", success=success, workspace=self.workspace_dir)
        return success

    def import_catalog(self, catalog_path: str, catalog_name: str = "nist-sp800-53") -> bool:
        """
        Import an OSCAL catalog JSON into the trestle workspace.
        catalog_path: absolute path to the OSCAL catalog JSON file.
        catalog_name: name to register within trestle (no spaces, no extension).
        """
        result = self._run([
            "import",
            "-f", catalog_path,
            "-o", catalog_name,
        ])
        success = result.returncode == 0
        log.info("trestle_import_catalog", name=catalog_name, success=success)
        return success

    def create_ssp(
        self,
        ssp_name: str,
        profile_name: str = "nist-sp800-53",
        output_name: str | None = None,
    ) -> bool:
        """
        Generate a System Security Plan (SSP) skeleton from a profile.
        ssp_name:    name for the SSP in the workspace
        profile_name: the imported profile name
        """
        out = output_name or ssp_name
        result = self._run([
            "author", "ssp-generate",
            "-p", profile_name,
            "-o", out,
            "--force-overwrite",
        ])
        success = result.returncode == 0
        log.info("trestle_create_ssp", ssp_name=ssp_name, success=success)
        return success

    def validate_ssp(self, ssp_name: str) -> ValidationResult:
        """
        Run trestle validate on an SSP and return a structured result.
        """
        result = self._run(["validate", "-f", f"system-security-plans/{ssp_name}/system-security-plan.json"])
        errors: list[str] = []
        warnings: list[str] = []

        for line in (result.stdout + result.stderr).splitlines():
            low = line.lower()
            if "error" in low:
                errors.append(line.strip())
            elif "warning" in low or "warn" in low:
                warnings.append(line.strip())

        vr = ValidationResult(
            valid=result.returncode == 0 and len(errors) == 0,
            errors=errors,
            warnings=warnings,
            stdout=result.stdout,
        )
        log.info(
            "trestle_validate_ssp",
            ssp_name=ssp_name,
            valid=vr.valid,
            errors=len(errors),
            warnings=len(warnings),
        )
        return vr

    def generate_ssp_markdown(self, ssp_name: str, output_dir: str | None = None) -> str | None:
        """
        Generate Markdown documentation from an SSP.
        Returns the output directory path on success, None on failure.
        """
        md_dir = output_dir or os.path.join(self.workspace_dir, "markdown", ssp_name)
        os.makedirs(md_dir, exist_ok=True)
        result = self._run([
            "author", "ssp-filter",
            "-n", ssp_name,
            "--output", md_dir,
        ])
        if result.returncode != 0:
            # Fallback: trestle author docs
            result = self._run([
                "author", "docs",
                "--name", ssp_name,
                "--output", md_dir,
            ])
        if result.returncode == 0:
            log.info("trestle_ssp_markdown_generated", path=md_dir)
            return md_dir
        log.warning("trestle_ssp_markdown_failed", ssp_name=ssp_name, stderr=result.stderr[:300])
        return None

    def assemble_ssp(self, ssp_name: str, markdown_dir: str) -> bool:
        """
        Assemble a Markdown SSP back into OSCAL JSON.
        """
        result = self._run([
            "author", "ssp-assemble",
            "-m", markdown_dir,
            "-o", ssp_name,
        ])
        success = result.returncode == 0
        log.info("trestle_assemble_ssp", ssp_name=ssp_name, success=success)
        return success

    def list_models(self, model_type: str = "catalog") -> list[str]:
        """List models of a given type (catalog, profile, system-security-plan, etc.)."""
        result = self._run(["list", "-t", model_type])
        names: list[str] = []
        for line in result.stdout.splitlines():
            name = line.strip()
            if name and not name.startswith("TRESTLE") and not name.startswith("Name"):
                names.append(name)
        return names


async def main() -> None:
    workspace = os.getenv("TRESTLE_WORKSPACE", "/tmp/trestle_workspace")
    automation = TrestleAutomation(workspace)

    print(f"Initialising trestle workspace at {workspace}...")
    if not automation.init_workspace():
        print("WARNING: init returned non-zero (may already be initialised)")

    catalog_path = os.getenv("OSCAL_CATALOG_PATH", "nist-sp800-53-rev5-catalog.json")
    if os.path.exists(catalog_path):
        print(f"Importing catalog from {catalog_path}...")
        automation.import_catalog(catalog_path, "nist-sp800-53")

    print("Listing catalogs in workspace:")
    for name in automation.list_models("catalog"):
        print(f"  {name}")

    ssp_name = "kubric-ssp"
    print(f"\nGenerating SSP '{ssp_name}'...")
    automation.create_ssp(ssp_name, profile_name="nist-sp800-53")

    print("Validating SSP...")
    vr = automation.validate_ssp(ssp_name)
    print(f"Valid: {vr.valid}, Errors: {len(vr.errors)}, Warnings: {len(vr.warnings)}")
    for e in vr.errors[:5]:
        print(f"  ERROR: {e}")


if __name__ == "__main__":
    asyncio.run(main())
