# K-DL-CH-005 — Arrow Bulk Insert

## Overview

Apache Arrow IPC format provides the fastest bulk insert path into ClickHouse. The `clickhouse-connect` Python client and ClickHouse Go driver both support Arrow-native inserts, eliminating serialization overhead for large OCSF event batches.

## Architecture

```
Rust Agent (OCSF events)
    │
    ▼ NATS JetStream (protobuf)
    │
    ▼ KAI Python / Go Ingestor
    │
    ├── PyArrow Table (in-memory)
    │   └── clickhouse_connect.insert_arrow(table)
    │
    └── Go Arrow IPC
        └── clickhouse.Conn.PrepareBatch() + Arrow columns
```

## Python Bulk Insert (clickhouse-connect + PyArrow)

```python
import clickhouse_connect
import pyarrow as pa

client = clickhouse_connect.get_client(
    host='clickhouse.kubric.svc.cluster.local',
    port=8123,
    database='kubric',
    username='ingestor',
    password=vault_secret('clickhouse/ingestor'),
)

# Build Arrow table from OCSF events
schema = pa.schema([
    ('event_id', pa.string()),
    ('tenant_id', pa.string()),
    ('event_class', pa.uint16()),
    ('severity', pa.uint8()),
    ('src_ip', pa.string()),
    ('dst_ip', pa.string()),
    ('timestamp', pa.timestamp('ms')),
    ('payload', pa.string()),     # JSON
    ('blake3_hash', pa.string()), # 64-char hex
])

table = pa.table({
    'event_id': event_ids,
    'tenant_id': tenant_ids,
    'event_class': classes,
    'severity': severities,
    'src_ip': src_ips,
    'dst_ip': dst_ips,
    'timestamp': timestamps,
    'payload': payloads,
    'blake3_hash': hashes,
}, schema=schema)

# Insert — zero-copy Arrow IPC to ClickHouse
client.insert_arrow('ocsf_events', table)
```

## Go Bulk Insert (clickhouse-go v2)

```go
batch, _ := conn.PrepareBatch(ctx, "INSERT INTO ocsf_events")
for _, event := range events {
    batch.Append(
        event.ID, event.TenantID, event.Class,
        event.Severity, event.SrcIP, event.DstIP,
        event.Timestamp, event.Payload, event.Blake3Hash,
    )
}
batch.Send()
```

## Performance Benchmarks

| Method | Rows/sec | Latency (1M rows) | CPU |
|--------|----------|-------------------|-----|
| Arrow IPC (Python) | ~500K | ~2s | Low |
| Native format (Go) | ~400K | ~2.5s | Low |
| JSON HTTP (fallback) | ~50K | ~20s | High |
| CSV (legacy) | ~100K | ~10s | Medium |

## Compression

OCSF event payloads are ZSTD-compressed before Arrow insert to reduce network I/O:

```python
import zstandard as zstd

compressor = zstd.ZstdCompressor(level=3)
compressed_payloads = [compressor.compress(p.encode()) for p in payloads]
```

## Batching Strategy

- **Batch size**: 50,000 events per insert
- **Flush interval**: 5 seconds (whichever comes first)
- **Retry**: 3 attempts with exponential backoff
- **Dead letter**: Failed batches written to MinIO for manual investigation
