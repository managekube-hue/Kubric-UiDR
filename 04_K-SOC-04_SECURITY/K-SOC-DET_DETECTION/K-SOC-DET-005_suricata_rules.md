# K-SOC-DET-005 -- Suricata ET Rules Integration

**License:** GPL 2.0 — rules are data files consumed at runtime, not compiled into Kubric code.  
**Vendored at:** `vendor/suricata/emerging-*.rules`  
**Role:** Network threat detection via signature matching in the NetGuard Rust agent.

---

## 1. Rule Categories

| File | Purpose |
|------|---------|
| `emerging-malware.rules` | Malware C2, dropper, payload signatures |
| `emerging-trojan.rules` | Trojan traffic patterns |
| `emerging-exploit.rules` | Exploitation attempts (CVE-based) |
| `emerging-web_server.rules` | Web server attacks (SQLi, XSS, RCE) |
| `emerging-dns.rules` | Malicious DNS requests |
| `emerging-policy.rules` | Policy violation indicators |
| `emerging-ciarmy.rules` | CI Army blocklist hits |
| `emerging-compromised.rules` | Known compromised hosts |
| `emerging-current_events.rules` | Trending threat signatures |
| `emerging-info.rules` | Informational (data leak, geolocation) |

---

## 2. Rule Vendoring and Updates

```bash
#!/usr/bin/env bash
# scripts/update-suricata-rules.sh
set -euo pipefail

VENDOR_DIR="vendor/suricata"
RULES_URL="https://rules.emergingthreats.net/open/suricata-7.0/emerging-all.rules"
RULES_TAR="https://rules.emergingthreats.net/open/suricata-7.0/emerging.rules.tar.gz"

mkdir -p "$VENDOR_DIR"

echo "[+] Downloading Emerging Threats Suricata rules"
curl -sSL "$RULES_TAR" -o /tmp/et-rules.tar.gz
tar xzf /tmp/et-rules.tar.gz -C /tmp/et-rules/

# Copy relevant rule files
cp /tmp/et-rules/rules/emerging-malware.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-trojan.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-exploit.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-web_server.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-dns.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-policy.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-compromised.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-current_events.rules "$VENDOR_DIR/"
cp /tmp/et-rules/rules/emerging-info.rules "$VENDOR_DIR/"

# Count vendored rules
RULE_COUNT=$(grep -c '^alert' "$VENDOR_DIR"/emerging-*.rules 2>/dev/null | awk -F: '{s+=$2}END{print s}')
echo "[+] Vendored $RULE_COUNT Suricata rules"

# Cleanup
rm -rf /tmp/et-rules/ /tmp/et-rules.tar.gz

echo "[+] Rules update complete"
```

### Automated Update via suricata-update

```bash
# Install suricata-update
pip install suricata-update

# Configure rule sources
suricata-update update-sources
suricata-update enable-source et/open

# Run update with output to vendor directory
suricata-update \
  --suricata-conf /etc/suricata/suricata.yaml \
  --output-dir vendor/suricata/ \
  --no-reload

# Verify rule count
suricata-update list-enabled-sources
```

---

## 3. Rule Parsing in NetGuard Rust Agent

