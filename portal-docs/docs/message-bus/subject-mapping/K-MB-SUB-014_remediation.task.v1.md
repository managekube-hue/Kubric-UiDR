# K-MB-SUB-014 — Remediation Task

> NATS subject mapping reference for auto-remediation directive dispatch within the Kubric UIDR platform.

## Subject Pattern

```
kubric.remediation.task.v1
kubric.remediation.task.>       # wildcard for all remediation task sub-subjects
```

Tokens:
| Position | Token           | Description                                    |
|----------|-----------------|------------------------------------------------|
| 1        | `kubric`        | Platform root namespace                        |
| 2        | `remediation`   | Domain — automated remediation                 |
| 3        | `task`          | Event type — remediation task directive        |
| 4        | `v1`            | Schema version                                 |

## Publisher

**KAI-KEEPER** — the AI-driven remediation orchestrator within the KAI crew. KAI-KEEPER evaluates triaged incidents from KAI-TRIAGE and generates remediation tasks based on playbook templates and AI-recommended actions.

Publish triggers:
| Trigger                   | Description                                                |
|---------------------------|------------------------------------------------------------|
| Auto-remediation rule hit | Preconfigured playbook matches a triage outcome            |
| AI recommendation         | KAI-KEEPER AI model recommends remediation with confidence >= 80 |
| Analyst approval          | Human analyst approves a queued remediation via K-SVC Portal |
| Scheduled maintenance     | Time-triggered remediation tasks (e.g., weekly patching)   |

## Consumer(s)

| Consumer                | Runtime | Role                                                          |
|-------------------------|---------|----------------------------------------------------------------|
| **CoreSec** (primary)   | Rust    | Executes endpoint-level remediation: isolate, quarantine, config fix |
| **KIC** (primary)       | Go      | Executes infrastructure-level remediation: patch, block IP, policy enforcement |
| K-SVC Portal            | TypeScript / Next.js | Displays remediation task status and history        |

## Payload

**Format:** JSON — Remediation command directive with approval and rollback metadata.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "task_id": "rtk_n3o4p5q6",
  "tenant_id": "tnt_a1b2c3d4",
  "target_asset": {
    "asset_id": "ast_j9k0l1m2",
    "hostname": "WORKSTATION-042",
    "ip": "10.0.5.42"
  },
  "action_type": "isolate",
  "parameters": {
    "isolation_level": "full",
    "allow_dns": true,
    "allow_kubric_agent": true,
    "reason": "Ransomware behavior detected — LockBit 3.0 TTP (T1486)"
  },
  "priority": "critical",
  "approved_by": "kai-keeper-auto",
  "approval_method": "playbook_auto",
  "timeout_seconds": 300,
  "rollback": {
    "enabled": true,
    "auto_rollback_on_failure": true,
    "rollback_action": "unisolate",
    "rollback_timeout_seconds": 120
  },
  "related_alert_id": "alt_e5f6g7h8",
  "timestamp": "2026-02-26T14:30:15Z"
}
```

## Fields

| Field                            | Type     | Required | Description                                                            |
|----------------------------------|----------|----------|------------------------------------------------------------------------|
| `schema_version`                 | string   | yes      | Payload schema version (semver)                                        |
| `task_id`                        | string   | yes      | Unique remediation task identifier (`rtk_` prefix)                     |
| `tenant_id`                      | string   | yes      | Tenant owning the target asset (`tnt_` prefix)                         |
| `target_asset.asset_id`          | string   | yes      | Target asset identifier (`ast_` prefix)                                |
| `target_asset.hostname`          | string   | yes      | Hostname of the target system                                          |
| `target_asset.ip`                | string   | yes      | IP address of the target system                                        |
| `action_type`                    | string   | yes      | Remediation action: `patch`, `isolate`, `block_ip`, `quarantine`, `config_fix` |
| `parameters`                     | object   | yes      | Action-specific parameters (schema varies by `action_type`)            |
| `priority`                       | string   | yes      | Task priority: `critical`, `high`, `medium`, `low`                     |
| `approved_by`                    | string   | yes      | Approver identity: `kai-keeper-auto`, analyst email, or API key name   |
| `approval_method`                | string   | yes      | How approval was granted: `playbook_auto`, `analyst_manual`, `api`     |
| `timeout_seconds`                | integer  | yes      | Execution timeout; task is marked failed if not completed within this window |
| `rollback.enabled`               | boolean  | yes      | Whether rollback is available for this action                          |
| `rollback.auto_rollback_on_failure` | boolean | no    | Automatically execute rollback if the primary action fails             |
| `rollback.rollback_action`       | string   | no       | Reverse action to execute on rollback                                  |
| `rollback.rollback_timeout_seconds` | integer | no    | Timeout for the rollback action                                        |
| `related_alert_id`               | string   | no       | Alert ID that triggered this remediation task                          |
| `timestamp`                      | datetime | yes      | ISO 8601 UTC timestamp of task creation                                |

### Action Type Reference

| Action Type   | Executor | Description                                                    | Rollback Action   |
|---------------|----------|----------------------------------------------------------------|-------------------|
| `patch`       | KIC      | Apply OS or application security patch                         | `rollback_patch`  |
| `isolate`     | CoreSec  | Network-isolate the endpoint (allow agent heartbeat only)      | `unisolate`       |
| `block_ip`    | KIC      | Add IP to firewall block list across perimeter infrastructure  | `unblock_ip`      |
| `quarantine`  | CoreSec  | Move malicious file to secure quarantine vault                 | `restore_file`    |
| `config_fix`  | KIC      | Apply configuration hardening (e.g., disable SMBv1, enforce MFA) | `revert_config` |

### Approval Workflow

Remediation tasks follow a tiered approval model based on action risk and tenant configuration:

```
Risk Assessment
  |
  +--> Low Risk (config_fix, block_ip)
  |      --> Auto-approved by KAI-KEEPER playbook
  |
  +--> Medium Risk (quarantine, patch)
  |      --> Auto-approved if confidence >= 90, else queued for analyst
  |
  +--> High Risk (isolate)
         --> Auto-approved ONLY for critical severity with confidence >= 95
         --> Otherwise queued for analyst approval via K-SVC Portal
