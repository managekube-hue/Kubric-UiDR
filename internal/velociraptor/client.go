// Package velociraptor provides an HTTP-only REST client for the Velociraptor
// DFIR platform. Velociraptor is AGPL-3.0 licensed — this package communicates
// exclusively via HTTP and imports no Velociraptor code.
package velociraptor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client communicates with Velociraptor via HTTP only (AGPL-3.0 boundary).
type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// VRClient represents a Velociraptor endpoint agent enrolment record.
type VRClient struct {
	ClientID     string                 `json:"client_id"`
	Hostname     string                 `json:"os_info.hostname"`
	OS           string                 `json:"os_info.system"`
	Release      string                 `json:"os_info.release"`
	Architecture string                 `json:"os_info.machine"`
	FirstSeen    int64                  `json:"first_seen_at"`
	LastSeen     int64                  `json:"last_seen_at"`
	LastIP       string                 `json:"last_ip"`
	Labels       []string               `json:"labels"`
	AgentInfo    map[string]interface{} `json:"agent_information,omitempty"`
}

// Hunt represents a Velociraptor hunt — a bulk artifact collection across many
// endpoints.
type Hunt struct {
	HuntID      string     `json:"hunt_id"`
	Description string     `json:"hunt_description"`
	Creator     string     `json:"creator"`
	State       string     `json:"state"` // RUNNING, PAUSED, STOPPED, COMPLETED
	Created     int64      `json:"create_time"`
	Started     int64      `json:"start_time"`
	Expires     int64      `json:"expires"`
	Stats       *HuntStats `json:"stats,omitempty"`
	Artifacts   []string   `json:"start_request.artifacts"`
}

// HuntStats provides aggregate client counts for a hunt.
type HuntStats struct {
	TotalClients     int `json:"total_clients_scheduled"`
	CompletedClients int `json:"total_clients_with_results"`
	ErrorClients     int `json:"total_clients_with_errors"`
}

// Artifact describes a Velociraptor artifact definition.
type Artifact struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Type        string   `json:"type"` // CLIENT, SERVER, CLIENT_EVENT
	Sources     []Source `json:"sources,omitempty"`
	Parameters  []Param  `json:"parameters,omitempty"`
}

// Source is a single VQL source block inside an artifact.
type Source struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

// Param describes one parameter an artifact accepts.
type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Default     string `json:"default"`
}

// VQLRequest is the payload for server-side VQL execution.
type VQLRequest struct {
	Query string            `json:"query"`
	Env   map[string]string `json:"env,omitempty"`
}

// VQLRow is a single result row returned by a VQL query.
type VQLRow map[string]interface{}

// FlowResult describes the state of an artifact collection flow.
type FlowResult struct {
	FlowID    string                 `json:"session_id"`
	ClientID  string                 `json:"client_id"`
	State     string                 `json:"state"`
	Status    string                 `json:"status"`
	Artifacts []string               `json:"artifacts_with_results"`
	Request   map[string]interface{} `json:"request,omitempty"`
}

// CollectRequest is the payload for scheduling an artifact collection on a
// specific client.
type CollectRequest struct {
	ClientID   string            `json:"client_id"`
	Artifacts  []string          `json:"artifacts"`
	Parameters map[string]string `json:"specs,omitempty"`
	Urgent     bool              `json:"urgent,omitempty"`
	MaxRows    int64             `json:"max_rows,omitempty"`
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// New creates a Velociraptor HTTP client. Returns nil, nil when baseURL is
// empty (integration disabled). The transport skips TLS verification because
// Velociraptor commonly uses self-signed certificates.
func New(baseURL, apiKey string) (*Client, error) {
	if baseURL == "" {
		return nil, nil
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("velociraptor: invalid base URL: %s", baseURL)
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Velociraptor self-signed certs
				},
			},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// doGet issues an authenticated GET and returns the raw response body.
func (c *Client) doGet(ctx context.Context, path string) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: build GET request: %w", err)
	}
	return c.execute(req)
}

