# K-PSA-BI-002 -- Profitability Analysis Engine

**Role:** Per-tenant and per-service profitability calculation. Computes margins, identifies cost drivers, and feeds data into QBR reports and executive dashboards.

---

## 1. Data Model

```go
// internal/psa/billing/profitability.go
package billing

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ProfitabilityReport contains margin analysis for a tenant.
type ProfitabilityReport struct {
	TenantID       string             `json:"tenant_id"`
	TenantName     string             `json:"tenant_name"`
	Period         string             `json:"period"` // "2025-Q1" or "2025-01"
	GeneratedAt    time.Time          `json:"generated_at"`
	Revenue        RevenueBreakdown   `json:"revenue"`
	Costs          CostBreakdown      `json:"costs"`
	Margins        MarginAnalysis     `json:"margins"`
	ServiceLines   []ServiceLineP     `json:"service_lines"`
	TrendMonths    []MonthlyMargin    `json:"trend_months"`
	CostDrivers    []CostDriver       `json:"cost_drivers"`
}

type RevenueBreakdown struct {
	MRR               float64 `json:"mrr"`
	OneTimeRevenue    float64 `json:"one_time_revenue"`
	ProjectRevenue    float64 `json:"project_revenue"`
	TotalRevenue      float64 `json:"total_revenue"`
	PerEndpointMRR    float64 `json:"per_endpoint_mrr"`
	EndpointCount     int64   `json:"endpoint_count"`
}

type CostBreakdown struct {
	LaborCost         float64 `json:"labor_cost"`
	ToolingCost       float64 `json:"tooling_cost"`
	InfrastructureCost float64 `json:"infrastructure_cost"`
	LicensingCost     float64 `json:"licensing_cost"`
	OverheadAllocation float64 `json:"overhead_allocation"`
	TotalCost         float64 `json:"total_cost"`
	PerEndpointCost   float64 `json:"per_endpoint_cost"`
}

type MarginAnalysis struct {
	GrossMargin       float64 `json:"gross_margin"`       // (Revenue - COGS) / Revenue
	GrossMarginPct    float64 `json:"gross_margin_pct"`
	NetMargin         float64 `json:"net_margin"`         // After overhead
	NetMarginPct      float64 `json:"net_margin_pct"`
	ContributionMargin float64 `json:"contribution_margin"`
	BreakevenEndpoints int64  `json:"breakeven_endpoints"`
}

type ServiceLineP struct {
	Name          string  `json:"name"` // SOC, NOC, vCISO, etc.
	Revenue       float64 `json:"revenue"`
	Cost          float64 `json:"cost"`
	MarginPct     float64 `json:"margin_pct"`
	HoursSpent    float64 `json:"hours_spent"`
	EffectiveRate float64 `json:"effective_rate"` // Revenue / Hours
}

type MonthlyMargin struct {
	Month     string  `json:"month"` // "2025-01"
	Revenue   float64 `json:"revenue"`
	Cost      float64 `json:"cost"`
	MarginPct float64 `json:"margin_pct"`
}

type CostDriver struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"` // of total cost
	Trend       string  `json:"trend"`      // increasing, stable, decreasing
}
```

---

## 2. PostgreSQL Schema

```sql
-- migrations/014_profitability.sql
CREATE TABLE IF NOT EXISTS service_costs (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    period       DATE NOT NULL,         -- first day of month
    category     TEXT NOT NULL,          -- labor, tooling, infra, licensing, overhead
    service_line TEXT NOT NULL,          -- SOC, NOC, vCISO, MDM, Backup
    amount       DECIMAL(12,2) NOT NULL,
    description  TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_service_costs_tenant_period 
    ON service_costs(tenant_id, period);

CREATE TABLE IF NOT EXISTS time_entries (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    technician_id UUID NOT NULL,
    service_line TEXT NOT NULL,
    date         DATE NOT NULL,
    hours        DECIMAL(5,2) NOT NULL,
    hourly_rate  DECIMAL(8,2) NOT NULL,  -- internal cost rate
    billable     BOOLEAN DEFAULT true,
    ticket_id    UUID,
    description  TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_time_entries_tenant ON time_entries(tenant_id, date);

-- Materialized view for monthly profitability
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_monthly_profitability AS
SELECT
    t.id AS tenant_id,
    t.name AS tenant_name,
    DATE_TRUNC('month', i.invoice_date)::DATE AS month,
    COALESCE(SUM(i.amount), 0) AS revenue,
    COALESCE(SUM(sc.total_cost), 0) AS cost,
    CASE
        WHEN COALESCE(SUM(i.amount), 0) > 0
        THEN (SUM(i.amount) - COALESCE(SUM(sc.total_cost), 0)) * 100.0 / SUM(i.amount)
        ELSE 0
    END AS margin_pct
FROM tenants t
LEFT JOIN invoices i ON i.tenant_id = t.id
LEFT JOIN (
    SELECT tenant_id, period, SUM(amount) AS total_cost
    FROM service_costs
    GROUP BY tenant_id, period
) sc ON sc.tenant_id = t.id AND sc.period = DATE_TRUNC('month', i.invoice_date)::DATE
GROUP BY t.id, t.name, DATE_TRUNC('month', i.invoice_date)::DATE;

CREATE UNIQUE INDEX idx_mv_profitability ON mv_monthly_profitability(tenant_id, month);
```

---

## 3. Profitability Calculator

```go
// internal/psa/billing/profitability_calc.go
package billing

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ProfitabilityCalculator struct {
	db *sql.DB // PostgreSQL
}

func NewProfitabilityCalculator(db *sql.DB) *ProfitabilityCalculator {
	return &ProfitabilityCalculator{db: db}
}

// Calculate computes the full profitability report for a tenant.
func (pc *ProfitabilityCalculator) Calculate(
	ctx context.Context,
	tenantID, tenantName string,
	periodStart, periodEnd time.Time,
) (*ProfitabilityReport, error) {
	report := &ProfitabilityReport{
		TenantID:    tenantID,
		TenantName:  tenantName,
		GeneratedAt: time.Now().UTC(),
	}

	// Revenue
	rev, err := pc.calculateRevenue(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("revenue: %w", err)
	}
	report.Revenue = *rev

	// Costs
	costs, err := pc.calculateCosts(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("costs: %w", err)
	}
	report.Costs = *costs

	// Margins
	report.Margins = computeMargins(rev, costs)

	// Service line breakdown
	lines, err := pc.calculateServiceLines(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("service lines: %w", err)
	}
	report.ServiceLines = lines

	// Monthly trend
	trend, err := pc.getMonthlTrend(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("trend: %w", err)
	}
	report.TrendMonths = trend

	return report, nil
}

func (pc *ProfitabilityCalculator) calculateRevenue(
	ctx context.Context,
	tenantID string,
	start, end time.Time,
) (*RevenueBreakdown, error) {
	rev := &RevenueBreakdown{}

	row := pc.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE invoice_type = 'recurring'), 0) AS mrr_total,
			COALESCE(SUM(amount) FILTER (WHERE invoice_type = 'one_time'), 0) AS one_time,
			COALESCE(SUM(amount) FILTER (WHERE invoice_type = 'project'), 0) AS project,
			COALESCE(SUM(amount), 0) AS total
		FROM invoices
		WHERE tenant_id = $1
		  AND invoice_date BETWEEN $2 AND $3
		  AND status = 'paid'
	`, tenantID, start, end)

	var mrrTotal float64
	if err := row.Scan(&mrrTotal, &rev.OneTimeRevenue,
		&rev.ProjectRevenue, &rev.TotalRevenue); err != nil {
		return nil, err
	}

	months := end.Sub(start).Hours() / (24 * 30)
	if months > 0 {
		rev.MRR = mrrTotal / months
	}

	// Endpoint count
	row = pc.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM endpoints
		WHERE tenant_id = $1 AND active = true
	`, tenantID)
	row.Scan(&rev.EndpointCount)

	if rev.EndpointCount > 0 {
		rev.PerEndpointMRR = rev.MRR / float64(rev.EndpointCount)
	}

	return rev, nil
}

