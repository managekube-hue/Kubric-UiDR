# K-DL-CH-005 — Apache Arrow Bulk Insert Pipeline

> **Language:** Go (primary), Python (KAI batch)  
> **Libraries:** `github.com/apache/arrow-go/v18`, `github.com/ClickHouse/clickhouse-go/v2`  
> **Transport:** ClickHouse native protocol (port 9440 TLS)  
> **Target Table:** `kubric_telemetry.security_events_distributed`  
> **Throughput Target:** 100K events/sec sustained, 500K burst

---

## 1. Architecture

```
┌──────────────┐     ┌──────────────┐     ┌───────────────────┐
│  NATS JetStream │──►│  Go Ingester  │──►│  ClickHouse        │
│  (events.>)     │   │  Arrow batch  │   │  Native bulk insert│
│                 │   │  100K rows/s  │   │  columnar format   │
└──────────────┘     └──────────────┘     └───────────────────┘
                           │
                     ┌─────┴─────┐
                     │ Arrow IPC  │
                     │ RecordBatch│
                     └───────────┘
```

---

## 2. Go Dependencies

```bash
go get github.com/apache/arrow-go/v18@latest
go get github.com/ClickHouse/clickhouse-go/v2@latest
go get github.com/nats-io/nats.go@latest
```

---

## 3. Go Arrow Bulk Inserter

File: `internal/ingester/arrow_inserter.go`

```go
package ingester

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SecurityEvent represents a parsed security event from NATS.
type SecurityEvent struct {
	TenantID       uuid.UUID
	EventTime      time.Time
	SourceType     string
	SourceHost     string
	SourceIP       net.IP
	Severity       string
	Category       string
	ClassUID       uint32
	SrcIP          net.IP
	DstIP          net.IP
	SrcPort        uint16
	DstPort        uint16
	Protocol       string
	ProcessName    string
	ProcessPID     uint32
	ProcessCmd     string
	UserName       string
	RuleName       string
	RuleID         string
	MitreTactic    string
	MitreTechnique string
	RawLog         string
	Metadata       string
}

// ArrowInserter batches SecurityEvents using Apache Arrow and bulk-inserts into ClickHouse.
type ArrowInserter struct {
	conn       driver.Conn
	logger     *zap.Logger
	allocator  memory.Allocator
	batchSize  int
	flushInterval time.Duration

	mu      sync.Mutex
	buffer  []SecurityEvent
	schema  *arrow.Schema
}

// NewArrowInserter creates a new bulk inserter.
func NewArrowInserter(cfg Config, logger *zap.Logger) (*ArrowInserter, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: cfg.ClickHouseHosts, // ["10.0.50.21:9440", "10.0.50.22:9440"]
		Auth: clickhouse.Auth{
			Database: "kubric_telemetry",
			Username: cfg.ClickHouseUser,
			Password: cfg.ClickHousePass,
		},
		TLS:         cfg.TLSConfig,
		DialTimeout: 10 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time":             300,
			"max_insert_block_size":          1_000_000,
			"async_insert":                   1,
			"wait_for_async_insert":          0,
			"async_insert_max_data_size":     10_485_760, // 10 MB
			"async_insert_busy_timeout_ms":   1000,
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse connect: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "tenant_id", Type: &arrow.FixedSizeBinaryType{ByteWidth: 16}},
		{Name: "event_time", Type: &arrow.TimestampType{Unit: arrow.Millisecond, TimeZone: "UTC"}},
		{Name: "source_type", Type: arrow.BinaryTypes.String},
		{Name: "source_host", Type: arrow.BinaryTypes.String},
		{Name: "source_ip", Type: arrow.BinaryTypes.String},
		{Name: "severity", Type: arrow.BinaryTypes.String},
		{Name: "category", Type: arrow.BinaryTypes.String},
		{Name: "class_uid", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "src_ip", Type: arrow.BinaryTypes.String},
		{Name: "dst_ip", Type: arrow.BinaryTypes.String},
		{Name: "src_port", Type: arrow.PrimitiveTypes.Uint16},
		{Name: "dst_port", Type: arrow.PrimitiveTypes.Uint16},
		{Name: "protocol", Type: arrow.BinaryTypes.String},
		{Name: "process_name", Type: arrow.BinaryTypes.String},
		{Name: "process_pid", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "process_cmd", Type: arrow.BinaryTypes.String},
		{Name: "user_name", Type: arrow.BinaryTypes.String},
		{Name: "rule_name", Type: arrow.BinaryTypes.String},
		{Name: "rule_id", Type: arrow.BinaryTypes.String},
		{Name: "mitre_tactic", Type: arrow.BinaryTypes.String},
		{Name: "mitre_technique", Type: arrow.BinaryTypes.String},
		{Name: "raw_log", Type: arrow.BinaryTypes.String},
		{Name: "metadata", Type: arrow.BinaryTypes.String},
	}, nil)

	return &ArrowInserter{
		conn:          conn,
		logger:        logger,
		allocator:     memory.NewGoAllocator(),
		batchSize:     cfg.BatchSize,     // default: 50_000
		flushInterval: cfg.FlushInterval, // default: 5s
		buffer:        make([]SecurityEvent, 0, cfg.BatchSize),
		schema:        schema,
	}, nil
}

// Add buffers an event. Flushes when batch size is reached.
func (ai *ArrowInserter) Add(event SecurityEvent) error {
	ai.mu.Lock()
	ai.buffer = append(ai.buffer, event)
	shouldFlush := len(ai.buffer) >= ai.batchSize
	ai.mu.Unlock()

	if shouldFlush {
		return ai.Flush(context.Background())
	}
	return nil
}

// Flush writes the current buffer to ClickHouse.
func (ai *ArrowInserter) Flush(ctx context.Context) error {
	ai.mu.Lock()
	if len(ai.buffer) == 0 {
		ai.mu.Unlock()
		return nil
	}
	events := make([]SecurityEvent, len(ai.buffer))
	copy(events, ai.buffer)
	ai.buffer = ai.buffer[:0]
	ai.mu.Unlock()

	start := time.Now()

	batch, err := ai.conn.PrepareBatch(ctx,
		"INSERT INTO security_events_distributed "+
			"(tenant_id, event_time, source_type, source_host, source_ip, "+
			"severity, category, class_uid, "+
			"src_ip, dst_ip, src_port, dst_port, protocol, "+
			"process_name, process_pid, process_cmd, user_name, "+
			"rule_name, rule_id, mitre_tactic, mitre_technique, "+
			"raw_log, metadata)")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, e := range events {
		err := batch.Append(
			e.TenantID,
			e.EventTime,
			e.SourceType,
			e.SourceHost,
			e.SourceIP.String(),
			e.Severity,
			e.Category,
			e.ClassUID,
			ipToString(e.SrcIP),
			ipToString(e.DstIP),
			e.SrcPort,
			e.DstPort,
			e.Protocol,
			e.ProcessName,
			e.ProcessPID,
			e.ProcessCmd,
			e.UserName,
			e.RuleName,
			e.RuleID,
			e.MitreTactic,
			e.MitreTechnique,
			e.RawLog,
			e.Metadata,
		)
		if err != nil {
			ai.logger.Error("batch append failed", zap.Error(err))
			continue
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("batch send (%d events): %w", len(events), err)
	}

	duration := time.Since(start)
	ai.logger.Info("batch flushed",
		zap.Int("events", len(events)),
		zap.Duration("duration", duration),
		zap.Float64("events_per_sec", float64(len(events))/duration.Seconds()),
	)

	return nil
}

// StartFlushLoop runs a periodic flush.
func (ai *ArrowInserter) StartFlushLoop(ctx context.Context) {
	ticker := time.NewTicker(ai.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush
			ai.Flush(context.Background())
			return
		case <-ticker.C:
			if err := ai.Flush(ctx); err != nil {
				ai.logger.Error("periodic flush failed", zap.Error(err))
			}
		}
	}
}

// Close flushes remaining events and closes the connection.
func (ai *ArrowInserter) Close() error {
	ai.Flush(context.Background())
	return ai.conn.Close()
}

func ipToString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

// Config holds ArrowInserter configuration.
type Config struct {
	ClickHouseHosts []string
	ClickHouseUser  string
	ClickHousePass  string
	TLSConfig       *tls.Config
	BatchSize       int
	FlushInterval   time.Duration
}
```

