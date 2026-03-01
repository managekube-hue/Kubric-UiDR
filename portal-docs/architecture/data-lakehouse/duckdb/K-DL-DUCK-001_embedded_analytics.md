# K-DL-DUCK-001 — DuckDB Embedded Analytics

## Overview

DuckDB provides embedded analytical SQL processing directly inside Kubric agents and KAI Python scripts. Zero-server deployment — runs in-process like SQLite but optimized for OLAP workloads.

## Architecture

```
                    ┌─────────────────────────┐
                    │   ClickHouse (cluster)   │  ← Hot telemetry
                    └─────────┬───────────────┘
                              │ Bulk export
                              ▼
┌──────────────┐    ┌─────────────────────────┐
│ CoreSec Agent│───▶│    DuckDB (embedded)     │  ← In-agent analytics
│  (Rust/Go)   │    │  • Local OCSF correlation│
└──────────────┘    │  • ML feature compute    │
                    │  • Offline querying       │
                    └─────────────────────────┘
```

## Use Cases

### 1. In-Agent OCSF Event Correlation (Go)

```go
import (
    "database/sql"
    _ "github.com/marcboeker/go-duckdb"
)

db, _ := sql.Open("duckdb", "")
defer db.Close()

// Create in-memory table from recent OCSF events
db.Exec(`CREATE TABLE events AS SELECT * FROM read_json_auto('events.jsonl')`)

// Correlate: processes that spawned after suspicious network connections
rows, _ := db.Query(`
    SELECT p.executable, p.cmdline, n.dst_ip, n.dst_port
    FROM events p
    JOIN events n ON p.pid = n.pid
    WHERE p.event_class = 4007  -- Process
      AND n.event_class = 4001  -- Network
      AND n.dst_port IN (4444, 5555, 8080, 1337)
      AND n.timestamp_ns > p.timestamp_ns
    ORDER BY p.timestamp_ns
`)
```

### 2. KAI Python Analytics

```python
import duckdb

con = duckdb.connect()

# Load Parquet from MinIO
con.execute("""
    SELECT
        tenant_id,
        event_class,
        count(*) as event_count,
        avg(severity) as avg_severity
    FROM read_parquet('s3://kubric-telemetry/2024/01/*.parquet',
                      s3_access_key_id='...', s3_secret_access_key='...')
    WHERE event_date >= '2024-01-01'
    GROUP BY tenant_id, event_class
    ORDER BY event_count DESC
""")
```

### 3. Air-Gapped Endpoint Querying

When the agent is disconnected from the cluster, DuckDB provides local query capabilities on cached OCSF events without requiring ClickHouse connectivity.

## Performance

| Query Type | DuckDB (embedded) | ClickHouse (network) |
|------------|-------------------|---------------------|
| 1M row scan | ~200ms | ~50ms + 10ms network |
| 10K row aggregation | ~5ms | ~2ms + 10ms network |
| Join (2 tables, 100K) | ~50ms | ~20ms + 10ms network |
| Startup time | ~1ms (in-process) | N/A (always running) |

DuckDB wins when network latency exceeds query time — common on remote endpoints.

## Storage

DuckDB files are stored in the agent's data directory:
- Linux: `/var/lib/kubric/analytics.duckdb`
- Windows: `%ProgramData%\Kubric\analytics.duckdb`
- macOS: `/Library/Application Support/Kubric/analytics.duckdb`

Maximum local DB size: configurable, default 1GB. Oldest events pruned on overflow.
