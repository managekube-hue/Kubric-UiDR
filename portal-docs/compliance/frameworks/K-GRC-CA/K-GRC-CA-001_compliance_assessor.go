// Package grc provides Governance, Risk, and Compliance tooling.
// K-GRC-CA-001 — Compliance Assessor: automated compliance framework assessment
// Supports: CIS Benchmarks, SOC 2, ISO 27001, NIST CSF, PCI DSS, HIPAA
package grc

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"text/template"
	"time"
)

// ─── Framework identifiers ────────────────────────────────────────────────────

// Framework identifies a compliance framework.
type Framework string

const (
	FrameworkCIS      Framework = "cis"
	FrameworkSOC2     Framework = "soc2"
	FrameworkISO27001 Framework = "iso27001"
	FrameworkNIST     Framework = "nist_csf"
	FrameworkPCIDSS   Framework = "pci_dss"
	FrameworkHIPAA    Framework = "hipaa"
)

// ─── Control status ──────────────────────────────────────────────────────────

// ControlStatus represents the compliance state of a single control.
type ControlStatus string

const (
	StatusCompliant      ControlStatus = "compliant"
	StatusNonCompliant   ControlStatus = "non_compliant"
	StatusPartial        ControlStatus = "partial"
	StatusNotApplicable  ControlStatus = "not_applicable"
	StatusUnknown        ControlStatus = "unknown"
)

// AutomationLevel describes how a control can be assessed.
type AutomationLevel string

const (
	AutomationManual    AutomationLevel = "manual"
	AutomationAutomated AutomationLevel = "automated"
	AutomationPartial   AutomationLevel = "partial"
)

// Severity of a control finding when non-compliant.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ─── Core data types ─────────────────────────────────────────────────────────

// Control is a single compliance requirement within a framework.
type Control struct {
	ID              string          `json:"id"`
	FrameworkID     Framework       `json:"framework_id"`
	Category        string          `json:"category"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Requirement     string          `json:"requirement"`
	AutomationLevel AutomationLevel `json:"automation_level"`
	Severity        Severity        `json:"severity"`
}

// ControlResult is the assessed state of a Control for a specific tenant + agent.
type ControlResult struct {
	ControlID     string        `json:"control_id"`
	TenantID      string        `json:"tenant_id"`
	AgentID       string        `json:"agent_id,omitempty"`
	Status        ControlStatus `json:"status"`
	EvidenceRef   string        `json:"evidence_ref,omitempty"`
	AssessedAt    time.Time     `json:"assessed_at"`
	Notes         string        `json:"notes,omitempty"`
	Remediations  []string      `json:"remediations,omitempty"`
}

// ComplianceReport is the aggregated assessment output for a framework and tenant.
type ComplianceReport struct {
	TenantID          string          `json:"tenant_id"`
	Framework         Framework       `json:"framework"`
	TotalControls     int             `json:"total_controls"`
	CompliantCount    int             `json:"compliant_count"`
	NonCompliantCount int             `json:"non_compliant_count"`
	PartialCount      int             `json:"partial_count"`
	NotApplicable     int             `json:"not_applicable_count"`
	Score             float64         `json:"score"` // 0–100, excludes N/A controls
	GeneratedAt       time.Time       `json:"generated_at"`
	Results           []ControlResult `json:"results"`
}

// ─── Client interfaces ───────────────────────────────────────────────────────

// WazuhClient abstracts the Wazuh API for SCA and agent queries.
type WazuhClient interface {
	// ListSCAChecks returns SCA policy results for an agent.
	ListSCAChecks(ctx context.Context, agentID string) ([]WazuhSCACheck, error)
	// ListAgents returns active agent IDs for a tenant.
	ListAgents(ctx context.Context, tenantID string) ([]string, error)
}

// WazuhSCACheck is a single Wazuh SCA policy item result.
type WazuhSCACheck struct {
	ID          int    `json:"id"`
	PolicyID    string `json:"policy_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Result      string `json:"result"` // "passed", "failed", "not applicable"
	Remediation string `json:"remediation"`
}

