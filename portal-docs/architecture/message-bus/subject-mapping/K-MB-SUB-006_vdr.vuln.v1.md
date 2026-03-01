# K-MB-SUB-006 — VDR Vulnerability Finding

> NATS subject mapping reference for vulnerability detection and remediation events in the Kubric UIDR platform.

## Subject Pattern

```
kubric.vdr.vuln.v1
kubric.vdr.vuln.v1.<tenant_id>
kubric.vdr.vuln.v1.<tenant_id>.<host_id>
```

Wildcard subscription: `kubric.vdr.vuln.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **VDR Scanner** | Go | Vulnerability Detection and Response scanner. Performs agent-based package enumeration (dpkg, rpm, apk, Windows Update), container image scanning (Syft + Grype), and Nuclei-based active checks. Enriches findings with EPSS scores and KEV catalog status before publishing. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-KEEPER** | Python (CrewAI) | Vulnerability management AI agent. Evaluates findings through SSVC decision tree, prioritizes remediation, triggers auto-patching for eligible vulnerabilities, and generates exception requests for risk-accepted findings. |

## Payload (OCSF Class)

- **OCSF Class**: Vulnerability Finding (`2002`)
- **OCSF Category**: Findings
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `cve_id` | `string` | `vulnerabilities[].cve.uid` | CVE identifier (e.g., `"CVE-2024-3094"`). |
| 2 | `cvss_score` | `float` | `vulnerabilities[].cve.cvss[].base_score` | CVSS v3.1 base score (0.0 to 10.0). |
| 3 | `severity` | `string` | `severity` | Human-readable severity: `"critical"`, `"high"`, `"medium"`, `"low"`, `"none"`. |
| 4 | `epss_score` | `float` | `unmapped.epss_score` | EPSS probability score (0.0 to 1.0). Represents the probability of exploitation in the next 30 days. |
| 5 | `in_kev` | `bool` | `unmapped.in_kev` | Whether this CVE is listed in CISA's Known Exploited Vulnerabilities catalog. |
| 6 | `package` | `string` | `vulnerabilities[].packages[].name` | Affected package or library name. |
| 7 | `version` | `string` | `vulnerabilities[].packages[].version` | Currently installed version of the affected package. |
| 8 | `fixed_version` | `string` | `vulnerabilities[].packages[].fix_version` | Version that remediates the vulnerability (empty if no fix available). |
| 9 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 10 | `host_id` | `string` | `device.uid` | Host or container image identifier where the vulnerability was found. |
| 11 | `timestamp_ns` | `uint64` | `time` | Scan result timestamp in nanoseconds since Unix epoch. |
| 12 | `scanner` | `string` | `finding_info.analytic.name` | Scanner that produced the finding: `"grype"`, `"nuclei"`, `"osv"`. |
| 13 | `cpe` | `string` | `vulnerabilities[].cve.cpe` | CPE string for the affected product. |
| 14 | `exploit_available` | `bool` | `unmapped.exploit_available` | Whether a public exploit is known to exist (from ExploitDB, Metasploit, nuclei templates). |
| 15 | `severity_id` | `uint32` | `severity_id` | OCSF severity derived from SSVC decision: 5=Critical (Act), 4=High (Attend), 3=Medium (Track*), 2=Low (Track). |

## SSVC Decision Tree Integration

KAI-KEEPER evaluates each vulnerability finding through the CISA SSVC (Stakeholder-Specific Vulnerability Categorization) decision tree:

```
                        Vulnerability Finding
                               |
                    +----------+-----------+
                    |                      |
              Exploitation             Exploitation
              Status: Active           Status: None/PoC
                    |                      |
              +-----+-----+         +-----+-----+
              |           |         |           |
          Automatable  Not Auto  Exposure:   Exposure:
              |           |      Internet     Internal
              |           |         |           |
           ACT         ATTEND    TRACK*       TRACK
        (Critical)     (High)   (Medium)      (Low)
```

**SSVC Input Sources:**
| SSVC Factor | Data Source |
|-------------|------------|
| Exploitation Status | EPSS score + KEV catalog + ExploitDB |
| Automatability | CVSS Attack Complexity + Attack Vector |
| Technical Impact | CVSS Impact subscore |
| Mission Prevalence | Tenant asset criticality tags (from CMDB) |
| Public Well-Being | Industry vertical classification |

**SSVC Decision to Action Mapping:**

| SSVC Decision | Action | SLA | Automation |
|---------------|--------|-----|------------|
| **Act** | Immediate remediation | 24 hours | Auto-patch if eligible, otherwise P1 ticket |
| **Attend** | Scheduled remediation | 7 days | Auto-patch if eligible, otherwise P2 ticket |
| **Track*** | Monitor closely | 30 days | Track for status change, P3 ticket |
| **Track** | Record and monitor | 90 days | Log only, re-evaluate on next scan |

## Auto-Patch Workflow

```
kubric.vdr.vuln.v1 (VDR Scanner)
        |
        v
   KAI-KEEPER
        |
        +-- SSVC Decision: ACT or ATTEND
        |       |
        |       +-- Package update available? (fixed_version != "")
        |       |       |
        |       |       +-- Yes --> Check auto-patch policy
        |       |       |              |
        |       |       |              +-- Approved --> Execute patch via SaltStack
        |       |       |              |                   |
        |       |       |              |                   +-- Publish result to kubric.vdr.vuln.v1 (status=remediated)
        |       |       |              |                   +-- Create ticket (kubric.svc.ticket.v1) documenting patch
        |       |       |              |
        |       |       |              +-- Not approved --> Create P1/P2 ticket for manual action
        |       |       |
        |       |       +-- No  --> Create ticket with workaround guidance
        |       |
        +-- SSVC Decision: TRACK* or TRACK
                |
                +-- Log finding, schedule re-evaluation
