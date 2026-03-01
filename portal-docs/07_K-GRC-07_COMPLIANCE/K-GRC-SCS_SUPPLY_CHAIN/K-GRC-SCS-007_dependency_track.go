package grc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DepTrackFinding is a vulnerability finding returned by Dependency-Track.
// Fields match the spec: VulnerabilityID, Component, Severity, Published, Description, Recommendation.
type DepTrackFinding struct {
	VulnerabilityID string `json:"vulnerabilityId"`
	Component       string `json:"component"`       // "name@version"
	Severity        string `json:"severity"`        // CRITICAL | HIGH | MEDIUM | LOW
	Published       string `json:"published"`       // RFC3339 date
	Description     string `json:"description"`
	Recommendation  string `json:"recommendation"`
}

// ProjectMetrics holds the current portfolio risk metrics for a Dependency-Track project.
type ProjectMetrics struct {
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Unassigned int `json:"unassigned"`
}

// DepTrackClient is an HTTP client for the Dependency-Track REST API v1.
// BaseURL and APIKey are exported so callers can override them directly.
type DepTrackClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	pgPool     *pgxpool.Pool
}

// NewDepTrackClient creates a DepTrackClient from environment variables.
// Required: DEPTRACK_URL, DEPTRACK_API_KEY, DATABASE_URL.
func NewDepTrackClient(ctx context.Context) (*DepTrackClient, error) {
	baseURL := os.Getenv("DEPTRACK_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("dep_track: DEPTRACK_URL is required")
	}
	apiKey := os.Getenv("DEPTRACK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("dep_track: DEPTRACK_API_KEY is required")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("dep_track: DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("dep_track: pgxpool: %w", err)
	}
	return &DepTrackClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		pgPool:     pool,
	}, nil
}

// doJSON executes an HTTP request against Dependency-Track with X-API-Key auth.
// Decodes the JSON response into dest (if non-nil).
func (c *DepTrackClient) doJSON(ctx context.Context, method, path string, body any, dest any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("dep_track doJSON encode: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("dep_track doJSON new request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("dep_track %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("dep_track %s %s status=%d: %s", method, path, resp.StatusCode, string(raw))
	}
	if dest != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("dep_track doJSON unmarshal: %w (%s)", err, string(raw[:min(200, len(raw))]))
		}
	}
	return nil
}

// CreateProject creates or updates a project in Dependency-Track.
// tenantID is stored as a tag so projects remain tenant-scoped.
// Returns the project UUID assigned by Dependency-Track.
func (c *DepTrackClient) CreateProject(ctx context.Context, name, version, tenantID string) (string, error) {
	body := map[string]any{
		"name":       name,
		"version":    version,
		"classifier": "APPLICATION",
		"active":     true,
		"tags": []map[string]string{
			{"name": "tenant:" + tenantID},
		},
	}
	var proj struct {
		UUID string `json:"uuid"`
	}
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/project", body, &proj); err != nil {
		return "", fmt.Errorf("CreateProject: %w", err)
	}
	return proj.UUID, nil
}

// UploadSBOM uploads a CycloneDX SBOM (as a JSON string) to a project.
// The BOM is base64-encoded per Dependency-Track's API contract.
// Returns the processing token used to poll status.
func (c *DepTrackClient) UploadSBOM(ctx context.Context, projectUUID, sbomJSON string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(sbomJSON))
	body := map[string]any{
		"project": projectUUID,
		"bom":     encoded,
	}
	return c.doJSON(ctx, http.MethodPut, "/api/v1/bom", body, nil)
}

// GetFindings returns vulnerability findings for a project.
func (c *DepTrackClient) GetFindings(ctx context.Context, projectUUID string) ([]DepTrackFinding, error) {
	type apiMatch struct {
		Vulnerability struct {
			VulnID          string `json:"vulnId"`
			Severity        string `json:"severity"`
			Description     string `json:"description"`
			Recommendation  string `json:"recommendation"`
			Published       string `json:"published"`
		} `json:"vulnerability"`
		Component struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"component"`
	}

	var apiFindings []apiMatch
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/finding/project/"+projectUUID, nil, &apiFindings); err != nil {
		return nil, fmt.Errorf("GetFindings: %w", err)
	}

	findings := make([]DepTrackFinding, 0, len(apiFindings))
	for _, f := range apiFindings {
		findings = append(findings, DepTrackFinding{
			VulnerabilityID: f.Vulnerability.VulnID,
			Component:       f.Component.Name + "@" + f.Component.Version,
			Severity:        f.Vulnerability.Severity,
			Published:       f.Vulnerability.Published,
			Description:     f.Vulnerability.Description,
			Recommendation:  f.Vulnerability.Recommendation,
		})
	}
	return findings, nil
}

// GetProjectMetrics fetches the current risk metrics for a project.
func (c *DepTrackClient) GetProjectMetrics(ctx context.Context, projectUUID string) (*ProjectMetrics, error) {
	type apiMetrics struct {
		Critical   int `json:"critical"`
		High       int `json:"high"`
		Medium     int `json:"medium"`
		Low        int `json:"low"`
		Unassigned int `json:"unassigned"`
	}
	var m apiMetrics
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/metrics/project/"+projectUUID+"/current", nil, &m); err != nil {
		return nil, fmt.Errorf("GetProjectMetrics: %w", err)
	}
	return &ProjectMetrics{
		Critical:   m.Critical,
		High:       m.High,
		Medium:     m.Medium,
		Low:        m.Low,
		Unassigned: m.Unassigned,
	}, nil
}

// SyncToPostgres fetches findings and metrics for a project and upserts to PostgreSQL.
func (c *DepTrackClient) SyncToPostgres(ctx context.Context, projectUUID, tenantID, assetID string) error {
	findings, err := c.GetFindings(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("SyncToPostgres GetFindings: %w", err)
	}

	for _, f := range findings {
		_, err := c.pgPool.Exec(ctx, `
			INSERT INTO deptrack_findings
				(tenant_id, asset_id, project_uuid, vulnerability_id, component,
				 severity, published, description, recommendation, synced_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
			ON CONFLICT (tenant_id, project_uuid, vulnerability_id, component) DO UPDATE SET
				severity        = EXCLUDED.severity,
				description     = EXCLUDED.description,
				recommendation  = EXCLUDED.recommendation,
				synced_at       = NOW()`,
			tenantID, assetID, projectUUID,
			f.VulnerabilityID, f.Component,
			f.Severity, f.Published, f.Description, f.Recommendation)
		if err != nil {
			return fmt.Errorf("SyncToPostgres insert finding: %w", err)
		}
	}

	metrics, err := c.GetProjectMetrics(ctx, projectUUID)
	if err != nil {
		// Non-fatal: metrics may be unavailable while Dependency-Track is still processing.
		return nil
	}
	_, err = c.pgPool.Exec(ctx, `
		INSERT INTO deptrack_metrics
			(tenant_id, asset_id, project_uuid, critical, high, medium, low, unassigned, synced_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		ON CONFLICT (tenant_id, project_uuid) DO UPDATE SET
			critical   = EXCLUDED.critical,
			high       = EXCLUDED.high,
			medium     = EXCLUDED.medium,
			low        = EXCLUDED.low,
			unassigned = EXCLUDED.unassigned,
			synced_at  = NOW()`,
		tenantID, assetID, projectUUID,
		metrics.Critical, metrics.High, metrics.Medium, metrics.Low, metrics.Unassigned)
	return err
}

// Close releases the PostgreSQL pool.
func (c *DepTrackClient) Close() {
	c.pgPool.Close()
}