// doPost issues an authenticated POST with a JSON body and returns the raw
// response body.
func (c *Client) doPost(ctx context.Context, path string, body interface{}) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("velociraptor: build POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.execute(req)
}

// execute adds auth headers, performs the request, and reads the body.
func (c *Client) execute(req *http.Request) ([]byte, error) {
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("velociraptor: %s %s returned %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, truncateBody(data, 512))
	}
	return data, nil
}

// truncateBody returns at most n bytes of data as a string, for error messages.
func truncateBody(data []byte, n int) string {
	if len(data) <= n {
		return string(data)
	}
	return string(data[:n]) + "...(truncated)"
}

// ---------------------------------------------------------------------------
// Client management
// ---------------------------------------------------------------------------

// SearchClients searches for Velociraptor clients by hostname, label, or
// client_id. The query string follows Velociraptor's search syntax
// (e.g. "host:web-server", "label:compromised", "C.1234").
func (c *Client) SearchClients(ctx context.Context, query string, limit int) ([]VRClient, error) {
	if c == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	path := fmt.Sprintf("/api/v1/SearchClients?query=%s&limit=%d",
		url.QueryEscape(query), limit)
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: search clients: %w", err)
	}

	var envelope struct {
		Items []VRClient `json:"items"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("velociraptor: decode search clients response: %w", err)
	}
	return envelope.Items, nil
}

// GetClient returns a single client by client_id (e.g. "C.abc123").
func (c *Client) GetClient(ctx context.Context, clientID string) (*VRClient, error) {
	if c == nil {
		return nil, nil
	}
	path := fmt.Sprintf("/api/v1/GetClient/%s", url.PathEscape(clientID))
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: get client %s: %w", clientID, err)
	}

	var client VRClient
	if err := json.Unmarshal(data, &client); err != nil {
		return nil, fmt.Errorf("velociraptor: decode get client response: %w", err)
	}
	return &client, nil
}

// LabelClient adds a label to a client. Labels are used by Velociraptor for
// grouping and targeting hunts.
func (c *Client) LabelClient(ctx context.Context, clientID, label string) error {
	if c == nil {
		return nil
	}
	body := map[string]interface{}{
		"client_ids": []string{clientID},
		"labels":     []string{label},
		"operation":  "set",
	}
	_, err := c.doPost(ctx, "/api/v1/LabelClients", body)
	if err != nil {
		return fmt.Errorf("velociraptor: label client %s: %w", clientID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Hunts
// ---------------------------------------------------------------------------

// CreateHunt creates a new hunt that collects the specified artifacts across
// all matching clients.
func (c *Client) CreateHunt(ctx context.Context, description string, artifacts []string, parameters map[string]string) (*Hunt, error) {
	if c == nil {
		return nil, nil
	}

	// Build the specs array — one entry per artifact with its parameters.
	specs := make([]map[string]interface{}, 0, len(artifacts))
	for _, a := range artifacts {
		spec := map[string]interface{}{
			"artifact": a,
		}
		if len(parameters) > 0 {
			envList := make([]map[string]string, 0, len(parameters))
			for k, v := range parameters {
				envList = append(envList, map[string]string{"key": k, "value": v})
			}
			spec["parameters"] = map[string]interface{}{"env": envList}
		}
		specs = append(specs, spec)
	}

	body := map[string]interface{}{
		"hunt_description": description,
		"start_request": map[string]interface{}{
			"artifacts": artifacts,
			"specs":     specs,
		},
		"state": "RUNNING",
	}

	data, err := c.doPost(ctx, "/api/v1/CreateHunt", body)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: create hunt: %w", err)
	}

	var hunt Hunt
	if err := json.Unmarshal(data, &hunt); err != nil {
		return nil, fmt.Errorf("velociraptor: decode create hunt response: %w", err)
	}
	return &hunt, nil
}

// ListHunts returns hunts ordered by creation time (newest first).
func (c *Client) ListHunts(ctx context.Context, limit, offset int) ([]Hunt, error) {
	if c == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	path := fmt.Sprintf("/api/v1/ListHunts?count=%d&offset=%d", limit, offset)
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: list hunts: %w", err)
	}

	var envelope struct {
		Items []Hunt `json:"items"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("velociraptor: decode list hunts response: %w", err)
	}
	return envelope.Items, nil
}

