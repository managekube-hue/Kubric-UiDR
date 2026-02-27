package noc

import (
	"context"
	"fmt"
	"time"

	kubricdb "github.com/managekube-hue/Kubric-UiDR/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cluster represents a registered Kubernetes (or Proxmox) cluster managed by Kubric.
type Cluster struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`  // k8s | eks | gke | aks | proxmox
	Version   string    `json:"version"`   // Kubernetes version e.g. "1.29.2"
	Status    string    `json:"status"`    // healthy | degraded | critical | unknown
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

// Agent represents a Kubric agent binary deployed to an endpoint or node.
type Agent struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ClusterID     string    `json:"cluster_id,omitempty"` // empty for standalone agents
	Hostname      string    `json:"hostname"`
	AgentType     string    `json:"agent_type"`     // coresec | netguard | perftrace | watchdog
	Version       string    `json:"version"`
	Status        string    `json:"status"`         // online | offline | degraded
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
}

// NOCStore provides CRUD access to noc_clusters and noc_agents tables.
// Both tables are managed through a single pgxpool for efficiency.
type NOCStore struct {
	pool *pgxpool.Pool
}

// NewNOCStore opens a pooled Postgres connection and auto-migrates both NOC tables.
func NewNOCStore(ctx context.Context, databaseURL string) (*NOCStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &NOCStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *NOCStore) Close()                        { s.pool.Close() }
func (s *NOCStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *NOCStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS noc_clusters (
			id         TEXT        PRIMARY KEY,
			tenant_id  TEXT        NOT NULL,
			name       TEXT        NOT NULL,
			provider   TEXT        NOT NULL DEFAULT 'k8s',
			version    TEXT        NOT NULL DEFAULT '',
			status     TEXT        NOT NULL DEFAULT 'unknown',
			last_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate noc_clusters: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS noc_agents (
			id             TEXT        PRIMARY KEY,
			tenant_id      TEXT        NOT NULL,
			cluster_id     TEXT        NOT NULL DEFAULT '',
			hostname       TEXT        NOT NULL,
			agent_type     TEXT        NOT NULL,
			version        TEXT        NOT NULL DEFAULT '',
			status         TEXT        NOT NULL DEFAULT 'online',
			last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate noc_agents: %w", err)
	}

	// Unique index enables the ON CONFLICT upsert in Heartbeat()
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS noc_clusters_tenant_id   ON noc_clusters (tenant_id)`)
	_, _ = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS noc_agents_tenant_id      ON noc_agents (tenant_id)`)
	_, _ = s.pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS noc_agents_identity_idx
		ON noc_agents (tenant_id, hostname, agent_type)
	`)
	return nil
}

// ── Cluster methods ──────────────────────────────────────────────────────────

// CreateCluster registers a new cluster. Returns it with generated UUID + timestamps.
func (s *NOCStore) CreateCluster(ctx context.Context, c Cluster) (Cluster, error) {
	c.ID = uuid.New().String()
	const q = `
		INSERT INTO noc_clusters (id, tenant_id, name, provider, version, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, tenant_id, name, provider, version, status, last_seen, created_at
	`
	var result Cluster
	err := kubricdb.RunWithTenant(ctx, s.pool, c.TenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, c.ID, c.TenantID, c.Name, c.Provider, c.Version, c.Status)
		var e error
		result, e = scanCluster(row)
		return e
	})
	return result, err
}

// GetCluster returns a cluster by ID. Returns pgx.ErrNoRows if not found.
func (s *NOCStore) GetCluster(ctx context.Context, id string) (Cluster, error) {
	const q = `
		SELECT id, tenant_id, name, provider, version, status, last_seen, created_at
		FROM noc_clusters WHERE id = $1
	`
	return scanCluster(s.pool.QueryRow(ctx, q, id))
}

