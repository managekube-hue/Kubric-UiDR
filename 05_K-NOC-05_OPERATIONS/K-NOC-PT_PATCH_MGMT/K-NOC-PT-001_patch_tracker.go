//go:build ignore

// Package nocops provides NOC operations tooling.
// K-NOC-PT-001 — Patch Tracker: CVE-to-endpoint state machine with NVD enrichment
// and Wazuh alert correlation.
package nocops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PatchStatus describes the remediation state of a CVE on a given endpoint.
type PatchStatus string

const (
	// StatusPatchable means a patch is available but not yet applied.
	StatusPatchable PatchStatus = "patchable"
	// StatusPatched means the patch has been successfully applied.
	StatusPatched PatchStatus = "patched"
	// StatusMitigated means no patch exists but a compensating control is active.
	StatusMitigated PatchStatus = "mitigated"
	// StatusAccepted means the risk has been accepted and no action is required.
	StatusAccepted PatchStatus = "accepted"
	// StatusUnknown means the patch status cannot be determined yet.
	StatusUnknown PatchStatus = "unknown"
)

// CVERecord holds NVD-sourced vulnerability information enriched from the NVD
// REST API v2 (https://services.nvd.nist.gov/rest/json/cves/2.0).
type CVERecord struct {
	CVEID            string    `json:"cve_id"`
	CVSS             float64   `json:"cvss"`
	Severity         string    `json:"severity"`          // NONE, LOW, MEDIUM, HIGH, CRITICAL
	Description      string    `json:"description"`
	AffectedSoftware []string  `json:"affected_software"` // CPE URIs
	PatchAvailable   bool      `json:"patch_available"`
	PatchVersion     string    `json:"patch_version,omitempty"`
	PublishedAt      time.Time `json:"published_at"`
}

// EndpointPatchState links a CVE record to the patch state on a specific asset.
type EndpointPatchState struct {
	AssetID      string      `json:"asset_id"`
	TenantID     string      `json:"tenant_id"`
	CVEID        string      `json:"cve_id"`
	Status       PatchStatus `json:"status"`
	DetectedAt   time.Time   `json:"detected_at"`
	PatchedAt    *time.Time  `json:"patched_at,omitempty"`
	WazuhAlertID string      `json:"wazuh_alert_id,omitempty"`
}

// PatchReport is a tenant-scoped summary generated on demand.
type PatchReport struct {
	TenantID       string         `json:"tenant_id"`
	TotalCVEs      int            `json:"total_cves"`
	PatchedCount   int            `json:"patched_count"`
	UnpatchedCount int            `json:"unpatched_count"`
	CriticalCount  int            `json:"critical_count"`
	ByAsset        map[string]int `json:"by_asset"`   // assetID → unpatched count
	BySeverity     map[string]int `json:"by_severity"` // severity → count
	GeneratedAt    time.Time      `json:"generated_at"`
}

// WazuhClient fetches active vulnerability alerts from a Wazuh manager.
type WazuhClient interface {
	// GetVulnAlerts returns raw Wazuh vulnerability alerts for a tenant's assets.
	GetVulnAlerts(ctx context.Context, tenantID string) ([]EndpointPatchState, error)
}

// PatchStore persists CVE records and endpoint patch states.
type PatchStore interface {
	UpsertCVE(ctx context.Context, rec CVERecord) error
	GetCVE(ctx context.Context, cveID string) (*CVERecord, error)
	UpsertState(ctx context.Context, state EndpointPatchState) error
	ListStates(ctx context.Context, tenantID string) ([]EndpointPatchState, error)
	GetStatesByAsset(ctx context.Context, tenantID, assetID string) ([]EndpointPatchState, error)
}

// PatchTracker correlates Wazuh alerts with NVD enrichment and maintains
// per-endpoint patch state.
type PatchTracker struct {
	wazuh      WazuhClient
	store      PatchStore
	nvdBaseURL string
	httpClient *http.Client
}

