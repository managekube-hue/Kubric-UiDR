"""
K-KAI-HU-002: Sigma Hunting Runner
Loads Sigma rules from a directory, converts them to Velociraptor VQL,
and dispatches hunts via VelociraptorArtifactRunner.
"""

import logging
import os
import sys
from pathlib import Path
from typing import Any

import yaml

sys.path.insert(0, os.path.dirname(__file__))
from K_KAI_HU_001_velociraptor_artifacts import VelociraptorArtifactRunner  # noqa: E402

logger = logging.getLogger("K-KAI-HU-002")

SIGMA_RULES_DIR: str = os.getenv("SIGMA_RULES_DIR", "/opt/kubric/sigma-rules")

# ---------------------------------------------------------------------------
# Sigma field → Velociraptor VQL field mapping
# ---------------------------------------------------------------------------
_FIELD_MAP: dict[str, str] = {
    # Windows Process
    "Image": "Exe",
    "CommandLine": "CommandLine",
    "ParentImage": "ParentExe",
    "ParentCommandLine": "ParentCommandLine",
    "ProcessId": "Pid",
    "ParentProcessId": "Ppid",
    # Windows Network
    "DestinationIp": "RemoteAddr",
    "DestinationPort": "RemotePort",
    "SourceIp": "LocalAddr",
    "SourcePort": "LocalPort",
    # Windows Event
    "EventID": "EventID",
    "Channel": "Channel",
    "Provider_Name": "Provider",
    # File
    "TargetFilename": "FullPath",
    "Hashes": "Hash",
    # User
    "User": "Username",
    "SubjectUserName": "Username",
    # Generic
    "Keywords": "Keywords",
    "Message": "Message",
}

_LOGSOURCE_TO_ARTIFACT: dict[str, str] = {
    "process_creation": "Windows.EventLogs.Evtx",
    "network_connection": "Windows.Network.Netstat",
    "file_event": "Windows.NTFS.MFT",
    "registry_event": "Windows.Registry.NTUser",
    "dns_query": "Windows.EventLogs.Evtx",
    "default": "Windows.EventLogs.Evtx",
}


class SigmaHuntingRunner:
    """
    Loads Sigma YAML rules, converts them to VQL WHERE clauses, and
    dispatches Velociraptor artifact hunts.
    """

    def __init__(self) -> None:
        self._runner = VelociraptorArtifactRunner()

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def load_rules(self, directory: str | None = None) -> list[dict]:
        """
        Recursively load all .yml / .yaml Sigma rules from *directory*.

        Returns a list of parsed rule dicts with an injected '_file' key.
        """
        rules_dir = Path(directory or SIGMA_RULES_DIR)
        if not rules_dir.exists():
            logger.error("Sigma rules directory not found: %s", rules_dir)
            return []

        rules: list[dict] = []
        for path in sorted(rules_dir.rglob("*.yml")) + sorted(rules_dir.rglob("*.yaml")):
            try:
                with open(path, encoding="utf-8") as fh:
                    rule: dict = yaml.safe_load(fh) or {}
                rule["_file"] = str(path)
                rules.append(rule)
                logger.debug("Loaded Sigma rule: %s", path.name)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Failed to load rule %s: %s", path, exc)

        logger.info("Loaded %d Sigma rules from %s", len(rules), rules_dir)
        return rules

    def rule_to_vql(self, rule: dict) -> str:
        """
        Convert a Sigma rule dict to a Velociraptor VQL WHERE clause string.

        Returns a VQL SELECT statement string for use in an artifact parameter.
        """
        detection: dict = rule.get("detection", {})
        condition: str = detection.get("condition", "")
        logsource: dict = rule.get("logsource", {})

        category = logsource.get("category", "default")
        artifact = _LOGSOURCE_TO_ARTIFACT.get(category, _LOGSOURCE_TO_ARTIFACT["default"])

        # Build WHERE conditions from named detection sections
        where_clauses: list[str] = []
        for section_name, section_value in detection.items():
            if section_name == "condition":
                continue
            clause = self._build_clause(section_value)
            if clause:
                where_clauses.append(f"/* {section_name} */ ({clause})")

        if not where_clauses:
            where_part = "TRUE"  # no filter – returns all events
        elif "all of" in condition.lower() or "and" in condition.lower():
            where_part = " AND ".join(where_clauses)
        else:
            where_part = " OR ".join(where_clauses)

        rule_title = rule.get("title", "unnamed").replace("'", "''")
        vql = (
            f"/* Sigma: {rule_title} */\n"
            f"SELECT * FROM artifact_definitions()\n"
            f"WHERE {where_part}"
        )
        return vql

    def run_hunt(self, rule: dict, client_ids: list[str]) -> list[dict]:
        """
        Dispatch a hunt for the given Sigma rule across a list of clients.

        Returns a list of flow result dicts, one per client.
        """
        logsource: dict = rule.get("logsource", {})
        category: str = logsource.get("category", "default")
        artifact: str = _LOGSOURCE_TO_ARTIFACT.get(category, _LOGSOURCE_TO_ARTIFACT["default"])

        vql: str = self.rule_to_vql(rule)
        parameters: dict[str, str] = {
            "Query": vql,
            "SigmaRuleTitle": rule.get("title", ""),
            "SigmaRuleId": rule.get("id", ""),
        }

        results: list[dict] = []
        for client_id in client_ids:
            try:
                flow_result = self._runner.hunt(client_id, artifact, parameters)
                flow_result["sigma_rule"] = rule.get("title")
                flow_result["sigma_id"] = rule.get("id")
                results.append(flow_result)
                logger.info(
                    "Hunt dispatched for rule '%s' on client %s → flow %s",
                    rule.get("title"),
                    client_id,
                    flow_result.get("flow_id"),
                )
            except Exception as exc:  # noqa: BLE001
                logger.error(
                    "Hunt failed for rule '%s' on client %s: %s",
                    rule.get("title"),
                    client_id,
                    exc,
                )
                results.append({
                    "client_id": client_id,
                    "sigma_rule": rule.get("title"),
                    "status": "ERROR",
                    "error": str(exc),
                    "results": [],
                })
        return results

    # ------------------------------------------------------------------
    # VQL clause builder
    # ------------------------------------------------------------------

    def _build_clause(self, section: Any) -> str:
        """Convert a Sigma detection section into a VQL condition fragment."""
        if isinstance(section, dict):
            parts: list[str] = []
            for sigma_field, values in section.items():
                vql_field = _FIELD_MAP.get(sigma_field, sigma_field)
                part = self._field_to_vql(vql_field, values)
                if part:
                    parts.append(part)
            return " AND ".join(parts)
        if isinstance(section, list):
            return " OR ".join(
                self._build_clause(item) for item in section if item
            )
        return ""

    @staticmethod
    def _field_to_vql(field: str, values: Any) -> str:
        """Build a VQL condition for a single field."""
        if isinstance(values, str):
            escaped = values.replace("'", "''")
            if "*" in values:
                pattern = escaped.replace("*", "%")
                return f"{field} LIKE '{pattern}'"
            return f"{field} = '{escaped}'"

        if isinstance(values, list):
            conditions: list[str] = []
            for v in values:
                if isinstance(v, str):
                    escaped = v.replace("'", "''")
                    if "*" in v:
                        pattern = escaped.replace("*", "%")
                        conditions.append(f"{field} LIKE '{pattern}'")
                    else:
                        conditions.append(f"{field} = '{escaped}'")
                else:
                    conditions.append(f"{field} = {v!r}")
            return "(" + " OR ".join(conditions) + ")" if conditions else ""

        return f"{field} = {values!r}"
