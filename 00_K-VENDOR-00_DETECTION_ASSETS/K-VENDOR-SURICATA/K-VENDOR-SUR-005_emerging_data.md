# K-VENDOR-SUR-005 -- Emerging Data Exfiltration Rules

## Category

The `emerging-policy.rules` and related rulesets detect data exfiltration, policy violations, and suspicious outbound transfers. These rules identify large data transfers, use of unauthorized cloud storage, and encoded data leaving the network.

## Detection Patterns

| Pattern | Description |
|---|---|
| **Cloud upload** | POST requests to Mega, Dropbox, Google Drive, Pastebin |
| **DNS exfil** | Abnormally long DNS queries (>50 chars), high-entropy subdomains |
| **Encoded transfers** | Base64-encoded payloads in HTTP headers or query strings |
| **ICMP tunneling** | Oversized ICMP echo payloads carrying embedded data |
| **FTP/SFTP bulk** | Large outbound file transfers to untrusted destinations |
| **Email exfil** | SMTP with large attachments to free-mail providers |

## Rule Structure

```
alert dns $HOME_NET any -> any any (
    msg:"ET POLICY DNS Query to Pastebin";
    dns.query; content:"pastebin.com"; nocase;
    sid:2032000; rev:1;
    classtype:policy-violation;
    metadata:attack_target Cloud, deployment Perimeter;
)
```

## Kubric Integration

NetGuard monitors outbound flows for exfiltration indicators. When a data-loss rule matches:

1. Alert published to `kubric.{tenant}.detection.network_ids.v1`
2. KAI-Risk assesses financial impact using FAIR loss magnitude estimation
3. KAI-Analyst reviews the flow for volume, destination reputation, and timing
4. NOC dashboard flags the event under the "Data Loss" category

## MITRE ATT&CK Mapping

- T1048 -- Exfiltration Over Alternative Protocol
- T1567 -- Exfiltration to Cloud Storage
- T1041 -- Exfiltration Over C2 Channel
- T1052 -- Exfiltration Over Physical Medium (USB policy rules)

## Cross-Reference

These rules complement YARA PII rules (YAR-003) which scan file content for sensitive data patterns like credit card numbers and SSNs. Together they provide detection at both the content and transport layers.
