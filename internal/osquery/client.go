// Package osquery provides a REST client for FleetDM, an osquery fleet
// management server.  It wraps the FleetDM v1 API to enumerate hosts, run
// live/distributed queries, and manage query packs.
//
// API reference: https://fleetdm.com/docs/rest-api/rest-api
package osquery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Client talks to a FleetDM server over its REST API.
type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// Host describes a single enrolled osquery host as returned by FleetDM.
type Host struct {
	ID            int    `json:"id"`
	Hostname      string `json:"hostname"`
	UUID          string `json:"uuid"`
	Platform      string `json:"platform"`       // darwin, ubuntu, windows, centos
	OSVersion     string `json:"os_version"`
	OSQueryVer    string `json:"osquery_version"`
	Status        string `json:"status"`          // online, offline
	LastEnrolled  string `json:"last_enrolled_at"`
	LastSeen      string `json:"seen_time"`
	Uptime        int64  `json:"uptime"`
	Memory        int64  `json:"memory"`
	CPUBrand      string `json:"cpu_brand"`
	HardwareModel string `json:"hardware_model"`
	TeamID        *int   `json:"team_id,omitempty"`
	LabelIDs      []int  `json:"label_ids,omitempty"`
}

// Query represents a saved osquery query inside FleetDM.
type Query struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
	Platform    string `json:"platform,omitempty"` // blank = all platforms
	Interval    int    `json:"interval,omitempty"` // seconds, 0 = on-demand only
}

// DistributedQueryResult holds results from a single host for an ad-hoc
// distributed query.
type DistributedQueryResult struct {
	HostID   int                 `json:"host_id"`
	Hostname string              `json:"hostname"`
	Rows     []map[string]string `json:"rows"`
	Error    string              `json:"error,omitempty"`
}

// LiveQueryResult is the top-level response from a live query campaign.
type LiveQueryResult struct {
	CampaignID int                      `json:"campaign_id"`
	Results    []DistributedQueryResult `json:"results"`
}

// Pack is an osquery query pack.
type Pack struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name"`
	Platform string `json:"platform,omitempty"`
	Disabled bool   `json:"disabled"`
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// New creates a FleetDM client.  baseURL is the server root
// (e.g. "https://fleet.internal:8080"), apiKey is a FleetDM API token.
// Returns nil, nil when baseURL is empty (osquery integration is optional).
func New(baseURL, apiKey string) (*Client, error) {
	if baseURL == "" {
		return nil, nil
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // internal cluster traffic
			},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

// newReq builds an http.Request with the common Authorization header.
func (c *Client) newReq(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// do executes a request and returns the response body bytes.  It returns an
// error for any non-2xx status code.
func (c *Client) do(req *http.Request, label string) ([]byte, error) {
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osquery: %s: %w", label, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("osquery: %s read body: %w", label, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("osquery: %s: status %d: %s", label, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Host methods
// ---------------------------------------------------------------------------

// hostsEnvelope wraps the JSON response from GET /api/v1/fleet/hosts.
type hostsEnvelope struct {
	Hosts []Host `json:"hosts"`
	Meta  struct {
		HasNextResults    bool `json:"has_next_results"`
		HasPreviousResults bool `json:"has_previous_results"`
		TotalCount        int  `json:"total_count"`
	} `json:"meta"`
}

// ListHosts returns hosts filtered by status ("online", "offline", or "" for
// all).  limit/offset provide pagination.  The second return value is the
// total host count reported by FleetDM.
func (c *Client) ListHosts(ctx context.Context, status string, limit, offset int) ([]Host, int, error) {
	if c == nil {
		return nil, 0, nil
	}

	path := fmt.Sprintf("/api/v1/fleet/hosts?per_page=%d&page=%d", limit, offset)
	if status != "" {
		path += "&status=" + status
	}

	req, err := c.newReq(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("osquery: list hosts build request: %w", err)
	}

	data, err := c.do(req, "list hosts")
	if err != nil {
		return nil, 0, err
	}

	var env hostsEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, 0, fmt.Errorf("osquery: list hosts decode: %w", err)
	}
	return env.Hosts, env.Meta.TotalCount, nil
}

// hostEnvelope wraps the JSON response from GET /api/v1/fleet/hosts/{id}.
type hostEnvelope struct {
	Host Host `json:"host"`
}

// GetHost returns a single host by its FleetDM numeric ID.
func (c *Client) GetHost(ctx context.Context, hostID int) (*Host, error) {
	if c == nil {
		return nil, nil
	}

	path := fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID)
	req, err := c.newReq(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("osquery: get host build request: %w", err)
	}

	data, err := c.do(req, "get host")
	if err != nil {
		return nil, err
	}

	var env hostEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: get host decode: %w", err)
	}
	return &env.Host, nil
}

