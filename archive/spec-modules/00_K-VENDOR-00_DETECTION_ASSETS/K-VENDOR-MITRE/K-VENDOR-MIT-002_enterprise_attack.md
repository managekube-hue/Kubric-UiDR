# K-VENDOR-MIT-002 -- MITRE ATT&CK Enterprise STIX Bundle

## Overview

The ATT&CK Enterprise bundle is a STIX 2.1 JSON file containing all tactics, techniques, sub-techniques, mitigations, groups, and software for the Enterprise matrix. It covers Windows, Linux, macOS, cloud (AWS/Azure/GCP), SaaS, containers, and network infrastructure.

## Data Structure

```json
{
  "type": "bundle",
  "id": "bundle--enterprise-attack",
  "objects": [
    {
      "type": "attack-pattern",
      "id": "attack-pattern--a1234",
      "name": "Phishing",
      "external_references": [
        { "source_name": "mitre-attack", "external_id": "T1566" }
      ],
      "kill_chain_phases": [
        { "kill_chain_name": "mitre-attack", "phase_name": "initial-access" }
      ]
    }
  ]
}
```

## STIX Object Types Consumed

| STIX Type | ATT&CK Concept | Count (approx) |
|---|---|---|
| `attack-pattern` | Techniques & sub-techniques | ~800 |
| `intrusion-set` | Threat actor groups | ~140 |
| `malware` | Malware families | ~600 |
| `tool` | Attacker tools | ~80 |
| `course-of-action` | Mitigations | ~280 |
| `relationship` | Technique-to-group, tool-to-technique links | ~15,000 |

## Kubric Integration

VDR parses the STIX bundle at startup and builds an in-memory graph of technique relationships. This graph powers:

1. **CVE-to-technique mapping** -- When a CVE is enriched with EPSS/NVD data, VDR looks up associated ATT&CK techniques via CWE bridging (see MIT-003).
2. **Attack chain reconstruction** -- KAI-Analyst traverses tactic phases (Initial Access -> Execution -> Persistence -> ...) to build timeline narratives in investigation reports.
3. **Coverage gap analysis** -- The NOC dashboard compares detected techniques against the full ATT&CK matrix to identify blind spots in detection coverage.
4. **Hunt hypothesis generation** -- KAI-Hunter selects under-detected techniques as starting points for proactive threat hunts.

## ATT&CK Tactic Order

| ID | Tactic | Common Kubric Detections |
|---|---|---|
| TA0001 | Initial Access | Suricata web rules, Nuclei CVE scans |
| TA0002 | Execution | Sigma process creation rules |
| TA0003 | Persistence | CoreSec FIM, Wazuh SCA rules |
| TA0011 | Command and Control | Suricata C2 rules, NetGuard TLS/JA3 |
| TA0010 | Exfiltration | Suricata data rules, YARA PII rules |
