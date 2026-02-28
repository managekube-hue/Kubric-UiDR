# K-SOC-DET-006 -- Custom Detection Rule Authoring

**Purpose:** Author, test, and deploy custom detection rules across the Kubric platform using Sigma, Suricata, and YARA formats.

---

## 1. Supported Rule Formats

| Format | Domain | Engine | Location |
|--------|--------|--------|----------|
| Sigma YAML | Log-based detections | sigma-rust (CoreSec) | `vendor/sigma/rules/custom/` |
| Suricata | Network detections | NetGuard rule matcher | `vendor/suricata/custom/` |
| YARA | File/malware detections | YARA-X (CoreSec) | `vendor/yara-rules/custom/` |

---

## 2. Sigma Custom Rule Template

```yaml
# vendor/sigma/rules/custom/kubric_lateral_movement_psexec.yml
title: Kubric - PsExec Lateral Movement Detection
id: a0b1c2d3-e4f5-6789-abcd-ef0123456789
status: stable
level: high
description: >
  Detects PsExec-style lateral movement commonly used by attackers
  after initial compromise. Maps to MITRE T1570 (Lateral Tool Transfer).
author: Kubric SOC Team
date: 2025/11/15
modified: 2026/01/20
references:
  - https://attack.mitre.org/techniques/T1570/
  - https://attack.mitre.org/techniques/T1021/002/

logsource:
  category: process_creation
  product: windows

detection:
  selection_parent:
    ParentImage|endswith:
      - '\services.exe'
  selection_child:
    Image|endswith:
      - '\PSEXESVC.exe'
      - '\cmd.exe'
      - '\powershell.exe'
    User|contains:
      - 'SYSTEM'
  selection_pipe:
    PipeName|startswith:
      - '\PSEXESVC'
  condition: selection_parent and (selection_child or selection_pipe)

falsepositives:
  - Legitimate remote administration by IT
  - PsExec in approved change management ticket

tags:
  - attack.lateral_movement
  - attack.t1570
  - attack.t1021.002
  - kubric.custom
```

### Additional Sigma Examples

```yaml
# vendor/sigma/rules/custom/kubric_credential_dumping_lsass.yml
title: Kubric - LSASS Memory Access for Credential Dumping
id: b1c2d3e4-f567-8901-abcd-ef2345678901
status: stable
level: critical
description: Detects suspicious access to LSASS process memory (T1003.001)
author: Kubric SOC Team
date: 2026/01/10

logsource:
  category: process_access
  product: windows

detection:
  selection:
    TargetImage|endswith: '\lsass.exe'
    GrantedAccess|contains:
      - '0x1010'   # PROCESS_VM_READ | PROCESS_QUERY_INFORMATION
      - '0x1038'   # PROCESS_VM_READ | PROCESS_VM_WRITE | PROCESS_VM_OPERATION
      - '0x1FFFFF' # PROCESS_ALL_ACCESS
  filter_system:
    SourceImage|endswith:
      - '\svchost.exe'
      - '\lsm.exe'
      - '\csrss.exe'
      - '\wininit.exe'
  condition: selection and not filter_system

tags:
  - attack.credential_access
  - attack.t1003.001
  - kubric.critical
```

---

## 3. Suricata Custom Rule Template

