# K-PSA-BI-001 -- QBR Engine (Quarterly Business Review)

**Role:** Automated Quarterly Business Review (QBR) generation engine. Aggregates tenant metrics from ClickHouse, computes KPIs, and produces data-driven review documents.

---

## 1. Architecture

```
┌──────────────┐              ┌──────────────┐              ┌──────────────┐
│  ClickHouse  │─────────────►│  Go Service  │─────────────►│  PDF / HTML  │
│  Telemetry   │  SQL queries │  (PSA/BI)    │  Template    │  QBR Report  │
│              │              │              │  render      │              │
│  PostgreSQL  │─────────────►│  Aggregate   │              │  Email       │
│  Billing     │  SQL queries │  KPIs        │              │  delivery    │
└──────────────┘              │  Score       │              └──────────────┘
                              └──────┬───────┘
                                     │ NATS
                              kubric.psa.qbr.{tenant_id}
```

---

## 2. QBR Data Model

```go
// internal/psa/qbr/engine.go
package qbr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// QBR represents a Quarterly Business Review report.
type QBR struct {
	TenantID    string      `json:"tenant_id"`
	TenantName  string      `json:"tenant_name"`
	Quarter     string      `json:"quarter"` // "2025-Q1"
	GeneratedAt time.Time   `json:"generated_at"`
	Period      DateRange   `json:"period"`
	Executive   Executive   `json:"executive_summary"`
	Security    SecurityKPI `json:"security"`
	Operations  OpsKPI      `json:"operations"`
	Compliance  CompKPI     `json:"compliance"`
	Financial   FinancialKPI `json:"financial"`
	Trends      []TrendPoint `json:"trends"`
	Recommendations []Recommendation `json:"recommendations"`
}

type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Executive struct {
	OverallScore     float64 `json:"overall_score"`     // 0-100
	ScoreChange      float64 `json:"score_change"`      // vs previous quarter
	RiskLevel        string  `json:"risk_level"`        // low, medium, high, critical
	TopAchievement   string  `json:"top_achievement"`
	TopConcern       string  `json:"top_concern"`
	SummaryNarrative string  `json:"summary_narrative"` // AI-generated
}

type SecurityKPI struct {
	TotalAlerts         int64   `json:"total_alerts"`
	CriticalAlerts      int64   `json:"critical_alerts"`
	MTTD                float64 `json:"mttd_minutes"`       // Mean Time to Detect
	MTTR                float64 `json:"mttr_minutes"`       // Mean Time to Respond
	IncidentsClosed     int64   `json:"incidents_closed"`
	IncidentsOpen       int64   `json:"incidents_open"`
	VulnsDiscovered     int64   `json:"vulns_discovered"`
	VulnsRemediated     int64   `json:"vulns_remediated"`
	PatchCompliance     float64 `json:"patch_compliance_pct"`
	MITRECoverage       float64 `json:"mitre_coverage_pct"`
	FalsePositiveRate   float64 `json:"false_positive_rate"`
}

type OpsKPI struct {
	Uptime              float64 `json:"uptime_pct"`
	TicketsCreated      int64   `json:"tickets_created"`
	TicketsResolved     int64   `json:"tickets_resolved"`
	AvgResolutionHours  float64 `json:"avg_resolution_hours"`
	BackupSuccessRate   float64 `json:"backup_success_rate"`
	EndpointsManaged    int64   `json:"endpoints_managed"`
	EndpointsCompliant  int64   `json:"endpoints_compliant"`
}

type CompKPI struct {
	FrameworksTracked   int     `json:"frameworks_tracked"`
	OverallCompliance   float64 `json:"overall_compliance_pct"`
	ControlsPassed      int     `json:"controls_passed"`
	ControlsFailed      int     `json:"controls_failed"`
	AuditFindings       int     `json:"audit_findings"`
	RiskScore           float64 `json:"risk_score"` // 0-100
}

type FinancialKPI struct {
	MRR                 float64 `json:"mrr"`
	TotalBilled         float64 `json:"total_billed"`
	CostPerEndpoint     float64 `json:"cost_per_endpoint"`
	CostPerAlert        float64 `json:"cost_per_alert"`
	ServiceHoursUsed    float64 `json:"service_hours_used"`
	ContractUtilization float64 `json:"contract_utilization_pct"`
}

type TrendPoint struct {
	Month string  `json:"month"` // "2025-01"
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

type Recommendation struct {
	Priority    int    `json:"priority"` // 1 = highest
	Category    string `json:"category"` // security, operations, compliance
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"` // low, medium, high
}
```

---

## 3. KPI Aggregation Queries

```go
// internal/psa/qbr/aggregator.go
package qbr

