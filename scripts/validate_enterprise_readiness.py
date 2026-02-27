#!/usr/bin/env python3
"""Enterprise readiness gate for Kubric monorepo.

Fails fast on structural blockers that should never reach production branches.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SKIP_DIR_NAMES = {
    ".git",
    ".next",
    "node_modules",
    "vendor",
    "target",
    "dist",
    "build",
    "coverage",
    "__pycache__",
    ".pytest_cache",
}


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8", errors="ignore")


def should_skip_path(path: Path) -> bool:
    return any(part in SKIP_DIR_NAMES for part in path.parts)


def fail(msg: str, errors: list[str]) -> None:
    errors.append(msg)


def check_sealed_secret_placeholders(errors: list[str]) -> None:
    k8s = ROOT / "infra" / "k8s"
    if not k8s.exists():
        return
    for path in k8s.rglob("*.yaml"):
        if should_skip_path(path):
            continue
        txt = read_text(path)
        if "<KUBESEAL_CIPHERTEXT_HERE>" in txt and "The files below are TEMPLATES" not in txt:
            fail(f"SealedSecrets placeholder found: {path.relative_to(ROOT)}", errors)


def check_duplicate_migration_versions(errors: list[str]) -> None:
    mig = ROOT / "migrations" / "postgres"
    if not mig.exists():
        return

    versions: dict[str, list[Path]] = {}
    for path in sorted(mig.glob("*.sql")):
        if should_skip_path(path):
            continue
        m = re.match(r"^(\d{3})_", path.name)
        if not m:
            continue
        versions.setdefault(m.group(1), []).append(path)

    for ver, files in versions.items():
        if len(files) > 1:
            rel = ", ".join(str(p.relative_to(ROOT)) for p in files)
            fail(f"Duplicate migration version {ver}: {rel}", errors)


def check_migration_tenant_model(errors: list[str]) -> None:
    mig = ROOT / "migrations" / "postgres"
    if not mig.exists():
        return

    bad_patterns = [
        r"tenant_id\s+UUID\s+NOT NULL REFERENCES tenants\(id\)",
        r"current_setting\('app\.tenant_id'\)::uuid",
        r"REFERENCES\s+tenants\(id\)",
    ]
    combined = re.compile("|".join(bad_patterns), re.IGNORECASE)

    for path in sorted(mig.glob("*.sql")):
        if should_skip_path(path):
            continue
        txt = read_text(path)
        if combined.search(txt):
            fail(
                f"Legacy tenant model reference in {path.relative_to(ROOT)} "
                f"(use kubric_tenants(tenant_id) + app.current_tenant_id)",
                errors,
            )


def check_critical_empty_files(errors: list[str]) -> None:
    critical = [
        ROOT / "06_K-PSA-06_BUSINESS" / "K-PSA-PTL_PORTAL" / "K-PSA-PTL-APP",
        ROOT / "frontend",
        ROOT / "kai",
        ROOT / "internal",
    ]
    exts = {".ts", ".tsx", ".py", ".go", ".sql", ".css", ".yaml", ".yml"}

    for base in critical:
        if not base.exists():
            continue
        for path in base.rglob("*"):
            if should_skip_path(path):
                continue
            if path.is_file() and path.suffix in exts and path.stat().st_size == 0:
                fail(f"Empty critical file: {path.relative_to(ROOT)}", errors)


def main() -> int:
    errors: list[str] = []
    check_sealed_secret_placeholders(errors)
    check_duplicate_migration_versions(errors)
    check_migration_tenant_model(errors)
    check_critical_empty_files(errors)

    if errors:
        print("ENTERPRISE READINESS: FAIL")
        for err in errors:
            print(f" - {err}")
        return 1

    print("ENTERPRISE READINESS: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