```
# vendor/suricata/custom/kubric_c2_beacon.rules

# Detect beacon-like periodic DNS lookups (potential C2)
alert dns $HOME_NET any -> any any (
  msg:"KUBRIC C2 - Periodic DNS beacon pattern detected";
  dns.query;
  content:".xyz"; nocase; endswith;
  threshold: type both, track by_src, count 10, seconds 60;
  classtype:trojan-activity;
  sid:9000001; rev:1;
  metadata: kubric_custom, mitre_t1071_004;
)

# Detect data exfiltration via DNS TXT queries
alert dns $HOME_NET any -> any any (
  msg:"KUBRIC EXFIL - Suspicious DNS TXT query with long subdomain";
  dns.query;
  content:"|00 10|"; # TXT record type
  pcre:"/^[a-z0-9]{32,}\./i";
  threshold: type threshold, track by_src, count 5, seconds 300;
  classtype:data-leak;
  sid:9000002; rev:1;
  metadata: kubric_custom, mitre_t1048_003;
)

# Detect unauthorized outbound SSH
alert tcp $HOME_NET any -> $EXTERNAL_NET 22 (
  msg:"KUBRIC POLICY - Unauthorized outbound SSH connection";
  flow:to_server,established;
  content:"SSH-"; depth:4;
  classtype:policy-violation;
  sid:9000003; rev:1;
  metadata: kubric_custom, mitre_t1021_004;
)

# Detect Cobalt Strike default beacon
alert http $HOME_NET any -> $EXTERNAL_NET any (
  msg:"KUBRIC C2 - Cobalt Strike HTTP beacon default profile";
  flow:to_server,established;
  http.uri; content:"/submit.php?id="; startswith;
  http.header; content:"Cookie:";
  pcre:"/Cookie:\s*[A-Za-z0-9+\/=]{60,}/";
  classtype:trojan-activity;
  sid:9000004; rev:1;
  metadata: kubric_custom, mitre_t1071_001;
)
```

---

## 4. YARA Custom Rule Template

```yara
// vendor/yara-rules/custom/kubric_webshell.yar

rule Kubric_Webshell_Generic : webshell {
    meta:
        author = "Kubric SOC"
        description = "Detects generic PHP/ASP webshell patterns"
        severity = "critical"
        mitre = "T1505.003"
        date = "2026-01-15"

    strings:
        $php_eval = /eval\s*\(\s*\$_(GET|POST|REQUEST|COOKIE)\[/ ascii nocase
        $php_system = /system\s*\(\s*\$_(GET|POST|REQUEST)/ ascii nocase
        $php_passthru = /passthru\s*\(\s*\$_(GET|POST|REQUEST)/ ascii nocase
        $php_base64 = /eval\s*\(\s*base64_decode\s*\(/ ascii nocase
        $asp_execute = /Execute\s*\(\s*Request\s*\(/ ascii nocase
        $asp_eval_request = /eval\s*request\s*\(/ ascii nocase

        $common1 = "FilesMan" ascii wide
        $common2 = "WSO " ascii wide
        $common3 = "c99shell" ascii wide
        $common4 = "r57shell" ascii wide
        $common5 = "b374k" ascii wide

    condition:
        filesize < 500KB and (
            any of ($php_*) or
            any of ($asp_*) or
            2 of ($common*)
        )
}

rule Kubric_Ransomware_Note : ransomware {
    meta:
        author = "Kubric SOC"
        description = "Detects common ransomware note filenames and content"
        severity = "critical"
        mitre = "T1486"
        date = "2026-01-15"

    strings:
        $note1 = "YOUR FILES HAVE BEEN ENCRYPTED" ascii wide nocase
        $note2 = "bitcoin" ascii wide nocase
        $note3 = "decrypt" ascii wide nocase
        $note4 = ".onion" ascii wide
        $ext1 = "README_TO_DECRYPT" ascii wide
        $ext2 = "HOW_TO_RECOVER" ascii wide
        $ext3 = "DECRYPT_INSTRUCTIONS" ascii wide

    condition:
        filesize < 100KB and (
            ($note1 and $note2) or
            ($note1 and $note4) or
            any of ($ext*)
        )
}
```

---

## 5. Rule Testing Against Mordor/OTRF Datasets