// ListClusters returns clusters for a tenant ordered by creation time.
func (s *NOCStore) ListClusters(ctx context.Context, tenantID string, limit int) ([]Cluster, error) {
	const q = `
		SELECT id, tenant_id, name, provider, version, status, last_seen, created_at
		FROM noc_clusters WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`
	rows, err := s.pool.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var clusters []Cluster
	for rows.Next() {
		c, err := scanCluster(rows)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

// UpdateCluster updates cluster status and/or version, and refreshes last_seen.
func (s *NOCStore) UpdateCluster(ctx context.Context, id, status, version string) (Cluster, error) {
	// Look up tenant_id so RunWithTenant can activate RLS for the correct tenant.
	var tenantID string
	_ = s.pool.QueryRow(ctx, `SELECT tenant_id FROM noc_clusters WHERE id = $1`, id).Scan(&tenantID)

	const q = `
		UPDATE noc_clusters
		SET status    = COALESCE(NULLIF($2,''), status),
		    version   = COALESCE(NULLIF($3,''), version),
		    last_seen = now()
		WHERE id = $1
		RETURNING id, tenant_id, name, provider, version, status, last_seen, created_at
	`
	var result Cluster
	err := kubricdb.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, id, status, version)
		var e error
		result, e = scanCluster(row)
		return e
	})
	return result, err
}

// DeleteCluster removes a cluster. Returns pgx.ErrNoRows if not found.
func (s *NOCStore) DeleteCluster(ctx context.Context, id string) error {
	// Look up tenant_id so RunWithTenant can activate RLS for the correct tenant.
	var tenantID string
	_ = s.pool.QueryRow(ctx, `SELECT tenant_id FROM noc_clusters WHERE id = $1`, id).Scan(&tenantID)

	return kubricdb.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM noc_clusters WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// ── Agent methods ─────────────────────────────────────────────────────────────

// Heartbeat upserts an agent by (tenant_id, hostname, agent_type).
// On first call it creates the agent; on subsequent calls it refreshes last_heartbeat and version.
func (s *NOCStore) Heartbeat(ctx context.Context, a Agent) (Agent, error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	const q = `
		INSERT INTO noc_agents (id, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat)
		VALUES ($1,$2,$3,$4,$5,$6,'online',now())
		ON CONFLICT (tenant_id, hostname, agent_type) DO UPDATE
		  SET version        = EXCLUDED.version,
		      cluster_id     = EXCLUDED.cluster_id,
		      status         = 'online',
		      last_heartbeat = now()
		RETURNING id, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat, created_at
	`
	var result Agent
	err := kubricdb.RunWithTenant(ctx, s.pool, a.TenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, a.ID, a.TenantID, a.ClusterID, a.Hostname, a.AgentType, a.Version)
		var e error
		result, e = scanAgent(row)
		return e
	})
	return result, err
}

// GetAgent returns an agent by ID. Returns pgx.ErrNoRows if not found.
func (s *NOCStore) GetAgent(ctx context.Context, id string) (Agent, error) {
	const q = `
		SELECT id, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat, created_at
		FROM noc_agents WHERE id = $1
	`
	return scanAgent(s.pool.QueryRow(ctx, q, id))
}

// UpdateAgent updates the mutable fields of an agent (status).
// Only non-empty values are applied; the caller is responsible for validation.
func (s *NOCStore) UpdateAgent(ctx context.Context, id, status string) (Agent, error) {
	// Look up tenant_id so RunWithTenant can activate RLS for the correct tenant.
	var tenantID string
	_ = s.pool.QueryRow(ctx, `SELECT tenant_id FROM noc_agents WHERE id = $1`, id).Scan(&tenantID)

	const q = `
		UPDATE noc_agents
		SET status = COALESCE(NULLIF($2,''), status)
		WHERE id = $1
		RETURNING id, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat, created_at
	`
	var result Agent
	err := kubricdb.RunWithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, id, status)
		var e error
		result, e = scanAgent(row)
		return e
	})
	return result, err
}

// ListAgents returns agents for a tenant, optionally filtered by cluster_id.
func (s *NOCStore) ListAgents(ctx context.Context, tenantID, clusterID string, limit int) ([]Agent, error) {
	const q = `
		SELECT id, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat, created_at
		FROM noc_agents
		WHERE tenant_id = $1
		  AND ($2 = '' OR cluster_id = $2)
		ORDER BY last_heartbeat DESC LIMIT $3
	`
	rows, err := s.pool.Query(ctx, q, tenantID, clusterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func scanCluster(row pgx.Row) (Cluster, error) {
	var c Cluster
	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.Provider, &c.Version,
		&c.Status, &c.LastSeen, &c.CreatedAt)
	return c, err
}

func scanAgent(row pgx.Row) (Agent, error) {
	var a Agent
	err := row.Scan(&a.ID, &a.TenantID, &a.ClusterID, &a.Hostname,
		&a.AgentType, &a.Version, &a.Status, &a.LastHeartbeat, &a.CreatedAt)
	return a, err
}
