# K-DL-DUCK-001 — DuckDB Embedded Analytics

> **Version:** DuckDB 1.1+  
> **Language:** Go (primary API), Python (KAI ML)  
> **Mode:** Embedded (in-process), no separate server  
> **Use Cases:** Ad-hoc queries on Parquet/CSV exports, ML feature computation, client-side OLAP, offline analytics  
> **Data Sources:** ClickHouse Parquet exports in MinIO, PostgreSQL via `postgres_scanner`, local CSV/JSON  
> **Go Library:** `github.com/marcboeker/go-duckdb`  
> **Python:** `duckdb` (pip)

---

## 1. Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Kubric API / KAI                                       │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │  PostgreSQL   │    │  ClickHouse   │                   │
│  │  kubric_core  │    │  telemetry    │                   │
│  │  (OLTP)       │    │  (OLAP)       │                   │
│  └──────┬───────┘    └──────┬───────┘                   │
│         │                    │                            │
│         │  postgres_scanner  │  Parquet export to MinIO   │
│         ▼                    ▼                            │
│  ┌──────────────────────────────────────┐               │
│  │  DuckDB (embedded, in-process)       │               │
│  │  - Ad-hoc joins across PG + CH data  │               │
│  │  - ML feature extraction             │               │
│  │  - Client-side dashboards            │               │
│  │  - Offline analytics (no network)    │               │
│  └──────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Go Integration

### 2.1 Dependencies

```bash
go get github.com/marcboeker/go-duckdb@latest
```

### 2.2 Embedded Analytics Engine

File: `internal/analytics/duckdb_engine.go`

```go
package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/marcboeker/go-duckdb"
	"go.uber.org/zap"
)

// DuckDBEngine provides embedded OLAP analytics.
type DuckDBEngine struct {
	db     *sql.DB
	mu     sync.RWMutex
	logger *zap.Logger
	dbPath string
}

// NewDuckDBEngine creates or opens a DuckDB database.
// Use ":memory:" for in-memory or a file path for persistent.
func NewDuckDBEngine(dbPath string, logger *zap.Logger) (*DuckDBEngine, error) {
	if dbPath != ":memory:" {
		os.MkdirAll(filepath.Dir(dbPath), 0755)
	}

	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_WRITE&threads=4&memory_limit=4GB")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	engine := &DuckDBEngine{
		db:     db,
		logger: logger,
		dbPath: dbPath,
	}

	if err := engine.init(); err != nil {
		db.Close()
		return nil, err
	}

	return engine, nil
}

// init loads required extensions and sets up configuration.
func (e *DuckDBEngine) init() error {
	initSQL := []string{
		// Extensions
		"INSTALL httpfs",
		"LOAD httpfs",
		"INSTALL postgres_scanner",
		"LOAD postgres_scanner",

		// S3/MinIO config for reading Parquet from MinIO
		"SET s3_endpoint='10.0.50.23:9000'",
		"SET s3_access_key_id='kubric-duckdb'",
		"SET s3_secret_access_key='" + os.Getenv("MINIO_DUCKDB_PASS") + "'",
		"SET s3_use_ssl=false",
		"SET s3_url_style='path'",

		// Performance
		"SET memory_limit='4GB'",
		"SET threads=4",
		"SET enable_progress_bar=false",
	}

	for _, stmt := range initSQL {
		if _, err := e.db.Exec(stmt); err != nil {
			e.logger.Warn("init statement failed (may be ok)", zap.String("sql", stmt), zap.Error(err))
		}
	}

	return nil
}

// QueryParquet runs a SQL query against Parquet files in MinIO.
func (e *DuckDBEngine) QueryParquet(ctx context.Context, query string) (*sql.Rows, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.db.QueryContext(ctx, query)
}

// AttachPostgres attaches PostgreSQL as a foreign data source.
func (e *DuckDBEngine) AttachPostgres(name, connStr string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	stmt := fmt.Sprintf(
		"ATTACH '%s' AS %s (TYPE POSTGRES, READ_ONLY)",
		connStr, name,
	)
	_, err := e.db.Exec(stmt)
	return err
}

// ExportToParquet exports a query result to a Parquet file.
func (e *DuckDBEngine) ExportToParquet(ctx context.Context, query, outputPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	stmt := fmt.Sprintf("COPY (%s) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD, ROW_GROUP_SIZE 100000)", query, outputPath)
	_, err := e.db.ExecContext(ctx, stmt)
	return err
}

// Close shuts down the DuckDB engine.
func (e *DuckDBEngine) Close() error {
	return e.db.Close()
}
```

