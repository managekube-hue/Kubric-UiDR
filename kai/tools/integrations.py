"""
KAI Integration Tools — CrewAI @tool definitions for NOC security-tool integrations.

Each function is decorated with @tool so CrewAI agents can invoke them
autonomously.  All tools are synchronous wrappers that call the NOC service's
/integrations routes via httpx.  The NOC service proxies requests to the
actual upstream tools (Wazuh, Velociraptor, TheHive, Cortex, osquery,
BloodHound, Shuffle).

Environment variables:
  NOC_BASE_URL       — NOC service URL (default: http://noc:8083)
  KUBRIC_JWT_TOKEN   — JWT bearer token for NOC API authentication

Tools available:
  Wazuh:
    query_wazuh_agents           — list Wazuh agents (optionally filtered by status)
    query_wazuh_alerts           — get high-severity alerts for a specific agent
    query_wazuh_sca              — get SCA compliance results for an agent
    run_wazuh_active_response    — execute an active-response command on an agent

  Velociraptor:
    search_velociraptor_clients  — search Velociraptor endpoints
    create_velociraptor_hunt     — start a new Velociraptor hunt
    run_vql_query                — execute an arbitrary VQL query
    collect_artifact             — collect artifacts from a specific client

  TheHive:
    create_thehive_alert         — create a new alert in TheHive
    promote_thehive_alert        — promote an alert to a case
    list_thehive_cases           — list cases filtered by status
    add_thehive_observable       — add an observable/IOC to a case

  Cortex:
    run_cortex_analyzer          — submit data for analysis and poll for results
    list_cortex_analyzers        — list available analyzers (optionally by data type)

  osquery (FleetDM):
    query_osquery_hosts          — list fleet hosts (optionally filtered by status)
    run_osquery_live             — execute a distributed live query across hosts

  BloodHound CE:
    query_bloodhound_domains     — list Active Directory domains
    run_bloodhound_cypher        — execute a Cypher graph query
    list_attack_paths            — list attack paths for a domain

  Shuffle SOAR:
    list_shuffle_workflows       — list available SOAR workflows
    execute_shuffle_workflow     — trigger a workflow execution

  Health:
    check_integration_health     — check connectivity status of all integrations
"""

from __future__ import annotations

import json
import os
import time

import httpx
import structlog
from crewai.tools import tool

log = structlog.get_logger(__name__)

# ─── Configuration ───────────────────────────────────────────────────────────
_NOC_BASE = os.getenv("NOC_BASE_URL", "http://noc:8083")
_JWT_TOKEN = os.getenv("KUBRIC_JWT_TOKEN", "")
_TIMEOUT = 10.0
_POLL_INTERVAL = 2.0
_POLL_MAX_WAIT = 60.0


def _headers() -> dict[str, str]:
    """Return standard request headers with JWT bearer auth."""
    h: dict[str, str] = {"Content-Type": "application/json"}
    if _JWT_TOKEN:
        h["Authorization"] = f"Bearer {_JWT_TOKEN}"
    return h


def _noc_url(path: str) -> str:
    """Build a full NOC integrations URL from a relative path."""
    return f"{_NOC_BASE}/integrations{path}"


# =============================================================================
# Wazuh
# =============================================================================

