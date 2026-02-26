// Package thehive provides a Go client for the TheHive 5.x REST API.
//
// TheHive is an AGPL-3.0-licensed incident response platform.
// This client communicates exclusively over HTTP (no code imports from TheHive)
// to maintain a clear AGPL-3.0 license boundary.
//
// Default endpoint: http://localhost:9000
// Auth: API key via "Authorization: Bearer {api-key}" header.
package thehive

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

// --------------------------------------------------------------------------
// Client
// --------------------------------------------------------------------------

// Client for TheHive 5.x REST API (AGPL-3.0 boundary: HTTP only).
type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// New creates a TheHive client.  If baseURL is empty the constructor returns
// (nil, nil) so callers can treat TheHive as an optional integration.
func New(baseURL, apiKey string) (*Client, error) {
	if baseURL == "" {
		return nil, nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed certs in lab
			},
		},
	}, nil
}

// Close is nil-safe and releases any resources held by the client.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.hc.CloseIdleConnections()
}

// --------------------------------------------------------------------------
// Domain types
// --------------------------------------------------------------------------

// Case represents a TheHive case.
type Case struct {
	ID           string                 `json:"_id,omitempty"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Severity     int                    `json:"severity"` // 1=Low, 2=Medium, 3=High, 4=Critical
	StartDate    int64                  `json:"startDate,omitempty"`
	TLP          int                    `json:"tlp"`    // 0=WHITE, 1=GREEN, 2=AMBER, 3=RED
	PAP          int                    `json:"pap"`    // 0=WHITE, 1=GREEN, 2=AMBER, 3=RED
	Status       string                 `json:"status"` // New, InProgress, Closed
	Tags         []string               `json:"tags,omitempty"`
	Owner        string                 `json:"owner,omitempty"`
	Flag         bool                   `json:"flag"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// Alert represents a TheHive alert that can be promoted to a case.
type Alert struct {
	ID          string       `json:"_id,omitempty"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Severity    int          `json:"severity"`
	Date        int64        `json:"date"`
	TLP         int          `json:"tlp"`
	PAP         int          `json:"pap"`
	Type        string       `json:"type"`      // "kubric-edr", "kubric-ndr", etc.
	Source      string       `json:"source"`     // "kubric"
	SourceRef   string       `json:"sourceRef"`  // unique event ID for dedup
	Status      string       `json:"status"`     // New, Updated, Ignored, Imported
	Tags        []string     `json:"tags,omitempty"`
	Observables []Observable `json:"observables,omitempty"`
}

// Observable represents an IOC/artifact attached to a case.
type Observable struct {
	DataType string   `json:"dataType"` // ip, domain, hash, filename, url, mail, etc.
	Data     string   `json:"data"`
	Message  string   `json:"message,omitempty"`
	TLP      int      `json:"tlp"`
	IOC      bool     `json:"ioc"`
	Sighted  bool     `json:"sighted"`
	Tags     []string `json:"tags,omitempty"`
}

// Task represents a case task.
type Task struct {
	ID          string `json:"_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"` // Waiting, InProgress, Completed, Cancel
	Group       string `json:"group,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Order       int    `json:"order"`
}

// CaseComment represents a comment on a case.
type CaseComment struct {
	Message string `json:"message"`
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// do builds, executes and decodes a JSON API request.
func (c *Client) do(ctx context.Context, method, path string, body, dst interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("thehive: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("thehive: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("thehive: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("thehive: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("thehive: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if dst != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dst); err != nil {
			return fmt.Errorf("thehive: decode response: %w", err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Alert operations
// --------------------------------------------------------------------------

// CreateAlert creates a new alert in TheHive.
// POST /api/v1/alert
func (c *Client) CreateAlert(ctx context.Context, alert Alert) (*Alert, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var created Alert
	if err := c.do(ctx, http.MethodPost, "/api/v1/alert", alert, &created); err != nil {
		return nil, fmt.Errorf("thehive: create alert: %w", err)
	}
	return &created, nil
}

// GetAlert retrieves an alert by ID.
// GET /api/v1/alert/{id}
func (c *Client) GetAlert(ctx context.Context, alertID string) (*Alert, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var alert Alert
	if err := c.do(ctx, http.MethodGet, "/api/v1/alert/"+alertID, nil, &alert); err != nil {
		return nil, fmt.Errorf("thehive: get alert %s: %w", alertID, err)
	}
	return &alert, nil
}

// PromoteAlert promotes an alert to a new case.
// POST /api/v1/alert/{id}/promote
func (c *Client) PromoteAlert(ctx context.Context, alertID string) (*Case, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var promoted Case
	if err := c.do(ctx, http.MethodPost, "/api/v1/alert/"+alertID+"/promote", struct{}{}, &promoted); err != nil {
		return nil, fmt.Errorf("thehive: promote alert %s: %w", alertID, err)
	}
	return &promoted, nil
}

// MergeAlertIntoCase merges an existing alert into an existing case.
// POST /api/v1/alert/{id}/merge/{caseId}
func (c *Client) MergeAlertIntoCase(ctx context.Context, alertID, caseID string) error {
	if c == nil {
		return fmt.Errorf("thehive: client is nil")
	}
	path := "/api/v1/alert/" + alertID + "/merge/" + caseID
	if err := c.do(ctx, http.MethodPost, path, struct{}{}, nil); err != nil {
		return fmt.Errorf("thehive: merge alert %s into case %s: %w", alertID, caseID, err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Case operations
// --------------------------------------------------------------------------

// CreateCase creates a new case in TheHive.
// POST /api/v1/case
func (c *Client) CreateCase(ctx context.Context, cas Case) (*Case, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var created Case
	if err := c.do(ctx, http.MethodPost, "/api/v1/case", cas, &created); err != nil {
		return nil, fmt.Errorf("thehive: create case: %w", err)
	}
	return &created, nil
}

// GetCase retrieves a case by ID.
// GET /api/v1/case/{id}
func (c *Client) GetCase(ctx context.Context, caseID string) (*Case, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var cas Case
	if err := c.do(ctx, http.MethodGet, "/api/v1/case/"+caseID, nil, &cas); err != nil {
		return nil, fmt.Errorf("thehive: get case %s: %w", caseID, err)
	}
	return &cas, nil
}

// UpdateCase patches specific fields on an existing case.
// PATCH /api/v1/case/{id}
func (c *Client) UpdateCase(ctx context.Context, caseID string, updates map[string]interface{}) error {
	if c == nil {
		return fmt.Errorf("thehive: client is nil")
	}
	if err := c.do(ctx, http.MethodPatch, "/api/v1/case/"+caseID, updates, nil); err != nil {
		return fmt.Errorf("thehive: update case %s: %w", caseID, err)
	}
	return nil
}

// searchQuery is the TheHive 5.x query DSL wrapper for _search endpoints.
type searchQuery struct {
	Query []map[string]interface{} `json:"query"`
}

// ListCases searches cases by status with pagination.
// POST /api/v1/case/_search
func (c *Client) ListCases(ctx context.Context, status string, limit, offset int) ([]Case, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}

	// Build TheHive 5.x query DSL
	query := searchQuery{
		Query: []map[string]interface{}{
			{"_name": "filter", "_field": "status", "_value": status},
			{"_name": "page", "from": offset, "to": offset + limit},
			{"_name": "sort", "_fields": []map[string]string{{"startDate": "desc"}}},
		},
	}

	var cases []Case
	if err := c.do(ctx, http.MethodPost, "/api/v1/case/_search", query, &cases); err != nil {
		return nil, fmt.Errorf("thehive: list cases (status=%s): %w", status, err)
	}
	return cases, nil
}

// --------------------------------------------------------------------------
// Observable operations
// --------------------------------------------------------------------------

// AddObservable adds an observable/artifact to a case.
// POST /api/v1/case/{id}/observable
func (c *Client) AddObservable(ctx context.Context, caseID string, obs Observable) (*Observable, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var created Observable
	path := "/api/v1/case/" + caseID + "/observable"
	if err := c.do(ctx, http.MethodPost, path, obs, &created); err != nil {
		return nil, fmt.Errorf("thehive: add observable to case %s: %w", caseID, err)
	}
	return &created, nil
}

// ListObservables returns all observables for a case.
// POST /api/v1/case/{id}/observable/_search
func (c *Client) ListObservables(ctx context.Context, caseID string) ([]Observable, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}

	query := searchQuery{
		Query: []map[string]interface{}{
			{"_name": "page", "from": 0, "to": 1000},
		},
	}

	var observables []Observable
	path := "/api/v1/case/" + caseID + "/observable/_search"
	if err := c.do(ctx, http.MethodPost, path, query, &observables); err != nil {
		return nil, fmt.Errorf("thehive: list observables for case %s: %w", caseID, err)
	}
	return observables, nil
}

// --------------------------------------------------------------------------
// Task operations
// --------------------------------------------------------------------------

// CreateTask creates a task within a case.
// POST /api/v1/case/{id}/task
func (c *Client) CreateTask(ctx context.Context, caseID string, task Task) (*Task, error) {
	if c == nil {
		return nil, fmt.Errorf("thehive: client is nil")
	}
	var created Task
	path := "/api/v1/case/" + caseID + "/task"
	if err := c.do(ctx, http.MethodPost, path, task, &created); err != nil {
		return nil, fmt.Errorf("thehive: create task in case %s: %w", caseID, err)
	}
	return &created, nil
}

// UpdateTask patches specific fields on an existing task.
// PATCH /api/v1/task/{id}
func (c *Client) UpdateTask(ctx context.Context, taskID string, updates map[string]interface{}) error {
	if c == nil {
		return fmt.Errorf("thehive: client is nil")
	}
	if err := c.do(ctx, http.MethodPatch, "/api/v1/task/"+taskID, updates, nil); err != nil {
		return fmt.Errorf("thehive: update task %s: %w", taskID, err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Comment operations
// --------------------------------------------------------------------------

// AddComment adds a comment/log entry to a case.
// POST /api/v1/case/{id}/comment
func (c *Client) AddComment(ctx context.Context, caseID string, comment CaseComment) error {
	if c == nil {
		return fmt.Errorf("thehive: client is nil")
	}
	path := "/api/v1/case/" + caseID + "/comment"
	if err := c.do(ctx, http.MethodPost, path, comment, nil); err != nil {
		return fmt.Errorf("thehive: add comment to case %s: %w", caseID, err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Health check
// --------------------------------------------------------------------------

// Health checks connectivity to the TheHive instance.
// GET /api/v1/status
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("thehive: client is nil")
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/status", nil, nil); err != nil {
		return fmt.Errorf("thehive: health check failed: %w", err)
	}
	return nil
}