```

Tenants can override this model to require manual approval for all actions (`approval_mode: manual_only`).

### Rollback Support

Every remediation action has a corresponding rollback action. Rollback can be triggered:

1. **Automatically** — if `auto_rollback_on_failure` is `true` and the primary action fails or times out.
2. **Manually** — by an analyst via the K-SVC Portal within 72 hours of execution.
3. **Scheduled** — for temporary actions (e.g., `isolate` can be set to auto-unisolate after investigation completes).

Rollback status is published to `kubric.remediation.status.v1`.

## JetStream Configuration

```
Stream:          REMEDIATION_TASK
Subjects:        kubric.remediation.task.>
Storage:         File
Retention:       Limits
Max Age:         30 days
Max Bytes:       5 GB
Replicas:        3
Discard Policy:  Old
Duplicate Window: 30 seconds
```

## Consumer Groups

| Consumer Group         | Deliver Policy | Ack Policy  | Max Deliver | Filter Subject                  |
|------------------------|---------------|-------------|-------------|----------------------------------|
| `coresec-executor`     | All           | Explicit    | 5           | `kubric.remediation.task.>`      |
| `kic-executor`         | All           | Explicit    | 5           | `kubric.remediation.task.>`      |
| `portal-task-tracker`  | All           | Explicit    | 3           | `kubric.remediation.task.>`      |

Note: CoreSec and KIC consumers filter tasks by `action_type` in application logic. CoreSec handles `isolate`, `quarantine`; KIC handles `patch`, `block_ip`, `config_fix`.

## Example (NATS CLI)

**Publish a remediation task:**

```bash
nats pub kubric.remediation.task.v1 '{
  "schema_version": "1.0.0",
  "task_id": "rtk_n3o4p5q6",
  "tenant_id": "tnt_a1b2c3d4",
  "target_asset": {
    "asset_id": "ast_j9k0l1m2",
    "hostname": "WORKSTATION-042",
    "ip": "10.0.5.42"
  },
  "action_type": "isolate",
  "parameters": {
    "isolation_level": "full",
    "allow_dns": true,
    "allow_kubric_agent": true,
    "reason": "Ransomware behavior detected"
  },
  "priority": "critical",
  "approved_by": "kai-keeper-auto",
  "approval_method": "playbook_auto",
  "timeout_seconds": 300,
  "rollback": {
    "enabled": true,
    "auto_rollback_on_failure": true,
    "rollback_action": "unisolate",
    "rollback_timeout_seconds": 120
  },
  "related_alert_id": "alt_e5f6g7h8",
  "timestamp": "2026-02-26T14:30:15Z"
}'
```

**Subscribe to all remediation tasks:**

```bash
nats sub "kubric.remediation.task.>"
```

**Create the JetStream stream:**

```bash
nats stream add REMEDIATION_TASK \
  --subjects "kubric.remediation.task.>" \
  --storage file \
  --retention limits \
  --max-age 30d \
  --max-bytes 5GB \
  --replicas 3 \
  --discard old \
  --dupe-window 30s
```

**Create the coresec-executor consumer:**

```bash
nats consumer add REMEDIATION_TASK coresec-executor \
  --deliver all \
  --ack explicit \
  --max-deliver 5 \
  --filter "kubric.remediation.task.>"
```

## Notes

- **Idempotency:** Each remediation task is identified by `task_id`. Consumers must treat re-delivered tasks idempotently — if the action has already been executed for that `task_id`, acknowledge without re-executing.
- **Execution Feedback:** After executing a task, CoreSec and KIC publish a status update to `kubric.remediation.status.v1` with outcome (`success`, `failed`, `timeout`, `rolled_back`).
- **Audit Compliance:** All remediation actions are logged with full provenance (`approved_by`, `approval_method`, `timestamp`) for SOC 2 and compliance audit trails.
- **Blast Radius Control:** KAI-KEEPER enforces a maximum concurrent isolation limit per tenant (default: 10 assets) to prevent accidental mass isolation from a false positive cascade.
- **Timeout Handling:** If a task exceeds `timeout_seconds`, the consumer publishes a `timeout` status. If `auto_rollback_on_failure` is enabled, the rollback action is automatically triggered.
- **Related Subjects:** `kubric.remediation.status.v1` (execution outcome), `kubric.security.alert.v1` (upstream alert trigger), `kubric.comm.alert.v1` (notification of remediation action taken).
