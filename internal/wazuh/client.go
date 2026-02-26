// Package wazuh implements a production HTTP client for the Wazuh Manager
// REST API (port 55000). It uses JWT authentication and supports agent
// management, alert retrieval, SCA compliance checks, FIM/syscheck, and
// active-response operations.
package wazuh

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Client is a thread-safe HTTP client for the Wazuh Manager REST API.
// A nil *Client is safe to call; every exported method returns immediately.
type Client struct {
	baseURL  string
	username string
	password string
	hc       *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// Agent represents a Wazuh agent registration.
type Agent struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	IP            string   `json:"ip"`
	Status        string   `json:"status"` // active, disconnected, never_connected, pending
	OS            *AgentOS `json:"os,omitempty"`
	Version       string   `json:"version"`
	LastKeepAlive string   `json:"lastKeepAlive"`
	Group         []string `json:"group"`
	NodeName      string   `json:"node_name"`
}

// AgentOS describes an agent's operating system.
type AgentOS struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Arch     string `json:"arch"`
}

// Alert represents a Wazuh alert event.
type Alert struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Rule      AlertRule              `json:"rule"`
	Agent     AlertAgent             `json:"agent"`
	Data      map[string]interface{} `json:"data,omitempty"`
	FullLog   string                 `json:"full_log,omitempty"`
}

// AlertRule contains rule metadata for an alert.
type AlertRule struct {
	ID          string   `json:"id"`
	Level       int      `json:"level"`
	Description string   `json:"description"`
	Groups      []string `json:"groups"`
	MITRE       *MITRE   `json:"mitre,omitempty"`
}

// MITRE holds MITRE ATT&CK enrichment data.
type MITRE struct {
	ID        []string `json:"id"`
	Tactic    []string `json:"tactic"`
	Technique []string `json:"technique"`
}

// AlertAgent identifies the agent that generated an alert.
type AlertAgent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// SCACheck represents a single Security Configuration Assessment check result.
type SCACheck struct {
	ID          int             `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Rationale   string          `json:"rationale"`
	Result      string          `json:"result"` // passed, failed, not_applicable
	Remediation string          `json:"remediation"`
	PolicyID    string          `json:"policy_id"`
	Compliance  []SCACompliance `json:"compliance,omitempty"`
}

// SCACompliance maps a compliance framework key to a value (e.g. "cis" -> "1.1.1").
type SCACompliance struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SCAPolicy represents aggregate results for an SCA policy on an agent.
type SCAPolicy struct {
	PolicyID    string `json:"policy_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pass        int    `json:"pass"`
	Fail        int    `json:"fail"`
	Invalid     int    `json:"invalid"`
	Score       int    `json:"score"`
}

// ActiveResponseRequest describes an active-response command to send to an agent.
type ActiveResponseRequest struct {
	Command   string                 `json:"command"`
	Arguments []string               `json:"arguments,omitempty"`
	Alert     map[string]interface{} `json:"alert,omitempty"`
}

