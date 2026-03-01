# K-MB-SUB-011 — Threat Intelligence IOC

> NATS subject mapping reference for STIX 2.1 Indicator of Compromise distribution within the Kubric UIDR platform.

## Subject Pattern

```
kubric.ti.ioc.v1
kubric.ti.ioc.>                 # wildcard for all TI IOC sub-subjects
```

Tokens:
| Position | Token       | Description                                  |
|----------|-------------|----------------------------------------------|
| 1        | `kubric`    | Platform root namespace                      |
| 2        | `ti`        | Domain — threat intelligence                 |
| 3        | `ioc`       | Event type — Indicator of Compromise         |
| 4        | `v1`        | Schema version                               |

## Publisher

**SIDR TI (Go)** — the Security Intelligence Detection and Response Threat Intelligence module. SIDR TI aggregates IOCs from multiple upstream sources and normalizes them into STIX 2.1 Indicator objects before publishing.

Upstream feed sources:
| Source       | Protocol      | Refresh Interval | Description                          |
|--------------|---------------|-------------------|--------------------------------------|
| MISP         | REST API      | 5 minutes         | Community and private MISP instances |
| OTX          | REST API      | 15 minutes        | AlienVault Open Threat Exchange      |
| AbuseIPDB    | REST API      | 10 minutes        | IP reputation and abuse reports      |
| CISA KEV     | REST API      | 1 hour            | Known Exploited Vulnerabilities catalog |

## Consumer(s)

| Consumer                  | Runtime  | Role                                                       |
|---------------------------|----------|-------------------------------------------------------------|
| **NetGuard** (primary)    | Rust     | Loads IOCs into IDS/NTA rule engine for network-level detection |
| **CoreSec** (primary)     | Rust     | Loads IOCs into FIM/EDR watchlists for endpoint-level detection |
| KAI-HUNTER                | Python   | Enriches threat hunts with latest IOC context               |

## Payload

**Format:** JSON — STIX 2.1 Indicator object with Kubric-specific extensions.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "stix_type": "indicator",
  "indicator_type": "ip",
  "pattern": "[ipv4-addr:value = '203.0.113.42']",
  "pattern_type": "stix",
  "confidence": 85,
  "source": "MISP",
  "valid_from": "2026-02-25T00:00:00Z",
  "valid_until": "2026-03-25T00:00:00Z",
  "threat_actor": "APT-29",
  "mitre_techniques": ["T1071.001", "T1059.001"],
  "labels": ["malicious-activity", "c2"],
  "created": "2026-02-25T12:00:00Z",
  "modified": "2026-02-26T08:15:00Z"
}
```

## Fields

| Field               | Type       | Required | Description                                                              |
|---------------------|------------|----------|--------------------------------------------------------------------------|
| `schema_version`    | string     | yes      | Payload schema version (semver)                                          |
| `stix_type`         | string     | yes      | Always `indicator` for this subject                                      |
| `indicator_type`    | string     | yes      | IOC type: `ip`, `domain`, `hash`, `url`                                  |
| `pattern`           | string     | yes      | STIX 2.1 pattern expression                                             |
| `pattern_type`      | string     | yes      | Pattern language: `stix`, `snort`, `sigma`, `yara`                       |
| `confidence`        | integer    | yes      | Confidence score (0-100) assigned by source and/or SIDR TI enrichment    |
| `source`            | string     | yes      | Originating feed: `MISP`, `OTX`, `AbuseIPDB`, `CISA_KEV`                |
| `valid_from`        | datetime   | yes      | ISO 8601 UTC start of indicator validity window                          |
| `valid_until`       | datetime   | yes      | ISO 8601 UTC end of indicator validity window                            |
| `threat_actor`      | string     | no       | Associated threat actor name (if attributed)                             |
| `mitre_techniques`  | string[]   | no       | MITRE ATT&CK technique IDs associated with this indicator               |
| `labels`            | string[]   | no       | STIX 2.1 indicator labels (e.g., `malicious-activity`, `c2`)            |
| `created`           | datetime   | yes      | ISO 8601 UTC timestamp when the indicator was first created              |
| `modified`          | datetime   | yes      | ISO 8601 UTC timestamp of last modification                              |

### NATS KV Store for Fast IOC Lookup

In addition to publishing IOCs on the JetStream subject for real-time push, SIDR TI maintains a **NATS KV Store** bucket for fast point-lookups by detection engines:

```
Bucket:     IOC_STORE
Key format: <indicator_type>.<value_hash_blake3>
Value:      Full STIX indicator JSON
TTL:        Matches valid_until of the indicator
```

NetGuard and CoreSec query the KV Store at detection time to check whether an observed artifact matches a known IOC, enabling sub-millisecond lookup without re-querying upstream feeds.

**KV operations:**

```bash
# Put an IOC
nats kv put IOC_STORE ip.a3f2b1c4d5e6 '{"indicator_type":"ip","pattern":"[ipv4-addr:value = '\''203.0.113.42'\'']",...}'