```bash
#!/usr/bin/env bash
# scripts/test-custom-rules.sh
# Test custom detection rules against known-good datasets.
set -euo pipefail

MORDOR_DIR="tests/datasets/mordor"
SIGMA_CUSTOM="vendor/sigma/rules/custom/"
YARA_CUSTOM="vendor/yara-rules/custom/"
RESULTS_DIR="tests/results/$(date +%F)"

mkdir -p "$RESULTS_DIR"

# --- Download Mordor datasets (if not present) ---
if [ ! -d "$MORDOR_DIR" ]; then
    echo "[+] Downloading Mordor test datasets..."
    mkdir -p "$MORDOR_DIR"
    # APT3 simulation dataset
    curl -sSL -o "$MORDOR_DIR/apt3_host.json.gz" \
        "https://raw.githubusercontent.com/OTRF/Security-Datasets/master/datasets/compound/apt3/host/apt3_host.json.gz"
    # Credential dumping dataset
    curl -sSL -o "$MORDOR_DIR/credential_access_lsass.json.gz" \
        "https://raw.githubusercontent.com/OTRF/Security-Datasets/master/datasets/atomic/windows/credential_access/host/psh_lsass_memory_access.json.gz"
    gunzip "$MORDOR_DIR"/*.gz 2>/dev/null || true
fi

# --- Test Sigma rules ---
echo "[+] Testing Sigma custom rules against Mordor..."
for rule in "$SIGMA_CUSTOM"*.yml; do
    rulename=$(basename "$rule" .yml)
    echo "  Testing: $rulename"
    # Use sigma-rust CLI or sigmac for validation
    sigma-rust validate "$rule" 2>&1 | tee "$RESULTS_DIR/sigma_${rulename}.log"

    # Test against Mordor JSON events
    for dataset in "$MORDOR_DIR"/*.json; do
        dsname=$(basename "$dataset" .json)
        sigma-rust eval \
            --rule "$rule" \
            --input "$dataset" \
            --output "$RESULTS_DIR/sigma_${rulename}_${dsname}_matches.json" \
            2>&1 || true
    done
done

# --- Test YARA rules ---
echo "[+] Testing YARA custom rules compilation..."
for rule in "$YARA_CUSTOM"*.yar; do
    rulename=$(basename "$rule" .yar)
    echo "  Validating: $rulename"
    yara-x check "$rule" 2>&1 | tee "$RESULTS_DIR/yara_${rulename}_validate.log"
done

# --- Summary ---
echo ""
echo "=== Test Results ==="
echo "Sigma matches:"
find "$RESULTS_DIR" -name 'sigma_*_matches.json' -exec sh -c \
    'echo "  $(basename {}): $(jq length {} 2>/dev/null || echo 0) matches"' \;
echo ""
echo "Results saved to: $RESULTS_DIR"
```

---

## 6. Rule Deployment Pipeline

```
┌─────────────┐    ┌──────────────┐    ┌───────────────┐    ┌─────────────┐
│  Author      │───►│  Test         │───►│  Vendor       │───►│  Deploy     │
│  Rule YAML   │    │  Against      │    │  Git commit   │    │  NATS push  │
│              │    │  Mordor/OTRF  │    │  vendor/...   │    │  hot-reload │
└─────────────┘    └──────────────┘    └───────────────┘    └─────────────┘
```

### Git Workflow

```bash
# 1. Create rule branch
git checkout -b detection/custom-c2-beacon

# 2. Author rule in appropriate vendor directory
vim vendor/suricata/custom/kubric_c2_beacon.rules

# 3. Test
./scripts/test-custom-rules.sh

# 4. Commit and push
git add vendor/suricata/custom/kubric_c2_beacon.rules
git commit -m "detection: add C2 beacon DNS pattern rule (T1071.004)"
git push origin detection/custom-c2-beacon

# 5. After merge, CI deploys via NATS
nats pub kubric.config.rules.reload \
    '{"type":"suricata","scope":"custom","ts":"2026-01-20T12:00:00Z"}'
```

---

## 7. Detection Coverage Measurement (MITRE ATT&CK)

