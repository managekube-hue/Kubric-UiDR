# K-MB-SUB-003 — NDR Network Flow

> NATS subject mapping reference for network flow telemetry in the Kubric UIDR platform.

## Subject Pattern

```
kubric.ndr.flow.v1
kubric.ndr.flow.v1.<tenant_id>
kubric.ndr.flow.v1.<tenant_id>.<sensor_id>
```

Wildcard subscription: `kubric.ndr.flow.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **NetGuard** | Rust | Network detection agent performing packet capture via `AF_PACKET` (Linux) / Npcap (Windows). Assembles flows from raw packets, classifies L7 protocols using nDPI, extracts JA3/JA3S fingerprints, and publishes completed flow records. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-FORESIGHT** | Python (CrewAI) | Behavioral baseline engine. Ingests flow records to build per-host and per-subnet communication profiles. Detects anomalies such as new external destinations, unusual port usage, data exfiltration volume spikes, and protocol misuse. |

## Payload (OCSF Class)

- **OCSF Class**: Network Activity (`4001`)
- **OCSF Category**: Network Activity
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `src_ip` | `string` | `src_endpoint.ip` | Source IP address (IPv4 or IPv6). |
| 2 | `src_port` | `uint32` | `src_endpoint.port` | Source port number. |
| 3 | `dst_ip` | `string` | `dst_endpoint.ip` | Destination IP address (IPv4 or IPv6). |
| 4 | `dst_port` | `uint32` | `dst_endpoint.port` | Destination port number. |
| 5 | `protocol` | `uint32` | `connection_info.protocol_num` | IANA protocol number (6=TCP, 17=UDP, 1=ICMP). |
| 6 | `bytes_sent` | `uint64` | `traffic.bytes_out` | Total bytes sent from source to destination. |
| 7 | `bytes_received` | `uint64` | `traffic.bytes_in` | Total bytes received from destination. |
| 8 | `packets` | `uint64` | `traffic.packets` | Total packet count for the flow. |
| 9 | `l7_protocol` | `string` | `app_name` | Application-layer protocol detected by nDPI (e.g., `"HTTP"`, `"TLS"`, `"DNS"`, `"SSH"`, `"QUIC"`). |
| 10 | `ja3_hash` | `string` | `tls.ja3_hash.value` | JA3 fingerprint of the TLS Client Hello (empty for non-TLS flows). |
| 11 | `sni` | `string` | `tls.sni` | TLS Server Name Indication value (empty for non-TLS flows). |
| 12 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 13 | `sensor_id` | `string` | `device.uid` | Unique sensor/host identifier for the NetGuard instance. |
| 14 | `timestamp_ns` | `uint64` | `time` | Flow start timestamp in nanoseconds since Unix epoch. |
| 15 | `duration_ms` | `uint64` | `duration` | Flow duration in milliseconds. |
| 16 | `ja3s_hash` | `string` | `tls.ja3s_hash.value` | JA3S fingerprint of the TLS Server Hello. |
| 17 | `tcp_flags` | `uint32` | `connection_info.tcp_flags` | Bitfield of observed TCP flags across the flow (SYN, ACK, FIN, RST, etc.). |
| 18 | `severity_id` | `uint32` | `severity_id` | OCSF severity: 0=Unknown through 5=Critical. |

## JetStream Configuration

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
    "duplicate_window": "2m",
    "deny_delete": true,
    "deny_purge": false
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_NDR` | Shared stream for all NDR subjects (`kubric.ndr.>`), including flows and beacon alerts. |
| `subjects` | `kubric.ndr.>` | Captures flow, beacon, and future NDR event types under one stream. |
| `max_age` | `30d` (2592000s) | Longer retention than EDR — flow baselines require 30 days of history for behavioral modeling. KAI-FORESIGHT needs rolling windows for anomaly detection. |
| `max_bytes` | `200 GB` (214748364800) | Flow telemetry is high volume. 200 GB budget accommodates ~10,000 flows/sec across a mid-size deployment. |
| `max_msg_size` | `32 KB` | Flow records are compact; 32 KB is generous. Prevents accidental large payloads. |
| `num_replicas` | `3` | R3 for durability across NATS cluster nodes. |

## Flow Volume Considerations

Network flow telemetry is the highest-volume data source in the Kubric UIDR platform. Design decisions to manage volume:

