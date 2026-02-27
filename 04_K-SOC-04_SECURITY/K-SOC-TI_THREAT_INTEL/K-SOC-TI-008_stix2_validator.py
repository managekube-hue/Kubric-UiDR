"""
K-SOC-TI-008 — STIX2 bundle validator.

Validates STIX2 JSON bundles using the stix2-validator library before
ingestion into the pipeline.

CLI usage:
    python K-SOC-TI-008_stix2_validator.py --file bundle.json --strict
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import structlog

log = structlog.get_logger(__name__)

try:
    from stix2validator import validate_string, ValidationOptions  # type: ignore[import]
    _HAS_VALIDATOR = True
except ImportError:
    _HAS_VALIDATOR = False
    validate_string = None  # type: ignore[assignment]
    ValidationOptions = None  # type: ignore[assignment]


@dataclass
class ValidationResult:
    is_valid: bool
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    object_count: int = 0


class STIXValidator:
    """
    Validates STIX2 bundles before ingestion.

    Uses stix2validator in strict mode when requested.
    Falls back to basic JSON + structural checks when the library
    is not available.
    """

    def __init__(self, strict: bool = False) -> None:
        self.strict = strict
        if not _HAS_VALIDATOR:
            log.warning("stix2_validator.library_missing", hint="pip install stix2-validator")

    def validate(self, json_str: str) -> ValidationResult:
        """Validate a STIX2 bundle JSON string."""
        # First pass: JSON parse check.
        try:
            raw = json.loads(json_str)
        except json.JSONDecodeError as exc:
            return ValidationResult(is_valid=False, errors=[f"JSON parse error: {exc}"])

        object_count = len(raw.get("objects") or [])

        if _HAS_VALIDATOR:
            return self._validate_with_library(json_str, object_count)

        # Fallback structural validation.
        return self._structural_validate(raw, object_count)

    def validate_file(self, path: str) -> ValidationResult:
        """Validate a STIX2 JSON bundle from a file path."""
        try:
            content = Path(path).read_text(encoding="utf-8")
        except OSError as exc:
            return ValidationResult(is_valid=False, errors=[f"File read error: {exc}"])
        return self.validate(content)

    def batch_validate(self, paths: list[str]) -> list[tuple[str, ValidationResult]]:
        """Validate multiple files. Returns list of (path, ValidationResult) tuples."""
        results: list[tuple[str, ValidationResult]] = []
        for path in paths:
            result = self.validate_file(path)
            results.append((path, result))
            if result.is_valid:
                log.info("stix2_validator.valid", path=path, objects=result.object_count)
            else:
                log.warning(
                    "stix2_validator.invalid",
                    path=path,
                    error_count=len(result.errors),
                )
        return results

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _validate_with_library(self, json_str: str, object_count: int) -> ValidationResult:
        opts_kwargs: dict[str, Any] = {}
        if self.strict:
            opts_kwargs["strict"] = True
            opts_kwargs["no_extensions"] = True
        else:
            opts_kwargs["strict"] = False

        opts = ValidationOptions(**opts_kwargs)
        results_obj = validate_string(json_str, options=opts)

        errors: list[str] = []
        warnings: list[str] = []

        for r in results_obj.results:
            for e in getattr(r, "errors", []):
                errors.append(str(e))
            for w in getattr(r, "warnings", []):
                warnings.append(str(w))

        is_valid = results_obj.is_valid
        return ValidationResult(
            is_valid=is_valid,
            errors=errors,
            warnings=warnings,
            object_count=object_count,
        )

    def _structural_validate(
        self, raw: dict[str, Any], object_count: int
    ) -> ValidationResult:
        """Basic structural checks when stix2validator is not installed."""
        errors: list[str] = []
        warnings: list[str] = []

        if raw.get("type") != "bundle":
            errors.append("Root object type must be 'bundle'")

        if "id" not in raw:
            errors.append("Bundle missing required 'id' field")

        if "spec_version" not in raw:
            warnings.append("Bundle missing 'spec_version' field")

        for i, obj in enumerate(raw.get("objects") or []):
            for required_field in ("type", "id", "spec_version"):
                if required_field not in obj:
                    errors.append(
                        f"objects[{i}] missing required field '{required_field}'"
                    )

            if self.strict:
                # In strict mode, disallow custom (x_ prefixed) properties.
                for key in obj.keys():
                    if key.startswith("x_"):
                        errors.append(
                            f"objects[{i}] has custom property '{key}' (strict mode)"
                        )

        return ValidationResult(
            is_valid=len(errors) == 0,
            errors=errors,
            warnings=warnings,
            object_count=object_count,
        )


def main() -> None:
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )

    parser = argparse.ArgumentParser(description="Validate a STIX2 bundle")
    parser.add_argument("--file", required=True, help="Path to STIX2 JSON bundle")
    parser.add_argument("--strict", action="store_true", help="Enable strict validation mode")
    args = parser.parse_args()

    validator = STIXValidator(strict=args.strict)
    result = validator.validate_file(args.file)

    print(f"Valid:    {result.is_valid}")
    print(f"Objects:  {result.object_count}")
    if result.errors:
        print(f"Errors ({len(result.errors)}):")
        for e in result.errors:
            print(f"  - {e}")
    if result.warnings:
        print(f"Warnings ({len(result.warnings)}):")
        for w in result.warnings:
            print(f"  - {w}")

    sys.exit(0 if result.is_valid else 1)


if __name__ == "__main__":
    main()
