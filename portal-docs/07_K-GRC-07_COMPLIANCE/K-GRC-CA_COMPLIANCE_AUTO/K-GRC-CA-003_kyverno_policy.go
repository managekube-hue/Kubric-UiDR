package grc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyViolation is a single Kyverno policy rule violation.
type PolicyViolation struct {
	PolicyName string
	RuleID     string
	Namespace  string
	Resource   string
	Message    string
	Severity   string
	Category   string
}

// ComplianceScore summarises policy compliance across a cluster.
type ComplianceScore struct {
	Total              int
	Compliant          int
	ViolationsCritical int
	ViolationsHigh     int
	ViolationsMedium   int
	ViolationsLow      int
	Score              float64 // 0–100
}

// KyvernoClient queries Kyverno policy reports via kubectl.
type KyvernoClient struct {
	KubeconfigPath string
}

// NewKyvernoClient creates a KyvernoClient using the KUBECONFIG env variable.
func NewKyvernoClient() *KyvernoClient {
	return &KyvernoClient{
		KubeconfigPath: os.Getenv("KUBECONFIG"),
	}
}

// kubectl executes a kubectl command and returns raw stdout.
func (k *KyvernoClient) kubectl(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if k.KubeconfigPath != "" {
		cmd.Env = append(os.Environ(), "KUBECONFIG="+k.KubeconfigPath)
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl %v: %w: %s", args, err, stderr.String())
	}
	return out, nil
}

// policyReportList is the JSON shape returned by kubectl get policyreport -o json.
type policyReportList struct {
	Items []policyReport `json:"items"`
}

type policyReport struct {
	Metadata struct {
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Results []policyReportResult `json:"results"`
}

type policyReportResult struct {
	Policy    string              `json:"policy"`
	Rule      string              `json:"rule"`
	Message   string              `json:"message"`
	Result    string              `json:"result"` // pass | fail | error
	Severity  string              `json:"severity"`
	Category  string              `json:"category"`
	Resources []resourceReference `json:"resources"`
}

type resourceReference struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// parseReportItems extracts PolicyViolation records from a raw policy-report JSON blob.
func parseReportItems(data []byte, namespace string) []PolicyViolation {
	var list policyReportList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil
	}

	var violations []PolicyViolation
	for _, report := range list.Items {
		ns := report.Metadata.Namespace
		if ns == "" {
			ns = namespace
		}
		for _, r := range report.Results {
			if r.Result != "fail" {
				continue
			}
			resource := ""
			if len(r.Resources) > 0 {
				res := r.Resources[0]
				resource = fmt.Sprintf("%s/%s/%s", res.Kind, res.Namespace, res.Name)
			}
			violations = append(violations, PolicyViolation{
				PolicyName: r.Policy,
				RuleID:     r.Rule,
				Namespace:  ns,
				Resource:   resource,
				Message:    r.Message,
				Severity:   r.Severity,
				Category:   r.Category,
			})
		}
	}
	return violations
}

// GetPolicyViolations returns namespace-scoped Kyverno policy violations.
// Pass namespace="" or "all" to query all namespaces.
func (k *KyvernoClient) GetPolicyViolations(ctx context.Context, namespace string) ([]PolicyViolation, error) {
	var args []string
	if namespace == "" || namespace == "all" {
		args = []string{"get", "policyreport", "-A", "-o", "json"}
	} else {
		args = []string{"get", "policyreport", "-n", namespace, "-o", "json"}
	}
	out, err := k.kubectl(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseReportItems(out, namespace), nil
}

// GetClusterPolicyViolations returns cluster-scoped Kyverno ClusterPolicyReport violations.
func (k *KyvernoClient) GetClusterPolicyViolations(ctx context.Context) ([]PolicyViolation, error) {
	out, err := k.kubectl(ctx, "get", "clusterpolicyreport", "-o", "json")
	if err != nil {
		return nil, err
	}
	return parseReportItems(out, "cluster"), nil
}

// ApplyPolicy applies a Kyverno policy supplied as a YAML string to the cluster.
func (k *KyvernoClient) ApplyPolicy(ctx context.Context, policyYAML string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(policyYAML)
	if k.KubeconfigPath != "" {
		cmd.Env = append(os.Environ(), "KUBECONFIG="+k.KubeconfigPath)
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w: %s", err, stderr.String())
	}
	return nil
}

// GetComplianceScore calculates an overall policy compliance score based on current violations.
func (k *KyvernoClient) GetComplianceScore(ctx context.Context) (*ComplianceScore, error) {
	nsViolations, err := k.GetPolicyViolations(ctx, "all")
	if err != nil {
		return nil, fmt.Errorf("get policy violations: %w", err)
	}
	clusterViolations, _ := k.GetClusterPolicyViolations(ctx)
	all := append(nsViolations, clusterViolations...)

	score := &ComplianceScore{}
	for _, v := range all {
		switch strings.ToLower(v.Severity) {
		case "critical":
			score.ViolationsCritical++
		case "high":
			score.ViolationsHigh++
		case "medium":
			score.ViolationsMedium++
		default:
			score.ViolationsLow++
		}
	}

	totalViolations := score.ViolationsCritical + score.ViolationsHigh + score.ViolationsMedium + score.ViolationsLow
	// Assume 100 baseline resources as the denominator when no other data is available.
	score.Total = totalViolations + 100
	score.Compliant = score.Total - totalViolations
	if score.Total > 0 {
		score.Score = float64(score.Compliant) / float64(score.Total) * 100
	} else {
		score.Score = 100
	}
	return score, nil
}

// SyncViolationsToAssessments writes PolicyViolations to the assessments PostgreSQL table.
func (k *KyvernoClient) SyncViolationsToAssessments(ctx context.Context, tenantID string, pgPool *pgxpool.Pool) error {
	nsViolations, err := k.GetPolicyViolations(ctx, "all")
	if err != nil {
		return fmt.Errorf("get policy violations: %w", err)
	}
	clusterViols, _ := k.GetClusterPolicyViolations(ctx)
	all := append(nsViolations, clusterViols...)

	for _, v := range all {
		_, err := pgPool.Exec(ctx, `
			INSERT INTO assessments
				(tenant_id, control_id, framework, status, message, assessed_at)
			VALUES ($1,$2,'kyverno','fail',$3,NOW())
			ON CONFLICT (tenant_id, control_id, framework) DO UPDATE SET
				status      = 'fail',
				message     = EXCLUDED.message,
				assessed_at = NOW()`,
			tenantID,
			v.PolicyName+"/"+v.RuleID,
			v.Message)
		if err != nil {
			log.Printf("kyverno_policy: save assessment error: %v", err)
		}
	}

	log.Printf("kyverno_policy: synced %d violations for tenant=%s at %s",
		len(all), tenantID, time.Now().Format(time.RFC3339))
	return nil
}
