// Package db provides shared Postgres transaction helpers for Kubric services.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunWithTenant opens a transaction on pool, activates Postgres RLS for the
// given tenant by executing SET LOCAL app.current_tenant_id, then invokes fn.
// The transaction is committed on success and rolled back on any error.
//
// Usage:
//
//	err := db.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT INTO ...", ...)
//	    return err
//	})
func RunWithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