# Get an IOC
nats kv get IOC_STORE ip.a3f2b1c4d5e6

# Watch for IOC updates
nats kv watch IOC_STORE
```

## JetStream Configuration

```
Stream:          TI_IOC
Subjects:        kubric.ti.ioc.>
Storage:         File
Retention:       Limits
Max Age:         30 days
Max Bytes:       5 GB
Replicas:        3
Discard Policy:  Old
Duplicate Window: 30 seconds
```

## Consumer Groups

| Consumer Group       | Deliver Policy | Ack Policy  | Max Deliver | Filter Subject         |
|----------------------|---------------|-------------|-------------|------------------------|
| `netguard-ioc-sync`  | All           | Explicit    | 5           | `kubric.ti.ioc.>`      |
| `coresec-ioc-sync`   | All           | Explicit    | 5           | `kubric.ti.ioc.>`      |
| `hunter-enrichment`  | All           | Explicit    | 3           | `kubric.ti.ioc.>`      |

## Example (NATS CLI)

**Publish an IOC indicator:**

```bash
nats pub kubric.ti.ioc.v1 '{
  "schema_version": "1.0.0",
  "stix_type": "indicator",
  "indicator_type": "ip",
  "pattern": "[ipv4-addr:value = '\''203.0.113.42'\'']",
  "pattern_type": "stix",
  "confidence": 85,
  "source": "MISP",
  "valid_from": "2026-02-25T00:00:00Z",
  "valid_until": "2026-03-25T00:00:00Z",
  "threat_actor": "APT-29",
  "mitre_techniques": ["T1071.001", "T1059.001"],
  "labels": ["malicious-activity", "c2"],
  "created": "2026-02-25T12:00:00Z",
  "modified": "2026-02-26T08:15:00Z"
}'
```

**Subscribe to all TI IOC events:**

```bash
nats sub "kubric.ti.ioc.>"
```

**Create the JetStream stream:**

```bash
nats stream add TI_IOC \
  --subjects "kubric.ti.ioc.>" \
  --storage file \
  --retention limits \
  --max-age 30d \
  --max-bytes 5GB \
  --replicas 3 \
  --discard old \
  --dupe-window 30s
```

**Create the netguard-ioc-sync consumer:**

```bash
nats consumer add TI_IOC netguard-ioc-sync \
  --deliver all \
  --ack explicit \
  --max-deliver 5 \
  --filter "kubric.ti.ioc.>"
```

## Notes

- **Deduplication:** SIDR TI deduplicates IOCs across feeds using a BLAKE3 hash of `(indicator_type, pattern)`. If the same IOC appears in MISP and OTX, only one message is published with the higher confidence score.
- **Confidence Decay:** IOCs with no reconfirmation from any feed within 7 days have their confidence score reduced by 10 points per day. IOCs falling below confidence 20 are removed from the KV Store.
- **False Positive Suppression:** CoreSec and NetGuard can publish to `kubric.ti.ioc.feedback.v1` to flag false positives. SIDR TI incorporates this feedback into confidence scoring.
- **Volume Estimate:** Approximately 5,000-15,000 IOCs per day across all feeds during normal operations.
- **CISA KEV Special Handling:** IOCs sourced from CISA KEV are always published with `confidence: 100` and are exempt from confidence decay, as they represent confirmed exploited vulnerabilities.
- **Related Subjects:** `kubric.ti.ioc.feedback.v1` (detection engine feedback), `kubric.ti.feed.status.v1` (feed health monitoring).