| Strategy | Description |
|----------|-------------|
| **Flow aggregation** | NetGuard aggregates packets into flows (5-tuple + L7 protocol). Only completed flows are published, not individual packets. This reduces message rate by 100-1000x compared to raw packet events. |
| **Idle timeout** | Flows are exported after 30 seconds of inactivity or 5 minutes of active duration (configurable). |
| **Sampling** | For extremely high-throughput links (>10 Gbps), NetGuard supports 1:N packet sampling. Sampling ratio is published in the flow record metadata so KAI-FORESIGHT can extrapolate. |
| **Subject partitioning** | The `<tenant_id>.<sensor_id>` subject hierarchy enables per-sensor consumer partitioning for horizontal scaling. |
| **Tiered storage** | JetStream R3 storage is NVMe for recent data. Flows older than 7 days are automatically migrated to object storage (S3/MinIO) via NATS subject-based mirroring. Queries against historical flows go through the Kubric query API, not direct NATS replay. |
| **Compression** | Protobuf wire format + JetStream file-based storage with compression enabled. Typical flow record is 200-400 bytes on wire. |

**Estimated throughput per deployment size:**

| Deployment | Hosts | Flows/sec | Messages/day | Daily storage |
|------------|-------|-----------|--------------|---------------|
| Small | 50 | ~500 | ~43M | ~12 GB |
| Medium | 500 | ~5,000 | ~432M | ~120 GB |
| Large | 5,000 | ~50,000 | ~4.3B | ~1.2 TB (requires tiered storage) |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-foresight-flow` | Yes | `kubric.ndr.flow.v1.>` | `all` | `explicit` | 3 | 60s |
| `siem-export-ndr` | Yes | `kubric.ndr.>` | `all` | `explicit` | 3 | 120s |
| `flow-archiver` | Yes | `kubric.ndr.flow.v1.>` | `all` | `explicit` | 5 | 120s |

- **kai-foresight-flow**: Primary consumer for behavioral baseline building. KAI-FORESIGHT maintains rolling statistical models (mean, stddev, percentiles) for each src/dst pair. Anomalies trigger alerts on `kubric.edr.process.>` correlation subjects.
- **siem-export-ndr**: Feeds flow summaries to external SIEM. May apply downsampling to reduce volume for customer-facing dashboards.
- **flow-archiver**: Writes flow records to S3/MinIO object storage for long-term retention and compliance evidence. Operates as a high-throughput batch consumer.

## Example (NATS CLI)

### Publish a test flow event

```bash
nats pub kubric.ndr.flow.v1.tenant-acme.sensor-01 \
  --header="Nats-Msg-Id:flow-$(uuidgen)" \
  '{"src_ip":"192.168.1.100","src_port":54321,"dst_ip":"93.184.216.34","dst_port":443,"protocol":6,"bytes_sent":15234,"bytes_received":892451,"packets":347,"l7_protocol":"TLS","ja3_hash":"e7d705a3286e19ea42f587b344ee6865","sni":"example.com","tenant_id":"tenant-acme","sensor_id":"sensor-01","timestamp_ns":1718900000000000000,"duration_ms":12450,"severity_id":1}'
```

### Subscribe to all flow events

```bash
nats sub "kubric.ndr.flow.>"
```

### Create the NDR stream

```bash
nats stream add KUBRIC_NDR \
  --subjects="kubric.ndr.>" \
  --retention=limits \
  --max-age=30d \
  --max-bytes=214748364800 \
  --max-msg-size=32768 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=2m
```

### Create the KAI-FORESIGHT consumer

```bash
nats consumer add KUBRIC_NDR kai-foresight-flow \
  --filter="kubric.ndr.flow.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=3 \
  --wait=60s \
  --pull \
  --durable="kai-foresight-flow"
```

### Monitor flow ingestion rate

```bash
nats stream info KUBRIC_NDR --json | jq '.state.messages, .state.bytes'
```

## Notes

- **nDPI integration**: NetGuard embeds nDPI for deep packet inspection. The `l7_protocol` field contains the nDPI protocol name. Over 300 protocols are supported including encrypted protocol classification (e.g., distinguishing Zoom from generic TLS).
- **JA3 fingerprinting**: JA3 hashes are computed for every TLS Client Hello observed. These are correlated against known-malicious JA3 databases (e.g., Abuse.ch JA3 feed) by KAI-FORESIGHT. JA3S (server) hashes are also captured for bidirectional fingerprinting.
- **IPv6 support**: All IP fields store both IPv4 and IPv6 addresses as strings. NetGuard normalizes IPv4-mapped IPv6 addresses to their IPv4 form.
- **DNS correlation**: When `l7_protocol` is `"DNS"`, additional fields (`dns_query`, `dns_rcode`, `dns_answers`) are populated in an extension object. This supports DNS tunneling detection by KAI-FORESIGHT.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
- **Baseline window**: KAI-FORESIGHT builds baselines over a 14-day rolling window by default. The 30-day stream retention provides buffer for baseline recomputation after model updates.
