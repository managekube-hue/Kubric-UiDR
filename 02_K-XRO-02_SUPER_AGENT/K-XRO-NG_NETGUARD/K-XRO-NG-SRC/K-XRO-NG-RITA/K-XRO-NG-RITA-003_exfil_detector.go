// K-XRO-NG-RITA-003 — RITA data exfiltration detector (HTTP client).
//
// RITA is GPL-3.0.  This file communicates via HTTP REST only.
//
// Data exfiltration detection identifies internal hosts sending unusually
// large amounts of data to external destinations over many connections.
// RITA flags pairs where the bytes-sent / bytes-received ratio is high
// (outbound-heavy) combined with a high connection count.
//
// Note: RITA's exfil analysis differs from its long-connection analysis.
// Long-connection analysis focuses on slow-and-low exfil; the exfil module
// focuses on bulk transfer indicators within the analysis window.

package rita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ExfilResult holds the exfiltration analysis output for one src/dst pair.
type ExfilResult struct {
	// SrcIP is the internal host suspected of exfiltrating data.
	SrcIP string `json:"src_ip"`
	// DstIP is the external destination receiving the data.
	DstIP string `json:"dst_ip"`
	// BytesSent is the total bytes sent by SrcIP to DstIP in the window.
	BytesSent int64 `json:"bytes_sent"`
	// BytesReceived is the total bytes received by SrcIP from DstIP.
	BytesReceived int64 `json:"bytes_received"`
	// Connections is the number of observed connections in the window.
	Connections int `json:"connections"`
	// Score is the RITA exfil composite score, 0.0–1.0.
	Score float64 `json:"score"`
	// TenantID is the Kubric tenant owning this detection.
	TenantID string `json:"tenant_id"`
	// DurationSecs is the total span of activity in the analysis window.
	DurationSecs int64 `json:"duration_secs,omitempty"`
}

// SendRecvRatio returns bytes_sent / bytes_received (0 if bytes_received == 0).
func (r *ExfilResult) SendRecvRatio() float64 {
	if r.BytesReceived == 0 {
		return 0
	}
	return float64(r.BytesSent) / float64(r.BytesReceived)
}

// ExfilDetector wraps a RITA HTTP Client for exfiltration queries.
type ExfilDetector struct {
	client *Client
}

// NewExfilDetector returns an ExfilDetector using the provided Client.
func NewExfilDetector(client *Client) *ExfilDetector {
	return &ExfilDetector{client: client}
}

// GetExfil returns all exfiltration detections for tenantID.
//
// Calls GET /api/v1/exfil?tenant={tenantID}
func (d *ExfilDetector) GetExfil(
	ctx context.Context,
	tenantID string,
) ([]ExfilResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/exfil", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create exfil request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get exfil: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get exfil: unexpected status %d", resp.StatusCode)
	}

	var results []ExfilResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode exfil results: %w", err)
	}

	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}

// CheckPair checks whether the srcIP → dstIP pair exhibits exfiltration
// behaviour for the given tenant.
//
// Calls GET /api/v1/exfil/check?tenant={tenantID}&src={srcIP}&dst={dstIP}
//
// Returns (isExfil, score, error).
func (d *ExfilDetector) CheckPair(
	ctx context.Context,
	srcIP, dstIP, tenantID string,
) (bool, float64, error) {
	endpoint := fmt.Sprintf("%s/api/v1/exfil/check", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, 0, fmt.Errorf("create exfil check request: %w", err)
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("src", srcIP)
	q.Set("dst", dstIP)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("exfil check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		return false, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("exfil check: unexpected status %d", resp.StatusCode)
	}

	var result ExfilResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("decode exfil check result: %w", err)
	}

	return result.Score >= ExfilThreshold, result.Score, nil
}

// GetTopExfil returns the top-N exfiltration pairs ordered by bytes_sent.
//
// Calls GET /api/v1/exfil/top?tenant={tenantID}&n={n}&sort={sort}
func (d *ExfilDetector) GetTopExfil(
	ctx context.Context,
	tenantID string,
	n int,
	sortBy string, // "bytes_sent" | "score" | "connections"
) ([]ExfilResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/exfil/top", d.client.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create top-exfil request: %w", err)
	}

	if sortBy == "" {
		sortBy = "bytes_sent"
	}

	q := url.Values{}
	q.Set("tenant", tenantID)
	q.Set("n", strconv.Itoa(n))
	q.Set("sort", sortBy)
	req.URL.RawQuery = q.Encode()

	d.client.setHeaders(req)

	resp, err := d.client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get top exfil: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get top exfil: status %d", resp.StatusCode)
	}

	var results []ExfilResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode top exfil: %w", err)
	}

	for i := range results {
		if results[i].TenantID == "" {
			results[i].TenantID = tenantID
		}
	}
	return results, nil
}

// ExfilThreshold is the minimum RITA exfil score to classify a pair as
// exfiltrating data.
const ExfilThreshold = 0.75