### 2.3 Analytics Use Cases

File: `internal/analytics/queries.go`

```go
package analytics

import (
	"context"
	"database/sql"
	"time"
)

// ThreatSummary holds aggregated threat data.
type ThreatSummary struct {
	TenantID      string
	Severity      string
	SourceType    string
	EventCount    int64
	UniqueHosts   int64
	FirstSeen     time.Time
	LastSeen      time.Time
}

// GetThreatSummaryFromParquet queries exported ClickHouse Parquet data.
func (e *DuckDBEngine) GetThreatSummaryFromParquet(ctx context.Context, tenantID string, days int) ([]ThreatSummary, error) {
	query := `
		SELECT
			tenant_id,
			severity,
			source_type,
			count(*) AS event_count,
			count(DISTINCT source_host) AS unique_hosts,
			min(event_time) AS first_seen,
			max(event_time) AS last_seen
		FROM read_parquet('s3://kubric-ch-cold/exports/security_events_*.parquet',
			hive_partitioning=true)
		WHERE tenant_id = ?
		  AND event_time >= current_timestamp - INTERVAL ? DAY
		GROUP BY tenant_id, severity, source_type
		ORDER BY event_count DESC
	`

	rows, err := e.db.QueryContext(ctx, query, tenantID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ThreatSummary
	for rows.Next() {
		var ts ThreatSummary
		if err := rows.Scan(
			&ts.TenantID, &ts.Severity, &ts.SourceType,
			&ts.EventCount, &ts.UniqueHosts,
			&ts.FirstSeen, &ts.LastSeen,
		); err != nil {
			return nil, err
		}
		results = append(results, ts)
	}
	return results, rows.Err()
}

// CrossJoinAlertsTelemetry joins PG alerts with CH telemetry (via Parquet).
func (e *DuckDBEngine) CrossJoinAlertsTelemetry(ctx context.Context, tenantID string) (*sql.Rows, error) {
	// Requires AttachPostgres("pg_core", connStr) first
	query := `
		SELECT
			a.id AS alert_id,
			a.title,
			a.severity AS alert_severity,
			a.mitre_technique,
			t.event_count,
			t.source_types
		FROM pg_core.core.alerts a
		LEFT JOIN (
			SELECT
				rule_name,
				count(*) AS event_count,
				list_distinct(source_type) AS source_types
			FROM read_parquet('s3://kubric-ch-cold/exports/security_events_*.parquet')
			WHERE tenant_id = ?
			GROUP BY rule_name
		) t ON a.title = t.rule_name
		WHERE a.tenant_id = ?::UUID
		  AND a.status = 'open'
		ORDER BY t.event_count DESC NULLS LAST
	`
	return e.db.QueryContext(ctx, query, tenantID, tenantID)
}
```

---

## 3. Python DuckDB (KAI ML Features)

File: `kai/features/duckdb_features.py`