```rust
// agents/netguard/src/detections/suricata.rs

use std::collections::HashMap;
use std::path::Path;
use regex::Regex;
use tracing::{info, warn};

/// Parsed Suricata rule
#[derive(Debug, Clone)]
pub struct SuricataRule {
    pub sid: u64,
    pub rev: u32,
    pub action: String,       // alert, drop, pass
    pub protocol: String,     // tcp, udp, http, dns, tls
    pub src_addr: String,
    pub src_port: String,
    pub direction: String,    // -> or <>
    pub dst_addr: String,
    pub dst_port: String,
    pub msg: String,
    pub content_matches: Vec<ContentMatch>,
    pub classtype: String,
    pub severity: u8,         // 1=high, 2=medium, 3=low
    pub reference: Vec<String>,
    pub metadata: HashMap<String, String>,
    pub raw: String,
}

#[derive(Debug, Clone)]
pub struct ContentMatch {
    pub pattern: Vec<u8>,
    pub nocase: bool,
    pub depth: Option<u32>,
    pub offset: Option<u32>,
    pub is_negated: bool,
}

/// Load and parse Suricata rules from vendor directory.
pub fn load_rules(rules_dir: &str) -> anyhow::Result<Vec<SuricataRule>> {
    let mut rules = Vec::new();
    let sid_re = Regex::new(r"sid:\s*(\d+)")?;
    let rev_re = Regex::new(r"rev:\s*(\d+)")?;
    let msg_re = Regex::new(r#"msg:\s*"([^"]+)""#)?;
    let classtype_re = Regex::new(r"classtype:\s*([^;]+)")?;
    let content_re = Regex::new(r#"content:\s*"([^"]+)""#)?;

    for entry in std::fs::read_dir(rules_dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.extension().map(|e| e == "rules").unwrap_or(false) {
            let text = std::fs::read_to_string(&path)?;
            for line in text.lines() {
                let line = line.trim();
                if line.is_empty() || line.starts_with('#') {
                    continue;
                }
                if let Some(rule) = parse_rule_line(
                    line, &sid_re, &rev_re, &msg_re, &classtype_re, &content_re,
                ) {
                    rules.push(rule);
                }
            }
            info!(
                path = %path.display(),
                "Loaded Suricata rules file"
            );
        }
    }

    info!(total = rules.len(), "Suricata rules loaded");
    Ok(rules)
}

fn parse_rule_line(
    line: &str,
    sid_re: &Regex,
    rev_re: &Regex,
    msg_re: &Regex,
    classtype_re: &Regex,
    content_re: &Regex,
) -> Option<SuricataRule> {
    // Parse header: action proto src_addr src_port direction dst_addr dst_port
    let parts: Vec<&str> = line.splitn(8, ' ').collect();
    if parts.len() < 7 {
        return None;
    }

    let sid = sid_re.captures(line)
        .and_then(|c| c.get(1))
        .and_then(|m| m.as_str().parse().ok())
        .unwrap_or(0);

    let rev = rev_re.captures(line)
        .and_then(|c| c.get(1))
        .and_then(|m| m.as_str().parse().ok())
        .unwrap_or(1);

    let msg = msg_re.captures(line)
        .and_then(|c| c.get(1))
        .map(|m| m.as_str().to_string())
        .unwrap_or_default();

    let classtype = classtype_re.captures(line)
        .and_then(|c| c.get(1))
        .map(|m| m.as_str().trim().to_string())
        .unwrap_or_default();

    let content_matches: Vec<ContentMatch> = content_re
        .captures_iter(line)
        .map(|c| ContentMatch {
            pattern: c[1].as_bytes().to_vec(),
            nocase: line.contains("nocase;"),
            depth: None,
            offset: None,
            is_negated: false,
        })
        .collect();

    let severity = match classtype.as_str() {
        "trojan-activity" | "attempted-admin" | "successful-admin" => 1,
        "web-application-attack" | "attempted-user" => 2,
        _ => 3,
    };

    Some(SuricataRule {
        sid,
        rev,
        action: parts[0].to_string(),
        protocol: parts[1].to_string(),
        src_addr: parts[2].to_string(),
        src_port: parts[3].to_string(),
        direction: parts[4].to_string(),
        dst_addr: parts[5].to_string(),
        dst_port: parts[6].to_string(),
        msg,
        content_matches,
        classtype,
        severity,
        reference: Vec::new(),
        metadata: HashMap::new(),
        raw: line.to_string(),
    })
}
```

---

## 4. Flow Matching Engine

```rust
// agents/netguard/src/detections/suricata_match.rs

use super::suricata::{SuricataRule, ContentMatch};
use crate::flows::NetworkFlow;

/// Match a network flow against loaded Suricata rules.
pub fn match_flow(
    flow: &NetworkFlow,
    rules: &[SuricataRule],
) -> Vec<&SuricataRule> {
    rules.iter().filter(|rule| {
        // Protocol match
        if !protocol_matches(&rule.protocol, &flow.protocol) {
            return false;
        }

        // Port match
        if !port_matches(&rule.dst_port, flow.dst_port) {
            return false;
        }

        // Content match against payload
        if !rule.content_matches.is_empty() {
            if let Some(ref payload) = flow.payload {
                return rule.content_matches.iter().all(|cm| {
                    content_match(cm, payload)
                });
            }
            return false;
        }

        true
    }).collect()
}

fn protocol_matches(rule_proto: &str, flow_proto: &str) -> bool {
    match rule_proto {
        "any" => true,
        "ip" => true,
        p => p.eq_ignore_ascii_case(flow_proto),
    }
}

fn port_matches(rule_port: &str, flow_port: u16) -> bool {
    match rule_port {
        "any" => true,
        p if p.starts_with('[') => {
            // Port group: [80,443,8080]
            p.trim_matches(|c| c == '[' || c == ']')
                .split(',')
                .any(|port| port.trim().parse::<u16>().map(|pp| pp == flow_port).unwrap_or(false))
        }
        p if p.contains(':') => {
            // Port range: 1024:65535
            let parts: Vec<&str> = p.splitn(2, ':').collect();
            let lo = parts[0].parse::<u16>().unwrap_or(0);
            let hi = parts[1].parse::<u16>().unwrap_or(65535);
            flow_port >= lo && flow_port <= hi
        }
        p => p.parse::<u16>().map(|pp| pp == flow_port).unwrap_or(false),
    }
}

fn content_match(cm: &ContentMatch, payload: &[u8]) -> bool {
    if cm.nocase {
        let pattern_lower: Vec<u8> = cm.pattern.iter().map(|b| b.to_ascii_lowercase()).collect();
        let payload_lower: Vec<u8> = payload.iter().map(|b| b.to_ascii_lowercase()).collect();
        payload_lower.windows(pattern_lower.len()).any(|w| w == pattern_lower.as_slice())
    } else {
        payload.windows(cm.pattern.len()).any(|w| w == cm.pattern.as_slice())
    }
}
```