// OsqueryClient abstracts osquery for system-state queries.
type OsqueryClient interface {
	// Query executes an osquery SQL statement and returns JSON rows.
	Query(ctx context.Context, agentID, sql string) ([]map[string]string, error)
}

// ComplianceStore persists and retrieves compliance reports.
type ComplianceStore interface {
	SaveReport(ctx context.Context, report *ComplianceReport) error
	LoadLatestReport(ctx context.Context, tenantID string, fw Framework) (*ComplianceReport, error)
}

// ─── Assessor ────────────────────────────────────────────────────────────────

// ComplianceAssessor orchestrates compliance assessment across frameworks.
type ComplianceAssessor struct {
	wazuh   WazuhClient
	osquery OsqueryClient
	store   ComplianceStore
}

// NewComplianceAssessor constructs an assessor with the provided backend clients.
func NewComplianceAssessor(w WazuhClient, o OsqueryClient, s ComplianceStore) *ComplianceAssessor {
	return &ComplianceAssessor{wazuh: w, osquery: o, store: s}
}

// ─── CIS Benchmark Assessment ────────────────────────────────────────────────

// cisControlMap maps Wazuh SCA policy prefixes to CIS control IDs.
var cisControlMap = map[string]string{
	"cis_debian10": "CIS-1",
	"cis_rhel8":    "CIS-1",
	"cis_ubuntu20": "CIS-1",
	"cis_win2019":  "CIS-1",
}

// AssessCISBenchmark evaluates the CIS Benchmarks for an agent via Wazuh SCA.
// It maps each SCA check to a CIS control and aggregates a ComplianceReport.
func (a *ComplianceAssessor) AssessCISBenchmark(ctx context.Context, tenantID, agentID string) (*ComplianceReport, error) {
	checks, err := a.wazuh.ListSCAChecks(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("wazuh sca list: %w", err)
	}

	report := &ComplianceReport{
		TenantID:    tenantID,
		Framework:   FrameworkCIS,
		GeneratedAt: time.Now().UTC(),
	}

	categorySet := map[string]bool{}
	for _, chk := range checks {
		report.TotalControls++

		res := ControlResult{
			ControlID:  fmt.Sprintf("CIS-%d", chk.ID),
			TenantID:   tenantID,
			AgentID:    agentID,
			AssessedAt: time.Now().UTC(),
		}

		switch strings.ToLower(chk.Result) {
		case "passed":
			res.Status = StatusCompliant
			report.CompliantCount++
		case "failed":
			res.Status = StatusNonCompliant
			report.NonCompliantCount++
			if chk.Remediation != "" {
				res.Remediations = []string{chk.Remediation}
			}
		case "not applicable":
			res.Status = StatusNotApplicable
			report.NotApplicable++
		default:
			res.Status = StatusUnknown
		}

		res.Notes = chk.Description
		report.Results = append(report.Results, res)
		categorySet[chk.PolicyID] = true
	}

	report.Score = calcScore(report)

	if err := a.store.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("save cis report: %w", err)
	}
	return report, nil
}

// ─── SOC 2 Assessment ────────────────────────────────────────────────────────

