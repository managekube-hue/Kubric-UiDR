# K-VENDOR-ZEK-004 -- Zeek HTTP Detection Scripts

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | HTTP traffic analysis and anomaly detection    |
| Format      | Zeek scripts (.zeek) + http.log analysis       |
| Consumer    | NetGuard agent, KAI-TRIAGE                     |

## Purpose

Custom and community Zeek scripts that detect suspicious HTTP patterns
including web shells, data exfiltration, exploit delivery, and C2
communication over HTTP/HTTPS.

## Key Detection Scripts

| Script                         | Detection Target                    |
|--------------------------------|-------------------------------------|
| detect-webshell.zeek           | Web shell upload/access patterns    |
| long-uri-strings.zeek          | Buffer overflow / SQLi indicators   |
| http-exe-download.zeek         | Executable download over HTTP       |
| suspicious-user-agents.zeek    | Known malware user-agent strings    |
| http-post-exfil.zeek           | Large POST bodies (data exfil)      |
| http-tunneling.zeek            | HTTP tunnel / CONNECT abuse         |

## Detection Outputs

- Matching events are written to Zeek `notice.log` with a
  categorized `Notice::Type` (e.g., `HTTP::Suspicious_Download`).
- NetGuard parses notice entries and maps them to MITRE ATT&CK
  technique IDs before publishing to NATS.

## Integration Flow

1. Zeek loads HTTP detection scripts at startup via `local.zeek`.
2. Scripts evaluate each HTTP transaction in real time.
3. Notices trigger JSON entries consumed by NetGuard.
4. Alerts are published to `kubric.netguard.http.alert` on NATS.
5. KAI-TRIAGE correlates with endpoint data for severity scoring.

## Notes

- Community scripts are BSD-licensed and may be bundled directly.
- Custom Kubric scripts follow the same `notice` output convention.
