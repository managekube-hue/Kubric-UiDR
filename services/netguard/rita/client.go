// Package rita provides an HTTP client for the RITA (Real Intelligence Threat
// Analytics) beacon analysis sidecar.
//
// # GPL 3.0 hard boundary
//
// RITA is licensed under GPL 3.0.  This package communicates with the RITA
// Docker sidecar exclusively over HTTP/JSON.  NO imports from
// github.com/activecm/rita or any other GPL-licensed package are permitted in
// this file or any file in this package.  The check-gpl-boundary Makefile
// target enforces this at CI time.
package rita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ── Client ────────────────────────────────────────────────────────────────────

// Client is a thin HTTP wrapper around the RITA REST API.
// Construct with [New]; reuse across goroutines (safe for concurrent use).
type Client struct {
	baseURL string
	hc      *http.Client
}

// New returns a Client pointed at baseURL (e.g. "http://rita:4096").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		hc: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ── Result types ──────────────────────────────────────────────────────────────

// Beacon represents a RITA-detected C2 beacon connection.
type Beacon struct {
	SrcIP       string  `json:"src_ip"`
	DstIP       string  `json:"dst_ip"`
	Score       float64 `json:"score"`
	Connections int     `json:"connection_count"`
	TenantID    string  `json:"tenant_id"`
}

// DNSTunnel represents a RITA-detected DNS-based data exfiltration channel.
type DNSTunnel struct {
	FQDN     string  `json:"fqdn"`
	SrcIP    string  `json:"src_ip"`
	Score    float64 `json:"score"`
	TenantID string  `json:"tenant_id"`
}

// ExfilEvent represents a RITA-detected large-volume data exfiltration event.
type ExfilEvent struct {
	SrcIP     string  `json:"src_ip"`
	DstIP     string  `json:"dst_ip"`
	BytesSent int64   `json:"bytes_sent"`
	Score     float64 `json:"score"`
	TenantID  string  `json:"tenant_id"`
}

// ── API methods ───────────────────────────────────────────────────────────────

// GetBeacons fetches all detected C2 beacon connections for the given tenant.
//
// RITA endpoint: GET /v1/beacons?tenant_id={tenantID}
func (c *Client) GetBeacons(ctx context.Context, tenantID string) ([]Beacon, error) {
	url := fmt.Sprintf("%s/v1/beacons?tenant_id=%s", c.baseURL, tenantID)
	var out []Beacon
	if err := c.get(ctx, url, &out); err != nil {
		return nil, fmt.Errorf("rita.GetBeacons: %w", err)
	}
	return out, nil
}

// GetDNSTunneling fetches all detected DNS tunneling events for the given tenant.
//
// RITA endpoint: GET /v1/dns-tunneling?tenant_id={tenantID}
func (c *Client) GetDNSTunneling(ctx context.Context, tenantID string) ([]DNSTunnel, error) {
	url := fmt.Sprintf("%s/v1/dns-tunneling?tenant_id=%s", c.baseURL, tenantID)
	var out []DNSTunnel
	if err := c.get(ctx, url, &out); err != nil {
		return nil, fmt.Errorf("rita.GetDNSTunneling: %w", err)
	}
	return out, nil
}

// GetExfil fetches all detected data exfiltration events for the given tenant.
//
// RITA endpoint: GET /v1/exfil?tenant_id={tenantID}
func (c *Client) GetExfil(ctx context.Context, tenantID string) ([]ExfilEvent, error) {
	url := fmt.Sprintf("%s/v1/exfil?tenant_id=%s", c.baseURL, tenantID)
	var out []ExfilEvent
	if err := c.get(ctx, url, &out); err != nil {
		return nil, fmt.Errorf("rita.GetExfil: %w", err)
	}
	return out, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
