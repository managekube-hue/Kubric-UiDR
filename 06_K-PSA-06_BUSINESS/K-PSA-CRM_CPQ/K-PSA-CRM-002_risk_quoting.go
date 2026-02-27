package psa

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuoteRequest holds the parameters for generating a risk-based quote.
type QuoteRequest struct {
	TenantID             string
	AssetCount           int
	HighRiskAssets       int
	Industry             string
	ComplianceFrameworks []string
}

// Quote is a generated price quote stored in PostgreSQL.
type Quote struct {
	QuoteID         string
	TenantID        string
	BasePrice       float64
	RiskMultiplier  float64
	ComplianceAddon float64
	TotalMonthly    float64
	Breakdown       map[string]float64
	CreatedAt       time.Time
}

// RiskQuoter generates quotes based on risk assessment scores and asset profiles.
type RiskQuoter struct {
	pgPool *pgxpool.Pool
}

// NewRiskQuoter creates a RiskQuoter from the DATABASE_URL environment variable.
func NewRiskQuoter() (*RiskQuoter, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	return &RiskQuoter{pgPool: pool}, nil
}

// IndustryMultiplier returns the pricing multiplier for a given industry vertical.
func (r *RiskQuoter) IndustryMultiplier(industry string) float64 {
	switch industry {
	case "healthcare":
		return 1.4
	case "finance", "banking", "insurance":
		return 1.3
	case "government", "federal", "defense":
		return 1.2
	default:
		return 1.0
	}
}

// ComplianceMultiplier returns an additive multiplier based on compliance framework count.
// Each framework adds 0.1, capped at 0.4 (i.e., 1.0 + min(n*0.1, 0.4)).
func (r *RiskQuoter) ComplianceMultiplier(frameworks []string) float64 {
	addon := float64(len(frameworks)) * 0.1
	if addon > 0.4 {
		addon = 0.4
	}
	return 1.0 + addon
}

// BaseMonthlyUSD returns the base monthly price for an asset count using tiered pricing.
// < 50 assets: $500 | < 200: $1,500 | < 1,000: $4,000 | >= 1,000: $8,000
func (r *RiskQuoter) BaseMonthlyUSD(assetCount int) float64 {
	switch {
	case assetCount < 50:
		return 500
	case assetCount < 200:
		return 1500
	case assetCount < 1000:
		return 4000
	default:
		return 8000
	}
}

// GenerateQuote creates a risk-adjusted Quote for the given request and saves it.
func (r *RiskQuoter) GenerateQuote(ctx context.Context, req QuoteRequest) (*Quote, error) {
	base := r.BaseMonthlyUSD(req.AssetCount)
	industryMult := r.IndustryMultiplier(req.Industry)
	complianceMult := r.ComplianceMultiplier(req.ComplianceFrameworks)

	// High-risk asset surcharge: $5/month per high-risk asset, capped at 50% of base.
	highRiskSurcharge := math.Min(float64(req.HighRiskAssets)*5, base*0.5)

	riskAdjustedBase := base * industryMult
	complianceAddon := riskAdjustedBase * (complianceMult - 1)
	total := riskAdjustedBase + complianceAddon + highRiskSurcharge

	now := time.Now().UTC()
	quote := &Quote{
		QuoteID:         uuid.NewString(),
		TenantID:        req.TenantID,
		BasePrice:       base,
		RiskMultiplier:  industryMult,
		ComplianceAddon: complianceAddon,
		TotalMonthly:    math.Round(total*100) / 100,
		CreatedAt:       now,
		Breakdown: map[string]float64{
			"base_price":          base,
			"industry_multiplier": industryMult,
			"risk_adjusted_base":  riskAdjustedBase,
			"compliance_addon":    complianceAddon,
			"high_risk_surcharge": highRiskSurcharge,
			"total_monthly":       math.Round(total*100) / 100,
		},
	}

	if err := r.SaveQuote(ctx, quote); err != nil {
		return nil, fmt.Errorf("save quote: %w", err)
	}
	return quote, nil
}

// SaveQuote persists a Quote to the risk_quotes table in PostgreSQL.
func (r *RiskQuoter) SaveQuote(ctx context.Context, quote *Quote) error {
	_, err := r.pgPool.Exec(ctx, `
		INSERT INTO risk_quotes
			(id, tenant_id, base_price, risk_multiplier, compliance_addon,
			 total_monthly, breakdown, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8)
		ON CONFLICT (id) DO NOTHING`,
		quote.QuoteID, quote.TenantID,
		quote.BasePrice, quote.RiskMultiplier,
		quote.ComplianceAddon, quote.TotalMonthly,
		breakdownToJSON(quote.Breakdown),
		quote.CreatedAt)
	return err
}

// GetQuoteHistory returns all quotes for a tenant ordered by creation date descending.
func (r *RiskQuoter) GetQuoteHistory(ctx context.Context, tenantID string) ([]Quote, error) {
	rows, err := r.pgPool.Query(ctx, `
		SELECT id, tenant_id, base_price, risk_multiplier, compliance_addon,
		       total_monthly, created_at
		FROM risk_quotes
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query quotes: %w", err)
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(
			&q.QuoteID, &q.TenantID, &q.BasePrice,
			&q.RiskMultiplier, &q.ComplianceAddon, &q.TotalMonthly,
			&q.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan quote: %w", err)
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

// breakdownToJSON converts a float64 map to a minimal JSON object string.
func breakdownToJSON(m map[string]float64) string {
	b := "{"
	i := 0
	for k, v := range m {
		if i > 0 {
			b += ","
		}
		b += fmt.Sprintf(`"%s":%g`, k, v)
		i++
	}
	return b + "}"
}
