package vdr

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register "clickhouse" sql driver
)

// EPSSScore holds the Exploit Prediction Scoring System data for a CVE.
type EPSSScore struct {
	CveID      string  `json:"cve_id"`
	Score      float64 `json:"epss_score"`
	Percentile float64 `json:"epss_percentile"`
}

// EPSSClient queries the kubric.epss_scores ClickHouse table populated by the
// ti_feeds.py EPSS feed ingestion pipeline.
type EPSSClient struct {
	db *sql.DB
}

// NewEPSSClient opens a ClickHouse connection. clickhouseURL must be in the
// form clickhouse://user:password@host:9000/database.
// Returns nil, nil when clickhouseURL is empty (EPSS enrichment is optional).
func NewEPSSClient(clickhouseURL string) (*EPSSClient, error) {
	if clickhouseURL == "" {
		return nil, nil
	}
	// Validate the URL scheme so we fail fast rather than at query time.
	if u, err := url.Parse(clickhouseURL); err != nil || u.Scheme != "clickhouse" {
		return nil, fmt.Errorf("epss: invalid clickhouse URL (expected clickhouse://...): %s", clickhouseURL)
	}
	db, err := sql.Open("clickhouse", clickhouseURL)
	if err != nil {
		return nil, fmt.Errorf("epss: open clickhouse: %w", err)
	}
	return &EPSSClient{db: db}, nil
}

// Close releases the ClickHouse connection pool.
func (c *EPSSClient) Close() {
	if c != nil {
		_ = c.db.Close()
	}
}

// Lookup returns the most recent EPSS score for cveID.
// Returns nil, nil when the CVE is not found in the scores table.
func (c *EPSSClient) Lookup(ctx context.Context, cveID string) (*EPSSScore, error) {
	if c == nil || cveID == "" {
		return nil, nil
	}
	const q = `
		SELECT cve_id, epss_score, percentile
		FROM kubric.epss_scores
		WHERE cve_id = ?
		ORDER BY fetched_at DESC
		LIMIT 1
	`
	row := c.db.QueryRowContext(ctx, q, cveID)
	var s EPSSScore
	if err := row.Scan(&s.CveID, &s.Score, &s.Percentile); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("epss lookup %s: %w", cveID, err)
	}
	return &s, nil
}

// LookupBatch returns EPSS scores keyed by CVE ID for multiple CVEs in one query.
func (c *EPSSClient) LookupBatch(ctx context.Context, cveIDs []string) (map[string]*EPSSScore, error) {
	result := make(map[string]*EPSSScore, len(cveIDs))
	if c == nil || len(cveIDs) == 0 {
		return result, nil
	}
	// Build IN list using ClickHouse array syntax
	const q = `
		SELECT cve_id, epss_score, percentile
		FROM kubric.epss_scores
		WHERE cve_id IN (?)
		ORDER BY cve_id ASC, fetched_at DESC
		LIMIT BY 1 cve_id
	`
	rows, err := c.db.QueryContext(ctx, q, cveIDs)
	if err != nil {
		return nil, fmt.Errorf("epss batch lookup: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s EPSSScore
		if err := rows.Scan(&s.CveID, &s.Score, &s.Percentile); err != nil {
			continue
		}
		cp := s
		result[s.CveID] = &cp
	}
	return result, rows.Err()
}