```python
"""
DuckDB-based feature extraction for KAI ML models.
Reads from MinIO Parquet exports and PostgreSQL directly.
"""

import duckdb
import os


def get_connection() -> duckdb.DuckDBPyConnection:
    """Create a DuckDB connection with MinIO and Postgres configured."""
    conn = duckdb.connect(":memory:")

    # Configure S3/MinIO
    conn.execute(f"""
        SET s3_endpoint='10.0.50.23:9000';
        SET s3_access_key_id='kubric-duckdb';
        SET s3_secret_access_key='{os.environ["MINIO_DUCKDB_PASS"]}';
        SET s3_use_ssl=false;
        SET s3_url_style='path';
        SET memory_limit='8GB';
        SET threads=4;
    """)

    # Load extensions
    conn.execute("INSTALL httpfs; LOAD httpfs;")
    conn.execute("INSTALL postgres_scanner; LOAD postgres_scanner;")

    return conn


def compute_network_features(conn: duckdb.DuckDBPyConnection, tenant_id: str, hours: int = 24):
    """
    Compute network behavior features for anomaly detection.
    Returns a Pandas DataFrame with features per source IP.
    """
    query = f"""
        SELECT
            src_ip,
            count(*) AS total_flows,
            count(DISTINCT dst_ip) AS unique_dst_ips,
            count(DISTINCT dst_port) AS unique_dst_ports,
            sum(bytes_in + bytes_out) AS total_bytes,
            avg(duration_ms) AS avg_duration_ms,
            stddev(duration_ms) AS stddev_duration_ms,
            max(bytes_out) AS max_bytes_out,
            count(*) FILTER (WHERE dst_port IN (22,23,3389,5900)) AS remote_access_count,
            count(*) FILTER (WHERE dst_port IN (53,443,80)) AS web_dns_count,
            count(*) FILTER (WHERE dst_port > 49152) AS high_port_count,
            entropy(dst_ip::VARCHAR) AS dst_ip_entropy,
            approx_count_distinct(dst_ip) AS cardinality_dst,
            -- Time features
            count(*) FILTER (WHERE hour(start_time) BETWEEN 0 AND 5) AS nighttime_flows,
            count(*) FILTER (WHERE hour(start_time) BETWEEN 9 AND 17) AS business_hours_flows,
        FROM read_parquet('s3://kubric-ch-cold/exports/network_flows_*.parquet')
        WHERE tenant_id = '{tenant_id}'
          AND start_time >= current_timestamp - INTERVAL {hours} HOUR
        GROUP BY src_ip
        HAVING total_flows > 10
        ORDER BY total_flows DESC
    """
    return conn.execute(query).fetchdf()


def compute_dns_features(conn: duckdb.DuckDBPyConnection, tenant_id: str, hours: int = 24):
    """Extract DNS-based features for DGA/exfiltration detection."""
    query = f"""
        SELECT
            src_ip,
            count(*) AS total_queries,
            count(DISTINCT query) AS unique_domains,
            avg(length(query)) AS avg_domain_length,
            max(length(query)) AS max_domain_length,
            count(*) FILTER (WHERE response_code = 'NXDOMAIN') AS nxdomain_count,
            count(*) FILTER (WHERE query_type = 'TXT') AS txt_query_count,
            count(*) FILTER (WHERE length(query) > 50) AS long_domain_count,
            -- Subdomain depth
            avg(length(query) - length(replace(query, '.', ''))) AS avg_subdomain_depth,
            -- High entropy domains (potential DGA)
            count(*) FILTER (WHERE entropy(split_part(query, '.', 1)) > 3.5) AS high_entropy_count,
        FROM read_parquet('s3://kubric-ch-cold/exports/dns_logs_*.parquet')
        WHERE tenant_id = '{tenant_id}'
          AND event_time >= current_timestamp - INTERVAL {hours} HOUR
        GROUP BY src_ip
        HAVING total_queries > 5
    """
    return conn.execute(query).fetchdf()


def export_training_dataset(conn: duckdb.DuckDBPyConnection, tenant_id: str, output_path: str):
    """
    Export a training dataset combining network + DNS features
    with PostgreSQL alert labels.
    """
    # Attach PostgreSQL for labels
    pg_conn_str = (
        f"host=10.0.50.22 port=5432 dbname=kubric_core "
        f"user=kubric_readonly password={os.environ['PG_READONLY_PASS']}"
    )
    conn.execute(f"ATTACH '{pg_conn_str}' AS pg (TYPE POSTGRES, READ_ONLY)")

    query = f"""
        COPY (
            SELECT
                nf.src_ip,
                nf.total_flows,
                nf.unique_dst_ips,
                nf.unique_dst_ports,
                nf.total_bytes,
                nf.avg_duration_ms,
                nf.high_port_count,
                nf.nighttime_flows,
                dns.total_queries AS dns_queries,
                dns.nxdomain_count,
                dns.high_entropy_count,
                CASE WHEN a.id IS NOT NULL THEN 1 ELSE 0 END AS is_malicious
            FROM (
                SELECT src_ip, count(*) AS total_flows,
                    count(DISTINCT dst_ip) AS unique_dst_ips,
                    count(DISTINCT dst_port) AS unique_dst_ports,
                    sum(bytes_in + bytes_out) AS total_bytes,
                    avg(duration_ms) AS avg_duration_ms,
                    count(*) FILTER (WHERE dst_port > 49152) AS high_port_count,
                    count(*) FILTER (WHERE hour(start_time) BETWEEN 0 AND 5) AS nighttime_flows
                FROM read_parquet('s3://kubric-ch-cold/exports/network_flows_*.parquet')
                WHERE tenant_id = '{tenant_id}'
                GROUP BY src_ip
            ) nf
            LEFT JOIN (
                SELECT src_ip, count(*) AS total_queries,
                    count(*) FILTER (WHERE response_code = 'NXDOMAIN') AS nxdomain_count,
                    count(*) FILTER (WHERE entropy(split_part(query, '.', 1)) > 3.5) AS high_entropy_count
                FROM read_parquet('s3://kubric-ch-cold/exports/dns_logs_*.parquet')
                WHERE tenant_id = '{tenant_id}'
                GROUP BY src_ip
            ) dns ON nf.src_ip = dns.src_ip
            LEFT JOIN pg.core.alerts a ON a.metadata->>'source_ip' = nf.src_ip::VARCHAR
                AND a.severity IN ('critical', 'high')
        ) TO '{output_path}' (FORMAT PARQUET, COMPRESSION ZSTD)
    """
    conn.execute(query)


if __name__ == "__main__":
    conn = get_connection()

    # Quick test
    result = conn.execute("""
        SELECT count(*), min(event_time), max(event_time)
        FROM read_parquet('s3://kubric-ch-cold/exports/security_events_*.parquet')
    """).fetchone()
    print(f"Events: {result[0]}, Range: {result[1]} → {result[2]}")

    conn.close()
```