import (
	"context"
	"database/sql"
	"time"
)

type Aggregator struct {
	clickhouse *sql.DB
	postgres   *sql.DB
}

func NewAggregator(ch *sql.DB, pg *sql.DB) *Aggregator {
	return &Aggregator{clickhouse: ch, postgres: pg}
}

// AggregateSecurityKPIs pulls security metrics from ClickHouse.
func (a *Aggregator) AggregateSecurityKPIs(
	ctx context.Context,
	tenantID string,
	period DateRange,
) (*SecurityKPI, error) {
	kpi := &SecurityKPI{}

	// Total and critical alerts
	row := a.clickhouse.QueryRowContext(ctx, `
		SELECT
			count() AS total_alerts,
			countIf(severity_id >= 4) AS critical_alerts,
			avg(time_to_detect_ms) / 60000 AS mttd_minutes,
			avg(time_to_respond_ms) / 60000 AS mttr_minutes,
			countIf(status = 'closed') AS incidents_closed,
			countIf(status = 'open') AS incidents_open
		FROM kubric.security_events
		WHERE tenant_id = $1
		  AND event_time BETWEEN $2 AND $3
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.TotalAlerts, &kpi.CriticalAlerts,
		&kpi.MTTD, &kpi.MTTR,
		&kpi.IncidentsClosed, &kpi.IncidentsOpen)

	// Vulnerability metrics
	row = a.clickhouse.QueryRowContext(ctx, `
		SELECT
			countIf(activity = 'discovered') AS vulns_discovered,
			countIf(activity = 'remediated') AS vulns_remediated
		FROM kubric.vulnerability_findings
		WHERE tenant_id = $1
		  AND event_time BETWEEN $2 AND $3
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.VulnsDiscovered, &kpi.VulnsRemediated)

	// Patch compliance
	row = a.clickhouse.QueryRowContext(ctx, `
		SELECT
			countIf(patch_status = 'current') * 100.0 / count() AS patch_compliance
		FROM kubric.endpoint_inventory
		WHERE tenant_id = $1
	`, tenantID)

	row.Scan(&kpi.PatchCompliance)

	// False positive rate
	row = a.clickhouse.QueryRowContext(ctx, `
		SELECT
			countIf(disposition = 'false_positive') * 100.0 /
			greatest(count(), 1) AS fp_rate
		FROM kubric.security_events
		WHERE tenant_id = $1
		  AND event_time BETWEEN $2 AND $3
		  AND disposition != ''
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.FalsePositiveRate)

	return kpi, nil
}

// AggregateOpsKPIs pulls operations metrics.
func (a *Aggregator) AggregateOpsKPIs(
	ctx context.Context,
	tenantID string,
	period DateRange,
) (*OpsKPI, error) {
	kpi := &OpsKPI{}

	row := a.clickhouse.QueryRowContext(ctx, `
		SELECT
			avg(uptime_pct) AS uptime,
			count() AS endpoints_managed,
			countIf(compliance_status = 'compliant') AS endpoints_compliant
		FROM kubric.endpoint_inventory
		WHERE tenant_id = $1
	`, tenantID)

	row.Scan(&kpi.Uptime, &kpi.EndpointsManaged, &kpi.EndpointsCompliant)

	// Tickets from PostgreSQL
	row = a.postgres.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at BETWEEN $2 AND $3) AS tickets_created,
			COUNT(*) FILTER (WHERE resolved_at BETWEEN $2 AND $3) AS tickets_resolved,
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600)
				FILTER (WHERE resolved_at BETWEEN $2 AND $3), 0) AS avg_resolution_hours
		FROM tickets
		WHERE tenant_id = $1
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.TicketsCreated, &kpi.TicketsResolved, &kpi.AvgResolutionHours)

	// Backup success rate
	row = a.clickhouse.QueryRowContext(ctx, `
		SELECT
			countIf(status = 'success') * 100.0 / greatest(count(), 1)
		FROM kubric.backup_jobs
		WHERE tenant_id = $1
		  AND start_time BETWEEN $2 AND $3
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.BackupSuccessRate)

	return kpi, nil
}