func (pc *ProfitabilityCalculator) calculateCosts(
	ctx context.Context,
	tenantID string,
	start, end time.Time,
) (*CostBreakdown, error) {
	costs := &CostBreakdown{}

	row := pc.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE category = 'labor'), 0),
			COALESCE(SUM(amount) FILTER (WHERE category = 'tooling'), 0),
			COALESCE(SUM(amount) FILTER (WHERE category = 'infrastructure'), 0),
			COALESCE(SUM(amount) FILTER (WHERE category = 'licensing'), 0),
			COALESCE(SUM(amount) FILTER (WHERE category = 'overhead'), 0),
			COALESCE(SUM(amount), 0)
		FROM service_costs
		WHERE tenant_id = $1
		  AND period BETWEEN $2 AND $3
	`, tenantID, start, end)

	if err := row.Scan(&costs.LaborCost, &costs.ToolingCost,
		&costs.InfrastructureCost, &costs.LicensingCost,
		&costs.OverheadAllocation, &costs.TotalCost); err != nil {
		return nil, err
	}

	// Add labor from time entries
	row = pc.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(hours * hourly_rate), 0)
		FROM time_entries
		WHERE tenant_id = $1
		  AND date BETWEEN $2 AND $3
	`, tenantID, start, end)

	var laborFromTime float64
	row.Scan(&laborFromTime)
	costs.LaborCost += laborFromTime
	costs.TotalCost += laborFromTime

	return costs, nil
}