---

## 4. ClickHouse → MinIO Parquet Export (Scheduled)

```bash
# Export script run daily via cron on ClickHouse node
ssh kubric@10.0.50.21 "tee /opt/kubric/ch-parquet-export.sh" <<'SCRIPT'
#!/bin/bash
set -euo pipefail

DATE=$(date -d yesterday +%Y-%m-%d)
EXPORT_DIR="/tmp/ch-export-${DATE}"
mkdir -p "${EXPORT_DIR}"

# Export security events
clickhouse-client --query "
  SELECT *
  FROM kubric_telemetry.security_events
  WHERE event_date = '${DATE}'
  INTO OUTFILE '${EXPORT_DIR}/security_events_${DATE}.parquet'
  FORMAT Parquet
  SETTINGS output_format_parquet_compression_method='zstd'
"

# Export network flows
clickhouse-client --query "
  SELECT *
  FROM kubric_telemetry.network_flows
  WHERE flow_date = '${DATE}'
  INTO OUTFILE '${EXPORT_DIR}/network_flows_${DATE}.parquet'
  FORMAT Parquet
  SETTINGS output_format_parquet_compression_method='zstd'
"

# Export DNS logs
clickhouse-client --query "
  SELECT *
  FROM kubric_telemetry.dns_logs
  WHERE event_date = '${DATE}'
  INTO OUTFILE '${EXPORT_DIR}/dns_logs_${DATE}.parquet'
  FORMAT Parquet
  SETTINGS output_format_parquet_compression_method='zstd'
"

# Upload to MinIO
mc cp --recursive "${EXPORT_DIR}/" kubric/kubric-ch-cold/exports/

rm -rf "${EXPORT_DIR}"
echo "[$(date)] Export complete for ${DATE}"
SCRIPT

chmod +x /opt/kubric/ch-parquet-export.sh

# Cron: daily at 03:00
echo '0 3 * * * /opt/kubric/ch-parquet-export.sh >> /var/log/kubric/ch-export.log 2>&1' | crontab -
```

---

## 5. Verification

```bash
# Go test
go test -v ./internal/analytics/... -run TestDuckDBEngine

# Python test
python -c "
import duckdb
conn = duckdb.connect(':memory:')
print(conn.execute('SELECT version()').fetchone())
conn.close()
"

# Parquet files exist in MinIO
mc ls kubric/kubric-ch-cold/exports/ | head -10

# DuckDB reads MinIO Parquet
python -c "
import duckdb, os
conn = duckdb.connect(':memory:')
conn.execute(\"SET s3_endpoint='10.0.50.23:9000'\")
conn.execute(\"SET s3_access_key_id='kubric-duckdb'\")
conn.execute(f\"SET s3_secret_access_key='{os.environ[\"MINIO_DUCKDB_PASS\"]}'\")
conn.execute(\"SET s3_use_ssl=false\")
conn.execute(\"SET s3_url_style='path'\")
conn.execute('INSTALL httpfs; LOAD httpfs;')
print(conn.execute(\"SELECT count(*) FROM read_parquet('s3://kubric-ch-cold/exports/security_events_*.parquet')\").fetchone())
"
```