// AggregateFinancialKPIs pulls billing metrics from PostgreSQL.
func (a *Aggregator) AggregateFinancialKPIs(
	ctx context.Context,
	tenantID string,
	period DateRange,
) (*FinancialKPI, error) {
	kpi := &FinancialKPI{}

	row := a.postgres.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount), 0) AS total_billed,
			COALESCE(AVG(amount) FILTER (WHERE period_type = 'monthly'), 0) AS mrr
		FROM invoices
		WHERE tenant_id = $1
		  AND invoice_date BETWEEN $2 AND $3
	`, tenantID, period.Start, period.End)

	row.Scan(&kpi.TotalBilled, &kpi.MRR)

	// Cost per endpoint
	row = a.postgres.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM endpoints WHERE tenant_id = $1 AND active = true
	`, tenantID)

	var endpointCount int64
	row.Scan(&endpointCount)
	if endpointCount > 0 {
		kpi.CostPerEndpoint = kpi.TotalBilled / float64(endpointCount)
	}

	return kpi, nil
}
```

---

## 4. QBR Generation Engine

```go
// internal/psa/qbr/generator.go
package qbr

import (
	"context"
	"fmt"
	"math"
	"time"
)

type Generator struct {
	aggregator *Aggregator
}

func NewGenerator(agg *Aggregator) *Generator {
	return &Generator{aggregator: agg}
}

// Generate produces a complete QBR for a tenant.
func (g *Generator) Generate(
	ctx context.Context,
	tenantID, tenantName string,
	quarter string, // "2025-Q1"
) (*QBR, error) {
	period := quarterToDateRange(quarter)

	security, err := g.aggregator.AggregateSecurityKPIs(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("security kpis: %w", err)
	}

	ops, err := g.aggregator.AggregateOpsKPIs(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("ops kpis: %w", err)
	}

	financial, err := g.aggregator.AggregateFinancialKPIs(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("financial kpis: %w", err)
	}

	score := computeOverallScore(security, ops)
	recommendations := generateRecommendations(security, ops)

	return &QBR{
		TenantID:    tenantID,
		TenantName:  tenantName,
		Quarter:     quarter,
		GeneratedAt: time.Now().UTC(),
		Period:      period,
		Executive: Executive{
			OverallScore: score,
			RiskLevel:    scoreToRiskLevel(score),
		},
		Security:        *security,
		Operations:      *ops,
		Financial:       *financial,
		Recommendations: recommendations,
	}, nil
}

func computeOverallScore(sec *SecurityKPI, ops *OpsKPI) float64 {
	// Weighted scoring: Security 40%, Operations 30%, Compliance 30%
	secScore := 100.0
	if sec.MTTR > 0 {
		secScore -= math.Min(sec.MTTR/10, 30) // Penalize slow response
	}
	secScore -= sec.FalsePositiveRate * 0.5
	if sec.PatchCompliance < 100 {
		secScore -= (100 - sec.PatchCompliance) * 0.3
	}

	opsScore := ops.Uptime
	if ops.BackupSuccessRate < 100 {
		opsScore -= (100 - ops.BackupSuccessRate) * 0.5
	}

	return math.Max(0, math.Min(100,
		secScore*0.4 + opsScore*0.6))
}

func scoreToRiskLevel(score float64) string {
	switch {
	case score >= 90:
		return "low"
	case score >= 70:
		return "medium"
	case score >= 50:
		return "high"
	default:
		return "critical"
	}
}

func generateRecommendations(sec *SecurityKPI, ops *OpsKPI) []Recommendation {
	var recs []Recommendation
	priority := 1

	if sec.PatchCompliance < 90 {
		recs = append(recs, Recommendation{
			Priority:    priority,
			Category:    "security",
			Title:       "Improve Patch Compliance",
			Description: fmt.Sprintf("Patch compliance at %.1f%%. Target 95%%+.", sec.PatchCompliance),
			Impact:      "Reduces exploitable attack surface",
			Effort:      "medium",
		})
		priority++
	}

	if sec.MTTR > 60 {
		recs = append(recs, Recommendation{
			Priority:    priority,
			Category:    "security",
			Title:       "Reduce Mean Time to Respond",
			Description: fmt.Sprintf("MTTR at %.0f minutes. Target <30 minutes.", sec.MTTR),
			Impact:      "Limits blast radius of incidents",
			Effort:      "medium",
		})
		priority++
	}

	if ops.BackupSuccessRate < 99 {
		recs = append(recs, Recommendation{
			Priority:    priority,
			Category:    "operations",
			Title:       "Improve Backup Reliability",
			Description: fmt.Sprintf("Backup success rate at %.1f%%. Target 99.9%%+.", ops.BackupSuccessRate),
			Impact:      "Ensures data recovery capability",
			Effort:      "low",
		})
		priority++
	}

	return recs
}

func quarterToDateRange(quarter string) DateRange {
	var year int
	var q int
	fmt.Sscanf(quarter, "%d-Q%d", &year, &q)

	startMonth := time.Month((q-1)*3 + 1)
	start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 3, 0).Add(-time.Nanosecond)

	return DateRange{Start: start, End: end}
}
```
