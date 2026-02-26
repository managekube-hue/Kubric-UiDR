// K-XRO-NG-RITA-002 — RITA DNS tunneling detector (HTTP client).
//
// RITA is GPL-3.0.  This file communicates via HTTP REST only.
//
// DNS tunneling uses DNS queries to exfiltrate data or establish covert C2
// channels.  Indicators include:
//   - High volume of queries to a single domain
//   - Unusually long subdomain labels (entropy-stuffed with base32/base64 data)
//   - Large number of unique subdomains (data fragments)
//   - High average query length
//
// RITA scores domains 0.0–1.0; scores >= 0.7 are classified as tunneling.

package rita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// DNSTunnelResult holds analysis output for one potentially tunneled FQDN.
type DNSTunnelResult struct {
	// FQDN is the root domain suspected of carrying tunnel traffic.
	FQDN string `json:"fqdn"`
	// SrcIP is the internal host making the suspicious DNS queries.
	SrcIP string `json:"src_ip"`
	// Score is the RITA DNS tunnel score, 0.0–1.0.
	Score float64 `json:"score"`
	// QueryCount is the total number of queries observed in the window.
	QueryCount int `json:"query_count"`
	// AvgQueryLen is the average total query length including subdomains.
	AvgQueryLen float64 `json:"avg_query_len"`
	// UniqueSubdomains is the count of distinct subdomain prefixes.
	UniqueSubdomains int `json:"unique_subdomains"`
	// TenantID is the Kubric tenant owning this detection.
	TenantID string `json:"tenant_id"`
}

// DNSTunnelDetector wraps a RITA HTTP Client for DNS tunnel queries.
type DNSTunnelDetector struct {
	client *Client
}

// NewDNSTunnelDetector returns a DNSTunnelDetector using the provided Client.
func NewDNSTunnelDetector(client *Client) *DNSTunnelDetector {
	return &DNSTunnelDetector{client: client}
}

// GetTunnels returns all DNS tunnel detections for tenantID.
//
// Calls GET /api/v1/dns/tunneling?tenant={tenantID}
func (d *DNSTunnelDetector) GetTunnels(
	ctx context.Context,
	tenantID string,
) ([]DNSTunnelResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/dns/tunneling", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create dns tunnel request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get dns tunnels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get dns tunnels: unexpected status %d", resp.StatusCode)
	}

	var results []DNSTunnelResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode dns tunnel results: %w", err)
	}

	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}

// CheckDomain checks whether fqdn is classified as a DNS tunnel carrier
// for the given tenant.
//
// Calls GET /api/v1/dns/tunneling/check?tenant={tenantID}&fqdn={fqdn}
//
// Returns (isTunnel, score, error).
func (d *DNSTunnelDetector) CheckDomain(
	ctx context.Context,
	fqdn, tenantID string,
) (bool, float64, error) {
	endpoint := fmt.Sprintf("%s/api/v1/dns/tunneling/check", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, 0, fmt.Errorf("create dns check request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("fqdn", fqdn)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("dns tunnel check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		return false, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("dns tunnel check: status %d", resp.StatusCode)
	}

	var result DNSTunnelResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("decode dns check result: %w", err)
	}

	return result.Score >= DNSTunnelThreshold, result.Score, nil
}

// GetTopTunnels returns the top-N DNS tunnel detections ordered by score.
//
// Calls GET /api/v1/dns/tunneling/top?tenant={tenantID}&n={n}&min_score={minScore}
func (d *DNSTunnelDetector) GetTopTunnels(
	ctx context.Context,
	tenantID string,
	n int,
	minScore float64,
) ([]DNSTunnelResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/dns/tunneling/top", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create top-tunnels request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("n", strconv.Itoa(n))
	q.Set("min_score", strconv.FormatFloat(minScore, 'f', 4, 64))
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get top dns tunnels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get top dns tunnels: status %d", resp.StatusCode)
	}

	var results []DNSTunnelResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode top tunnels: %w", err)
	}

	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}

// DNSTunnelThreshold is the minimum RITA dns-tunnel score to classify a
// domain as tunneling.
const DNSTunnelThreshold = 0.7