// soc2Controls is the minimal set of SOC 2 Trust Service Criteria controls
// that can be assessed via osquery and API checks.
var soc2Controls = []Control{
	{ID: "CC6.1", FrameworkID: FrameworkSOC2, Category: "Logical Access", Title: "Logical access security software", Requirement: "Access controls restrict access to production systems", AutomationLevel: AutomationAutomated, Severity: SeverityHigh},
	{ID: "CC6.2", FrameworkID: FrameworkSOC2, Category: "Logical Access", Title: "New access provisioning", Requirement: "New access is provisioned via formal request and approval", AutomationLevel: AutomationPartial, Severity: SeverityHigh},
	{ID: "CC6.3", FrameworkID: FrameworkSOC2, Category: "Logical Access", Title: "Access removal", Requirement: "Access is removed when no longer required", AutomationLevel: AutomationPartial, Severity: SeverityHigh},
	{ID: "CC7.1", FrameworkID: FrameworkSOC2, Category: "Operations", Title: "Detection of configuration changes", Requirement: "Unauthorized configuration changes are detected", AutomationLevel: AutomationAutomated, Severity: SeverityCritical},
	{ID: "CC7.2", FrameworkID: FrameworkSOC2, Category: "Operations", Title: "Anomaly detection", Requirement: "Security events are detected and responded to", AutomationLevel: AutomationAutomated, Severity: SeverityCritical},
	{ID: "CC8.1", FrameworkID: FrameworkSOC2, Category: "Change Management", Title: "Change management process", Requirement: "Changes follow approved change management process", AutomationLevel: AutomationManual, Severity: SeverityMedium},
	{ID: "A1.1", FrameworkID: FrameworkSOC2, Category: "Availability", Title: "Uptime monitoring", Requirement: "System availability is monitored and maintained", AutomationLevel: AutomationAutomated, Severity: SeverityHigh},
	{ID: "PI1.1", FrameworkID: FrameworkSOC2, Category: "Processing Integrity", Title: "Data processing completeness", Requirement: "Data is processed completely and accurately", AutomationLevel: AutomationPartial, Severity: SeverityMedium},
}

// AssessSOC2 evaluates SOC 2 Trust Service Criteria for a tenant.
// Automated controls are verified via osquery; manual controls are marked partial.
func (a *ComplianceAssessor) AssessSOC2(ctx context.Context, tenantID string) (*ComplianceReport, error) {
	agents, err := a.wazuh.ListAgents(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("wazuh list agents: %w", err)
	}

	report := &ComplianceReport{
		TenantID:      tenantID,
		Framework:     FrameworkSOC2,
		TotalControls: len(soc2Controls),
		GeneratedAt:   time.Now().UTC(),
	}

	for _, ctrl := range soc2Controls {
		res := ControlResult{
			ControlID:  ctrl.ID,
			TenantID:   tenantID,
			AssessedAt: time.Now().UTC(),
		}

		switch ctrl.AutomationLevel {
		case AutomationAutomated:
			res.Status = assessSOC2AutomatedControl(ctx, a.osquery, agents, ctrl)
		case AutomationPartial:
			res.Status = StatusPartial
			res.Notes = "Partial evidence available; human review required for full attestation"
		case AutomationManual:
			res.Status = StatusUnknown
			res.Notes = "Manual assessment required — provide evidence via GRC Evidence Collector"
		}

		switch res.Status {
		case StatusCompliant:
			report.CompliantCount++
		case StatusNonCompliant:
			report.NonCompliantCount++
		case StatusPartial:
			report.PartialCount++
		}

		report.Results = append(report.Results, res)
	}

	report.Score = calcScore(report)

	if err := a.store.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("save soc2 report: %w", err)
	}
	return report, nil
}

// assessSOC2AutomatedControl runs an osquery probe on all tenant agents to
// determine the status of an automated SOC 2 control.
func assessSOC2AutomatedControl(ctx context.Context, oq OsqueryClient, agents []string, ctrl Control) ControlStatus {
	if oq == nil || len(agents) == 0 {
		return StatusUnknown
	}

	var query string
	switch ctrl.ID {
	case "CC6.1":
		query = "SELECT name, status FROM services WHERE name IN ('ssh','sshd','ufw','firewalld') AND status='running';"
	case "CC7.1":
		query = "SELECT count(*) AS cnt FROM file_events WHERE time > strftime('%s','now','-24 hours');"
	case "CC7.2":
		query = "SELECT count(*) AS cnt FROM process_events WHERE time > strftime('%s','now','-1 hour');"
	case "A1.1":
		query = "SELECT load_average FROM system_info;"
	default:
		return StatusPartial
	}

	passCount := 0
	for _, agentID := range agents {
		rows, err := oq.Query(ctx, agentID, query)
		if err != nil || len(rows) == 0 {
			continue
		}
		passCount++
	}

	if passCount == 0 {
		return StatusNonCompliant
	}
	if passCount < len(agents) {
		return StatusPartial
	}
	return StatusCompliant
}

