# K-VENDOR-SHF-002 -- Shuffle SOAR Workflows

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Security orchestration playbooks               |
| Format      | Shuffle workflow JSON definitions              |
| Consumer    | KAI-KEEPER, KAI-COMM                           |

## Purpose

Pre-built Shuffle workflows that automate multi-step incident response,
notification, and remediation sequences. KAI-KEEPER triggers workflows
via the Shuffle REST API.

## Key Workflows

| Workflow                     | Steps                                    |
|------------------------------|------------------------------------------|
| Phishing Response            | Extract IOCs, check reputation, block    |
|                              | sender domain, notify user, close alert  |
| Endpoint Isolation           | Verify alert, isolate host via EDR API,  |
|                              | collect forensics, notify SOC            |
| Malware Containment          | Hash lookup, quarantine file, scan       |
|                              | neighbors, update case in TheHive        |
| Brute Force Mitigation       | Block source IP, reset credentials,      |
|                              | enable MFA, notify account owner         |
| Escalation Notification      | Route to on-call via PagerDuty, send     |
|                              | Slack/Teams summary, update case TLP     |

## Integration Flow

1. KAI-KEEPER selects appropriate workflow based on alert category.
2. POSTs execution request to `https://shuffle-{tenant}/api/v1/workflows/{id}/execute`.
3. Passes alert context and tenant metadata as workflow arguments.
4. Polls `GET /api/v1/workflows/{id}/executions/{exec_id}` for status.
5. Publishes workflow outcome to `kubric.kai.keeper.result`.

## Workflow Design

- Each workflow is idempotent and includes rollback steps on failure.
- Approval gates can be inserted for high-impact actions.
- Shuffle apps (Wazuh, TheHive, Cortex, etc.) are used as action nodes.
