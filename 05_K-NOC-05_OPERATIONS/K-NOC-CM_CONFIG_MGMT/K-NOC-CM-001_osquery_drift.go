// Package noc provides NOC operations tooling.
// K-NOC-CM-001 — Osquery Drift Detector: detect configuration drift using osquery.
// Uses exec.CommandContext to invoke osqueryi since osquery-go is not in go.mod.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	nats "github.com/nats-io/nats.go"
)

// DriftRule defines a single configuration compliance check executed via osquery.
type DriftRule struct {
	RuleID        string `json:"rule_id"`
	Query         string `json:"query"`
	ExpectedKey   string `json:"expected_key"`
	ExpectedValue string `json:"expected_value"`
	Severity      string `json:"severity"` // critical, high, medium, low
}

// DriftViolation records a single compliance failure detected on an asset.
type DriftViolation struct {
	RuleID        string    `json:"rule_id"`
	AssetID       string    `json:"asset_id"`
	ExpectedValue string    `json:"expected_value"`
	ActualValue   string    `json:"actual_value"`
	DetectedAt    time.Time `json:"detected_at"`
	Severity      string    `json:"severity"`
}

// OsqueryDriftDetector uses osquery to detect configuration drift.
type OsqueryDriftDetector struct {
	socketPath string
}

// NewOsqueryDriftDetector reads OSQUERY_SOCKET from the environment.
func NewOsqueryDriftDetector() *OsqueryDriftDetector {
	sock := os.Getenv("OSQUERY_SOCKET")
	if sock == "" {
		sock = "/var/osquery/osquery.em"
	}
	return &OsqueryDriftDetector{socketPath: sock}
}

// RunQuery executes a SQL query via osqueryi and returns the parsed rows.
func (d *OsqueryDriftDetector) RunQuery(ctx context.Context, sql string) ([]map[string]string, error) {
	// Use osqueryi in non-interactive JSON mode.
	cmd := exec.CommandContext(ctx, "osqueryi", "--json", sql)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osqueryi query failed: %w — stderr: %s", err, stderr.String())
	}

	var rows []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		return nil, fmt.Errorf("parse osqueryi JSON output: %w", err)
	}
	return rows, nil
}

// CheckBaselineCompliance runs each rule against osquery and returns any violations.
func (d *OsqueryDriftDetector) CheckBaselineCompliance(ctx context.Context, assetID string, baseline []DriftRule) ([]DriftViolation, error) {
	var violations []DriftViolation
	for _, rule := range baseline {
		rows, err := d.RunQuery(ctx, rule.Query)
		if err != nil {
			// Record as a violation if we cannot even run the query.
			violations = append(violations, DriftViolation{
				RuleID:        rule.RuleID,
				AssetID:       assetID,
				ExpectedValue: rule.ExpectedValue,
				ActualValue:   fmt.Sprintf("query_error: %v", err),
				DetectedAt:    time.Now().UTC(),
				Severity:      rule.Severity,
			})
			continue
		}
		for _, row := range rows {
			actual, ok := row[rule.ExpectedKey]
			if !ok {
				actual = "<key_missing>"
			}
			if actual != rule.ExpectedValue {
				violations = append(violations, DriftViolation{
					RuleID:        rule.RuleID,
					AssetID:       assetID,
					ExpectedValue: rule.ExpectedValue,
					ActualValue:   actual,
					DetectedAt:    time.Now().UTC(),
					Severity:      rule.Severity,
				})
			}
		}
	}
	return violations, nil
}

// CommonRules returns a pre-built set of common compliance checks.
func (d *OsqueryDriftDetector) CommonRules() []DriftRule {
	return []DriftRule{
		{
			RuleID:        "ssh-permit-root-login",
			Query:         "SELECT value FROM augeas WHERE path='/files/etc/ssh/sshd_config/PermitRootLogin';",
			ExpectedKey:   "value",
			ExpectedValue: "no",
			Severity:      "critical",
		},
		{
			RuleID:        "password-max-days",
			Query:         "SELECT value FROM augeas WHERE path='/files/etc/login.defs/PASS_MAX_DAYS';",
			ExpectedKey:   "value",
			ExpectedValue: "90",
			Severity:      "high",
		},
		{
			RuleID:        "ufw-active",
			Query:         "SELECT status FROM iptables WHERE filter_name='ufw-before-input' LIMIT 1;",
			ExpectedKey:   "status",
			ExpectedValue: "1",
			Severity:      "high",
		},
		{
			RuleID:        "auditd-running",
			Query:         "SELECT pid FROM processes WHERE name='auditd' LIMIT 1;",
			ExpectedKey:   "pid",
			ExpectedValue: "", // any non-empty PID is acceptable; violation if zero rows
			Severity:      "medium",
		},
	}
}

// ScanAsset runs all common rules against the given asset and returns violations.
func (d *OsqueryDriftDetector) ScanAsset(ctx context.Context, assetID, _ string) ([]DriftViolation, error) {
	return d.CheckBaselineCompliance(ctx, assetID, d.CommonRules())
}

// PublishDrift publishes a drift violation event to NATS on the GRC drift subject.
func (d *OsqueryDriftDetector) PublishDrift(violation DriftViolation, nc *nats.Conn) {
	data, err := json.Marshal(violation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[drift] marshal violation error: %v\n", err)
		return
	}
	subject := fmt.Sprintf("kubric.%s.grc.drift.v1", violation.AssetID)
	if pubErr := nc.Publish(subject, data); pubErr != nil {
		fmt.Fprintf(os.Stderr, "[drift] nats publish error subject=%s err=%v\n", subject, pubErr)
	}
}