// SearchHosts searches hosts by hostname or IP substring.
func (c *Client) SearchHosts(ctx context.Context, query string) ([]Host, error) {
	if c == nil {
		return nil, nil
	}

	path := "/api/v1/fleet/hosts?query=" + query
	req, err := c.newReq(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("osquery: search hosts build request: %w", err)
	}

	data, err := c.do(req, "search hosts")
	if err != nil {
		return nil, err
	}

	var env hostsEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: search hosts decode: %w", err)
	}
	return env.Hosts, nil
}

// ---------------------------------------------------------------------------
// Query methods
// ---------------------------------------------------------------------------

// queryEnvelope wraps the JSON response from POST /api/v1/fleet/queries.
type queryEnvelope struct {
	Query Query `json:"query"`
}

// queriesEnvelope wraps the JSON response from GET /api/v1/fleet/queries.
type queriesEnvelope struct {
	Queries []Query `json:"queries"`
}

// CreateQuery saves a new named query in FleetDM.
func (c *Client) CreateQuery(ctx context.Context, q Query) (*Query, error) {
	if c == nil {
		return nil, nil
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("osquery: create query marshal: %w", err)
	}

	req, err := c.newReq(ctx, http.MethodPost, "/api/v1/fleet/queries", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("osquery: create query build request: %w", err)
	}

	data, err := c.do(req, "create query")
	if err != nil {
		return nil, err
	}

	var env queryEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: create query decode: %w", err)
	}
	return &env.Query, nil
}

// ListQueries returns all saved queries.
func (c *Client) ListQueries(ctx context.Context) ([]Query, error) {
	if c == nil {
		return nil, nil
	}

	req, err := c.newReq(ctx, http.MethodGet, "/api/v1/fleet/queries", nil)
	if err != nil {
		return nil, fmt.Errorf("osquery: list queries build request: %w", err)
	}

	data, err := c.do(req, "list queries")
	if err != nil {
		return nil, err
	}

	var env queriesEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: list queries decode: %w", err)
	}
	return env.Queries, nil
}

// runLiveQueryPayload is the POST body for /api/v1/fleet/queries/run.
type runLiveQueryPayload struct {
	Query   string `json:"query"`
	HostIDs []int  `json:"host_ids"`
}

// RunLiveQuery executes an ad-hoc SQL query on the specified hosts and waits
// for the campaign to return results.
func (c *Client) RunLiveQuery(ctx context.Context, sql string, hostIDs []int) (*LiveQueryResult, error) {
	if c == nil {
		return nil, nil
	}

	payload, err := json.Marshal(runLiveQueryPayload{
		Query:   sql,
		HostIDs: hostIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("osquery: run live query marshal: %w", err)
	}

	req, err := c.newReq(ctx, http.MethodPost, "/api/v1/fleet/queries/run", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("osquery: run live query build request: %w", err)
	}

	data, err := c.do(req, "run live query")
	if err != nil {
		return nil, err
	}

	var result LiveQueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("osquery: run live query decode: %w", err)
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Pack methods
// ---------------------------------------------------------------------------

// packsEnvelope wraps the JSON response from GET /api/v1/fleet/packs.
type packsEnvelope struct {
	Packs []Pack `json:"packs"`
}

// packEnvelope wraps the JSON response from POST /api/v1/fleet/packs.
type packEnvelope struct {
	Pack Pack `json:"pack"`
}

// ListPacks returns all query packs.
func (c *Client) ListPacks(ctx context.Context) ([]Pack, error) {
	if c == nil {
		return nil, nil
	}

	req, err := c.newReq(ctx, http.MethodGet, "/api/v1/fleet/packs", nil)
	if err != nil {
		return nil, fmt.Errorf("osquery: list packs build request: %w", err)
	}

	data, err := c.do(req, "list packs")
	if err != nil {
		return nil, err
	}

	var env packsEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: list packs decode: %w", err)
	}
	return env.Packs, nil
}

// CreatePack creates a new query pack.
func (c *Client) CreatePack(ctx context.Context, p Pack) (*Pack, error) {
	if c == nil {
		return nil, nil
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("osquery: create pack marshal: %w", err)
	}

	req, err := c.newReq(ctx, http.MethodPost, "/api/v1/fleet/packs", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("osquery: create pack build request: %w", err)
	}

	data, err := c.do(req, "create pack")
	if err != nil {
		return nil, err
	}

	var env packEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("osquery: create pack decode: %w", err)
	}
	return &env.Pack, nil
}

// ---------------------------------------------------------------------------
// Health / lifecycle
// ---------------------------------------------------------------------------

// Health performs a lightweight connectivity check against the FleetDM server
// by hitting GET /healthz.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return nil
	}

	req, err := c.newReq(ctx, http.MethodGet, "/healthz", nil)
	if err != nil {
		return fmt.Errorf("osquery: health build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("osquery: health request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("osquery: health check returned %d", resp.StatusCode)
	}
	return nil
}

// Close releases any idle HTTP connections held by the client.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.hc.CloseIdleConnections()
}
