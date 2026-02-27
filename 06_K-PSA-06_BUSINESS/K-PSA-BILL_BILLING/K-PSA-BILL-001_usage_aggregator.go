package psa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BillingPeriod represents a monthly billing window.
type BillingPeriod struct {
	Start     time.Time
	End       time.Time
	PeriodKey string // e.g. "2025-01"
}

// UsageSummary holds aggregated usage data for a tenant over a billing period.
type UsageSummary struct {
	TenantID       string
	Period         BillingPeriod
	AgentSeats     float64
	EventsIngested float64
	MLCalls        float64
	StorageGB      float64
	TotalUSD       float64
}

// UsageAggregator aggregates ClickHouse usage into PostgreSQL billing tables
// via the ClickHouse HTTP API endpoint.
type UsageAggregator struct {
	chURL  string // e.g. "http://clickhouse:8123"
	chDB   string // e.g. "kubric"
	pgPool *pgxpool.Pool
	client *http.Client
}

// NewUsageAggregator creates a UsageAggregator from environment variables.
// Required env: CLICKHOUSE_URL, CLICKHOUSE_DB, DATABASE_URL.
func NewUsageAggregator(ctx context.Context) (*UsageAggregator, error) {
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		chURL = "http://localhost:8123"
	}
	chDB := os.Getenv("CLICKHOUSE_DB")
	if chDB == "" {
		chDB = "default"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pgPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	return &UsageAggregator{
		chURL:  strings.TrimRight(chURL, "/"),
		chDB:   chDB,
		pgPool: pgPool,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// chQuery executes a ClickHouse SQL query via the HTTP API and returns the first JSON row.
// The query must produce a single numeric result row with column "result".
func (a *UsageAggregator) chQueryFloat(ctx context.Context, query string) (float64, error) {
	endpoint := fmt.Sprintf("%s/?database=%s&default_format=JSON", a.chURL, url.QueryEscape(a.chDB))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(query))
	if err != nil {
		return 0, fmt.Errorf("ch request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("ch http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("ch read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ch status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("ch json: %w", err)
	}
	if len(result.Data) == 0 {
		return 0, nil
	}

	// The first field of the first row is our scalar result.
	for _, v := range result.Data[0] {
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			// May be a string like "null" or "0"
			var s string
			if err2 := json.Unmarshal(v, &s); err2 == nil {
				if s == "" || s == "null" {
					return 0, nil
				}
				fmt.Sscanf(s, "%f", &f)
			}
		}
		return f, nil
	}
	return 0, nil
}

// AggregateForTenant queries ClickHouse for tenant usage via HTTP API within the given period.
func (a *UsageAggregator) AggregateForTenant(ctx context.Context, tenantID string, p BillingPeriod) (*UsageSummary, error) {
	summary := &UsageSummary{TenantID: tenantID, Period: p}

	startStr := p.Start.UTC().Format("2006-01-02 15:04:05")
	endStr := p.End.UTC().Format("2006-01-02 15:04:05")

	// Agent seats — max concurrent seats in period
	seatsQ := fmt.Sprintf(
		`SELECT max(seat_count) AS result FROM agent_seat_events WHERE tenant_id='%s' AND event_time BETWEEN '%s' AND '%s'`,
		tenantID, startStr, endStr)
	if v, err := a.chQueryFloat(ctx, seatsQ); err == nil {
		summary.AgentSeats = v
	}

	// Events ingested
	eventsQ := fmt.Sprintf(
		`SELECT sum(event_count) AS result FROM events_ingested WHERE tenant_id='%s' AND event_time BETWEEN '%s' AND '%s'`,
		tenantID, startStr, endStr)
	if v, err := a.chQueryFloat(ctx, eventsQ); err == nil {
		summary.EventsIngested = v
	}

	// ML calls
	mlQ := fmt.Sprintf(
		`SELECT sum(call_count) AS result FROM ml_calls WHERE tenant_id='%s' AND event_time BETWEEN '%s' AND '%s'`,
		tenantID, startStr, endStr)
	if v, err := a.chQueryFloat(ctx, mlQ); err == nil {
		summary.MLCalls = v
	}

	// Storage GB — max recorded value in period
	storageQ := fmt.Sprintf(
		`SELECT max(storage_gb) AS result FROM storage_snapshots WHERE tenant_id='%s' AND snapshot_time BETWEEN '%s' AND '%s'`,
		tenantID, startStr, endStr)
	if v, err := a.chQueryFloat(ctx, storageQ); err == nil {
		summary.StorageGB = v
	}

	summary.TotalUSD = a.ComputeCharge(summary)
	return summary, nil
}

// ComputeCharge calculates the total USD charge for a usage summary using HLE pricing.
func (a *UsageAggregator) ComputeCharge(s *UsageSummary) float64 {
	hle := CalculateHLE(int(s.AgentSeats), s.EventsIngested/1_000_000, int(s.MLCalls))
	return MonthlyCharge(hle)
}

// SaveSummary upserts a UsageSummary into the billing_usage_summary table.
func (a *UsageAggregator) SaveSummary(ctx context.Context, s *UsageSummary) error {
	_, err := a.pgPool.Exec(ctx, `
		INSERT INTO billing_usage_summary
			(tenant_id, period_key, period_start, period_end,
			 agent_seats, events_ingested, ml_calls, storage_gb, total_usd, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (tenant_id, period_key) DO UPDATE SET
			agent_seats      = EXCLUDED.agent_seats,
			events_ingested  = EXCLUDED.events_ingested,
			ml_calls         = EXCLUDED.ml_calls,
			storage_gb       = EXCLUDED.storage_gb,
			total_usd        = EXCLUDED.total_usd,
			updated_at       = NOW()`,
		s.TenantID, s.Period.PeriodKey,
		s.Period.Start, s.Period.End,
		s.AgentSeats, s.EventsIngested,
		s.MLCalls, s.StorageGB, s.TotalUSD)
	return err
}

// RunMonthly aggregates the previous month for all tenants.
// Called on the 1st of each month.
func (a *UsageAggregator) RunMonthly(ctx context.Context) error {
	now := time.Now().UTC()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonthEnd := firstOfMonth.Add(-time.Second)
	prevMonthStart := time.Date(prevMonthEnd.Year(), prevMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC)
	period := BillingPeriod{
		Start:     prevMonthStart,
		End:       firstOfMonth,
		PeriodKey: prevMonthStart.Format("2006-01"),
	}

	rows, err := a.pgPool.Query(ctx, `SELECT id FROM tenants WHERE deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			tenantIDs = append(tenantIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan tenants: %w", err)
	}

	var lastErr error
	for _, tid := range tenantIDs {
		summary, err := a.AggregateForTenant(ctx, tid, period)
		if err != nil {
			log.Printf("usage_aggregator: tenant %s aggregate error: %v", tid, err)
			lastErr = err
			continue
		}
		if err := a.SaveSummary(ctx, summary); err != nil {
			log.Printf("usage_aggregator: tenant %s save error: %v", tid, err)
			lastErr = err
		}
	}
	log.Printf("usage_aggregator: monthly run complete for %d tenants period=%s", len(tenantIDs), period.PeriodKey)
	return lastErr
}