// NewPatchTracker constructs a PatchTracker. Pass an empty nvdBaseURL to use the
// official NVD API endpoint.
func NewPatchTracker(wazuh WazuhClient, store PatchStore, nvdBaseURL string) *PatchTracker {
	if nvdBaseURL == "" {
		nvdBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	}
	return &PatchTracker{
		wazuh:      wazuh,
		store:      store,
		nvdBaseURL: nvdBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// EnrichFromNVD fetches CVE metadata from the NVD REST API and persists it.
// It returns ErrNotFound if the NVD does not know the CVE.
func (pt *PatchTracker) EnrichFromNVD(ctx context.Context, cveID string) (*CVERecord, error) {
	url := fmt.Sprintf("%s?cveId=%s", pt.nvdBaseURL, cveID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build NVD request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := pt.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("NVD HTTP call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cve %s not found in NVD", cveID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("NVD returned %d: %s", resp.StatusCode, body)
	}

	var nvdResp nvdResponse
	if err := json.NewDecoder(resp.Body).Decode(&nvdResp); err != nil {
		return nil, fmt.Errorf("decode NVD response: %w", err)
	}
	if len(nvdResp.Vulnerabilities) == 0 {
		return nil, fmt.Errorf("cve %s returned no vulnerabilities", cveID)
	}

	rec := parseNVDVuln(nvdResp.Vulnerabilities[0])
	if err := pt.store.UpsertCVE(ctx, rec); err != nil {
		return nil, fmt.Errorf("persist CVE %s: %w", cveID, err)
	}
	return &rec, nil
}

// GetUnpatchedByAsset returns all EndpointPatchStates for a given asset that are
// not yet in StatusPatched or StatusAccepted.
func (pt *PatchTracker) GetUnpatchedByAsset(
	ctx context.Context, tenantID, assetID string,
) ([]EndpointPatchState, error) {
	states, err := pt.store.GetStatesByAsset(ctx, tenantID, assetID)
	if err != nil {
		return nil, fmt.Errorf("query states for asset %s: %w", assetID, err)
	}
	var unpatched []EndpointPatchState
	for _, s := range states {
		if s.Status != StatusPatched && s.Status != StatusAccepted {
			unpatched = append(unpatched, s)
		}
	}
	return unpatched, nil
}

// GetCriticalUnpatched returns all unpatched states whose CVE CVSS score meets
// or exceeds minCVSS (e.g. 9.0 for critical).
func (pt *PatchTracker) GetCriticalUnpatched(
	ctx context.Context, tenantID string, minCVSS float64,
) ([]EndpointPatchState, error) {
	states, err := pt.store.ListStates(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list states: %w", err)
	}

	var critical []EndpointPatchState
	for _, s := range states {
		if s.Status == StatusPatched || s.Status == StatusAccepted {
			continue
		}
		rec, err := pt.store.GetCVE(ctx, s.CVEID)
		if err != nil || rec == nil {
			continue // enrichment not yet done; skip silently
		}
		if rec.CVSS >= minCVSS {
			critical = append(critical, s)
		}
	}
	return critical, nil
}

// MarkPatched transitions an endpoint/CVE pair to StatusPatched and records the
// timestamp.
func (pt *PatchTracker) MarkPatched(
	ctx context.Context, tenantID, assetID, cveID string,
) error {
	now := time.Now().UTC()
	state := EndpointPatchState{
		AssetID:   assetID,
		TenantID:  tenantID,
		CVEID:     cveID,
		Status:    StatusPatched,
		PatchedAt: &now,
	}
	// Preserve original DetectedAt by fetching existing record first.
	existing, err := pt.store.GetStatesByAsset(ctx, tenantID, assetID)
	if err == nil {
		for _, s := range existing {
			if s.CVEID == cveID {
				state.DetectedAt = s.DetectedAt
				state.WazuhAlertID = s.WazuhAlertID
				break
			}
		}
	}
	if state.DetectedAt.IsZero() {
		state.DetectedAt = now
	}
	return pt.store.UpsertState(ctx, state)
}

// GeneratePatchReport builds a tenant-scoped patch report from the current
// store contents. CVE records are enriched on-the-fly if missing.
func (pt *PatchTracker) GeneratePatchReport(
	ctx context.Context, tenantID string,
) (*PatchReport, error) {
	states, err := pt.store.ListStates(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list states for report: %w", err)
	}

	report := &PatchReport{
		TenantID:    tenantID,
		ByAsset:     make(map[string]int),
		BySeverity:  make(map[string]int),
		GeneratedAt: time.Now().UTC(),
	}

	for _, s := range states {
		report.TotalCVEs++
		if s.Status == StatusPatched {
			report.PatchedCount++
		} else {
			report.UnpatchedCount++
			report.ByAsset[s.AssetID]++
		}

		rec, _ := pt.store.GetCVE(ctx, s.CVEID)
		if rec == nil {
			// best-effort enrichment; ignore errors in report generation
			rec, _ = pt.EnrichFromNVD(ctx, s.CVEID)
		}
		if rec != nil {
			report.BySeverity[rec.Severity]++
			if rec.CVSS >= 9.0 && s.Status != StatusPatched && s.Status != StatusAccepted {
				report.CriticalCount++
			}
		}
	}
	return report, nil
}

// ---- NVD response deserialization ----

type nvdResponse struct {
	Vulnerabilities []nvdVulnWrapper `json:"vulnerabilities"`
}

type nvdVulnWrapper struct {
	CVE nvdCVE `json:"cve"`
}

type nvdCVE struct {
	ID          string        `json:"id"`
	Published   time.Time     `json:"published"`
	Descriptions []nvdLangStr `json:"descriptions"`
	Metrics      nvdMetrics   `json:"metrics"`
	Configurations []nvdConfig `json:"configurations"`
}

type nvdLangStr struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type nvdMetrics struct {
	CVSSMetricV31 []nvdCVSSMetric `json:"cvssMetricV31"`
	CVSSMetricV2  []nvdCVSSMetric `json:"cvssMetricV2"`
}

type nvdCVSSMetric struct {
	CVSSData nvdCVSSData `json:"cvssData"`
}

type nvdCVSSData struct {
	BaseScore           float64 `json:"baseScore"`
	BaseSeverity        string  `json:"baseSeverity"`
}

type nvdConfig struct {
	Nodes []nvdNode `json:"nodes"`
}

type nvdNode struct {
	CPEMatch []nvdCPEMatch `json:"cpeMatch"`
}

type nvdCPEMatch struct {
	Criteria string `json:"criteria"`
}

func parseNVDVuln(w nvdVulnWrapper) CVERecord {
	c := w.CVE
	rec := CVERecord{
		CVEID:       c.ID,
		PublishedAt: c.Published,
	}

	// Extract English description.
	for _, d := range c.Descriptions {
		if d.Lang == "en" {
			rec.Description = d.Value
			break
		}
	}

	// Prefer CVSSv3.1; fall back to v2.
	if len(c.Metrics.CVSSMetricV31) > 0 {
		m := c.Metrics.CVSSMetricV31[0].CVSSData
		rec.CVSS = m.BaseScore
		rec.Severity = strings.ToUpper(m.BaseSeverity)
	} else if len(c.Metrics.CVSSMetricV2) > 0 {
		m := c.Metrics.CVSSMetricV2[0].CVSSData
		rec.CVSS = m.BaseScore
		rec.Severity = cvssScoreToSeverity(m.BaseScore)
	}

	// Collect CPE affected software URIs.
	for _, cfg := range c.Configurations {
		for _, node := range cfg.Nodes {
			for _, cpe := range node.CPEMatch {
				rec.AffectedSoftware = append(rec.AffectedSoftware, cpe.Criteria)
			}
		}
	}

	rec.PatchAvailable = rec.CVSS > 0 // heuristic; real impl checks EPSS/patch DB
	return rec
}

func cvssScoreToSeverity(score float64) string {
	switch {
	case score == 0:
		return "NONE"
	case score < 4.0:
		return "LOW"
	case score < 7.0:
		return "MEDIUM"
	case score < 9.0:
		return "HIGH"
	default:
		return "CRITICAL"
	}
}
