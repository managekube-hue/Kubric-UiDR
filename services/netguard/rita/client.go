// Package rita provides an HTTP client for the RITA beacon analysis sidecar.
// RITA is GPL 3.0 — this package communicates via HTTP only.
// No imports from github.com/activecm/rita are permitted.
package rita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is an HTTP client for the RITA analysis sidecar REST API.
// RITA exposes results on port 4096. This client never imports RITA's Go packages
// to maintain GPL 3.0 license isolation.
type Client struct {
	baseURL string
	hc      *http.Client
}

// Beacon represents a detected periodic/beaconing connection from RITA analysis.
type Beacon struct {
	SrcIP       string  `json:"src"`
	DstIP       string  `json:"dst"`
	Score       float64 `json:"score"`
	Connections int     `json:"connection_count"`
	TenantID    string  `json:"tenant_id"`
}

// DNSTunnel represents a detected DNS tunneling session.
type DNSTunnel struct {
	FQDN     string  `json:"fqdn"`
	SrcIP    string  `json:"src"`
	Score    float64 `json:"score"`
	TenantID string  `json:"tenant_id"`
}

// ExfilEvent represents a detected data exfiltration event.
type ExfilEvent struct {
	SrcIP     string `json:"src"`
	DstIP     string `json:"dst"`
	BytesSent int64  `json:"bytes_sent"`
	TenantID  string `json:"tenant_id"`
}

// New creates a RITA HTTP client. baseURL should be e.g. "http://rita:4096".
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		hc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetBeacons fetches beacon analysis results for the given tenant from RITA.
func (c *Client) GetBeacons(ctx context.Context, tenantID string) ([]Beacon, error) {
	// GET {baseURL}/beacons?tenant_id={tenantID}
	url := fmt.Sprintf("%s/beacons?tenant_id=%s", c.baseURL, tenantID)
	var results []Beacon
	if err := c.get(ctx, url, &results); err != nil {
		return nil, fmt.Errorf("rita beacons: %w", err)
	}
	// Stamp tenant ID on each result
	for i := range results {
		results[i].TenantID = tenantID
	}
	return results, nil
}

// GetDNSTunneling fetches DNS tunneling detections for the given tenant.
func (c *Client) GetDNSTunneling(ctx context.Context, tenantID string) ([]DNSTunnel, error) {
	url := fmt.Sprintf("%s/dns-tunneling?tenant_id=%s", c.baseURL, tenantID)
	var results []DNSTunnel
	if err := c.get(ctx, url, &results); err != nil {
		return nil, fmt.Errorf("rita dns-tunneling: %w", err)
	}
	for i := range results {
		results[i].TenantID = tenantID
	}
	return results, nil
}

// GetExfil fetches data exfiltration detections for the given tenant.
func (c *Client) GetExfil(ctx context.Context, tenantID string) ([]ExfilEvent, error) {
	url := fmt.Sprintf("%s/exfil?tenant_id=%s", c.baseURL, tenantID)
	var results []ExfilEvent
	if err := c.get(ctx, url, &results); err != nil {
		return nil, fmt.Errorf("rita exfil: %w", err)
	}
	for i := range results {
		results[i].TenantID = tenantID
	}
	return results, nil
}

// get performs an HTTP GET request and JSON-decodes the response body into dest.
func (c *Client) get(ctx context.Context, url string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}
