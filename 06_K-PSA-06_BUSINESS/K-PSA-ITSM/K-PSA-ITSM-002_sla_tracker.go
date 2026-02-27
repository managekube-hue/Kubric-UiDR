package psa

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	nats "github.com/nats-io/nats.go"
)

// SLABreach describes a ticket that is about to breach or has already breached its SLA.
type SLABreach struct {
	TicketID           string
	TenantID           string
	Priority           string
	CreatedAt          time.Time
	BreachAt           time.Time
	MinutesUntilBreach int
}

// SLAMetrics holds SLA compliance statistics over a period.
type SLAMetrics struct {
	TotalTickets    int
	BreachedTickets int
	CompliancePct   float64
	P1Met           int
	P2Met           int
	P3Met           int
	P4Met           int
}

// SLAReport is a full compliance report for a tenant over a calendar month.
type SLAReport struct {
	Period             string
	Metrics            SLAMetrics
	BreachedTickets    []Ticket
	AvgResolutionHours float64
}

// SLATracker monitors SLA compliance and publishes breach alerts via NATS.
type SLATracker struct {
	pgPool   *pgxpool.Pool
	natsConn *nats.Conn
}

// NewSLATracker creates an SLATracker from environment variables.
// Required env: DATABASE_URL. Optional: NATS_URL (defaults to nats://localhost:4222).
func NewSLATracker() (*SLATracker, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	return &SLATracker{pgPool: pool, natsConn: nc}, nil
}

// CheckBreaches returns tickets that will breach within 30 minutes or have already breached
// and are not in a terminal state.
func (s *SLATracker) CheckBreaches(ctx context.Context) ([]SLABreach, error) {
	rows, err := s.pgPool.Query(ctx, `
		SELECT id, tenant_id, priority, created_at, sla_breach_at,
		       EXTRACT(EPOCH FROM (sla_breach_at - NOW()))::int / 60 AS minutes_until_breach
		FROM service_tickets
		WHERE sla_breach_at < NOW() + INTERVAL '30 minutes'
		  AND state NOT IN ('resolved','closed','cancelled')
		ORDER BY sla_breach_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("check breaches query: %w", err)
	}
	defer rows.Close()

	var breaches []SLABreach
	for rows.Next() {
		var b SLABreach
		if err := rows.Scan(
			&b.TicketID, &b.TenantID, &b.Priority,
			&b.CreatedAt, &b.BreachAt, &b.MinutesUntilBreach,
		); err != nil {
			return nil, fmt.Errorf("scan breach: %w", err)
		}
		breaches = append(breaches, b)
	}
	return breaches, rows.Err()
}

// AlertOnBreach publishes a JSON breach alert to NATS on kubric.{tenantID}.comm.alert.v1.
func (s *SLATracker) AlertOnBreach(ctx context.Context, breach SLABreach) error {
	subject := fmt.Sprintf("kubric.%s.comm.alert.v1", breach.TenantID)
	payload := fmt.Sprintf(
		`{"type":"sla_breach","ticket_id":"%s","priority":"%s","breach_at":"%s","minutes_until_breach":%d}`,
		breach.TicketID, breach.Priority,
		breach.BreachAt.Format(time.RFC3339),
		breach.MinutesUntilBreach,
	)
	if err := s.natsConn.Publish(subject, []byte(payload)); err != nil {
		return fmt.Errorf("nats publish: %w", err)
	}
	log.Printf("sla_tracker: alert published ticket=%s priority=%s minutes=%d",
		breach.TicketID, breach.Priority, breach.MinutesUntilBreach)
	return nil
}

// GetComplianceMetrics returns SLA compliance statistics for a tenant over a duration.
func (s *SLATracker) GetComplianceMetrics(ctx context.Context, tenantID string, period time.Duration) (*SLAMetrics, error) {
	since := time.Now().UTC().Add(-period)
	row := s.pgPool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE resolved_at > sla_breach_at) AS breached,
			COUNT(*) FILTER (WHERE priority = 'P1' AND (resolved_at IS NULL OR resolved_at <= sla_breach_at)) AS p1_met,
			COUNT(*) FILTER (WHERE priority = 'P2' AND (resolved_at IS NULL OR resolved_at <= sla_breach_at)) AS p2_met,
			COUNT(*) FILTER (WHERE priority = 'P3' AND (resolved_at IS NULL OR resolved_at <= sla_breach_at)) AS p3_met,
			COUNT(*) FILTER (WHERE priority = 'P4' AND (resolved_at IS NULL OR resolved_at <= sla_breach_at)) AS p4_met
		FROM service_tickets
		WHERE tenant_id = $1 AND created_at >= $2 AND state IN ('resolved','closed')`,
		tenantID, since)

	var m SLAMetrics
	if err := row.Scan(
		&m.TotalTickets, &m.BreachedTickets,
		&m.P1Met, &m.P2Met, &m.P3Met, &m.P4Met,
	); err != nil {
		return nil, fmt.Errorf("compliance metrics: %w", err)
	}
	if m.TotalTickets > 0 {
		m.CompliancePct = float64(m.TotalTickets-m.BreachedTickets) / float64(m.TotalTickets) * 100
	} else {
		m.CompliancePct = 100
	}
	return &m, nil
}

// GenerateComplianceReport builds a full SLA compliance report for a tenant for the given month.
func (s *SLATracker) GenerateComplianceReport(ctx context.Context, tenantID string, month time.Time) (*SLAReport, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	metrics, err := s.GetComplianceMetrics(ctx, tenantID, end.Sub(start))
	if err != nil {
		return nil, err
	}

	// Fetch detail rows for tickets that breached their SLA.
	rows, err := s.pgPool.Query(ctx, `
		SELECT id, tenant_id, title, description, priority, state,
		       COALESCE(assigned_to::text,''), created_at, updated_at,
		       resolved_at, sla_breach_at, COALESCE(external_ticket_id,'')
		FROM service_tickets
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
		  AND resolved_at > sla_breach_at`,
		tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("breached tickets query: %w", err)
	}
	defer rows.Close()

	var breached []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.Title, &t.Description,
			&t.Priority, &t.State, &t.AssignedTo,
			&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
			&t.SLABreachAt, &t.ExternalTicketID,
		); err != nil {
			return nil, fmt.Errorf("scan breached ticket: %w", err)
		}
		breached = append(breached, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var avgHours float64
	_ = s.pgPool.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))/3600), 0)
		FROM service_tickets
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3 AND resolved_at IS NOT NULL`,
		tenantID, start, end).Scan(&avgHours)

	return &SLAReport{
		Period:             start.Format("2006-01"),
		Metrics:            *metrics,
		BreachedTickets:    breached,
		AvgResolutionHours: avgHours,
	}, nil
}

// RunForever starts the SLA breach check loop, ticking every interval (default 5 min).
func (s *SLATracker) RunForever(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("sla_tracker: starting breach check loop interval=%v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("sla_tracker: shutting down")
			return
		case <-ticker.C:
			breaches, err := s.CheckBreaches(ctx)
			if err != nil {
				log.Printf("sla_tracker: check error: %v", err)
				continue
			}
			for _, b := range breaches {
				if err := s.AlertOnBreach(ctx, b); err != nil {
					log.Printf("sla_tracker: alert error ticket=%s: %v", b.TicketID, err)
				}
			}
			if len(breaches) > 0 {
				log.Printf("sla_tracker: alerted on %d SLA breaches", len(breaches))
			}
		}
	}
}
