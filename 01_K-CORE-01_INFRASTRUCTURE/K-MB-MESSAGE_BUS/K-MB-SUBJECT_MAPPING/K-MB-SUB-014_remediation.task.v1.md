# K-MB-SUB-014 — remediation.task.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.remediation.task.v1`                        |
| **Stream**         | `REMEDIATION-TASK`                                              |
| **Version**        | v1                                                              |
| **Purpose**        | Carries lifecycle events for every remediation task created on the Kubric platform. A remediation task represents a concrete action to eliminate a risk: applying a patch, correcting a configuration drift finding, rolling back a failed change, executing a config change playbook, or quarantining a compromised asset. Publishers create tasks when risks are identified and update them as execution progresses through pending, running, complete, failed, or cancelled states. Consumers drive automated execution via SaltStack, Ansible, and Temporal workflows, and feed the audit log for SOC accountability. |
| **Owner**          | KAI Housekeeper / Remediation Engineering                       |
| **SLA**            | At-least-once delivery; `immediate` priority tasks must be acknowledged by `deploy-agent` within 30 seconds of publication |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: REMEDIATION-TASK
  subjects:
    - "kubric.*.remediation.task.v1"
  retention: limits
  max_age: 31536000       # 365 days
  max_bytes: 10737418240  # 10 GB
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: deploy-agent
  durable: deploy-agent
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> Additional durable consumers `saltstack-client`, `ansible-runner`, and `audit-log` share this stream with identical ack settings.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/remediation/task/v1",
  "title": "RemediationTaskEvent",
  "type": "object",
  "required": ["tenant_id", "task_id", "type", "asset_id", "priority", "status", "created_at"],
  "additionalProperties": false,
  "properties": {
    "tenant_id": { "type": "string", "format": "uuid" },
    "task_id": { "type": "string", "format": "uuid" },
    "type": {
      "type": "string",
      "enum": ["patch", "drift_fix", "rollback", "config_change", "quarantine"]
    },
    "asset_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID of the target asset from the ASSET-EVENTS stream."
    },
    "priority": {
      "type": "string",
      "enum": ["immediate", "out_of_cycle", "scheduled", "defer"],
      "description": "Execution urgency: immediate=within 4h, out_of_cycle=within 72h, scheduled=next maintenance window, defer=risk-accepted."
    },
    "status": {
      "type": "string",
      "enum": ["pending", "running", "complete", "failed", "cancelled"]
    },
    "playbook_ref": {
      "type": ["string", "null"],
      "maxLength": 512,
      "description": "Reference to the Ansible playbook or SaltStack state file, e.g. 'ansible/playbooks/patch_windows.yml'. Null if not applicable."
    },
    "temporal_workflow_id": {
      "type": ["string", "null"],
      "maxLength": 256,
      "description": "Temporal workflow run ID for orchestrated tasks. Null for simple one-shot executions."
    },
    "ansible_run_id": {
      "type": ["string", "null"],
      "maxLength": 256,
      "description": "AWX or Ansible Controller job ID for tracked executions. Null if not using Ansible Controller."
    },
    "created_at": { "type": "string", "format": "date-time" },
    "started_at": { "type": ["string", "null"], "format": "date-time" },
    "completed_at": { "type": ["string", "null"], "format": "date-time" },
    "error_message": {
      "type": ["string", "null"],
      "maxLength": 2048,
      "description": "Human-readable failure reason. Null unless status is failed."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — Patch task created (pending), immediate priority
```json
{
  "tenant_id": "d5e6f7a8-b9c0-4d1e-a2f3-b4c5d6e7f8a9",
  "task_id": "aabb1122-ccdd-4eef-f001-223344556677",
  "type": "patch",
  "asset_id": "cc3344ff-5566-4778-99aa-bbccddeeff00",
  "priority": "immediate",
  "status": "pending",
  "playbook_ref": "ansible/playbooks/patch_windows_critical.yml",
  "temporal_workflow_id": null,
  "ansible_run_id": null,
  "created_at": "2026-02-26T03:16:00.000Z",
  "started_at": null,
  "completed_at": null,
  "error_message": null
}
```

### 4b — Patch task running via Temporal workflow
```json
{
  "tenant_id": "d5e6f7a8-b9c0-4d1e-a2f3-b4c5d6e7f8a9",
  "task_id": "aabb1122-ccdd-4eef-f001-223344556677",
  "type": "patch",
  "asset_id": "cc3344ff-5566-4778-99aa-bbccddeeff00",
  "priority": "immediate",
  "status": "running",
  "playbook_ref": "ansible/playbooks/patch_windows_critical.yml",
  "temporal_workflow_id": "patch-wf-01HX9MZPQR2STV4UWXY5",
  "ansible_run_id": "AWX-JOB-10847",
  "created_at": "2026-02-26T03:16:00.000Z",
  "started_at": "2026-02-26T03:16:28.441Z",
  "completed_at": null,
  "error_message": null
}
```

### 4c — Drift fix completed successfully
```json
{
  "tenant_id": "d5e6f7a8-b9c0-4d1e-a2f3-b4c5d6e7f8a9",
  "task_id": "bb2233cc-ddee-4ff0-a112-233445566778",
  "type": "drift_fix",
  "asset_id": "ee5566ff-7788-4990-aabb-001122334455",
  "priority": "out_of_cycle",
  "status": "complete",
  "playbook_ref": "salt/states/linux_cis_hardening.sls",
  "temporal_workflow_id": null,
  "ansible_run_id": null,
  "created_at": "2026-02-25T10:00:00.000Z",
  "started_at": "2026-02-25T10:00:45.000Z",
  "completed_at": "2026-02-25T10:07:33.821Z",
  "error_message": null
}
```

### 4d — Quarantine task for compromised host, failed with error
```json
{
  "tenant_id": "d5e6f7a8-b9c0-4d1e-a2f3-b4c5d6e7f8a9",
  "task_id": "cc3344dd-eeff-4001-b223-344556677889",
  "type": "quarantine",
  "asset_id": "dd4455ee-6677-4889-aabb-ccddee001122",
  "priority": "immediate",
  "status": "failed",
  "playbook_ref": "ansible/playbooks/network_quarantine.yml",
  "temporal_workflow_id": "quarantine-wf-01HX9NZABCD2EFG4HIJK",
  "ansible_run_id": "AWX-JOB-10851",
  "created_at": "2026-02-26T09:23:00.000Z",
  "started_at": "2026-02-26T09:23:15.000Z",
  "completed_at": null,
  "error_message": "Ansible playbook failed on task 'Update FortiGate address group': Connection timeout to firewall API 10.0.0.1:443 after 30s (attempt 3/3)"
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                                   |
|--------------------|-------------------------------------------------------------------------------------------------------------------------|
| **Service A**      | KAI Housekeeper agent                                                                                                   |
| **File A**         | `K-KAI-HS-001_housekeeper_agent.py`                                                                                    |
| **Service B**      | Temporal patch workflow                                                                                                 |
| **File B**         | `K-KAI-WF-TEMP-001_patch_workflow.py`                                                                                  |
| **Trigger A**      | Housekeeper agent publishes on task creation (status=pending) and on all status transitions                             |
| **Trigger B**      | Temporal workflow publishes status updates at each workflow activity milestone                                           |
| **Subject format** | `kubric.{tenant_id}.remediation.task.v1`                                                                               |
| **Library**        | `nats.py` async JetStream `publish_async`; `Nats-Msg-Id` = `{task_id}:{status}:{created_at_epoch_ms}`                 |
| **Serialisation**  | UTF-8 JSON                                                                                                             |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`, `Kubric-Task-Type: {type}`, `Kubric-Priority: {priority}`  |

Each task lifecycle event uses the same `task_id` with updated `status`. Consumers reconstruct full task history by replaying all messages for a given `task_id`.

---

## 6. Consumer Details

### deploy-agent
| Attribute          | Value                                                                                                   |
|--------------------|---------------------------------------------------------------------------------------------------------|
| **File**           | `K-KAI-DEP-001_deploy_agent.py`                                                                         |
| **Role**           | Receives `status: pending` tasks with `priority: immediate` or `out_of_cycle`. Initiates execution by launching the appropriate Temporal workflow or Ansible AWX job, then publishes a `running` status update. |
| **Filter**         | Only processes `status: pending` messages                                                               |
| **Batch size**     | 10 messages per pull                                                                                    |
| **Error behaviour**| NACKs on Temporal client timeout; dead-letters after 5 redeliveries                                    |

### saltstack-client
| Attribute          | Value                                                                                                   |
|--------------------|---------------------------------------------------------------------------------------------------------|
| **File**           | `K-KAI-DEP-002_saltstack_client.py`                                                                     |
| **Role**           | Handles `type: drift_fix` tasks by executing the referenced SaltStack state via the Salt REST API. Updates status to `running` on job submit and `complete` or `failed` on job result. |
| **Filter**         | Only processes `type: drift_fix` and `status: pending`                                                  |
| **Batch size**     | 20 messages per pull                                                                                    |
| **Error behaviour**| NACKs on Salt API unreachable; dead-letters after 5 redeliveries                                       |

### ansible-runner
| Attribute          | Value                                                                                                   |
|--------------------|---------------------------------------------------------------------------------------------------------|
| **File**           | `K-KAI-HS-002_ansible_runner.py`                                                                        |
| **Role**           | Handles `type: patch`, `type: config_change`, and `type: quarantine` tasks via Ansible AWX job templates. Polls AWX for job status and publishes final `complete` or `failed` task updates. |
| **Filter**         | Processes `type IN (patch, config_change, quarantine)` and `status: pending`                            |
| **Batch size**     | 10 messages per pull                                                                                    |
| **Error behaviour**| NACKs on AWX API connection failure; dead-letters after 5 redeliveries                                 |

### audit-log
| Attribute          | Value                                                                                                   |
|--------------------|---------------------------------------------------------------------------------------------------------|
| **Role**           | Appends every task event to Postgres `audit.remediation_log` for compliance tracking. Provides complete task history with immutable append semantics. |
| **Batch size**     | 500 messages per pull                                                                                   |
| **Error behaviour**| NACKs on Postgres write failure; dead-letters after 5 redeliveries                                     |

---

## 7. Retention Policy

| Parameter            | Value                                                                                             |
|----------------------|---------------------------------------------------------------------------------------------------|
| **Stream retention** | `limits`                                                                                          |
| **max_age**          | 31,536,000 seconds (365 days)                                                                     |
| **max_bytes**        | 10 GB                                                                                             |
| **Replicas**         | 3                                                                                                 |
| **Storage tier**     | `file`                                                                                            |
| **Archive policy**   | Postgres `audit.remediation_log` retains full task history indefinitely. Monthly Parquet exports to S3 for analytics on remediation velocity and SLA compliance. |

---

## 8. Error Handling

| Scenario                              | Behaviour                                                                                                                                       |
|---------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|
| **Temporal service unavailable**      | deploy-agent queues task IDs locally and retries every 60 seconds with exponential backoff. Prometheus fires `remediation_temporal_unavailable` after 5 failed attempts. |
| **Immediate priority SLA breach**     | A watchdog compares `created_at` to current time for `status: pending` tasks with `priority: immediate`. If age > 30 seconds, emits `kubric.<tenant_id>.ops.remediation.sla_breach.v1` and pages the on-call engineer. |
| **Schema validation failure**         | Invalid messages forwarded immediately to `kubric.<tenant_id>.dlq.remediation.task.v1` with `Kubric-Error-Reason: SCHEMA_INVALID`. |
| **Max redeliveries exceeded**         | NATS routes to `kubric.<tenant_id>.dlq.remediation.task.v1`. PagerDuty alert at DLQ depth > 3 for `immediate` priority. |
| **Ansible AWX job failure**           | ansible-runner publishes `status: failed` with `error_message` from AWX job stdout. Housekeeper agent evaluates the failure and may auto-retry once before escalating via the comm-agent. |
| **Duplicate task submission**         | `Nats-Msg-Id: {task_id}:{status}:{epoch}` deduplicates within the 120-second window. deploy-agent performs an idempotency check in Postgres before executing. |
| **Dead-letter subject**               | `kubric.<tenant_id>.dlq.remediation.task.v1`                                                                                                    |
