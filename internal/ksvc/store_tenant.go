package ksvc

import (
	"context"
	"fmt"
	"time"

	kubricdb "github.com/managekube-hue/Kubric-UiDR/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenant is the canonical tenant record stored in PostgreSQL.
// tenant_id follows the Kubernetes namespace naming convention enforced by schema.ValidateTenantID.
type Tenant struct {
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantStore provides CRUD access to the kubric_tenants table via pgxpool.
type TenantStore struct {
	pool *pgxpool.Pool
}

// NewTenantStore opens a connection pool to Postgres, pings the server,
// and runs the one-time idempotent migration that creates the tenants table.
func NewTenantStore(ctx context.Context, databaseURL string) (*TenantStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &TenantStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying connection pool.
func (s *TenantStore) Close() {
	s.pool.Close()
}

// Ping verifies the database connection is alive.
func (s *TenantStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// migrate creates the kubric_tenants table if it does not exist.
// This is idempotent — safe to call on every startup.
func (s *TenantStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS kubric_tenants (
			tenant_id   TEXT        PRIMARY KEY,
			name        TEXT        NOT NULL,
			plan        TEXT        NOT NULL DEFAULT 'starter',
			status      TEXT        NOT NULL DEFAULT 'active',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate kubric_tenants: %w", err)
	}
	return nil
}

// Create inserts a new tenant row. Returns an error if tenant_id already exists.
func (s *TenantStore) Create(ctx context.Context, t Tenant) (Tenant, error) {
	const q = `
		INSERT INTO kubric_tenants (tenant_id, name, plan, status)
		VALUES ($1, $2, $3, $4)
		RETURNING tenant_id, name, plan, status, created_at, updated_at
	`
	var result Tenant
	err := kubricdb.RunWithTenant(ctx, s.pool, t.TenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, t.TenantID, t.Name, t.Plan, t.Status)
		var e error
		result, e = scanTenant(row)
		return e
	})
	return result, err
}

// Get returns a tenant by tenant_id. Returns pgx.ErrNoRows if not found.
func (s *TenantStore) Get(ctx context.Context, tenantID string) (Tenant, error) {
	const q = `
		SELECT tenant_id, name, plan, status, created_at, updated_at
		FROM kubric_tenants
		WHERE tenant_id = $1
	`
	row := s.pool.QueryRow(ctx, q, tenantID)
	return scanTenant(row)
}

// List returns up to limit tenants ordered by created_at descending.
func (s *TenantStore) List(ctx context.Context, limit int) ([]Tenant, error) {
	const q = `
		SELECT tenant_id, name, plan, status, created_at, updated_at
		FROM kubric_tenants
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// Update applies partial patches to name, plan, and/or status.
// Empty strings leave the existing values unchanged.
// Returns pgx.ErrNoRows if the tenant does not exist.
func (s *TenantStore) Update(ctx context.Context, tenantID, name, plan, status string) (Tenant, error) {
	const q = `
		UPDATE kubric_tenants
		SET
			name       = COALESCE(NULLIF($2, ''), name),
			plan       = COALESCE(NULLIF($3, ''), plan),
			status     = COALESCE(NULLIF($4, ''), status),
			updated_at = now()
		WHERE tenant_id = $1
		RETURNING tenant_id, name, plan, status, created_at, updated_at
	`
	var result Tenant
	err := kubricdb.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, tenantID, name, plan, status)
		var e error
		result, e = scanTenant(row)
		return e
	})
	return result, err
}

// Delete removes a tenant by tenant_id. Returns pgx.ErrNoRows if not found.
func (s *TenantStore) Delete(ctx context.Context, tenantID string) error {
	return kubricdb.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM kubric_tenants WHERE tenant_id = $1`, tenantID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// scanTenant scans a single pgx.Row into a Tenant struct.
func scanTenant(row pgx.Row) (Tenant, error) {
	var t Tenant
	err := row.Scan(&t.TenantID, &t.Name, &t.Plan, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
