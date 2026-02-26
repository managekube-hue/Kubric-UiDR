// K-XRO-NG-RITA-001 — RITA beacon detector (HTTP client).
//
// RITA (Real Intelligence Threat Analytics) is GPL-3.0.
// This file communicates with a RITA sidecar ONLY over HTTP REST.
// No Go imports from github.com/activecm/rita are used here.
//
// RITA REST API is assumed to run on KUBRIC_RITA_URL (default http://rita:4096).
//
// Beacon analysis identifies hosts communicating with C2 servers on regular
// intervals.  RITA calculates a composite score based on:
//   - ts_score:  regularity of connection timestamps (jitter coefficient)
//   - ds_score:  consistency of data sizes per connection
//   - dur_score: consistency of connection duration
//
// A composite score >= 0.8 is treated as high-confidence beaconing.

package rita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// BeaconResult holds the beacon analysis output for one src/dst pair.
type BeaconResult struct {
	// SrcIP is the internal host initiating connections.
	SrcIP string `json:"src_ip"`
	// DstIP is the external destination (potential C2 server).
	DstIP string `json:"dst_ip"`
	// Score is the composite RITA beacon score, 0.0-1.0.
	Score float64 `json:"score"`
	// Connections is the number of observed connections in the analysis window.
	Connections int `json:"connections"`
	// Interval is the average connection interval in seconds.
	Interval float64 `json:"interval"`
	// JitterPct is the coefficient of variation (stddev/mean) of connection
	// intervals as a percentage.  Low jitter means high confidence.
	JitterPct float64 `json:"jitter_pct"`
	// TenantID is the Kubric tenant owning this detection.
	TenantID string `json:"tenant_id"`
	// TSScore is the timestamp regularity sub-score.
	TSScore float64 `json:"ts_score,omitempty"`
	// DSScore is the data-size consistency sub-score.
	DSScore float64 `json:"ds_score,omitempty"`
	// DurScore is the duration consistency sub-score.
	DurScore float64 `json:"dur_score,omitempty"`
}

// BeaconThreshold is the minimum composite score to classify a pair as
// beaconing.
const BeaconThreshold = 0.8

// Client is the shared HTTP client used by all RITA detectors.
type Client struct {
	baseURL string
	http    *http.Client
	apiKey  string
}

// NewClient creates a RITA HTTP client pointing at baseURL.
// If apiKey is non-empty it is sent as the X-API-Key header.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// NewClientFromEnv creates a Client from environment variables:
//
//	KUBRIC_RITA_URL  — base URL (default http://rita:4096)
//	KUBRIC_RITA_KEY  — optional API key
func NewClientFromEnv() *Client {
	baseURL := lookupEnvDefault("KUBRIC_RITA_URL", "http://rita:4096")
	apiKey := lookupEnvDefault("KUBRIC_RITA_KEY", "")
	return NewClient(baseURL, apiKey)
}

// HealthCheck returns nil if the RITA API is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/api/v1/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("RITA health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RITA health check: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

// lookupEnvDefault returns the value of key, or defaultVal if not set.
func lookupEnvDefault(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

// BeaconDetector wraps a RITA HTTP Client and exposes beacon queries.
type BeaconDetector struct {
	client *Client
}

// NewBeaconDetector returns a BeaconDetector using the provided Client.
func NewBeaconDetector(client *Client) *BeaconDetector {
	return &BeaconDetector{client: client}
}

// GetBeacons returns all beacon pairs for tenantID with score >= minScore.
//
// Calls GET /api/v1/beacons?tenant={tenantID}&min_score={minScore}
func (d *BeaconDetector) GetBeacons(
	ctx context.Context,
	tenantID string,
	minScore float64,
) ([]BeaconResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/beacons", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create beacon request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("min_score", strconv.FormatFloat(minScore, 'f', 4, 64))
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get beacons: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get beacons: unexpected status %d", resp.StatusCode)
	}

	var results []BeaconResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode beacon results: %w", err)
	}

	// Stamp tenant_id if the RITA API omits it in the response
	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}

// IsBeaconing checks whether the srcIP to dstIP pair is classified as beaconing
// for the given tenant.
//
// Calls GET /api/v1/beacons/check?tenant={tenantID}&src={srcIP}&dst={dstIP}
//
// Returns (isBeaconing, score, error).
func (d *BeaconDetector) IsBeaconing(
	ctx context.Context,
	srcIP, dstIP, tenantID string,
) (bool, float64, error) {
	endpoint := fmt.Sprintf("%s/api/v1/beacons/check", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, 0, fmt.Errorf("create beacon check request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("src", srcIP)
	q.Set("dst", dstIP)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("beacon check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		// Pair not observed in analysis window
		return false, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("beacon check: unexpected status %d", resp.StatusCode)
	}

	var result BeaconResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("decode beacon check result: %w", err)
	}

	return result.Score >= BeaconThreshold, result.Score, nil
}

// GetTopBeacons returns the top-N beacon pairs ordered by score descending.
//
// Calls GET /api/v1/beacons/top?tenant={tenantID}&n={n}
func (d *BeaconDetector) GetTopBeacons(
	ctx context.Context,
	tenantID string,
	n int,
) ([]BeaconResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/beacons/top", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create top-beacons request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("n", strconv.Itoa(n))
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get top beacons: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get top beacons: status %d", resp.StatusCode)
	}

	var results []BeaconResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode top beacons: %w", err)
	}

	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}
