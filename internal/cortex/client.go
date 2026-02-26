// Package cortex provides a Go client for the Cortex 3.x REST API.
//
// Cortex is an AGPL-3.0-licensed observable analysis and active response engine.
// This client communicates exclusively over HTTP (no code imports from Cortex)
// to maintain a clear AGPL-3.0 license boundary.
//
// Default endpoint: http://localhost:9001
// Auth: API key via "Authorization: Bearer {api-key}" header.
package cortex

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

// Client for Cortex 3.x REST API (AGPL-3.0 boundary: HTTP only).
type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// New creates a Cortex client.  If baseURL is empty the constructor returns
// (nil, nil) so callers can treat Cortex as an optional integration.
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

// Analyzer represents a Cortex analyzer that can inspect observables.
type Analyzer struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	DataTypeList []string `json:"dataTypeList"` // ip, domain, hash, url, etc.
}

// Responder represents a Cortex responder that can execute active response actions.
type Responder struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	DataTypeList []string `json:"dataTypeList"`
}

// Job represents a Cortex analysis job.
type Job struct {
	ID         string                 `json:"id"`
	AnalyzerID string                 `json:"analyzerId"`
	Status     string                 `json:"status"` // Waiting, InProgress, Success, Failure
	Data       string                 `json:"data"`
	DataType   string                 `json:"dataType"`
	TLP        int                    `json:"tlp"`
	PAP        int                    `json:"pap"`
	Report     map[string]interface{} `json:"report,omitempty"`
	CreatedAt  int64                  `json:"createdAt"`
	StartDate  int64                  `json:"startDate"`
	EndDate    int64                  `json:"endDate"`
}

// AnalyzeRequest is the payload sent to run an analyzer on an observable.
type AnalyzeRequest struct {
	Data     string `json:"data"`
	DataType string `json:"dataType"` // ip, domain, hash, url, mail, filename
	TLP      int    `json:"tlp"`
	PAP      int    `json:"pap"`
	Message  string `json:"message,omitempty"`
}

