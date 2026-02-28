//go:build !cgo

package analytics

import (
	"context"
	"fmt"
	"time"
)

var errDuckDBRequiresCGO = fmt.Errorf("duckdb analytics requires cgo-enabled build")

// Engine is a no-op placeholder when cgo is disabled.
type Engine struct{}

// EventSummary is a time-bucketed counts row from the events table.
type EventSummary struct {
	Bucket   time.Time `json:"bucket"`
	Source   string    `json:"source"`
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

// New returns an error when cgo support is unavailable.
func New(dbPath string) (*Engine, error) {
	_ = dbPath
	return nil, errDuckDBRequiresCGO
}

// Close is a no-op when cgo is disabled.
func (e *Engine) Close() error {
	_ = e
	return nil
}

// IngestEvent is unavailable without cgo-backed DuckDB.
func (e *Engine) IngestEvent(ctx context.Context, id, tenantID, source, severity, category, summary, rawJSON string) error {
	_, _, _, _, _, _, _, _ = e, ctx, id, tenantID, source, severity, category, summary
	_ = rawJSON
	return errDuckDBRequiresCGO
}

// IngestComplianceSnapshot is unavailable without cgo-backed DuckDB.
func (e *Engine) IngestComplianceSnapshot(ctx context.Context, tenantID, frameworkID string, pass, fail, total int) error {
	_, _, _, _, _, _, _ = e, ctx, tenantID, frameworkID, pass, fail, total
	return errDuckDBRequiresCGO
}

// IngestMetric is unavailable without cgo-backed DuckDB.
func (e *Engine) IngestMetric(ctx context.Context, tenantID, metricName string, value float64, labels string) error {
	_, _, _, _, _, _ = e, ctx, tenantID, metricName, value, labels
	return errDuckDBRequiresCGO
}

// EventSummaryByHour returns an error when cgo is disabled.
func (e *Engine) EventSummaryByHour(ctx context.Context, tenantID string, since time.Time) ([]EventSummary, error) {
	_, _, _, _ = e, ctx, tenantID, since
	return nil, errDuckDBRequiresCGO
}

// ComplianceTrendDaily returns an error when cgo is disabled.
func (e *Engine) ComplianceTrendDaily(ctx context.Context, tenantID, frameworkID string, since time.Time) ([]ComplianceTrend, error) {
	_, _, _, _, _ = e, ctx, tenantID, frameworkID, since
	return nil, errDuckDBRequiresCGO
}

// RawQuery returns an error when cgo is disabled.
func (e *Engine) RawQuery(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	_, _, _ = e, ctx, query
	_ = args
	return nil, errDuckDBRequiresCGO
}
