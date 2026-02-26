# K-VENDOR-YAR-003 -- PII & Sensitive Data Rules

## Category

PII (Personally Identifiable Information) YARA rules detect sensitive data patterns in files and network streams. These rules support data-loss prevention (DLP) and compliance requirements (PCI DSS, HIPAA, GDPR).

## Rule Structure

```yara
rule PII_CreditCard_Visa {
    meta:
        author      = "Kubric DLP"
        description = "Visa credit card number pattern"
        compliance  = "PCI-DSS-3.4"
        severity    = "HIGH"
    strings:
        $visa = /4[0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}/
    condition:
        $visa
}
```

## Detection Patterns

| Data Type | Pattern | Compliance |
|---|---|---|
| **Credit card numbers** | Visa, MC, Amex, Discover (Luhn-valid) | PCI DSS 3.4 |
| **Social Security Numbers** | `XXX-XX-XXXX` formats | HIPAA, SOX |
| **Email addresses** | Bulk email harvesting indicators | GDPR Art. 4 |
| **API keys / secrets** | AWS `AKIA*`, GitHub `ghp_*`, Stripe `sk_live_*` | SOC 2 CC6.1 |
| **Medical record numbers** | MRN patterns per HIPAA identifiers | HIPAA 164.514 |
| **Passport / ID numbers** | Country-specific formats (US, UK, EU) | GDPR Art. 9 |

## Kubric Integration

PII rules serve two scan paths:

1. **Network DLP** -- NetGuard scans outbound HTTP/SMTP payloads. A match triggers a `PolicyViolation` alert on `kubric.{tenant}.detection.dlp.v1`, which the NOC dashboard flags under "Data Loss Prevention."

2. **Endpoint DLP** -- CoreSec scans new files in sensitive directories. FIM change events that trigger YARA PII matches are escalated with `severity: HIGH` and forwarded to KAI-Risk for FAIR-based loss estimation.

## Cross-Reference

- Suricata data exfiltration rules (SUR-005) detect the transport layer
- YARA PII rules detect the content layer
- OSCAL PCI DSS controls (OSC-003) map DLP requirements to control families
- Together they provide defense-in-depth for sensitive data protection

## False Positive Handling

PII rules can be noisy. Kubric applies tenant-configurable allowlists for known-safe destinations (e.g., payment processor IPs) and exempted file paths. Allowlists are managed via the K-SVC tenant configuration API.
