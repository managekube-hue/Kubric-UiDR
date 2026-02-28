// Package analytics provides an embedded DuckDB analytical engine for Kubric.
// DuckDB runs in-process (no external server) and excels at OLAP-style queries
// over event logs, metric summaries, and compliance timeseries — exactly the
// workloads that are too expensive for row-oriented Postgres.
//
// Depends on: github.com/marcboeker/go-duckdb
package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/marcboeker/go-duckdb" // register DuckDB SQL driver
)

// Engine wraps a DuckDB database/sql connection.
type Engine struct {
	db *sql.DB
}

// EventSummary is a time-bucketed counts row from the events table.
type EventSummary struct {
	Bucket   time.Time `json:"bucket"`
	Source   string    `json:"source"` // noc, soc, grc, kai
	Severity string    `json:"severity"`
	Count    int64     `json:"count"`
}

// ComplianceTrend tracks framework pass-rate over time.
type ComplianceTrend struct {
	Bucket      time.Time `json:"bucket"`
	FrameworkID string    `json:"framework_id"`
	PassRate    float64   `json:"pass_rate"`
	TotalChecks int64     `json:"total_checks"`
}

// New opens a DuckDB database at the given path.
// Use ":memory:" for an ephemeral in-memory database or a file path for
// persistence (e.g. "/data/kubric-analytics.duckdb").
func New(dbPath string) (*Engine, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("duckdb open %s: %w", dbPath, err)
	}
	// Verify connection
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("duckdb ping: %w", err)
	}
	e := &Engine{db: db}
	if err := e.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("duckdb init: %w", err)
	}
	return e, nil
}

// Close releases the DuckDB connection.
func (e *Engine) Close() error {
	if e == nil || e.db == nil {
		return nil
	}
	return e.db.Close()
}

// initSchema creates analytical tables if they don't exist.
func (e *Engine) initSchema() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id          VARCHAR,
			tenant_id   VARCHAR,
			source      VARCHAR,
			severity    VARCHAR,
			category    VARCHAR,
			summary     VARCHAR,
			raw_json    VARCHAR,
			created_at  TIMESTAMP DEFAULT current_timestamp
		)`,
		`CREATE TABLE IF NOT EXISTS compliance_snapshots (
			tenant_id     VARCHAR,
			framework_id  VARCHAR,
			pass_count    INTEGER,
			fail_count    INTEGER,
			total_checks  INTEGER,
			snapshot_at   TIMESTAMP DEFAULT current_timestamp
		)`,
		`CREATE TABLE IF NOT EXISTS metric_samples (
			tenant_id   VARCHAR,
			metric_name VARCHAR,
			value       DOUBLE,
			labels      VARCHAR,
			sampled_at  TIMESTAMP DEFAULT current_timestamp
		)`,
	}
	for _, stmt := range ddl {
		if _, err := e.db.Exec(stmt); err != nil {
			return fmt.Errorf("duckdb DDL: %w", err)
		}
	}
	return nil
}

// IngestEvent appends an event row for analytical queries.
func (e *Engine) IngestEvent(ctx context.Context, id, tenantID, source, severity, category, summary, rawJSON string) error {
	_, err := e.db.ExecContext(ctx,
		`INSERT INTO events (id, tenant_id, source, severity, category, summary, raw_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, source, severity, category, summary, rawJSON)
	return err
}

// IngestComplianceSnapshot records a point-in-time compliance measurement.
func (e *Engine) IngestComplianceSnapshot(ctx context.Context, tenantID, frameworkID string, pass, fail, total int) error {
	_, err := e.db.ExecContext(ctx,
		`INSERT INTO compliance_snapshots (tenant_id, framework_id, pass_count, fail_count, total_checks)
		 VALUES (?, ?, ?, ?, ?)`,
		tenantID, frameworkID, pass, fail, total)
	return err
}

// IngestMetric records a timestamped metric sample.
func (e *Engine) IngestMetric(ctx context.Context, tenantID, metricName string, value float64, labels string) error {
	_, err := e.db.ExecContext(ctx,
		`INSERT INTO metric_samples (tenant_id, metric_name, value, labels)
		 VALUES (?, ?, ?, ?)`,
		tenantID, metricName, value, labels)
	return err
}

// EventSummaryByHour returns hourly event counts grouped by source+severity.
func (e *Engine) EventSummaryByHour(ctx context.Context, tenantID string, since time.Time) ([]EventSummary, error) {
	rows, err := e.db.QueryContext(ctx,
		`SELECT date_trunc('hour', created_at) AS bucket,
		        source, severity, COUNT(*) AS cnt
		 FROM events
		 WHERE tenant_id = ? AND created_at >= ?
		 GROUP BY bucket, source, severity
		 ORDER BY bucket DESC`,
		tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("duckdb event summary: %w", err)
	}
	defer rows.Close()
	var out []EventSummary
	for rows.Next() {
		var s EventSummary
		if err := rows.Scan(&s.Bucket, &s.Source, &s.Severity, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ComplianceTrendDaily returns daily compliance pass-rate trend for a framework.
func (e *Engine) ComplianceTrendDaily(ctx context.Context, tenantID, frameworkID string, since time.Time) ([]ComplianceTrend, error) {
	rows, err := e.db.QueryContext(ctx,
		`SELECT date_trunc('day', snapshot_at) AS bucket,
		        framework_id,
		        AVG(CAST(pass_count AS DOUBLE) / NULLIF(total_checks, 0)) AS pass_rate,
		        SUM(total_checks) AS total_checks
		 FROM compliance_snapshots
		 WHERE tenant_id = ? AND framework_id = ? AND snapshot_at >= ?
		 GROUP BY bucket, framework_id
		 ORDER BY bucket DESC`,
		tenantID, frameworkID, since)
	if err != nil {
		return nil, fmt.Errorf("duckdb compliance trend: %w", err)
	}
	defer rows.Close()
	var out []ComplianceTrend
	for rows.Next() {
		var t ComplianceTrend
		if err := rows.Scan(&t.Bucket, &t.FrameworkID, &t.PassRate, &t.TotalChecks); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RawQuery executes an arbitrary read-only analytics SQL and returns JSON-like maps.
// Guarded: callers should validate the query is SELECT-only before calling.
func (e *Engine) RawQuery(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdb raw query: %w", err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