// GetHunt returns a single hunt by ID (e.g. "H.abc123").
func (c *Client) GetHunt(ctx context.Context, huntID string) (*Hunt, error) {
	if c == nil {
		return nil, nil
	}
	path := fmt.Sprintf("/api/v1/GetHunt?hunt_id=%s", url.QueryEscape(huntID))
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: get hunt %s: %w", huntID, err)
	}

	var hunt Hunt
	if err := json.Unmarshal(data, &hunt); err != nil {
		return nil, fmt.Errorf("velociraptor: decode get hunt response: %w", err)
	}
	return &hunt, nil
}

// GetHuntResults returns result rows from a completed (or in-progress) hunt.
// artifact specifies which collected artifact's results to retrieve.
func (c *Client) GetHuntResults(ctx context.Context, huntID, artifact string, limit int) ([]VQLRow, error) {
	if c == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 1000
	}
	path := fmt.Sprintf("/api/v1/GetHuntResults?hunt_id=%s&artifact=%s&count=%d",
		url.QueryEscape(huntID), url.QueryEscape(artifact), limit)
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: get hunt results %s: %w", huntID, err)
	}
	return decodeRows(data, "velociraptor: decode hunt results")
}

// ---------------------------------------------------------------------------
// Artifact collection (per-client flows)
// ---------------------------------------------------------------------------

// CollectArtifact schedules an artifact collection on a specific client and
// returns the resulting flow metadata.
func (c *Client) CollectArtifact(ctx context.Context, req CollectRequest) (*FlowResult, error) {
	if c == nil {
		return nil, nil
	}

	// Build the request body matching Velociraptor's CollectArtifact API
	// envelope which expects specs with per-artifact parameters.
	specs := make([]map[string]interface{}, 0, len(req.Artifacts))
	for _, a := range req.Artifacts {
		spec := map[string]interface{}{
			"artifact": a,
		}
		if len(req.Parameters) > 0 {
			envList := make([]map[string]string, 0, len(req.Parameters))
			for k, v := range req.Parameters {
				envList = append(envList, map[string]string{"key": k, "value": v})
			}
			spec["parameters"] = map[string]interface{}{"env": envList}
		}
		specs = append(specs, spec)
	}

	body := map[string]interface{}{
		"client_id": req.ClientID,
		"artifacts": req.Artifacts,
		"specs":     specs,
	}
	if req.Urgent {
		body["urgent"] = true
	}
	if req.MaxRows > 0 {
		body["max_rows"] = req.MaxRows
	}

	data, err := c.doPost(ctx, "/api/v1/CollectArtifact", body)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: collect artifact on %s: %w", req.ClientID, err)
	}

	var flow FlowResult
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("velociraptor: decode collect artifact response: %w", err)
	}
	return &flow, nil
}

// GetFlowResults retrieves the result rows for a completed artifact collection
// flow on a specific client.
func (c *Client) GetFlowResults(ctx context.Context, clientID, flowID, artifact string) ([]VQLRow, error) {
	if c == nil {
		return nil, nil
	}
	path := fmt.Sprintf("/api/v1/GetTable?client_id=%s&flow_id=%s&artifact=%s",
		url.QueryEscape(clientID), url.QueryEscape(flowID), url.QueryEscape(artifact))
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: get flow results %s/%s: %w", clientID, flowID, err)
	}
	return decodeTableResponse(data, "velociraptor: decode flow results")
}

// ---------------------------------------------------------------------------
// Artifacts
// ---------------------------------------------------------------------------

