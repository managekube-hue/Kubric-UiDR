// Package shuffle provides a REST client for the Shuffle SOAR platform
// (AGPL-3.0).  It manages workflows, triggers executions, and polls for
// execution results via the Shuffle HTTP API (v1).
//
// API reference: https://shuffler.io/docs/api
package shuffle

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

// Client talks to a Shuffle SOAR instance over its REST API.
type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// Workflow represents a Shuffle automation workflow.
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsValid     bool      `json:"is_valid"`
	Status      string    `json:"status"` // running, stopped
	Actions     []Action  `json:"actions,omitempty"`
	Triggers    []Trigger `json:"triggers,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// Action is a single step inside a Shuffle workflow.
type Action struct {
	ID       string             `json:"id"`
	AppName  string             `json:"app_name"`
	AppID    string             `json:"app_id"`
	Name     string             `json:"name"`
	Position map[string]float64 `json:"position"`
}

// Trigger describes a webhook, schedule, user-input gate, or email trigger
// that can start a workflow.
type Trigger struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"trigger_type"` // WEBHOOK, SCHEDULE, USERINPUT, EMAIL
	Status string `json:"status"`
}

// Execution tracks a single run of a workflow.
type Execution struct {
	ID                string `json:"execution_id"`
	WorkflowID        string `json:"workflow_id"`
	Status            string `json:"status"` // EXECUTING, FINISHED, ABORTED
	StartedAt         int64  `json:"started_at"`
	CompletedAt       int64  `json:"completed_at"`
	ExecutionArgument string `json:"execution_argument"`
	Result            string `json:"result"`
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// New creates a Shuffle client.  baseURL is the Shuffle server root
// (e.g. "https://shuffle.internal:3443"), apiKey is a Shuffle API key.
// Returns nil, nil when baseURL is empty (Shuffle integration is optional).
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
		return nil, fmt.Errorf("shuffle: %s: %w", label, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shuffle: %s read body: %w", label, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("shuffle: %s: status %d: %s", label, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Workflow methods
// ---------------------------------------------------------------------------

// ListWorkflows returns all workflows visible to the authenticated API key.
func (c *Client) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	if c == nil {
		return nil, nil
	}

	req, err := c.newReq(ctx, http.MethodGet, "/api/v1/workflows", nil)
	if err != nil {
		return nil, fmt.Errorf("shuffle: list workflows build request: %w", err)
	}

	data, err := c.do(req, "list workflows")
	if err != nil {
		return nil, err
	}

	var workflows []Workflow
	if err := json.Unmarshal(data, &workflows); err != nil {
		return nil, fmt.Errorf("shuffle: list workflows decode: %w", err)
	}
	return workflows, nil
}

// GetWorkflow returns a single workflow by ID.
func (c *Client) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	if c == nil {
		return nil, nil
	}

	path := "/api/v1/workflows/" + workflowID
	req, err := c.newReq(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("shuffle: get workflow build request: %w", err)
	}

	data, err := c.do(req, "get workflow")
	if err != nil {
		return nil, err
	}

	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("shuffle: get workflow decode: %w", err)
	}
	return &wf, nil
}

// ---------------------------------------------------------------------------
// Execution methods
// ---------------------------------------------------------------------------

// executePayload is the POST body for workflow execution.
type executePayload struct {
	ExecutionArgument string                 `json:"execution_argument"`
	StartNode         string                 `json:"start,omitempty"`
}

// ExecuteWorkflow triggers a workflow run.  argument is passed as JSON to the
// workflow's start node.  Returns the Execution metadata (initially status
// EXECUTING).
func (c *Client) ExecuteWorkflow(ctx context.Context, workflowID string, argument map[string]interface{}) (*Execution, error) {
	if c == nil {
		return nil, nil
	}

	// Marshal the caller-supplied argument map into a JSON string that Shuffle
	// expects in execution_argument.
	argBytes, err := json.Marshal(argument)
	if err != nil {
		return nil, fmt.Errorf("shuffle: execute workflow marshal argument: %w", err)
	}

	payload, err := json.Marshal(executePayload{
		ExecutionArgument: string(argBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("shuffle: execute workflow marshal payload: %w", err)
	}

	path := "/api/v1/workflows/" + workflowID + "/execute"
	req, err := c.newReq(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("shuffle: execute workflow build request: %w", err)
	}

	data, err := c.do(req, "execute workflow")
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, fmt.Errorf("shuffle: execute workflow decode: %w", err)
	}
	return &exec, nil
}

// GetExecution retrieves the current state of a workflow execution.
func (c *Client) GetExecution(ctx context.Context, executionID string) (*Execution, error) {
	if c == nil {
		return nil, nil
	}

	// The Shuffle API uses /api/v1/streams/results with a POST body
	// containing the execution_id, but the simpler path also works for
	// individual execution lookup.
	path := "/api/v1/streams/results"
	payload, err := json.Marshal(map[string]string{
		"execution_id": executionID,
	})
	if err != nil {
		return nil, fmt.Errorf("shuffle: get execution marshal: %w", err)
	}

	req, err := c.newReq(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("shuffle: get execution build request: %w", err)
	}

	data, err := c.do(req, "get execution")
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, fmt.Errorf("shuffle: get execution decode: %w", err)
	}
	return &exec, nil
}

// AbortExecution requests cancellation of a running workflow execution.
func (c *Client) AbortExecution(ctx context.Context, executionID string) error {
	if c == nil {
		return nil
	}

	path := "/api/v1/streams/" + executionID + "/abort"
	req, err := c.newReq(ctx, http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("shuffle: abort execution build request: %w", err)
	}

	_, err = c.do(req, "abort execution")
	return err
}

// ---------------------------------------------------------------------------
// Health / lifecycle
// ---------------------------------------------------------------------------

// Health performs a lightweight connectivity check against the Shuffle server
// by hitting GET /api/v1/health.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return nil
	}

	req, err := c.newReq(ctx, http.MethodGet, "/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("shuffle: health build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("shuffle: health request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shuffle: health check returned %d", resp.StatusCode)
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