func computeMargins(rev *RevenueBreakdown, costs *CostBreakdown) MarginAnalysis {
	m := MarginAnalysis{}
	if rev.TotalRevenue > 0 {
		m.GrossMargin = rev.TotalRevenue - (costs.TotalCost - costs.OverheadAllocation)
		m.GrossMarginPct = m.GrossMargin * 100.0 / rev.TotalRevenue
		m.NetMargin = rev.TotalRevenue - costs.TotalCost
		m.NetMarginPct = m.NetMargin * 100.0 / rev.TotalRevenue
		m.ContributionMargin = rev.TotalRevenue - costs.LaborCost - costs.ToolingCost
	}

	// Breakeven: how many endpoints needed at current MRR/endpoint
	if rev.PerEndpointMRR > 0 && costs.PerEndpointCost > 0 {
		monthlyFixed := costs.OverheadAllocation + costs.InfrastructureCost + costs.LicensingCost
		m.BreakevenEndpoints = int64(monthlyFixed / (rev.PerEndpointMRR - costs.PerEndpointCost))
	}

	return m
}

func (pc *ProfitabilityCalculator) calculateServiceLines(
	ctx context.Context,
	tenantID string,
	start, end time.Time,
) ([]ServiceLineP, error) {
	rows, err := pc.db.QueryContext(ctx, `
		SELECT
			sc.service_line,
			COALESCE(SUM(sc.amount), 0) AS cost,
			COALESCE(SUM(te.hours), 0) AS hours
		FROM service_costs sc
		LEFT JOIN time_entries te ON te.tenant_id = sc.tenant_id
			AND te.service_line = sc.service_line
			AND te.date BETWEEN $2 AND $3
		WHERE sc.tenant_id = $1
		  AND sc.period BETWEEN $2 AND $3
		GROUP BY sc.service_line
	`, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []ServiceLineP
	for rows.Next() {
		var sl ServiceLineP
		if err := rows.Scan(&sl.Name, &sl.Cost, &sl.HoursSpent); err != nil {
			continue
		}
		if sl.HoursSpent > 0 {
			sl.EffectiveRate = sl.Revenue / sl.HoursSpent
		}
		lines = append(lines, sl)
	}
	return lines, nil
}

func (pc *ProfitabilityCalculator) getMonthlTrend(
	ctx context.Context,
	tenantID string,
	start, end time.Time,
) ([]MonthlyMargin, error) {
	rows, err := pc.db.QueryContext(ctx, `
		SELECT month::TEXT, revenue, cost, margin_pct
		FROM mv_monthly_profitability
		WHERE tenant_id = $1
		  AND month BETWEEN $2 AND $3
		ORDER BY month
	`, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []MonthlyMargin
	for rows.Next() {
		var m MonthlyMargin
		rows.Scan(&m.Month, &m.Revenue, &m.Cost, &m.MarginPct)
		trend = append(trend, m)
	}
	return trend, nil
}
```

---

## 4. Healthy Margin Targets

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| Gross Margin | > 60% | 45-60% | < 45% |
| Net Margin | > 30% | 15-30% | < 15% |
| Per-Endpoint MRR | > $15 | $8-15 | < $8 |
| Effective Hourly Rate | > $150 | $100-150 | < $100 |
| Backup Cost Ratio | < 5% of MRR | 5-10% | > 10% |
