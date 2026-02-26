package kic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Assessment is a single control evaluation result from OSCAL, Lula, kube-bench, or OpenSCAP.
type Assessment struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Framework    string    `json:"framework"`     // NIST-800-53 | CIS-K8s-1.8 | PCI-DSS-4.0 | SOC2 | ISO-27001
	ControlID    string    `json:"control_id"`    // AC-2 | CIS.5.1.1 | REQ-6.3.3
	Title        string    `json:"title"`
	Status       string    `json:"status"`        // pass | fail | not-applicable | not-reviewed
	EvidenceJSON string    `json:"evidence_json"` // OSCAL assessment result evidence blob
	AssessedBy   string    `json:"assessed_by"`   // lula | kube-bench | openscap | manual
	AssessedAt   time.Time `json:"assessed_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AssessmentStore provides CRUD access to the kic_assessments table.
type AssessmentStore struct {
	pool *pgxpool.Pool
}

// NewAssessmentStore opens a pooled Postgres connection and auto-migrates the assessments table.
func NewAssessmentStore(ctx context.Context, databaseURL string) (*AssessmentStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &AssessmentStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *AssessmentStore) Close()                        { s.pool.Close() }
func (s *AssessmentStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *AssessmentStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS kic_assessments (
			id            TEXT        PRIMARY KEY,
			tenant_id     TEXT        NOT NULL,
			framework     TEXT        NOT NULL,
			control_id    TEXT        NOT NULL,
			title         TEXT        NOT NULL,
			status        TEXT        NOT NULL DEFAULT 'not-reviewed',
			evidence_json TEXT        NOT NULL DEFAULT '',
			assessed_by   TEXT        NOT NULL DEFAULT 'manual',
			assessed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate kic_assessments: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS kic_assessments_tenant_id ON kic_assessments (tenant_id)`)
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS kic_assessments_framework  ON kic_assessments (framework)`)
	return nil
}

// Create inserts a new assessment and returns it with a generated UUID.
func (s *AssessmentStore) Create(ctx context.Context, a Assessment) (Assessment, error) {
	a.ID = uuid.New().String()
	const q = `
		INSERT INTO kic_assessments (id, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q,
		a.ID, a.TenantID, a.Framework, a.ControlID, a.Title,
		a.Status, a.EvidenceJSON, a.AssessedBy, a.AssessedAt)
	return scanAssessment(row)
}

// Get returns a single assessment by ID. Returns pgx.ErrNoRows if not found.
func (s *AssessmentStore) Get(ctx context.Context, id string) (Assessment, error) {
	const q = `
		SELECT id, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at, created_at, updated_at
		FROM kic_assessments WHERE id = $1
	`
	return scanAssessment(s.pool.QueryRow(ctx, q, id))
}

// List returns assessments filtered by tenant_id, and optionally by framework and status.
func (s *AssessmentStore) List(ctx context.Context, tenantID, framework, status string, limit int) ([]Assessment, error) {
	const q = `
		SELECT id, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at, created_at, updated_at
		FROM kic_assessments
		WHERE tenant_id = $1
		  AND ($2 = '' OR framework = $2)
		  AND ($3 = '' OR status    = $3)
		ORDER BY assessed_at DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, tenantID, framework, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []Assessment
	for rows.Next() {
		a, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

// UpdateStatus updates the pass/fail status and optional evidence of an assessment.
func (s *AssessmentStore) UpdateStatus(ctx context.Context, id, status, evidenceJSON string) (Assessment, error) {
	const q = `
		UPDATE kic_assessments
		SET status        = $2,
		    evidence_json = CASE WHEN $3 = '' THEN evidence_json ELSE $3 END,
		    updated_at    = now()
		WHERE id = $1
		RETURNING id, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at, created_at, updated_at
	`
	return scanAssessment(s.pool.QueryRow(ctx, q, id, status, evidenceJSON))
}

func scanAssessment(row pgx.Row) (Assessment, error) {
	var a Assessment
	err := row.Scan(&a.ID, &a.TenantID, &a.Framework, &a.ControlID, &a.Title,
		&a.Status, &a.EvidenceJSON, &a.AssessedBy, &a.AssessedAt,
		&a.CreatedAt, &a.UpdatedAt)
	return a, err
}