```

**Auto-patch eligibility criteria:**
1. Package manager patch available (`fixed_version` is not empty).
2. Host is tagged as `auto-patch: true` in the asset inventory.
3. Package is not in the tenant's patch exclusion list (e.g., kernel updates may require manual approval).
4. Host is not in a maintenance window blackout period.
5. Patch has been available for >= 48 hours (configurable cool-down to avoid zero-day patch regressions).

## JetStream Configuration

```json
{
  "stream": {
    "name": "KUBRIC_VDR",
    "subjects": ["kubric.vdr.>"],
    "retention": "limits",
    "max_age": "90d",
    "max_bytes": 53687091200,
    "max_msg_size": 32768,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "5m",
    "deny_delete": true,
    "deny_purge": false
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_VDR` | Dedicated stream for vulnerability findings. |
| `subjects` | `kubric.vdr.>` | Captures all VDR event types. |
| `max_age` | `90d` | Vulnerability findings retained for 90 days for trend analysis and compliance evidence. |
| `max_bytes` | `50 GB` | Vulnerability events are moderate volume (thousands per scan cycle, not continuous). |
| `duplicate_window` | `5m` | Longer dedup window since scans can produce bursts of findings. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-keeper-vuln` | Yes | `kubric.vdr.vuln.v1.>` | `all` | `explicit` | 5 | 120s |
| `vuln-dashboard` | Yes | `kubric.vdr.vuln.v1.>` | `all` | `explicit` | 3 | 60s |
| `siem-export-vdr` | Yes | `kubric.vdr.>` | `all` | `explicit` | 3 | 60s |

- **kai-keeper-vuln**: Primary consumer for SSVC evaluation and auto-remediation. 120s ack wait accommodates SSVC computation time (EPSS lookup, KEV check, asset criticality lookup).
- **vuln-dashboard**: Feeds the customer-facing vulnerability dashboard with real-time scan results and remediation status.
- **siem-export-vdr**: External SIEM forwarding for compliance reporting.

## Example (NATS CLI)

### Publish a test vulnerability finding

```bash
nats pub kubric.vdr.vuln.v1.tenant-acme.host-01 \
  --header="Nats-Msg-Id:vuln-$(uuidgen)" \
  '{"cve_id":"CVE-2024-3094","cvss_score":10.0,"severity":"critical","epss_score":0.97,"in_kev":true,"package":"xz-utils","version":"5.6.1","fixed_version":"5.6.1+really5.4.5-0+deb12u1","tenant_id":"tenant-acme","host_id":"host-01","timestamp_ns":1718900000000000000,"scanner":"grype","cpe":"cpe:2.3:a:tukaani:xz:5.6.1:*:*:*:*:*:*:*","exploit_available":true,"severity_id":5}'
```

### Subscribe to all vulnerability findings

```bash
nats sub "kubric.vdr.vuln.>"
```

### Create the VDR stream

```bash
nats stream add KUBRIC_VDR \
  --subjects="kubric.vdr.>" \
  --retention=limits \
  --max-age=90d \
  --max-bytes=53687091200 \
  --max-msg-size=32768 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=5m \
  --deny-delete
```

### Create the KAI-KEEPER consumer

```bash
nats consumer add KUBRIC_VDR kai-keeper-vuln \
  --filter="kubric.vdr.vuln.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=120s \
  --pull \
  --durable="kai-keeper-vuln"
```

## Notes

- **EPSS enrichment**: VDR Scanner fetches EPSS scores from the FIRST.org EPSS API daily and caches them locally. Each vulnerability finding is enriched with the current EPSS score at publish time. EPSS scores change daily; historical scores are preserved in the published event.
- **KEV catalog**: CISA KEV catalog is synced every 6 hours. The `in_kev` boolean is authoritative at publish time. Newly added KEV entries trigger a re-scan of existing findings.
- **Scan deduplication**: The `Nats-Msg-Id` includes the scan run ID and CVE+host combination. If a vulnerability persists across scans, each scan produces a new event (with updated EPSS score). KAI-KEEPER tracks first-seen and last-seen timestamps in its state store.
- **Container scanning**: For container images, `host_id` is set to the image digest (`sha256:...`). KAI-KEEPER correlates container vulnerabilities with running deployments via the Kubernetes API to determine actual exposure.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