// ─── Generic Report Generator ────────────────────────────────────────────────

// GenerateReport dispatches to the appropriate framework assessor and returns
// a fresh ComplianceReport. For frameworks without a dedicated automated
// assessor, it returns the last persisted report or an empty scaffold.
func (a *ComplianceAssessor) GenerateReport(ctx context.Context, tenantID string, framework Framework) (*ComplianceReport, error) {
	switch framework {
	case FrameworkSOC2:
		return a.AssessSOC2(ctx, tenantID)
	case FrameworkCIS:
		// CIS requires an agentID; load most recent persisted report instead.
		existing, err := a.store.LoadLatestReport(ctx, tenantID, FrameworkCIS)
		if err == nil && existing != nil {
			return existing, nil
		}
		return scaffoldReport(tenantID, framework), nil
	default:
		// ISO 27001, NIST CSF, PCI DSS, HIPAA: load persisted or scaffold.
		existing, err := a.store.LoadLatestReport(ctx, tenantID, framework)
		if err == nil && existing != nil {
			return existing, nil
		}
		return scaffoldReport(tenantID, framework), nil
	}
}

// scaffoldReport returns an empty report structure for frameworks that require
// manual evidence input via the GRC Evidence Collector.
func scaffoldReport(tenantID string, fw Framework) *ComplianceReport {
	return &ComplianceReport{
		TenantID:    tenantID,
		Framework:   fw,
		GeneratedAt: time.Now().UTC(),
		Score:       0,
	}
}

// ─── Export: CSV ─────────────────────────────────────────────────────────────

// ExportToCSV writes the ComplianceReport results as RFC 4180 CSV to w.
// Columns: ControlID, Status, AssessedAt, AgentID, EvidenceRef, Notes, Remediations.
func ExportToCSV(w io.Writer, report *ComplianceReport) error {
	cw := csv.NewWriter(w)

	header := []string{"ControlID", "Status", "AssessedAt", "AgentID", "EvidenceRef", "Notes", "Remediations"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("csv write header: %w", err)
	}

	// Sort results by ControlID for deterministic output.
	sorted := make([]ControlResult, len(report.Results))
	copy(sorted, report.Results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ControlID < sorted[j].ControlID
	})

	for _, r := range sorted {
		row := []string{
			r.ControlID,
			string(r.Status),
			r.AssessedAt.Format(time.RFC3339),
			r.AgentID,
			r.EvidenceRef,
			r.Notes,
			strings.Join(r.Remediations, "; "),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("csv write row %s: %w", r.ControlID, err)
		}
	}

	cw.Flush()
	return cw.Error()
}

// ─── Export: HTML/PDF-like ────────────────────────────────────────────────────

const pdfReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<title>Kubric Compliance Report — {{.Framework}} — {{.TenantID}}</title>
<style>
  body { font-family: Arial, sans-serif; margin: 40px; color: #1a1a2e; }
  h1   { color: #0f3460; border-bottom: 2px solid #e94560; padding-bottom:8px; }
  h2   { color: #16213e; margin-top: 32px; }
  .scorecard { background:#f8f9fa; border-radius:8px; padding:20px; margin:20px 0; display:flex; gap:32px; }
  .metric    { text-align:center; }
  .metric .val { font-size:2.4em; font-weight:bold; color:#0f3460; }
  .metric .lbl { font-size:0.85em; color:#666; }
  table { border-collapse:collapse; width:100%; margin-top:16px; }
  th    { background:#0f3460; color:#fff; padding:10px 12px; text-align:left; font-size:0.9em; }
  td    { padding:8px 12px; border-bottom:1px solid #dee2e6; font-size:0.85em; vertical-align:top; }
  tr:nth-child(even) td { background:#f8f9fa; }
  .status-compliant     { color:#28a745; font-weight:bold; }
  .status-non_compliant { color:#dc3545; font-weight:bold; }
  .status-partial       { color:#fd7e14; font-weight:bold; }
  .status-not_applicable{ color:#6c757d; }
  .status-unknown       { color:#adb5bd; }
  .footer { margin-top:48px; font-size:0.75em; color:#aaa; text-align:center; }
</style>
</head>
<body>
<h1>Kubric Compliance Report</h1>
<p><strong>Framework:</strong> {{.Framework}} &nbsp;|&nbsp;
   <strong>Tenant:</strong> {{.TenantID}} &nbsp;|&nbsp;
   <strong>Generated:</strong> {{.GeneratedAtFmt}}</p>

<div class="scorecard">
  <div class="metric"><div class="val">{{printf "%.1f" .Score}}%</div><div class="lbl">Overall Score</div></div>
  <div class="metric"><div class="val">{{.TotalControls}}</div><div class="lbl">Total Controls</div></div>
  <div class="metric"><div class="val" style="color:#28a745">{{.CompliantCount}}</div><div class="lbl">Compliant</div></div>
  <div class="metric"><div class="val" style="color:#dc3545">{{.NonCompliantCount}}</div><div class="lbl">Non-Compliant</div></div>
  <div class="metric"><div class="val" style="color:#fd7e14">{{.PartialCount}}</div><div class="lbl">Partial</div></div>
</div>

<h2>Control Results</h2>
<table>
  <tr>
    <th>Control ID</th><th>Status</th><th>Agent</th>
    <th>Evidence Ref</th><th>Assessed At</th><th>Notes / Remediations</th>
  </tr>
  {{range .Results}}
  <tr>
    <td>{{.ControlID}}</td>
    <td class="status-{{.Status}}">{{.Status}}</td>
    <td>{{.AgentID}}</td>
    <td>{{.EvidenceRef}}</td>
    <td>{{.AssessedAtFmt}}</td>
    <td>{{.Notes}}{{if .Remediations}}<br/><em>Remediation:</em> {{joinRemediations .Remediations}}{{end}}</td>
  </tr>
  {{end}}
</table>

<div class="footer">
  Generated by Kubric GRC Compliance Assessor (K-GRC-CA-001) &mdash; {{.GeneratedAtFmt}}
</div>
</body>
</html>
`

// pdfTemplateData is the view-model fed into pdfReportTemplate.
type pdfTemplateData struct {
	*ComplianceReport
	GeneratedAtFmt string
	Results        []pdfResultRow
}

type pdfResultRow struct {
	ControlResult
	AssessedAtFmt string
}

// ExportToPDF renders the ComplianceReport as a self-contained HTML document
// suitable for headless Chrome / wkhtmltopdf conversion. The returned bytes
// are UTF-8 HTML. Callers that require true PDF should pass the output to a
// PDF renderer (e.g. chromedp, gotenberg).
func ExportToPDF(_ context.Context, report *ComplianceReport) ([]byte, error) {
	funcMap := template.FuncMap{
		"joinRemediations": func(r []string) string { return strings.Join(r, "; ") },
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(pdfReportTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse pdf template: %w", err)
	}

	rows := make([]pdfResultRow, len(report.Results))
	for i, r := range report.Results {
		rows[i] = pdfResultRow{
			ControlResult: r,
			AssessedAtFmt: r.AssessedAt.Format("2006-01-02 15:04:05 UTC"),
		}
	}

	data := pdfTemplateData{
		ComplianceReport: report,
		GeneratedAtFmt:   report.GeneratedAt.Format("2006-01-02 15:04:05 UTC"),
		Results:          rows,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render pdf template: %w", err)
	}
	return buf.Bytes(), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// calcScore computes the compliance score as a percentage of compliant controls
// over the total assessable controls (excludes NotApplicable and Unknown).
func calcScore(r *ComplianceReport) float64 {
	assessable := r.TotalControls - r.NotApplicable
	if assessable <= 0 {
		return 0
	}
	// Partial controls count as 0.5 toward compliance.
	numerator := float64(r.CompliantCount) + float64(r.PartialCount)*0.5
	raw := (numerator / float64(assessable)) * 100
	return math.Round(raw*10) / 10 // round to 1 decimal place
}