@tool("query_wazuh_agents")
def query_wazuh_agents(status: str = "") -> str:
    """
    List Wazuh agents registered in the environment, optionally filtered by
    status (e.g. 'active', 'disconnected', 'never_connected').

    Returns a JSON object with 'items' (list of agents) and 'total' count.
    Use this to discover which endpoints are being monitored and their health.
    """
    try:
        params: dict[str, str] = {}
        if status:
            params["status"] = status
        resp = httpx.get(
            _noc_url("/wazuh/agents"),
            params=params,
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.query_wazuh_agents.failed", error=str(exc))
        return json.dumps({"error": f"query_wazuh_agents failed: {exc}"})


@tool("query_wazuh_alerts")
def query_wazuh_alerts(agent_id: str, min_level: int = 7) -> str:
    """
    Retrieve high-severity alerts for a specific Wazuh agent.  The min_level
    parameter filters to alerts at or above that Wazuh rule level (default 7,
    which corresponds roughly to HIGH severity).

    agent_id: the Wazuh agent identifier (e.g. '001')
    min_level: minimum Wazuh rule level to include (0-15, default 7)

    Returns a JSON object with 'items' (list of alerts) and 'total' count.
    Use this to investigate alerts on a specific endpoint.
    """
    try:
        resp = httpx.get(
            _noc_url(f"/wazuh/agents/{agent_id}/alerts"),
            params={"min_level": str(min_level)},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.query_wazuh_alerts.failed", error=str(exc))
        return json.dumps({"error": f"query_wazuh_alerts failed: {exc}"})


@tool("query_wazuh_sca")
def query_wazuh_sca(agent_id: str) -> str:
    """
    Retrieve Security Configuration Assessment (SCA) compliance results for a
    Wazuh agent.  SCA evaluates the agent against CIS benchmarks and other
    hardening policies.

    agent_id: the Wazuh agent identifier (e.g. '001')

    Returns a JSON array of SCA policy results including pass/fail counts,
    compliance percentages, and policy descriptions.
    Use this to assess endpoint hardening and compliance posture.
    """
    try:
        resp = httpx.get(
            _noc_url(f"/wazuh/agents/{agent_id}/sca"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.query_wazuh_sca.failed", error=str(exc))
        return json.dumps({"error": f"query_wazuh_sca failed: {exc}"})


@tool("run_wazuh_active_response")
def run_wazuh_active_response(
    agent_id: str,
    command: str,
    arguments: list[str] = [],  # noqa: B006
) -> str:
    """
    Execute an active-response command on a Wazuh agent.  Active responses can
    block IPs, kill processes, or run custom remediation scripts.

    WARNING: This takes action on a live endpoint.  Ensure the command and
    arguments are correct before invoking.

    agent_id:  the Wazuh agent identifier (e.g. '001')
    command:   the active-response command name (e.g. 'firewall-drop0')
    arguments: optional list of command arguments (e.g. ['srcip', '10.0.0.5'])

    Returns {"status": "ok"} on success or an error description.
    """
    try:
        payload: dict = {"command": command}
        if arguments:
            payload["arguments"] = arguments
        resp = httpx.post(
            _noc_url(f"/wazuh/agents/{agent_id}/active-response"),
            json=payload,
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.run_wazuh_active_response.failed", error=str(exc))
        return json.dumps({"error": f"run_wazuh_active_response failed: {exc}"})


# =============================================================================
# Velociraptor
# =============================================================================

@tool("search_velociraptor_clients")
def search_velociraptor_clients(query: str) -> str:
    """
    Search Velociraptor-enrolled endpoints using a search query string.
    Supports hostname, label, and OS-based queries (e.g. 'host:prod-web',
    'label:servers', 'os:windows').

    query: the Velociraptor search query string

    Returns a JSON array of matching client records with client IDs,
    hostnames, OS info, and last-seen timestamps.
    Use this to find specific endpoints for investigation or artifact collection.
    """
    try:
        resp = httpx.get(
            _noc_url("/velociraptor/clients"),
            params={"q": query},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.search_velociraptor_clients.failed", error=str(exc))
        return json.dumps({"error": f"search_velociraptor_clients failed: {exc}"})


@tool("create_velociraptor_hunt")
def create_velociraptor_hunt(
    description: str,
    artifacts: list[str],
) -> str:
    """
    Create a new Velociraptor hunt that runs specified artifacts across all
    enrolled clients (or a targeted subset).  Hunts enable fleet-wide
    investigation and data collection.

    description: human-readable hunt description (e.g. 'Hunt for Log4Shell IOCs')
    artifacts:   list of Velociraptor artifact names to collect
                 (e.g. ['Generic.Detection.Yara.Glob', 'Windows.System.Pslist'])

    Returns the created hunt object with hunt_id, state, and artifact details.
    Use this to initiate proactive threat hunts across the fleet.
    """
    try:
        resp = httpx.post(
            _noc_url("/velociraptor/hunts"),
            json={"description": description, "artifacts": artifacts},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.create_velociraptor_hunt.failed", error=str(exc))
        return json.dumps({"error": f"create_velociraptor_hunt failed: {exc}"})


@tool("run_vql_query")
def run_vql_query(query: str) -> str:
    """
    Execute an arbitrary VQL (Velociraptor Query Language) query on the
    Velociraptor server.  VQL provides powerful access to endpoint telemetry
    including process listings, file searches, registry queries, and more.

    query: a valid VQL statement (e.g. 'SELECT * FROM info()')

    Returns a JSON array of result rows.
    Use this for ad-hoc investigation queries and forensic data retrieval.
    """
    try:
        resp = httpx.post(
            _noc_url("/velociraptor/vql"),
            json={"query": query},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.run_vql_query.failed", error=str(exc))
        return json.dumps({"error": f"run_vql_query failed: {exc}"})


@tool("collect_artifact")
def collect_artifact(client_id: str, artifacts: list[str]) -> str:
    """
    Collect specific artifacts from a single Velociraptor client.  This
    creates a collection flow that gathers the requested data from the
    target endpoint.

    client_id: the Velociraptor client identifier (e.g. 'C.1234abcd')
    artifacts: list of artifact names to collect
               (e.g. ['Windows.System.Pslist', 'Generic.Client.Info'])

    Returns the created flow object with flow_id and status.
    Use this to gather forensic evidence from a specific endpoint.
    """
    try:
        resp = httpx.post(
            _noc_url("/velociraptor/collect"),
            json={"client_id": client_id, "artifacts": artifacts},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.collect_artifact.failed", error=str(exc))
        return json.dumps({"error": f"collect_artifact failed: {exc}"})


# =============================================================================
# TheHive
# =============================================================================

@tool("create_thehive_alert")
def create_thehive_alert(
    title: str,
    description: str,
    severity: int,
    source_ref: str,
    tags: list[str] = [],  # noqa: B006
) -> str:
    """
    Create a new alert in TheHive incident response platform.  Alerts can
    later be promoted to full cases for investigation.

    title:       short alert title (e.g. 'Suspicious PowerShell execution')
    description: detailed alert description with context and evidence
    severity:    1 (low), 2 (medium), 3 (high), or 4 (critical)
    source_ref:  unique reference ID from the source system (prevents duplicates)
    tags:        optional list of tags for categorisation (e.g. ['malware', 'T1059'])

    Returns the created alert object with alert ID and status.
    Use this to escalate findings into the incident response workflow.
    """
    try:
        payload: dict = {
            "title": title,
            "description": description,
            "severity": severity,
            "sourceRef": source_ref,
            "source": "kubric-kai",
            "type": "external",
        }
        if tags:
            payload["tags"] = tags
        resp = httpx.post(
            _noc_url("/thehive/alerts"),
            json=payload,
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.create_thehive_alert.failed", error=str(exc))
        return json.dumps({"error": f"create_thehive_alert failed: {exc}"})


@tool("promote_thehive_alert")
def promote_thehive_alert(alert_id: str) -> str:
    """
    Promote a TheHive alert to a full investigation case.  This creates a new
    case linked to the original alert, copies observables, and enables the
    full incident response workflow (tasks, evidence, reporting).

    alert_id: the TheHive alert identifier to promote

    Returns the created case object with case ID, title, and status.
    Use this when an alert warrants full investigation.
    """
    try:
        resp = httpx.post(
            _noc_url(f"/thehive/alerts/{alert_id}/promote"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.promote_thehive_alert.failed", error=str(exc))
        return json.dumps({"error": f"promote_thehive_alert failed: {exc}"})


@tool("list_thehive_cases")
def list_thehive_cases(status: str = "New") -> str:
    """
    List cases in TheHive filtered by status.

    status: case status filter — 'New', 'InProgress', 'Resolved', 'Closed'
            (default 'New')

    Returns a JSON array of case objects with IDs, titles, severities,
    assignees, and timestamps.
    Use this to review active investigations and workload.
    """
    try:
        resp = httpx.get(
            _noc_url("/thehive/cases"),
            params={"status": status},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.list_thehive_cases.failed", error=str(exc))
        return json.dumps({"error": f"list_thehive_cases failed: {exc}"})


@tool("add_thehive_observable")
def add_thehive_observable(
    case_id: str,
    data_type: str,
    data: str,
    ioc: bool = True,
) -> str:
    """
    Add an observable (indicator of compromise) to an existing TheHive case.
    Observables are automatically enriched by Cortex analyzers when configured.

    case_id:   the TheHive case identifier
    data_type: observable type — 'ip', 'domain', 'hash', 'url', 'mail',
               'filename', 'fqdn', 'registry', 'user-agent', etc.
    data:      the observable value (e.g. '192.168.1.100', 'evil.com', 'abc123...')
    ioc:       whether this observable is a confirmed IOC (default True)

    Returns the created observable object.
    Use this to attach IOCs discovered during investigation to the case.
    """
    try:
        resp = httpx.post(
            _noc_url(f"/thehive/cases/{case_id}/observables"),
            json={
                "dataType": data_type,
                "data": data,
                "ioc": ioc,
                "message": f"Added by KAI at {int(time.time())}",
            },
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.add_thehive_observable.failed", error=str(exc))
        return json.dumps({"error": f"add_thehive_observable failed: {exc}"})


# =============================================================================
# Cortex
# =============================================================================

@tool("run_cortex_analyzer")
def run_cortex_analyzer(analyzer_id: str, data: str, data_type: str) -> str:
    """
    Submit data to a Cortex analyzer for enrichment and wait for the analysis
    results.  Cortex analyzers provide threat intelligence lookups, sandbox
    analysis, reputation checks, and more.

    analyzer_id: the Cortex analyzer identifier (e.g. 'VirusTotal_GetReport_3_1')
    data:        the data to analyse (e.g. an IP address, domain, file hash)
    data_type:   the type of data — 'ip', 'domain', 'hash', 'url', 'mail',
                 'filename', 'fqdn'

    Returns the analyzer report JSON including taxonomies, summary, and
    full analysis results.  Polls until the job completes (up to 60 seconds).
    Use this to enrich IOCs with threat intelligence context.
    """
    try:
        # Submit the analysis job
        submit_resp = httpx.post(
            _noc_url(f"/cortex/analyzers/{analyzer_id}/run"),
            json={"data": data, "dataType": data_type},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        submit_resp.raise_for_status()
        job = submit_resp.json()
        job_id = job.get("id") or job.get("job_id") or job.get("_id", "")
        if not job_id:
            return json.dumps(job)

        # Poll for completion
        start = time.monotonic()
        while time.monotonic() - start < _POLL_MAX_WAIT:
            time.sleep(_POLL_INTERVAL)
            status_resp = httpx.get(
                _noc_url(f"/cortex/jobs/{job_id}"),
                headers=_headers(),
                timeout=_TIMEOUT,
            )
            status_resp.raise_for_status()
            job_status = status_resp.json()
            state = job_status.get("status", "").lower()
            if state in ("success", "failure"):
                # Fetch the full report
                report_resp = httpx.get(
                    _noc_url(f"/cortex/jobs/{job_id}/report"),
                    headers=_headers(),
                    timeout=_TIMEOUT,
                )
                report_resp.raise_for_status()
                return json.dumps(report_resp.json())

        return json.dumps({"error": "cortex job timed out", "job_id": job_id})
    except Exception as exc:
        log.debug("tool.run_cortex_analyzer.failed", error=str(exc))
        return json.dumps({"error": f"run_cortex_analyzer failed: {exc}"})


@tool("list_cortex_analyzers")
def list_cortex_analyzers(data_type: str = "") -> str:
    """
    List available Cortex analyzers, optionally filtered by the data type
    they accept (e.g. 'ip', 'domain', 'hash', 'url').

    data_type: optional data type filter (empty string returns all analyzers)

    Returns a JSON array of analyzer objects with IDs, names, descriptions,
    supported data types, and version information.
    Use this to discover which enrichment analyzers are available before
    running an analysis.
    """
    try:
        params: dict[str, str] = {}
        if data_type:
            params["dataType"] = data_type
        resp = httpx.get(
            _noc_url("/cortex/analyzers"),
            params=params,
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.list_cortex_analyzers.failed", error=str(exc))
        return json.dumps({"error": f"list_cortex_analyzers failed: {exc}"})


# =============================================================================
# osquery (FleetDM)
# =============================================================================

@tool("query_osquery_hosts")
def query_osquery_hosts(status: str = "") -> str:
    """
    List osquery-enrolled hosts managed by FleetDM, optionally filtered by
    status (e.g. 'online', 'offline').

    status: optional status filter (empty string returns all hosts)

    Returns a JSON object with 'items' (list of hosts with hostnames, OS info,
    osquery version, and last-seen timestamps) and 'total' count.
    Use this to get an overview of the osquery fleet and host health.
    """
    try:
        params: dict[str, str] = {}
        if status:
            params["status"] = status
        resp = httpx.get(
            _noc_url("/osquery/hosts"),
            params=params,
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.query_osquery_hosts.failed", error=str(exc))
        return json.dumps({"error": f"query_osquery_hosts failed: {exc}"})


@tool("run_osquery_live")
def run_osquery_live(sql: str, host_ids: list[int] = []) -> str:  # noqa: B006
    """
    Execute a distributed live SQL query across osquery-enrolled hosts via
    FleetDM.  osquery provides real-time visibility into endpoint state
    including processes, network connections, installed software, and more.

    sql:      a valid osquery SQL statement
              (e.g. 'SELECT * FROM processes WHERE name = \"cmd.exe\"')
    host_ids: list of FleetDM host IDs to target (required, at least one)

    Returns a JSON object with query results grouped by host.
    Use this for real-time endpoint investigation and posture checks.
    """
    try:
        resp = httpx.post(
            _noc_url("/osquery/queries/run"),
            json={"query": sql, "host_ids": host_ids},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.run_osquery_live.failed", error=str(exc))
        return json.dumps({"error": f"run_osquery_live failed: {exc}"})


# =============================================================================
# BloodHound CE
# =============================================================================

@tool("query_bloodhound_domains")
def query_bloodhound_domains() -> str:
    """
    List Active Directory domains that have been ingested into BloodHound CE.
    Each domain entry includes the domain SID, name, collected node counts,
    and last-ingestion timestamp.

    Returns a JSON array of domain objects.
    Use this to discover which AD environments are available for attack-path
    analysis.
    """
    try:
        resp = httpx.get(
            _noc_url("/bloodhound/domains"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.query_bloodhound_domains.failed", error=str(exc))
        return json.dumps({"error": f"query_bloodhound_domains failed: {exc}"})


@tool("run_bloodhound_cypher")
def run_bloodhound_cypher(query: str) -> str:
    """
    Execute a Cypher graph query against the BloodHound CE database.  Cypher
    queries can traverse the Active Directory attack graph to find privilege
    escalation paths, kerberoastable accounts, AS-REP roastable users, and
    other AD security issues.

    query: a valid Cypher query string
           (e.g. 'MATCH (n:User {hasspn:true}) RETURN n.name LIMIT 10')

    Returns a JSON object with the query result set.
    Use this for custom AD security analysis beyond the built-in attack paths.
    """
    try:
        resp = httpx.post(
            _noc_url("/bloodhound/cypher"),
            json={"query": query},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.run_bloodhound_cypher.failed", error=str(exc))
        return json.dumps({"error": f"run_bloodhound_cypher failed: {exc}"})


@tool("list_attack_paths")
def list_attack_paths(domain_id: str) -> str:
    """
    List pre-computed attack paths for a specific Active Directory domain in
    BloodHound CE.  Attack paths show how an adversary could escalate
    privileges from a compromised user/computer to Domain Admin or other
    high-value targets.

    domain_id: the BloodHound domain identifier (typically the domain SID)

    Returns a JSON array of attack path objects with source, target, path
    length, and the relationship chain.
    Use this to identify and prioritise AD security weaknesses.
    """
    try:
        resp = httpx.get(
            _noc_url(f"/bloodhound/domains/{domain_id}/attack-paths"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.list_attack_paths.failed", error=str(exc))
        return json.dumps({"error": f"list_attack_paths failed: {exc}"})


# =============================================================================
# Shuffle SOAR
# =============================================================================

@tool("list_shuffle_workflows")
def list_shuffle_workflows() -> str:
    """
    List all workflows configured in the Shuffle SOAR platform.  Workflows
    define automated response playbooks (e.g. phishing triage, IOC
    enrichment, ticket creation).

    Returns a JSON array of workflow objects with IDs, names, descriptions,
    trigger types, and enabled status.
    Use this to discover available automation playbooks before triggering one.
    """
    try:
        resp = httpx.get(
            _noc_url("/shuffle/workflows"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.list_shuffle_workflows.failed", error=str(exc))
        return json.dumps({"error": f"list_shuffle_workflows failed: {exc}"})


@tool("execute_shuffle_workflow")
def execute_shuffle_workflow(
    workflow_id: str,
    argument: dict = {},  # noqa: B006
) -> str:
    """
    Trigger execution of a Shuffle SOAR workflow.  The argument dict is passed
    as the workflow's input payload and can contain any key/value pairs the
    workflow expects (e.g. alert details, IOCs, case IDs).

    workflow_id: the Shuffle workflow identifier
    argument:    optional dict of input parameters for the workflow
                 (e.g. {"alert_id": "abc123", "severity": "CRITICAL"})

    Returns the execution object with execution_id and initial status.
    Use this to trigger automated response playbooks.
    """
    try:
        resp = httpx.post(
            _noc_url(f"/shuffle/workflows/{workflow_id}/execute"),
            json={"argument": argument},
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.execute_shuffle_workflow.failed", error=str(exc))
        return json.dumps({"error": f"execute_shuffle_workflow failed: {exc}"})


# =============================================================================
# Aggregate health check
# =============================================================================

@tool("check_integration_health")
def check_integration_health() -> str:
    """
    Check the connectivity and health status of all configured security-tool
    integrations (Wazuh, Velociraptor, TheHive, Cortex, Falco, osquery,
    Shuffle, BloodHound).

    Returns a JSON object with an overall 'status' ('ok' or 'degraded')
    and an 'integrations' array where each entry shows the integration name,
    its status ('ok', 'unavailable', or 'not_configured'), and any error
    message.

    Use this as a first step to verify which integrations are available
    before attempting to use their specific tools.
    """
    try:
        resp = httpx.get(
            _noc_url("/health"),
            headers=_headers(),
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        return json.dumps(resp.json())
    except Exception as exc:
        log.debug("tool.check_integration_health.failed", error=str(exc))
        return json.dumps({"error": f"check_integration_health failed: {exc}"})