// ResponderAction represents the result of running a responder.
type ResponderAction struct {
	ID          string                 `json:"id"`
	ResponderID string                 `json:"responderId"`
	Status      string                 `json:"status"`
	ObjectType  string                 `json:"objectType"`
	ObjectID    string                 `json:"objectId"`
	Report      map[string]interface{} `json:"report,omitempty"`
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
			return fmt.Errorf("cortex: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("cortex: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("cortex: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cortex: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cortex: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if dst != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dst); err != nil {
			return fmt.Errorf("cortex: decode response: %w", err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Analyzer operations
// --------------------------------------------------------------------------

// ListAnalyzers returns all analyzers that support the given data type.
// If dataType is empty, all analyzers are returned.
// GET /api/analyzer?dataType={dt}
func (c *Client) ListAnalyzers(ctx context.Context, dataType string) ([]Analyzer, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	path := "/api/analyzer"
	if dataType != "" {
		path += "?dataType=" + dataType
	}
	var analyzers []Analyzer
	if err := c.do(ctx, http.MethodGet, path, nil, &analyzers); err != nil {
		return nil, fmt.Errorf("cortex: list analyzers (dataType=%s): %w", dataType, err)
	}
	return analyzers, nil
}

// GetAnalyzer retrieves a single analyzer by ID.
// GET /api/analyzer/{id}
func (c *Client) GetAnalyzer(ctx context.Context, analyzerID string) (*Analyzer, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	var analyzer Analyzer
	if err := c.do(ctx, http.MethodGet, "/api/analyzer/"+analyzerID, nil, &analyzer); err != nil {
		return nil, fmt.Errorf("cortex: get analyzer %s: %w", analyzerID, err)
	}
	return &analyzer, nil
}

// Analyze submits an observable to an analyzer for inspection and returns
// the created job.
// POST /api/analyzer/{id}/run
func (c *Client) Analyze(ctx context.Context, analyzerID string, req AnalyzeRequest) (*Job, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	var job Job
	path := "/api/analyzer/" + analyzerID + "/run"
	if err := c.do(ctx, http.MethodPost, path, req, &job); err != nil {
		return nil, fmt.Errorf("cortex: analyze via %s: %w", analyzerID, err)
	}
	return &job, nil
}

// --------------------------------------------------------------------------
// Job operations
// --------------------------------------------------------------------------

// GetJob retrieves the current state of an analysis job.
// GET /api/job/{id}
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	var job Job
	if err := c.do(ctx, http.MethodGet, "/api/job/"+jobID, nil, &job); err != nil {
		return nil, fmt.Errorf("cortex: get job %s: %w", jobID, err)
	}
	return &job, nil
}

// WaitForJob polls a job until it reaches a terminal state (Success or Failure)
// or the timeout expires. It polls every 2 seconds.
func (c *Client) WaitForJob(ctx context.Context, jobID string, timeout time.Duration) (*Job, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		job, err := c.GetJob(ctx, jobID)
		if err != nil {
			return nil, fmt.Errorf("cortex: wait for job %s: %w", jobID, err)
		}
		switch job.Status {
		case "Success", "Failure":
			return job, nil
		}

		if time.Now().After(deadline) {
			return job, fmt.Errorf("cortex: wait for job %s: timed out after %s (status=%s)", jobID, timeout, job.Status)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("cortex: wait for job %s: %w", jobID, ctx.Err())
		case <-ticker.C:
			// continue polling
		}
	}
}

// GetJobReport retrieves the full report for a completed job.
// GET /api/job/{id}/report
func (c *Client) GetJobReport(ctx context.Context, jobID string) (map[string]interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	var report map[string]interface{}
	if err := c.do(ctx, http.MethodGet, "/api/job/"+jobID+"/report", nil, &report); err != nil {
		return nil, fmt.Errorf("cortex: get job report %s: %w", jobID, err)
	}
	return report, nil
}

// --------------------------------------------------------------------------
// Responder operations
// --------------------------------------------------------------------------

// ListResponders returns all responders that support the given data type.
// If dataType is empty, all responders are returned.
// GET /api/responder?dataType={dt}
func (c *Client) ListResponders(ctx context.Context, dataType string) ([]Responder, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	path := "/api/responder"
	if dataType != "" {
		path += "?dataType=" + dataType
	}
	var responders []Responder
	if err := c.do(ctx, http.MethodGet, path, nil, &responders); err != nil {
		return nil, fmt.Errorf("cortex: list responders (dataType=%s): %w", dataType, err)
	}
	return responders, nil
}

// responderRunRequest is the internal payload for POST /api/responder/{id}/run.
type responderRunRequest struct {
	ObjectType string                 `json:"objectType"`
	ObjectID   string                 `json:"objectId"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// RunResponder executes a responder action against a TheHive object
// (case, alert, task, etc.) identified by objectType and objectID.
// POST /api/responder/{id}/run
func (c *Client) RunResponder(ctx context.Context, responderID string, objectType, objectID string, params map[string]interface{}) (*ResponderAction, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	body := responderRunRequest{
		ObjectType: objectType,
		ObjectID:   objectID,
		Parameters: params,
	}
	var action ResponderAction
	path := "/api/responder/" + responderID + "/run"
	if err := c.do(ctx, http.MethodPost, path, body, &action); err != nil {
		return nil, fmt.Errorf("cortex: run responder %s: %w", responderID, err)
	}
	return &action, nil
}

// GetResponderAction retrieves the current state of a responder action.
// GET /api/responder/action/{id}
func (c *Client) GetResponderAction(ctx context.Context, actionID string) (*ResponderAction, error) {
	if c == nil {
		return nil, fmt.Errorf("cortex: client is nil")
	}
	var action ResponderAction
	if err := c.do(ctx, http.MethodGet, "/api/responder/action/"+actionID, nil, &action); err != nil {
		return nil, fmt.Errorf("cortex: get responder action %s: %w", actionID, err)
	}
	return &action, nil
}

// --------------------------------------------------------------------------
// Health check
// --------------------------------------------------------------------------

// Health checks connectivity to the Cortex instance.
// GET /api/status
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cortex: client is nil")
	}
	if err := c.do(ctx, http.MethodGet, "/api/status", nil, nil); err != nil {
		return fmt.Errorf("cortex: health check failed: %w", err)
	}
	return nil
}
