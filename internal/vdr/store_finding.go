package vdr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Finding is a normalized vulnerability result ingested from Nuclei, Trivy, or Grype.
type Finding struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Target      string    `json:"target"`      // hostname, image, IP, URL
	Scanner     string    `json:"scanner"`     // nuclei | trivy | grype | manual
	Severity    string    `json:"severity"`    // critical | high | medium | low | informational
	CVEID       string    `json:"cve_id"`      // "CVE-2021-44228" or ""
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`   // open | acknowledged | resolved | false-positive
	RawJSON     string    `json:"raw_json"` // raw scanner output blob
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FindingStore provides CRUD access to the vdr_findings table.
type FindingStore struct {
	pool *pgxpool.Pool
}

// NewFindingStore opens a pooled connection to Postgres and auto-migrates the findings table.
func NewFindingStore(ctx context.Context, databaseURL string) (*FindingStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &FindingStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *FindingStore) Close()                        { s.pool.Close() }
func (s *FindingStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *FindingStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS vdr_findings (
			id          TEXT        PRIMARY KEY,
			tenant_id   TEXT        NOT NULL,
			target      TEXT        NOT NULL,
			scanner     TEXT        NOT NULL,
			severity    TEXT        NOT NULL,
			cve_id      TEXT        NOT NULL DEFAULT '',
			title       TEXT        NOT NULL,
			description TEXT        NOT NULL DEFAULT '',
			status      TEXT        NOT NULL DEFAULT 'open',
			raw_json    TEXT        NOT NULL DEFAULT '',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate vdr_findings: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS vdr_findings_tenant_id ON vdr_findings (tenant_id)`)
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS vdr_findings_severity   ON vdr_findings (severity)`)
	return nil
}

// Create inserts a new finding and returns it with a generated UUID and timestamps.
func (s *FindingStore) Create(ctx context.Context, f Finding) (Finding, error) {
	f.ID = uuid.New().String()
	const q = `
		INSERT INTO vdr_findings (id, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json, created_at, updated_at
	`
	row := s.pool.QueryRow(ctx, q,
		f.ID, f.TenantID, f.Target, f.Scanner, f.Severity,
		f.CVEID, f.Title, f.Description, f.Status, f.RawJSON)
	return scanFinding(row)
}

// Get returns a single finding by ID. Returns pgx.ErrNoRows if not found.
func (s *FindingStore) Get(ctx context.Context, id string) (Finding, error) {
	const q = `
		SELECT id, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json, created_at, updated_at
		FROM vdr_findings WHERE id = $1
	`
	return scanFinding(s.pool.QueryRow(ctx, q, id))
}

// List returns findings filtered by tenant_id and optionally by severity and status.
func (s *FindingStore) List(ctx context.Context, tenantID, severity, status string, limit int) ([]Finding, error) {
	const q = `
		SELECT id, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json, created_at, updated_at
		FROM vdr_findings
		WHERE tenant_id = $1
		  AND ($2 = '' OR severity = $2)
		  AND ($3 = '' OR status   = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, tenantID, severity, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var findings []Finding
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

// UpdateStatus changes the triage status of a finding.
func (s *FindingStore) UpdateStatus(ctx context.Context, id, status string) (Finding, error) {
	const q = `
		UPDATE vdr_findings
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json, created_at, updated_at
	`
	return scanFinding(s.pool.QueryRow(ctx, q, id, status))
}

func scanFinding(row pgx.Row) (Finding, error) {
	var f Finding
	err := row.Scan(&f.ID, &f.TenantID, &f.Target, &f.Scanner, &f.Severity,
		&f.CVEID, &f.Title, &f.Description, &f.Status, &f.RawJSON,
		&f.CreatedAt, &f.UpdatedAt)
	return f, err
}
