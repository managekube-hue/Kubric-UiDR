// Package noc provides NOC operations tooling.
// K-NOC-INV-002 — FleetDM Policy Management: manage FleetDM host policies via REST API.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// FleetHost represents a host enrolled in FleetDM.
type FleetHost struct {
	ID           int       `json:"id"`
	Hostname     string    `json:"hostname"`
	UUID         string    `json:"uuid"`
	Platform     string    `json:"platform"`
	OSVersion    string    `json:"os_version"`
	LastEnrolled time.Time `json:"last_enrolled_at"`
	Memory       int64     `json:"memory"`
	CPUBrand     string    `json:"cpu_brand"`
}

// FleetPolicy represents a FleetDM policy query applied to hosts.
type FleetPolicy struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Query            string `json:"query"`
	Platform         string `json:"platform"`
	PassingHostCount int    `json:"passing_host_count"`
	FailingHostCount int    `json:"failing_host_count"`
}

// PolicyHostResult holds the pass/fail result for a single host against a policy.
type PolicyHostResult struct {
	HostID   int    `json:"host_id"`
	Hostname string `json:"hostname"`
	Response string `json:"response"` // "pass" or "fail"
}

// FleetDMClient manages FleetDM resources via the REST API.
type FleetDMClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewFleetDMClient reads FLEETDM_URL and FLEETDM_TOKEN from the environment.
func NewFleetDMClient() *FleetDMClient {
	baseURL := os.Getenv("FLEETDM_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &FleetDMClient{
		BaseURL:    baseURL,
		Token:      os.Getenv("FLEETDM_TOKEN"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *FleetDMClient) authHeader() string {
	return "Bearer " + f.Token
}

func (f *FleetDMClient) doJSON(ctx context.Context, method, path string, body interface{}) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, f.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", f.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FleetDM %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respData, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("FleetDM %s %s returned %d: %s", method, path, resp.StatusCode, respData)
	}
	return json.RawMessage(respData), nil
}

// ListHosts returns a page of FleetDM hosts.
func (f *FleetDMClient) ListHosts(ctx context.Context, page, perPage int) ([]FleetHost, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	path := "/api/v1/fleet/hosts?" + query.Encode()

	raw, err := f.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Hosts []FleetHost `json:"hosts"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse hosts response: %w", parseErr)
	}
	return envelope.Hosts, nil
}

// GetHost retrieves a single host by ID.
func (f *FleetDMClient) GetHost(ctx context.Context, hostID int) (*FleetHost, error) {
	raw, err := f.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Host FleetHost `json:"host"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse host response: %w", parseErr)
	}
	return &envelope.Host, nil
}

// ListPolicies returns all global FleetDM policies.
func (f *FleetDMClient) ListPolicies(ctx context.Context) ([]FleetPolicy, error) {
	raw, err := f.doJSON(ctx, http.MethodGet, "/api/v1/fleet/global/policies", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Policies []FleetPolicy `json:"policies"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse policies response: %w", parseErr)
	}
	return envelope.Policies, nil
}

// CreatePolicy creates a new FleetDM global policy.
func (f *FleetDMClient) CreatePolicy(ctx context.Context, name, query, platform string) (*FleetPolicy, error) {
	payload := map[string]string{
		"name":     name,
		"query":    query,
		"platform": platform,
	}
	raw, err := f.doJSON(ctx, http.MethodPost, "/api/v1/fleet/global/policies", payload)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Policy FleetPolicy `json:"policy"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse create policy response: %w", parseErr)
	}
	return &envelope.Policy, nil
}

// GetPolicyResults returns the pass/fail status for every host against a policy.
func (f *FleetDMClient) GetPolicyResults(ctx context.Context, policyID int) ([]PolicyHostResult, error) {
	path := fmt.Sprintf("/api/v1/fleet/global/policies/%d/results", policyID)
	raw, err := f.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Results []PolicyHostResult `json:"results"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse policy results: %w", parseErr)
	}
	return envelope.Results, nil
}