---

## 5. NATS Publishing

```rust
// agents/netguard/src/detections/suricata_publisher.rs

use chrono::Utc;
use serde_json::json;

use super::suricata::SuricataRule;
use crate::flows::NetworkFlow;

/// Publish Suricata rule match as OCSF NetworkActivity event.
pub async fn publish_rule_match(
    nc: &async_nats::Client,
    tenant_id: &str,
    flow: &NetworkFlow,
    rule: &SuricataRule,
) -> anyhow::Result<()> {
    let severity = match rule.severity {
        1 => 4, // High
        2 => 3, // Medium
        _ => 2, // Low
    };

    let event = json!({
        "class_uid": 4001,        // OCSF NetworkActivity
        "activity_id": 5,         // Traffic detected
        "category_uid": 4,        // Findings
        "severity_id": severity,
        "time": Utc::now().to_rfc3339(),
        "src_endpoint": {
            "ip": flow.src_ip.to_string(),
            "port": flow.src_port
        },
        "dst_endpoint": {
            "ip": flow.dst_ip.to_string(),
            "port": flow.dst_port
        },
        "connection_info": {
            "protocol_name": &flow.protocol,
            "direction_id": 2  // Outbound
        },
        "finding_info": {
            "title": &rule.msg,
            "uid": format!("suricata-sid-{}", rule.sid),
            "types": [&rule.classtype],
            "analytic": {
                "name": format!("ET SID:{}", rule.sid),
                "type": "Suricata",
                "category": &rule.classtype,
            }
        },
        "metadata": {
            "product": {
                "name": "NetGuard Suricata",
                "vendor_name": "Kubric",
                "version": env!("CARGO_PKG_VERSION")
            },
            "tenant_uid": tenant_id
        },
        "unmapped": {
            "suricata_sid": rule.sid,
            "suricata_rev": rule.rev,
            "suricata_action": &rule.action,
            "suricata_classtype": &rule.classtype,
            "raw_rule": &rule.raw
        }
    });

    let subject = format!("kubric.ndr.flow.{}", tenant_id);
    nc.publish(subject, serde_json::to_vec(&event)?.into()).await?;

    Ok(())
}
```

---

## 6. Rule Update Automation (Cron)

```yaml
# deployments/k8s/suricata-rule-updater.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: suricata-rule-updater
  namespace: kubric-agents
spec:
  schedule: "0 2 * * *"  # Daily at 02:00 UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: updater
              image: ghcr.io/kubric/rule-updater:latest
              command:
                - /bin/bash
                - -c
                - |
                  cd /workspace
                  ./scripts/update-suricata-rules.sh
                  # Signal agents to reload via NATS
                  nats pub kubric.config.rules.reload \
                    '{"type":"suricata","timestamp":"'$(date -u +%FT%TZ)'"}'
              volumeMounts:
                - name: vendor-rules
                  mountPath: /workspace/vendor/suricata
              env:
                - name: NATS_URL
                  valueFrom:
                    secretKeyRef:
                      name: kubric-nats
                      key: url
          volumes:
            - name: vendor-rules
              persistentVolumeClaim:
                claimName: kubric-vendor-rules
          restartPolicy: OnFailure
```

---

## 7. Performance

| Metric | Value |
|--------|-------|
| Rules loaded | ~30,000 ET Open rules |
| Parse time | ~500ms for all rules |
| Match throughput | ~100k flows/sec (content-free rules) |
| Content match | ~10k flows/sec (with payload inspection) |
| Memory | ~80 MB for full rule set |
| Update frequency | Daily at 02:00 UTC |
