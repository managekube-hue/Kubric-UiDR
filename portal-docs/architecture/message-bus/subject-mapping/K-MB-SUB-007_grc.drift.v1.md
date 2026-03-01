# K-MB-SUB-007 — GRC Configuration Drift

> NATS subject mapping reference for compliance configuration drift events in the Kubric UIDR platform.

## Subject Pattern

```
kubric.grc.drift.v1
kubric.grc.drift.v1.<tenant_id>
kubric.grc.drift.v1.<tenant_id>.<host_id>
```

Wildcard subscription: `kubric.grc.drift.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **KIC** | Go | Kubric Integrity & Compliance engine. Evaluates host configurations against compliance baselines using osquery-based checks and SCA (Security Configuration Assessment) policies. Publishes drift events when a host deviates from its expected compliance posture. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-KEEPER** | Python (CrewAI) | Compliance management AI agent. Evaluates drift severity, determines appropriate remediation strategy (auto-remediate vs. manual review), triggers configuration management actions via SaltStack or Ansible, and maintains compliance posture dashboards. |

## Payload (OCSF Class)

- **OCSF Class**: Compliance Finding (`5003`)
- **OCSF Category**: Discovery
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `framework` | `string` | `compliance.requirements[].control.framework` | Compliance framework identifier (e.g., `"CIS_Ubuntu_22.04_L1"`, `"DISA_STIG_RHEL9"`, `"PCI-DSS_4.0"`, `"HIPAA"`, `"SOC2"`). |
| 2 | `control_id` | `string` | `compliance.requirements[].control.uid` | Framework-specific control identifier (e.g., `"1.1.1.1"` for CIS, `"V-230221"` for STIG, `"2.2.1"` for PCI-DSS). |
| 3 | `check_id` | `string` | `finding_info.uid` | Unique KIC check identifier linking to the osquery/SCA policy rule (e.g., `"kic-cis-ubuntu-1.1.1.1"`). |
| 4 | `status` | `string` | `status` | Check result: `"pass"`, `"fail"`, `"error"`, `"not_applicable"`, `"manual_review"`. Drift events are published when status transitions to `"fail"`. |
| 5 | `evidence` | `string` | `finding_info.desc` | Evidence string capturing the actual vs. expected state (e.g., `"expected: permissions 0600, actual: permissions 0644 on /etc/shadow"`). |
| 6 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 7 | `host_id` | `string` | `device.uid` | Unique host identifier where the drift was detected. |
| 8 | `timestamp_ns` | `uint64` | `time` | Timestamp of the compliance check in nanoseconds since Unix epoch. |
| 9 | `previous_status` | `string` | `unmapped.previous_status` | Previous check status before the drift (typically `"pass"`). Enables transition tracking. |
| 10 | `control_title` | `string` | `compliance.requirements[].control.title` | Human-readable control title (e.g., `"Ensure permissions on /etc/shadow are configured"`). |
| 11 | `remediation_hint` | `string` | `unmapped.remediation_hint` | Suggested remediation command or guidance (e.g., `"chmod 0600 /etc/shadow"`). |
| 12 | `severity_id` | `uint32` | `severity_id` | OCSF severity based on control criticality: 5=Critical, 4=High, 3=Medium, 2=Low. |
| 13 | `risk_score` | `float` | `risk_score` | Computed risk score (0-100) factoring control severity, asset criticality, and exposure. |

## Drift Detection via osquery/SCA

KIC performs compliance checks using two complementary mechanisms:

### osquery-based checks

KIC manages an osquery daemon on each endpoint with a compliance-specific query pack:

```sql
-- Example: CIS Ubuntu 22.04 L1 - 1.1.1.1 Ensure cramfs is disabled
SELECT 'fail' AS status,
       'CIS_Ubuntu_22.04_L1' AS framework,
       '1.1.1.1' AS control_id
FROM kernel_modules
WHERE name = 'cramfs' AND status = 'live';
```

Query packs are generated from framework-specific templates and pushed to endpoints via the CoreSec policy mechanism. osquery queries run on a configurable schedule (default: every 15 minutes for critical controls, every 1 hour for standard controls).

### SCA (Security Configuration Assessment) policies

For checks that go beyond osquery's capabilities (e.g., file content parsing, registry deep inspection, multi-step evaluations), KIC uses Wazuh-compatible SCA policy files:

```yaml
# Example SCA check
- id: 10001
  title: "Ensure SSH root login is disabled"
  condition: all
  rules:
    - 'f:/etc/ssh/sshd_config -> r:^PermitRootLogin\s+no$'
```

### Drift detection logic

```
Scheduled Check Execution
        |
        v
  Current Status
        |
        +-- Status = "pass"  --> Compare with previous
        |                           |
        |                           +-- Previous = "pass"  --> No event (stable)
        |                           +-- Previous = "fail"  --> Publish REMEDIATED event
        |
        +-- Status = "fail"  --> Compare with previous
                                    |
                                    +-- Previous = "pass"  --> Publish DRIFT event
                                    +-- Previous = "fail"  --> No event (already drifted)
                                    +-- Previous = null    --> Publish INITIAL FAIL event