```python
#!/usr/bin/env python3
# scripts/measure_attack_coverage.py
"""Measure custom rule coverage against MITRE ATT&CK Enterprise matrix."""

import os
import re
import yaml
import json
from pathlib import Path
from collections import defaultdict

SIGMA_DIR = "vendor/sigma/rules/custom"
SURICATA_DIR = "vendor/suricata/custom"
YARA_DIR = "vendor/yara-rules/custom"

ATTACK_TACTICS = [
    "reconnaissance", "resource_development", "initial_access",
    "execution", "persistence", "privilege_escalation",
    "defense_evasion", "credential_access", "discovery",
    "lateral_movement", "collection", "command_and_control",
    "exfiltration", "impact",
]

def extract_sigma_techniques(rule_dir: str) -> dict[str, list[str]]:
    """Extract MITRE technique IDs from Sigma rule tags."""
    coverage = defaultdict(list)
    for f in Path(rule_dir).glob("*.yml"):
        with open(f) as fh:
            rule = yaml.safe_load(fh)
        tags = rule.get("tags", [])
        title = rule.get("title", f.stem)
        for tag in tags:
            match = re.match(r"attack\.t(\d{4})(?:\.(\d{3}))?", tag)
            if match:
                tid = f"T{match.group(1)}"
                if match.group(2):
                    tid += f".{match.group(2)}"
                coverage[tid].append(title)
    return dict(coverage)

def extract_suricata_techniques(rule_dir: str) -> dict[str, list[str]]:
    """Extract MITRE IDs from Suricata rule metadata."""
    coverage = defaultdict(list)
    for f in Path(rule_dir).glob("*.rules"):
        for line in open(f):
            msg_match = re.search(r'msg:"([^"]+)"', line)
            meta_match = re.search(r"mitre_(t\d{4}(?:_\d{3})?)", line)
            if meta_match:
                tid = meta_match.group(1).upper().replace("_", ".")
                msg = msg_match.group(1) if msg_match else f.stem
                coverage[tid].append(msg)
    return dict(coverage)

def extract_yara_techniques(rule_dir: str) -> dict[str, list[str]]:
    """Extract MITRE IDs from YARA rule metadata."""
    coverage = defaultdict(list)
    for f in Path(rule_dir).glob("*.yar"):
        content = open(f).read()
        for match in re.finditer(r'mitre\s*=\s*"(T\d{4}(?:\.\d{3})?)"', content):
            tid = match.group(1)
            rule_match = re.search(r"rule\s+(\w+)", content)
            name = rule_match.group(1) if rule_match else f.stem
            coverage[tid].append(name)
    return dict(coverage)

if __name__ == "__main__":
    all_coverage = defaultdict(list)

    for tid, rules in extract_sigma_techniques(SIGMA_DIR).items():
        all_coverage[tid].extend([f"[Sigma] {r}" for r in rules])
    for tid, rules in extract_suricata_techniques(SURICATA_DIR).items():
        all_coverage[tid].extend([f"[Suricata] {r}" for r in rules])
    for tid, rules in extract_yara_techniques(YARA_DIR).items():
        all_coverage[tid].extend([f"[YARA] {r}" for r in rules])

    total_techniques = len(all_coverage)
    print(f"Total MITRE techniques covered: {total_techniques}")
    print(f"\nCoverage by technique:")
    for tid in sorted(all_coverage):
        rules = all_coverage[tid]
        print(f"  {tid}: {len(rules)} rules")
        for r in rules:
            print(f"    - {r}")

    # Export for ATT&CK Navigator
    navigator_layer = {
        "name": "Kubric Custom Detection Coverage",
        "versions": {"attack": "14", "navigator": "4.9.1", "layer": "4.5"},
        "domain": "enterprise-attack",
        "techniques": [
            {"techniqueID": tid, "score": len(rules), "comment": "; ".join(rules)}
            for tid, rules in all_coverage.items()
        ],
    }
    with open("tests/results/attack_navigator_layer.json", "w") as f:
        json.dump(navigator_layer, f, indent=2)
    print(f"\nATT&CK Navigator layer saved to tests/results/attack_navigator_layer.json")
```