// SyscheckFile represents a file integrity monitoring (FIM) entry.
type SyscheckFile struct {
	File   string `json:"file"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	Perm   string `json:"perm"`
	UID    string `json:"uid"`
	GID    string `json:"gid"`
	User   string `json:"uname"`
	Group  string `json:"gname"`
	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	Date   string `json:"date"`
	MTime  string `json:"mtime"`
}

// apiResponse is the generic envelope returned by all Wazuh REST endpoints.
type apiResponse struct {
	Data struct {
		AffectedItems      json.RawMessage `json:"affected_items"`
		TotalAffectedItems int             `json:"total_affected_items"`
	} `json:"data"`
	Message string `json:"message"`
	Error   int    `json:"error"`
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// New creates a Wazuh API client. If baseURL is empty the integration is
// considered disabled and (nil, nil) is returned. The caller must validate
// that username and password are non-empty when baseURL is provided.
func New(baseURL, username, password string) (*Client, error) {
	if baseURL == "" {
		return nil, nil // optional integration
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("wazuh: invalid base URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("wazuh: unsupported scheme %q (need http or https)", u.Scheme)
	}

	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Wazuh ships with self-signed certs
				},
				MaxIdleConns:        20,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

// Close releases idle transport connections. Safe to call on a nil receiver.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.hc.CloseIdleConnections()
}

// ---------------------------------------------------------------------------
// Authentication
// ---------------------------------------------------------------------------

// authenticate obtains or reuses a JWT token. Tokens are cached until 55 min
// after issue (Wazuh tokens expire at 60 min by default).
func (c *Client) authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExp) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/security/user/authenticate", nil)
	if err != nil {
		return fmt.Errorf("wazuh: auth request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("wazuh: auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("wazuh: auth failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("wazuh: auth decode: %w", err)
	}
	if result.Data.Token == "" {
		return fmt.Errorf("wazuh: auth returned empty token")
	}

	c.token = result.Data.Token
	c.tokenExp = time.Now().Add(55 * time.Minute)
	return nil
}

// ---------------------------------------------------------------------------
// Internal HTTP helpers
// ---------------------------------------------------------------------------

// doGet performs an authenticated GET and returns the raw body.
func (c *Client) doGet(ctx context.Context, path string) ([]byte, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wazuh: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wazuh: %s returned HTTP %d: %s", path, resp.StatusCode, string(body))
	}

	return body, nil
}

// doPost performs an authenticated POST with a JSON body.
func (c *Client) doPost(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("wazuh: marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wazuh: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wazuh: %s returned HTTP %d: %s", path, resp.StatusCode, string(body))
	}

	return body, nil
}

// doPut performs an authenticated PUT with a JSON body.
func (c *Client) doPut(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("wazuh: marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wazuh: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wazuh: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wazuh: %s returned HTTP %d: %s", path, resp.StatusCode, string(body))
	}

	return body, nil
}

// decodeItems unmarshals the affected_items array from an apiResponse envelope.
func decodeItems[T any](raw []byte) ([]T, int, error) {
	var resp apiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, 0, fmt.Errorf("wazuh: decode envelope: %w", err)
	}
	if resp.Error != 0 {
		return nil, 0, fmt.Errorf("wazuh: API error %d: %s", resp.Error, resp.Message)
	}

	var items []T
	if resp.Data.AffectedItems != nil {
		if err := json.Unmarshal(resp.Data.AffectedItems, &items); err != nil {
			return nil, 0, fmt.Errorf("wazuh: decode items: %w", err)
		}
	}
	return items, resp.Data.TotalAffectedItems, nil
}

// ---------------------------------------------------------------------------
// Agent Methods
// ---------------------------------------------------------------------------

// ListAgents returns agents, optionally filtered by status (e.g. "active").
// Pass an empty status string to list all agents. limit and offset control
// pagination; use limit=0 for the server default (500).
func (c *Client) ListAgents(ctx context.Context, status string, limit, offset int) ([]Agent, int, error) {
	if c == nil {
		return nil, 0, nil
	}

	path := "/agents?"
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	path += params.Encode()

	raw, err := c.doGet(ctx, path)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list agents: %w", err)
	}

	agents, total, err := decodeItems[Agent](raw)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list agents: %w", err)
	}
	return agents, total, nil
}

// GetAgent returns a single agent by ID.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	if c == nil {
		return nil, nil
	}

	raw, err := c.doGet(ctx, fmt.Sprintf("/agents?agents_list=%s", url.QueryEscape(agentID)))
	if err != nil {
		return nil, fmt.Errorf("wazuh: get agent %s: %w", agentID, err)
	}

	agents, _, err := decodeItems[Agent](raw)
	if err != nil {
		return nil, fmt.Errorf("wazuh: get agent %s: %w", agentID, err)
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("wazuh: agent %s not found", agentID)
	}
	return &agents[0], nil
}

// RestartAgent sends a restart command to a specific agent.
func (c *Client) RestartAgent(ctx context.Context, agentID string) error {
	if c == nil {
		return nil
	}

	path := fmt.Sprintf("/agents/%s/restart", url.PathEscape(agentID))
	raw, err := c.doPut(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("wazuh: restart agent %s: %w", agentID, err)
	}

	var resp apiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("wazuh: restart agent decode: %w", err)
	}
	if resp.Error != 0 {
		return fmt.Errorf("wazuh: restart agent %s: API error %d: %s", agentID, resp.Error, resp.Message)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Alert Methods
// ---------------------------------------------------------------------------

// ListAlerts returns alerts, optionally filtered by agent ID and minimum rule
// level. Pass agentID="" and minLevel=0 to skip those filters.
func (c *Client) ListAlerts(ctx context.Context, agentID string, minLevel int, limit, offset int) ([]Alert, int, error) {
	if c == nil {
		return nil, 0, nil
	}

	params := url.Values{}
	if agentID != "" {
		params.Set("agents_list", agentID)
	}
	if minLevel > 0 {
		params.Set("min_level", fmt.Sprintf("%d", minLevel))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}

	path := "/alerts?" + params.Encode()
	raw, err := c.doGet(ctx, path)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list alerts: %w", err)
	}

	alerts, total, err := decodeItems[Alert](raw)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list alerts: %w", err)
	}
	return alerts, total, nil
}

// ---------------------------------------------------------------------------
// SCA Methods
// ---------------------------------------------------------------------------

// ListSCAPolicies returns SCA policy summaries for the given agent.
func (c *Client) ListSCAPolicies(ctx context.Context, agentID string) ([]SCAPolicy, error) {
	if c == nil {
		return nil, nil
	}

	path := fmt.Sprintf("/sca/%s", url.PathEscape(agentID))
	raw, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("wazuh: list SCA policies for agent %s: %w", agentID, err)
	}

	policies, _, err := decodeItems[SCAPolicy](raw)
	if err != nil {
		return nil, fmt.Errorf("wazuh: list SCA policies decode: %w", err)
	}
	return policies, nil
}

// ListSCAChecks returns individual SCA check results for an agent's policy.
func (c *Client) ListSCAChecks(ctx context.Context, agentID, policyID string) ([]SCACheck, error) {
	if c == nil {
		return nil, nil
	}

	path := fmt.Sprintf("/sca/%s/checks/%s",
		url.PathEscape(agentID), url.PathEscape(policyID))
	raw, err := c.doGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("wazuh: list SCA checks for agent %s policy %s: %w", agentID, policyID, err)
	}

	checks, _, err := decodeItems[SCACheck](raw)
	if err != nil {
		return nil, fmt.Errorf("wazuh: list SCA checks decode: %w", err)
	}
	return checks, nil
}

// ---------------------------------------------------------------------------
// FIM / Syscheck Methods
// ---------------------------------------------------------------------------

// ListSyscheckFiles returns file integrity monitoring results for an agent.
func (c *Client) ListSyscheckFiles(ctx context.Context, agentID string, limit, offset int) ([]SyscheckFile, int, error) {
	if c == nil {
		return nil, 0, nil
	}

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}

	path := fmt.Sprintf("/syscheck/%s?%s", url.PathEscape(agentID), params.Encode())
	raw, err := c.doGet(ctx, path)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list syscheck files for agent %s: %w", agentID, err)
	}

	files, total, err := decodeItems[SyscheckFile](raw)
	if err != nil {
		return nil, 0, fmt.Errorf("wazuh: list syscheck decode: %w", err)
	}
	return files, total, nil
}

// ---------------------------------------------------------------------------
// Active Response
// ---------------------------------------------------------------------------

// RunActiveResponse sends an active-response command to the specified agent.
func (c *Client) RunActiveResponse(ctx context.Context, agentID string, ar ActiveResponseRequest) error {
	if c == nil {
		return nil
	}

	path := fmt.Sprintf("/active-response?agents_list=%s", url.QueryEscape(agentID))
	raw, err := c.doPut(ctx, path, ar)
	if err != nil {
		return fmt.Errorf("wazuh: active response on agent %s: %w", agentID, err)
	}

	var resp apiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("wazuh: active response decode: %w", err)
	}
	if resp.Error != 0 {
		return fmt.Errorf("wazuh: active response on agent %s: API error %d: %s",
			agentID, resp.Error, resp.Message)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

// Health checks whether the Wazuh Manager API is reachable and responding.
// Safe to call on a nil receiver (returns nil).
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return nil
	}

	raw, err := c.doGet(ctx, "/manager/status")
	if err != nil {
		return fmt.Errorf("wazuh: health check: %w", err)
	}

	var resp apiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("wazuh: health decode: %w", err)
	}
	if resp.Error != 0 {
		return fmt.Errorf("wazuh: health check: API error %d: %s", resp.Error, resp.Message)
	}
	return nil
}