```

## Remediation via SaltStack/Ansible

KAI-KEEPER triggers automated remediation using configuration management tools:

```
kubric.grc.drift.v1 (KIC)
        |
        v
   KAI-KEEPER
        |
        +-- Evaluate remediation eligibility
        |       |
        |       +-- Auto-remediate policy: ENABLED for this control?
        |       |       |
        |       |       +-- Yes --> Check remediation playbook exists
        |       |       |              |
        |       |       |              +-- SaltStack target?
        |       |       |              |       --> salt '<host_id>' state.apply <control_state>
        |       |       |              |
        |       |       |              +-- Ansible target?
        |       |       |                      --> ansible-playbook -l <host_id> <control_playbook>.yml
        |       |       |
        |       |       +-- No  --> Create manual remediation ticket
        |       |
        |       +-- Record remediation attempt in audit log
        |
        +-- Re-check compliance after remediation (scheduled T+5min)
                |
                +-- Pass --> Close ticket, publish remediated event
                +-- Fail --> Escalate, create P1 ticket
```

**Remediation tool selection:**

| Criterion | SaltStack | Ansible |
|-----------|-----------|---------|
| Agent presence | Salt minion installed (Linux/Windows) | Agentless (SSH-based) |
| Use case | Real-time state enforcement, continuous drift correction | Ad-hoc remediation, complex multi-step playbooks |
| Kubric default | Primary for managed endpoints | Fallback for agentless environments |

## JetStream Configuration

```json
{
  "stream": {
    "name": "KUBRIC_GRC",
    "subjects": ["kubric.grc.>"],
    "retention": "limits",
    "max_age": "365d",
    "max_bytes": 53687091200,
    "max_msg_size": 16384,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "5m",
    "deny_delete": true,
    "deny_purge": true
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_GRC` | Dedicated stream for GRC (Governance, Risk, Compliance) events. |
| `subjects` | `kubric.grc.>` | Captures all GRC event types. |
| `max_age` | `365d` | Compliance evidence retained for 1 year. Regulatory requirements (SOC 2, PCI-DSS, HIPAA) mandate long-term retention of compliance posture data. |
| `max_bytes` | `50 GB` | Drift events are low-to-moderate volume (hundreds to thousands per check cycle). |
| `deny_delete` | `true` | Compliance audit trail integrity — individual messages cannot be deleted. |
| `deny_purge` | `true` | Compliance audit trail integrity — stream cannot be purged. Only `max_age` expiry removes messages. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-keeper-drift` | Yes | `kubric.grc.drift.v1.>` | `all` | `explicit` | 5 | 120s |
| `compliance-dashboard` | Yes | `kubric.grc.drift.v1.>` | `all` | `explicit` | 3 | 60s |
| `audit-archive-grc` | Yes | `kubric.grc.>` | `all` | `explicit` | 10 | 120s |
| `siem-export-grc` | Yes | `kubric.grc.>` | `all` | `explicit` | 3 | 60s |

- **kai-keeper-drift**: Primary consumer for remediation decision-making. Evaluates each drift event against the remediation policy and triggers SaltStack/Ansible actions.
- **compliance-dashboard**: Feeds the customer-facing compliance posture dashboard showing per-framework pass/fail status and drift trends.
- **audit-archive-grc**: Writes all compliance events to immutable object storage for audit evidence. Aggressive retry to ensure completeness.
- **siem-export-grc**: External SIEM forwarding for aggregated compliance reporting.

## Example (NATS CLI)

### Publish a test drift event

```bash
nats pub kubric.grc.drift.v1.tenant-acme.host-01 \
  --header="Nats-Msg-Id:drift-$(uuidgen)" \
  '{"framework":"CIS_Ubuntu_22.04_L1","control_id":"6.1.3","check_id":"kic-cis-ubuntu-6.1.3","status":"fail","evidence":"expected: permissions 0600, actual: permissions 0644 on /etc/shadow","tenant_id":"tenant-acme","host_id":"host-01","timestamp_ns":1718900000000000000,"previous_status":"pass","control_title":"Ensure permissions on /etc/shadow are configured","remediation_hint":"chmod 0600 /etc/shadow","severity_id":4,"risk_score":78.5}'
```

### Subscribe to all drift events

```bash
nats sub "kubric.grc.drift.>"
```

### Create the GRC stream

```bash
nats stream add KUBRIC_GRC \
  --subjects="kubric.grc.>" \
  --retention=limits \
  --max-age=365d \
  --max-bytes=53687091200 \
  --max-msg-size=16384 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=5m \
  --deny-delete \
  --deny-purge
```

### Create the KAI-KEEPER consumer

```bash
nats consumer add KUBRIC_GRC kai-keeper-drift \
  --filter="kubric.grc.drift.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=120s \
  --pull \
  --durable="kai-keeper-drift"
```

## Notes

- **FIM integration**: File integrity events from `kubric.edr.file.v1` (K-MB-SUB-002) feed into KIC's drift detection. When a FIM event indicates a change to a compliance-monitored file (e.g., `/etc/shadow`, `/etc/ssh/sshd_config`), KIC immediately runs the relevant compliance checks rather than waiting for the next scheduled cycle. This provides near-real-time drift detection for file-based controls.
- **Framework coverage**: KIC ships with built-in support for CIS Benchmarks (Level 1 and Level 2 for Ubuntu, RHEL, Windows Server, macOS), DISA STIG (RHEL 8/9, Ubuntu 22.04, Windows Server 2019/2022), PCI-DSS 4.0, HIPAA technical safeguards, and SOC 2 Type II common criteria. Custom frameworks can be defined using the KIC policy language.
- **Check scheduling**: Critical controls (password policy, SSH config, firewall rules) are evaluated every 15 minutes. Standard controls are evaluated every 1 hour. Full framework scans run daily. Schedules are configurable per tenant.
- **Remediation safety**: Auto-remediation is disabled by default for all controls. Tenants must explicitly opt-in per control or per control category. Remediation actions are always logged with before/after state for audit purposes.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
