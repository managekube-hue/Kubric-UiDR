# K-MB-SUB-005 — ITDR Authentication Activity

> NATS subject mapping reference for identity and authentication telemetry in the Kubric UIDR platform.

## Subject Pattern

```
kubric.itdr.auth.v1
kubric.itdr.auth.v1.<tenant_id>
kubric.itdr.auth.v1.<tenant_id>.<source_id>
```

Wildcard subscription: `kubric.itdr.auth.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **CoreSec ITDR** | Go | Identity Threat Detection and Response module. Collects authentication events from OS-level sources (PAM on Linux, Windows Security Event Log, SSH daemon logs), directory services (Active Directory, LDAP, Authentik), and cloud IdPs (Azure AD, Okta via webhook ingestion). |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-TRIAGE** | Python (CrewAI) | Correlates authentication events with endpoint and network telemetry. Detects brute force patterns, credential stuffing, and pass-the-hash/pass-the-ticket attacks. |
| **KAI Identity** | Python (CrewAI) | Specialized identity analytics agent. Builds user behavior profiles, detects impossible travel, identifies privilege escalation anomalies, and monitors service account misuse. |

## Payload (OCSF Class)

- **OCSF Class**: Authentication (`3002`)
- **OCSF Category**: Identity & Access Management
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `user` | `string` | `actor.user.name` | Username or UPN attempting authentication. |
| 2 | `domain` | `string` | `actor.user.domain` | Authentication domain (AD domain, Authentik realm, or local hostname). |
| 3 | `action` | `uint32` | `activity_id` | OCSF activity: 1=Logon, 2=Logoff, 3=Authentication Ticket, 4=Service Ticket. |
| 4 | `success` | `bool` | `status_id` | Whether the authentication succeeded. Maps to OCSF `status_id`: true=1(Success), false=2(Failure). |
| 5 | `src_ip` | `string` | `src_endpoint.ip` | Source IP address of the authentication attempt. |
| 6 | `auth_protocol` | `string` | `auth_protocol_id` | Authentication protocol: `"kerberos"`, `"ntlm"`, `"ldap"`, `"ssh_pubkey"`, `"ssh_password"`, `"oauth2"`, `"saml"`, `"mfa_totp"`, `"mfa_webauthn"`. |
| 7 | `factors` | `repeated string` | `actor.authorizations[].decision` | List of authentication factors used (e.g., `["password", "totp"]`, `["pubkey"]`, `["password", "webauthn"]`). |
| 8 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 9 | `source_id` | `string` | `device.uid` | Source system identifier (hostname, IdP name, or directory service name). |
| 10 | `timestamp_ns` | `uint64` | `time` | Event timestamp in nanoseconds since Unix epoch. |
| 11 | `session_id` | `string` | `session.uid` | Session identifier (logon ID on Windows, session ID on Linux, token ID for OAuth2). |
| 12 | `target_resource` | `string` | `dst_endpoint.name` | Resource being accessed (hostname for interactive logon, service name for Kerberos tickets, application name for OAuth2). |
| 13 | `failure_reason` | `string` | `status_detail` | Reason for authentication failure (e.g., `"invalid_password"`, `"account_locked"`, `"expired_cert"`, `"mfa_timeout"`). Empty on success. |
| 14 | `geo_city` | `string` | `src_endpoint.location.city` | GeoIP city of the source IP (MaxMind GeoLite2). |
| 15 | `geo_country` | `string` | `src_endpoint.location.country` | GeoIP country code (ISO 3166-1 alpha-2). |
| 16 | `severity_id` | `uint32` | `severity_id` | OCSF severity: successful logins are Informational(1), failures vary based on pattern analysis. |

## JetStream Configuration

```json
{
  "stream": {
    "name": "KUBRIC_ITDR",
    "subjects": ["kubric.itdr.>"],
    "retention": "limits",
    "max_age": "90d",
    "max_bytes": 107374182400,
    "max_msg_size": 16384,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "2m",
    "deny_delete": true,
    "deny_purge": true
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_ITDR` | Dedicated stream for identity/authentication events. |
| `subjects` | `kubric.itdr.>` | Captures all ITDR event types. |
| `max_age` | `90d` | Extended retention for identity events. Compliance frameworks (SOC 2, HIPAA, PCI-DSS) require 90-day audit trails for authentication activity. |
| `max_bytes` | `100 GB` (107374182400) | Authentication events are small (~500 bytes each) but 90-day retention requires larger budget. |
| `max_msg_size` | `16 KB` | Auth events are compact. 16 KB ceiling prevents abuse. |
| `deny_delete` | `true` | Audit trail integrity — individual messages cannot be deleted. |
| `deny_purge` | `true` | Audit trail integrity — stream cannot be purged. Only `max_age` expiry removes messages. |

## Impossible Travel Detection

KAI Identity performs impossible travel analysis on authentication events:

```
Auth Event 1:  user=jsmith, src_ip=203.0.113.10, geo=New York, US    @ T
Auth Event 2:  user=jsmith, src_ip=198.51.100.5,  geo=Moscow, RU     @ T + 30min
                                                                        ^
                                                    Distance: 7,510 km
                                                    Min travel time: ~10 hours
                                                    Actual gap: 30 minutes
                                                    --> IMPOSSIBLE TRAVEL ALERT
```

**Algorithm:**
1. For each user, maintain a sliding window of authentication events with GeoIP data.
2. Compute great-circle distance between consecutive authentication locations.
3. If distance / time_delta exceeds maximum feasible travel speed (configured default: 900 km/h to account for air travel), flag as impossible travel.
4. Exclude known VPN egress IPs and corporate proxy ranges from analysis (tenant-configurable allowlist).
5. Impossible travel events are published as high-severity alerts back to `kubric.itdr.auth.v1` with `severity_id=4` and a metadata annotation indicating the detection type.

## Brute Force Correlation

KAI-TRIAGE detects brute force patterns using sliding window counters:

| Pattern | Threshold | Window | Severity |
|---------|-----------|--------|----------|
| Failed logins (same user) | >= 10 failures | 5 minutes | High (4) |
| Failed logins (same src_ip, different users) | >= 20 failures | 10 minutes | Critical (5) — credential stuffing |
| Failed logins followed by success | >= 5 failures then 1 success | 15 minutes | Critical (5) — compromised account |
| Kerberos ticket anomaly | TGT request without prior AS-REQ | single event | High (4) — pass-the-ticket |
| NTLM downgrade | NTLM auth when Kerberos is expected | single event | Medium (3) — potential relay attack |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-triage-auth` | Yes | `kubric.itdr.auth.v1.>` | `all` | `explicit` | 5 | 30s |
| `kai-identity-auth` | Yes | `kubric.itdr.auth.v1.>` | `all` | `explicit` | 5 | 60s |
| `audit-log-archive` | Yes | `kubric.itdr.>` | `all` | `explicit` | 10 | 120s |
| `siem-export-itdr` | Yes | `kubric.itdr.>` | `all` | `explicit` | 3 | 60s |

- **kai-triage-auth**: Real-time brute force and credential attack detection. Low latency requirement — ack wait is 30s.
- **kai-identity-auth**: Behavioral analysis consumer. Builds user authentication profiles over time. Slightly more tolerant of latency (60s ack wait) as it operates on statistical windows rather than individual events.
- **audit-log-archive**: Writes all authentication events to immutable object storage for compliance audit trails. Aggressive retry (max_deliver=10) to ensure no events are lost.
- **siem-export-itdr**: External SIEM forwarding for customer-facing identity dashboards.

## Example (NATS CLI)

### Publish a test authentication event (failed login)

```bash
nats pub kubric.itdr.auth.v1.tenant-acme.dc-01 \
  --header="Nats-Msg-Id:auth-$(uuidgen)" \
  '{"user":"jsmith","domain":"acme.local","action":1,"success":false,"src_ip":"203.0.113.10","auth_protocol":"kerberos","factors":["password"],"tenant_id":"tenant-acme","source_id":"dc-01","timestamp_ns":1718900000000000000,"session_id":"","target_resource":"dc-01.acme.local","failure_reason":"invalid_password","geo_city":"New York","geo_country":"US","severity_id":2}'
```

### Publish a test authentication event (successful MFA login)

```bash
nats pub kubric.itdr.auth.v1.tenant-acme.authentik-01 \
  --header="Nats-Msg-Id:auth-$(uuidgen)" \
  '{"user":"jsmith","domain":"acme.local","action":1,"success":true,"src_ip":"203.0.113.10","auth_protocol":"oauth2","factors":["password","webauthn"],"tenant_id":"tenant-acme","source_id":"authentik-01","timestamp_ns":1718900060000000000,"session_id":"sess-abc123","target_resource":"kubric-dashboard","failure_reason":"","geo_city":"New York","geo_country":"US","severity_id":1}'
```

### Subscribe to all ITDR auth events

```bash
nats sub "kubric.itdr.auth.>"
```

### Create the ITDR stream

```bash
nats stream add KUBRIC_ITDR \
  --subjects="kubric.itdr.>" \
  --retention=limits \
  --max-age=90d \
  --max-bytes=107374182400 \
  --max-msg-size=16384 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=2m \
  --deny-delete \
  --deny-purge
```

### Create the KAI Identity consumer

```bash
nats consumer add KUBRIC_ITDR kai-identity-auth \
  --filter="kubric.itdr.auth.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=60s \
  --pull \
  --durable="kai-identity-auth"
```

## Notes

- **Multi-source correlation**: CoreSec ITDR normalizes authentication events from heterogeneous sources into a unified OCSF schema. A Kerberos TGT from Active Directory, an SSH key-based login from PAM, and an OAuth2 token from Authentik all produce the same message structure, enabling cross-source correlation.
- **GeoIP enrichment**: GeoIP lookup is performed at publish time by CoreSec ITDR using MaxMind GeoLite2 databases (updated weekly). This avoids consumer-side enrichment latency and ensures consistent geo attribution even if the GeoIP database is updated between event generation and consumption.
- **Service account monitoring**: KAI Identity maintains a registry of known service accounts. Authentication events from service accounts are monitored for interactive logons (which should never occur), source IP changes, and off-hours activity.
- **Audit immutability**: The `deny_delete` and `deny_purge` flags ensure that authentication events cannot be tampered with once written to JetStream. This is critical for compliance audit trails.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
