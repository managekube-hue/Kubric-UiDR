# K-VENDOR-SIG-005 -- Hunting Sigma Rules

| Field | Value |
|-------|-------|
| **Rule Path** | `vendor/sigma/rules/*/hunting/` (across all platform dirs) |
| **Purpose** | Proactive threat hunting and anomaly detection |
| **Sigma Status** | `experimental` or `test` -- not promoted to alerting by default |

## Scope

Hunting rules are lower-confidence, higher-volume detections designed for
analyst-driven investigation rather than automated alerting. CoreSec's
`SigmaEngine` loads them alongside production rules but their `level` is
typically `informational` or `low`, which the downstream alert routing layer
uses to suppress pager notifications and route matches to the Kai `hunter`
agent for triage.

## How Hunting Rules Differ from Detection Rules

| Aspect | Detection Rules (SIG-002/003/004) | Hunting Rules (SIG-005) |
|--------|-----------------------------------|-------------------------|
| Confidence | Medium-Critical | Low-Informational |
| False-positive rate | Tuned for low FP | Accepts higher FP |
| Alert routing | SOC queue / pager | Kai hunter agent review |
| Sigma `status` | `stable` / `production` | `experimental` / `test` |
| Purpose | Real-time blocking/alerting | Proactive threat discovery |

## Key Hunting Rule Families

### Living-off-the-Land Discovery

Rules that flag uncommon but not inherently malicious use of built-in OS
utilities (e.g., `nltest`, `dsquery`, `klist`) that frequently appear in
hands-on-keyboard intrusions.

**MITRE ATT&CK**: T1018 (Remote System Discovery), T1069 (Permission Groups
Discovery), T1087 (Account Discovery)

### Rare Process Lineage

Rules matching unusual parent-child process chains (e.g., `outlook.exe` spawning
`cmd.exe`, `svchost.exe` spawning `whoami.exe`) that may indicate code execution
from a compromised application.

**MITRE ATT&CK**: T1204.002 (Malicious File), T1055 (Process Injection)

### Anomalous Authentication Patterns

Cloud and SaaS hunting rules targeting sign-in anomalies such as off-hours
admin logins, first-time use of a service principal, or logins from hosting
provider ASNs.

**MITRE ATT&CK**: T1078 (Valid Accounts), T1078.004 (Cloud Accounts)

### Network Beaconing Indicators

Rules identifying periodic outbound connections with uniform intervals and
payload sizes, a hallmark of C2 beaconing frameworks (Cobalt Strike, Sliver).

**MITRE ATT&CK**: T1071.001 (Web Protocols), T1573 (Encrypted Channel)

### Persistence Enumeration

Rules that detect enumeration of scheduled tasks, startup folders, and
registry Run keys from non-standard parent processes.

**MITRE ATT&CK**: T1053.005 (Scheduled Task), T1547.001 (Registry Run Keys)

## CoreSec Integration

Hunting rules flow through the same `SigmaEngine::evaluate()` pipeline.
The `SigmaMatch.level` field determines routing:

```
SigmaMatch { level: "informational", tags: ["attack.t1018"], ... }
  --> alert router checks level
  --> level in [informational, low] --> route to kai.hunter queue
  --> level in [medium, high, critical] --> route to SOC alert queue
```

## Authoring Custom Hunting Rules

Place new YAML files in the appropriate `hunting/` subdirectory. CoreSec's
file watcher will pick them up automatically. Each rule must include:

- `id`: UUID v4
- `title`: Descriptive name
- `status`: `experimental`
- `level`: `informational` or `low`
- `tags`: At least one `attack.tNNNN` MITRE reference
- `detection`: Standard Sigma detection block with `condition`