---

## 4. NATS Consumer Integration

File: `internal/ingester/nats_consumer.go`

```go
package ingester

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// StartNATSConsumer subscribes to security events and feeds them to ArrowInserter.
func StartNATSConsumer(ctx context.Context, natsURL string, inserter *ArrowInserter, logger *zap.Logger) error {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstream new: %w", err)
	}

	// Create or bind to stream
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "SECURITY_EVENTS",
		Subjects:    []string{"events.security.>"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		MaxBytes:    10 * 1024 * 1024 * 1024, // 10 GB
		Replicas:    3,
		Compression: jetstream.S2Compression,
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}

	// Durable consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "clickhouse-ingester",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 10000,
		FilterSubject: "events.security.>",
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	// Consume messages
	iter, err := consumer.Messages(
		jetstream.PullMaxMessages(500),
		jetstream.PullExpiry(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	// Start flush loop
	go inserter.StartFlushLoop(ctx)

	logger.Info("NATS consumer started", zap.String("stream", "SECURITY_EVENTS"))

	for {
		select {
		case <-ctx.Done():
			iter.Stop()
			return nil
		default:
			msg, err := iter.Next()
			if err != nil {
				continue
			}

			var event SecurityEvent
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				logger.Warn("unmarshal failed", zap.Error(err))
				msg.Term()
				continue
			}

			if err := inserter.Add(event); err != nil {
				logger.Error("insert failed", zap.Error(err))
				msg.Nak()
				continue
			}

			msg.Ack()
		}
	}
}
```

---

## 5. Python Arrow Bulk Insert (KAI Batch)

File: `kai/ingestion/arrow_bulk.py`

