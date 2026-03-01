package grc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LulaResult is the outcome of a single Lula control validation.
type LulaResult struct {
	ControlID    string   `json:"controlId"`
	Status       string   `json:"status"` // pass | fail | not-applicable
	Message      string   `json:"message"`
	Observations []string `json:"observations"`
	Passing      bool     `json:"passing"`
}

// AssessmentReport aggregates all Lula results for a tenant.
type AssessmentReport struct {
	Framework      string
	TenantID       string
	TotalControls  int
	PassedControls int
	FailedControls int
	Results        []LulaResult
	GeneratedAt    time.Time
}

// LulaValidator runs Lula OSCAL compliance validation against Kubernetes clusters.
type LulaValidator struct {
	LulaBinary     string
	KubeconfigPath string
}

// NewLulaValidator creates a LulaValidator from environment variables.
// LULA_BINARY defaults to "lula"; KUBECONFIG is read from the environment.
func NewLulaValidator() *LulaValidator {
	lulaBin := os.Getenv("LULA_BINARY")
	if lulaBin == "" {
		lulaBin = "lula"
	}
	return &LulaValidator{
		LulaBinary:     lulaBin,
		KubeconfigPath: os.Getenv("KUBECONFIG"),
	}
}

// Validate runs `lula validate -f <lulaDocPath> --output json` and returns a parsed LulaResult.
func (v *LulaValidator) Validate(ctx context.Context, lulaDocPath string) (*LulaResult, error) {
	args := []string{"validate", "-f", lulaDocPath, "--output", "json"}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, v.LulaBinary, args...)
	if v.KubeconfigPath != "" {
		cmd.Env = append(os.Environ(), "KUBECONFIG="+v.KubeconfigPath)
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lula validate %s: %w: %s", lulaDocPath, err, stderr.String())
	}

	// lula may emit a single object or an array; try both.
	var result LulaResult
	if jsonErr := json.Unmarshal(out, &result); jsonErr == nil {
		result.Passing = result.Status == "pass"
		return &result, nil
	}

	var arr []LulaResult
	if jsonErr := json.Unmarshal(out, &arr); jsonErr == nil && len(arr) > 0 {
		arr[0].Passing = arr[0].Status == "pass"
		return &arr[0], nil
	}

	return nil, fmt.Errorf("parse lula output: cannot decode JSON from %s", lulaDocPath)
}

// BatchValidate validates all *.yaml / *.yml files directly inside docDir.
func (v *LulaValidator) BatchValidate(ctx context.Context, docDir string) ([]LulaResult, error) {
	entries, err := os.ReadDir(docDir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", docDir, err)
	}

	var results []LulaResult
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		r, err := v.Validate(ctx, filepath.Join(docDir, e.Name()))
		if err != nil {
			// Record as a fail rather than aborting the batch.
			results = append(results, LulaResult{
				ControlID: e.Name(),
				Status:    "fail",
				Message:   err.Error(),
				Passing:   false,
			})
			continue
		}
		results = append(results, *r)
	}
	return results, nil
}

// GenerateAssessmentResults validates all docs in docDir and builds an AssessmentReport.
func (v *LulaValidator) GenerateAssessmentResults(ctx context.Context, docDir, tenantID string) (*AssessmentReport, error) {
	results, err := v.BatchValidate(ctx, docDir)
	if err != nil {
		return nil, err
	}
	report := &AssessmentReport{
		Framework:     "OSCAL",
		TenantID:      tenantID,
		TotalControls: len(results),
		Results:       results,
		GeneratedAt:   time.Now().UTC(),
	}
	for _, r := range results {
		if r.Passing {
			report.PassedControls++
		} else {
			report.FailedControls++
		}
	}
	return report, nil
}

// SaveAssessmentReport persists an AssessmentReport to the assessments table in PostgreSQL.
// Each control result is upserted individually so repeated runs remain idempotent.
func (v *LulaValidator) SaveAssessmentReport(ctx context.Context, report *AssessmentReport, pgPool *pgxpool.Pool) error {
	for _, r := range report.Results {
		obsJSON, _ := json.Marshal(r.Observations)
		_, err := pgPool.Exec(ctx, `
			INSERT INTO assessments
				(tenant_id, control_id, framework, status, message, observations, assessed_at)
			VALUES ($1,$2,'OSCAL',$3,$4,$5::jsonb,NOW())
			ON CONFLICT (tenant_id, control_id, framework) DO UPDATE SET
				status       = EXCLUDED.status,
				message      = EXCLUDED.message,
				observations = EXCLUDED.observations,
				assessed_at  = NOW()`,
			report.TenantID, r.ControlID, r.Status, r.Message, string(obsJSON))
		if err != nil {
			return fmt.Errorf("save assessment row for %s: %w", r.ControlID, err)
		}
	}
	return nil
}