// ListArtifacts returns available artifact definitions, optionally filtered by
// searchTerm (matched against artifact name and description).
func (c *Client) ListArtifacts(ctx context.Context, searchTerm string) ([]Artifact, error) {
	if c == nil {
		return nil, nil
	}
	path := "/api/v1/GetArtifacts"
	if searchTerm != "" {
		path += "?search_term=" + url.QueryEscape(searchTerm)
	}
	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: list artifacts: %w", err)
	}

	var envelope struct {
		Items []Artifact `json:"items"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("velociraptor: decode list artifacts response: %w", err)
	}
	return envelope.Items, nil
}

// ---------------------------------------------------------------------------
// VQL execution
// ---------------------------------------------------------------------------

// RunVQL executes a server-side VQL query and returns the result rows. The
// query runs under the credentials of the API key (org admin context).
func (c *Client) RunVQL(ctx context.Context, vql VQLRequest) ([]VQLRow, error) {
	if c == nil {
		return nil, nil
	}

	// Build the VQL query body. Velociraptor's /api/v1/VQLResponse expects
	// a JSON object with "query" (array of VQL statements) and optional "env".
	body := map[string]interface{}{
		"query": []map[string]string{
			{"vql": vql.Query},
		},
	}
	if len(vql.Env) > 0 {
		envList := make([]map[string]string, 0, len(vql.Env))
		for k, v := range vql.Env {
			envList = append(envList, map[string]string{"key": k, "value": v})
		}
		body["env"] = envList
	}

	data, err := c.doPost(ctx, "/api/v1/VQLResponse", body)
	if err != nil {
		return nil, fmt.Errorf("velociraptor: run VQL: %w", err)
	}
	return decodeRows(data, "velociraptor: decode VQL response")
}

// ---------------------------------------------------------------------------
// Health check
// ---------------------------------------------------------------------------

// Health checks Velociraptor server connectivity by requesting the server
// status endpoint. Returns nil on success.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return nil
	}
	_, err := c.doGet(ctx, "/api/v1/GetServerMonitoringState")
	if err != nil {
		return fmt.Errorf("velociraptor: health check: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

// Close is a nil-safe no-op provided for interface symmetry. The underlying
// http.Client does not hold persistent resources requiring explicit release.
func (c *Client) Close() {}

// ---------------------------------------------------------------------------
// Decode helpers
// ---------------------------------------------------------------------------

// decodeRows unmarshals a bare JSON array of objects into []VQLRow.
func decodeRows(data []byte, errPrefix string) ([]VQLRow, error) {
	var rows []VQLRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return rows, nil
}

// decodeTableResponse unmarshals Velociraptor's GetTable envelope which
// returns rows as a columnar structure with "columns" and "rows" fields.
// Falls back to decodeRows if the response is a plain array.
func decodeTableResponse(data []byte, errPrefix string) ([]VQLRow, error) {
	// Try the columnar envelope first.
	var table struct {
		Columns []string   `json:"columns"`
		Rows    [][]string `json:"rows"`
	}
	if err := json.Unmarshal(data, &table); err == nil && len(table.Columns) > 0 {
		rows := make([]VQLRow, 0, len(table.Rows))
		for _, row := range table.Rows {
			m := make(VQLRow, len(table.Columns))
			for i, col := range table.Columns {
				if i < len(row) {
					m[col] = tryParseJSON(row[i])
				}
			}
			rows = append(rows, m)
		}
		return rows, nil
	}

	// Fall back to plain array of objects.
	return decodeRows(data, errPrefix)
}

// tryParseJSON attempts to parse s as JSON (number, bool, object, array). If
// parsing fails the original string is returned.
func tryParseJSON(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Fast path: try numeric
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		// Preserve integers when possible
		if i, err2 := strconv.ParseInt(s, 10, 64); err2 == nil {
			return i
		}
		return n
	}

	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Attempt structured JSON (object / array)
	if (s[0] == '{' || s[0] == '[') && len(s) > 1 {
		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}

	return s
}