```python
"""
Arrow-based bulk insert for KAI ML pipeline results.
Uses clickhouse-connect with Arrow/Pandas interop.
"""

import uuid
from datetime import datetime, timezone
from typing import Optional

import clickhouse_connect
import pyarrow as pa


# ClickHouse connection
client = clickhouse_connect.get_client(
    host="10.0.50.21",
    port=8123,
    username="kubric_writer",
    password="CHANGEME_CH_WRITER_PASS",
    database="kubric_telemetry",
    compress=True,
    send_receive_timeout=300,
)


def build_arrow_table(events: list[dict]) -> pa.Table:
    """Convert list of event dicts to Arrow table for columnar insert."""
    schema = pa.schema([
        ("tenant_id", pa.string()),
        ("event_time", pa.timestamp("ms", tz="UTC")),
        ("source_type", pa.string()),
        ("source_host", pa.string()),
        ("source_ip", pa.string()),
        ("severity", pa.string()),
        ("category", pa.string()),
        ("class_uid", pa.uint32()),
        ("raw_log", pa.string()),
        ("metadata", pa.string()),
        ("rule_name", pa.string()),
        ("mitre_tactic", pa.string()),
        ("mitre_technique", pa.string()),
    ])

    arrays = {
        "tenant_id": [str(e.get("tenant_id", "")) for e in events],
        "event_time": [e.get("event_time", datetime.now(timezone.utc)) for e in events],
        "source_type": [e.get("source_type", "kai-ml") for e in events],
        "source_host": [e.get("source_host", "") for e in events],
        "source_ip": [e.get("source_ip", "0.0.0.0") for e in events],
        "severity": [e.get("severity", "info") for e in events],
        "category": [e.get("category", "ml_detection") for e in events],
        "class_uid": [e.get("class_uid", 0) for e in events],
        "raw_log": [e.get("raw_log", "") for e in events],
        "metadata": [e.get("metadata", "{}") for e in events],
        "rule_name": [e.get("rule_name", "") for e in events],
        "mitre_tactic": [e.get("mitre_tactic", "") for e in events],
        "mitre_technique": [e.get("mitre_technique", "") for e in events],
    }

    return pa.table(arrays, schema=schema)


def bulk_insert_events(events: list[dict], table_name: str = "security_events_distributed") -> int:
    """
    Bulk insert events using Arrow columnar format.
    Returns number of inserted rows.
    """
    if not events:
        return 0

    arrow_table = build_arrow_table(events)

    columns = [
        "tenant_id", "event_time", "source_type", "source_host", "source_ip",
        "severity", "category", "class_uid", "raw_log", "metadata",
        "rule_name", "mitre_tactic", "mitre_technique",
    ]

    client.insert_arrow(
        table_name,
        arrow_table,
        column_names=columns,
    )

    return len(events)


def bulk_insert_from_parquet(parquet_path: str, table_name: str = "security_events_distributed") -> int:
    """Insert directly from Parquet file (zero-copy Arrow)."""
    table = pa.parquet.read_table(parquet_path)
    client.insert_arrow(table_name, table)
    return table.num_rows


# ─── Benchmark ───
if __name__ == "__main__":
    import time

    # Generate test events
    test_events = [
        {
            "tenant_id": str(uuid.uuid4()),
            "event_time": datetime.now(timezone.utc),
            "source_type": "benchmark",
            "source_host": f"host-{i}",
            "source_ip": f"10.0.0.{i % 255}",
            "severity": "info",
            "category": "test",
            "class_uid": 1001,
            "raw_log": f"benchmark event {i}",
            "metadata": '{"benchmark": true}',
        }
        for i in range(100_000)
    ]

    start = time.perf_counter()
    count = bulk_insert_events(test_events)
    elapsed = time.perf_counter() - start

    print(f"Inserted {count} events in {elapsed:.2f}s ({count/elapsed:.0f} events/sec)")
```

---

## 6. Benchmark Results

Expected performance on Dell R630 (NVMe Ceph):

| Method | Batch Size | Events/sec | Latency (p99) |
|--------|-----------|------------|---------------|
| Go Arrow bulk | 50,000 | ~120K | 450ms |
| Go Arrow bulk | 100,000 | ~150K | 800ms |
| Python Arrow | 100,000 | ~80K | 1.2s |
| Python Parquet import | 1M | ~200K | 5s |
| Go async insert | 1 (streaming) | ~30K | 2ms |

---

## 7. Verification

```bash
# Go benchmark
cd /path/to/kubric-uidr
go test -bench=BenchmarkArrowInsert -benchmem ./internal/ingester/...

# Python benchmark
python kai/ingestion/arrow_bulk.py

# Verify inserted rows
clickhouse-client --host 10.0.50.21 --secure --query "
  SELECT count(), formatReadableSize(sum(bytes_on_disk))
  FROM system.parts
  WHERE database='kubric_telemetry' AND table='security_events' AND active"

# Check insertion rate (last 5 minutes)
clickhouse-client --query "
  SELECT
    toStartOfMinute(event_time) AS minute,
    count() AS events,
    uniqExact(source_type) AS sources
  FROM kubric_telemetry.security_events_distributed
  WHERE event_time > now() - INTERVAL 5 MINUTE
  GROUP BY minute
  ORDER BY minute"
```
