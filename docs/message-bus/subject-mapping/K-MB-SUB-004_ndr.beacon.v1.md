# K-MB-SUB-004 — NDR Beacon Detection

> NATS subject mapping reference for C2 beacon detection events in the Kubric UIDR platform.

## Subject Pattern

```
kubric.ndr.beacon.v1
kubric.ndr.beacon.v1.<tenant_id>
kubric.ndr.beacon.v1.<tenant_id>.<sensor_id>
```

Wildcard subscription: `kubric.ndr.beacon.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **NetGuard RITA** | Go (sidecar wrapper) | Go service that queries RITA (Real Intelligence Threat Analytics) beacon analysis results via HTTP API and publishes scored beacon events to NATS. RITA itself runs as a separate process analyzing Zeek/NetGuard flow logs. See GPL boundary note below. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-TRIAGE** | Python (CrewAI) | Receives beacon alerts and correlates them with endpoint telemetry (process, file, DNS) to build C2 kill-chain narratives. High beacon scores trigger automated containment workflows. |

## Payload (OCSF Class)

- **OCSF Class**: Network Activity (`4001`) + RITA beacon score extension
- **OCSF Category**: Network Activity
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

The payload extends the standard OCSF Network Activity class with RITA-specific beacon scoring fields in an `unmapped` extension object.

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `src_ip` | `string` | `src_endpoint.ip` | Internal source IP exhibiting beaconing behavior. |
| 2 | `dst_ip` | `string` | `dst_endpoint.ip` | External destination IP suspected of being a C2 server. |
| 3 | `beacon_score` | `float` | `unmapped.beacon_score` | RITA beacon score (0.0 to 1.0). Scores above 0.7 are considered high confidence. |
| 4 | `connections` | `uint64` | `unmapped.connections` | Total number of connections observed between src and dst in the analysis window. |
| 5 | `interval_stddev` | `float` | `unmapped.interval_stddev` | Standard deviation of connection intervals in seconds. Low stddev indicates regular beaconing. |
| 6 | `payload_cv` | `float` | `unmapped.payload_cv` | Coefficient of variation of payload sizes. Low CV suggests automated/scripted communication rather than human browsing. |
| 7 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 8 | `sensor_id` | `string` | `device.uid` | NetGuard sensor that captured the underlying flows. |
| 9 | `timestamp_ns` | `uint64` | `time` | Timestamp of the beacon analysis result in nanoseconds since Unix epoch. |
| 10 | `analysis_window` | `string` | `unmapped.analysis_window` | Time window analyzed (e.g., `"24h"`, `"7d"`). |
| 11 | `dst_port` | `uint32` | `dst_endpoint.port` | Most common destination port used in the beaconing connections. |
| 12 | `sni` | `string` | `tls.sni` | Most common TLS SNI value observed in beaconing connections (if TLS). |
| 13 | `ja3_hash` | `string` | `tls.ja3_hash.value` | JA3 hash of the beaconing client (if TLS). |
| 14 | `severity_id` | `uint32` | `severity_id` | OCSF severity, derived from beacon_score: >=0.9=Critical(5), >=0.7=High(4), >=0.5=Medium(3), <0.5=Low(2). |

## JetStream Configuration

Beacon events share the **KUBRIC_NDR** JetStream stream with flow events (K-MB-SUB-003). Both subjects fall under the `kubric.ndr.>` wildcard.

```json
{
  "stream": {
    "name": "KUBRIC_NDR",
    "subjects": ["kubric.ndr.>"],
    "retention": "limits",
    "max_age": "30d",
    "max_bytes": 214748364800,
    "max_msg_size": 32768,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "2m"
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_NDR` | **Shared stream** — flow (`kubric.ndr.flow.>`) and beacon (`kubric.ndr.beacon.>`) events coexist. Beacon events are low volume relative to flows. |
| `max_age` | `30d` | Beacon events should be retained at least as long as flow data for retrospective correlation. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-triage-beacon` | Yes | `kubric.ndr.beacon.v1.>` | `all` | `explicit` | 5 | 30s |
| `siem-export-ndr` | Yes | `kubric.ndr.>` | `all` | `explicit` | 3 | 120s |
| `alert-escalation` | Yes | `kubric.ndr.beacon.v1.>` | `all` | `explicit` | 10 | 15s |

- **kai-triage-beacon**: Primary consumer. KAI-TRIAGE receives beacon detections and performs multi-source correlation:
  1. Looks up the `src_ip` in EDR process events to identify the beaconing process.
  2. Checks DNS resolution history for the `dst_ip` to find associated domains.
  3. Queries threat intelligence feeds (MISP, OTX) for the `dst_ip`, `ja3_hash`, and `sni`.
  4. Generates a triage report with MITRE ATT&CK mapping (T1071 — Application Layer Protocol, T1573 — Encrypted Channel).
- **siem-export-ndr**: Shared consumer for external SIEM forwarding.
- **alert-escalation**: High-priority consumer with aggressive retry. Routes beacon_score >= 0.9 events to the SOC notification pipeline (PagerDuty/Slack).

## GPL Boundary

```
+-------------------+       HTTP API        +-------------------+
|                   | <-------------------> |                   |
|  RITA (GPL v3)    |   localhost:8080       | NetGuard RITA     |
|  Beacon Analyzer  |   /api/v1/beacons     | Go Sidecar        |
|                   |                       | (proprietary)     |
+-------------------+                       +-------------------+
                                                    |
                                                    | NATS Publish
                                                    v
                                            kubric.ndr.beacon.v1
```

**Important licensing note**: RITA is licensed under GPL v3. To maintain license compliance:

- **RITA runs as a standalone process**. It is not linked, embedded, or compiled into any Kubric proprietary code.
- **Communication is via HTTP API only**. The Go sidecar (`netguard-rita-bridge`) queries RITA's REST API at `localhost:8080/api/v1/beacons` to retrieve beacon analysis results.
- **The Go sidecar is proprietary Kubric code**. It transforms RITA JSON responses into OCSF-formatted Protobuf messages and publishes them to NATS. No RITA code is included in the sidecar binary.
- **No RITA code touches NATS**. The GPL boundary is cleanly separated at the HTTP API layer.
- **RITA source attribution**: RITA is developed by Active Countermeasures and licensed under GPL v3. Kubric distributes RITA as a separate container image with full source availability per GPL requirements.

## Example (NATS CLI)

### Publish a test beacon event

```bash
nats pub kubric.ndr.beacon.v1.tenant-acme.sensor-01 \
  --header="Nats-Msg-Id:beacon-$(uuidgen)" \
  '{"src_ip":"192.168.1.42","dst_ip":"198.51.100.77","beacon_score":0.92,"connections":1847,"interval_stddev":0.34,"payload_cv":0.08,"tenant_id":"tenant-acme","sensor_id":"sensor-01","timestamp_ns":1718900000000000000,"analysis_window":"24h","dst_port":443,"sni":"cdn-update.example.com","ja3_hash":"e7d705a3286e19ea42f587b344ee6865","severity_id":5}'
```

### Subscribe to beacon alerts only

```bash
nats sub "kubric.ndr.beacon.>"
```

### Create the beacon triage consumer

```bash
nats consumer add KUBRIC_NDR kai-triage-beacon \
  --filter="kubric.ndr.beacon.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=30s \
  --pull \
  --durable="kai-triage-beacon"
```

### Pull and inspect beacon events

```bash
nats consumer next KUBRIC_NDR kai-triage-beacon --count=5
```

## Notes

- **Beacon score thresholds**: Default alert thresholds are configurable per tenant. The platform defaults are: >= 0.9 Critical, >= 0.7 High, >= 0.5 Medium. Scores below 0.5 are logged but do not generate alerts.
- **Analysis cadence**: RITA beacon analysis runs on a configurable schedule (default: every 1 hour). Each run analyzes the previous 24 hours of flow data. The Go sidecar polls RITA after each analysis cycle completes.
- **False positive tuning**: Known CDN endpoints, update servers, and NTP sources generate regular connection patterns that can trigger false beacon detections. KAI-TRIAGE maintains a tenant-configurable allowlist. High beacon scores against allowlisted destinations are downgraded to informational severity.
- **Kill chain correlation**: A beacon detection alone is a medium-confidence indicator. KAI-TRIAGE upgrades severity when beacon events correlate with: (a) suspicious process spawning on the source host, (b) lateral movement attempts from the source IP, or (c) data staging/exfiltration volume anomalies on the source host.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
