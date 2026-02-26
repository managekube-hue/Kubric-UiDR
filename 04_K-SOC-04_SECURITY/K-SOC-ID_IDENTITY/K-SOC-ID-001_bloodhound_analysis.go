// K-SOC-ID-001 — BloodHound Analysis: AD attack path analysis for SOC investigations.
// Wraps internal/bloodhound client with SOC-specific investigation workflows,
// NATS alerting for new paths, and TheHive case creation.
package soc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"

	"github.com/managekube-hue/Kubric-UiDR/internal/bloodhound"
	"github.com/managekube-hue/Kubric-UiDR/internal/thehive"
)

// ---------------------------------------------------------------------------
// SOC-layer types
// ---------------------------------------------------------------------------

// RiskSummary is a scored overview of Active Directory risk in a single domain.
type RiskSummary struct {
	// DomainName is the FQDN or BloodHound domain object ID.
	DomainName string `json:"domain_name"`

	// Counts
	KerberoastableCount int `json:"kerberoastable_count"`
	DCSyncCount         int `json:"dcsync_count"`
	AttackPathCount     int `json:"attack_path_count"`
	CriticalPathCount   int `json:"critical_path_count"`

	// RiskScore is a composite [0–100] value derived from the counts above.
	// Formula: min(100, 10*kerberoastable + 20*dcsync + 5*attack_paths + 15*critical_paths)
	RiskScore float64 `json:"risk_score"`

	// CollectedAt is when this summary was computed.
	CollectedAt time.Time `json:"collected_at"`
}

// computeRiskScore derives the RiskScore from the summary counts.
func (r *RiskSummary) computeRiskScore() {
	score := float64(r.KerberoastableCount)*10 +
		float64(r.DCSyncCount)*20 +
		float64(r.AttackPathCount)*5 +
		float64(r.CriticalPathCount)*15
	r.RiskScore = math.Min(100, score)
}

// AttackPath is a single attack route within a BloodHound graph.
// The path is reconstructed from the graph edge labels.
type AttackPath struct {
	// PathID is the BloodHound attack path finding ID (if sourced from a
	// pre-computed finding) or a synthetic ID derived from the path hash.
	PathID string `json:"path_id"`

	// Steps contains the node labels (or principals) along the path.
	Steps []string `json:"steps"`

	// Techniques lists the MITRE ATT&CK technique IDs implied by each edge
	// kind encountered (e.g. "T1558.003" for Kerberoasting).
	Techniques []string `json:"techniques"`

	// Severity is a 1–4 integer (Low/Medium/High/Critical) derived from
	// the BloodHound ImpactValue / exposure score.
	Severity int `json:"severity"`
}

// ServiceAccount represents a Kerberoastable account in Active Directory.
type ServiceAccount struct {
	Name        string    `json:"name"`
	SPN         string    `json:"spn,omitempty"`
	LastChanged time.Time `json:"last_changed,omitempty"`
	Enabled     bool      `json:"enabled"`
}

// adAttackPathNATSMessage is published when new attack paths are detected.
type adAttackPathNATSMessage struct {
	TenantID    string       `json:"tenant_id"`
	DomainName  string       `json:"domain_name"`
	NewPaths    []AttackPath `json:"new_paths"`
	DetectedAt  time.Time    `json:"detected_at"`
}

// ---------------------------------------------------------------------------
// BloodHoundInvestigator
// ---------------------------------------------------------------------------

// BloodHoundInvestigator wraps bloodhound.Client with SOC investigation
// workflows.  A nil inner client causes all methods to return a descriptive
// error rather than panicking, so callers can treat BloodHound as optional.
type BloodHoundInvestigator struct {
	bh *bloodhound.Client
}

// NewBloodHoundInvestigator creates an investigator backed by the given
// BloodHound CE client.  Pass nil to create a no-op investigator.
func NewBloodHoundInvestigator(bh *bloodhound.Client) *BloodHoundInvestigator {
	return &BloodHoundInvestigator{bh: bh}
}

// ---------------------------------------------------------------------------
// Domain risk summary
// ---------------------------------------------------------------------------

// DomainRiskSummary computes a composite risk summary for the named AD domain.
// It queries BloodHound for Kerberoastable accounts, DCSync principals, and
// pre-computed attack path findings.
func (inv *BloodHoundInvestigator) DomainRiskSummary(ctx context.Context, domainName string) (*RiskSummary, error) {
	if inv.bh == nil {
		return nil, fmt.Errorf("bloodhound: client not configured")
	}

	summary := &RiskSummary{
		DomainName:  domainName,
		CollectedAt: time.Now().UTC(),
	}

	// Resolve domain ID from name.
	domainID, err := inv.resolveDomainID(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: resolve domain %q: %w", domainName, err)
	}

	// Kerberoastable accounts.
	kerb, err := inv.bh.ListKerberoastable(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: kerberoastable accounts: %w", err)
	}
	summary.KerberoastableCount = len(kerb)

	// DCSync principals.
	dcsync, err := inv.bh.ListDCSync(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: dcsync principals: %w", err)
	}
	summary.DCSyncCount = len(dcsync)

	// Pre-computed attack paths.
	paths, err := inv.bh.ListAttackPaths(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: attack paths: %w", err)
	}
	summary.AttackPathCount = len(paths)
	for _, p := range paths {
		if p.ImpactValue >= 7.5 {
			summary.CriticalPathCount++
		}
	}

	summary.computeRiskScore()
	return summary, nil
}

// ---------------------------------------------------------------------------
// Attack paths to DA
// ---------------------------------------------------------------------------

// FindAttackPathToDA finds all attack paths that lead to a Domain Admin
// principal in the same domain as fromPrincipal.  It uses BloodHound's
// pre-computed attack path findings filtered by principal name.
func (inv *BloodHoundInvestigator) FindAttackPathToDA(ctx context.Context, fromPrincipal string) ([]AttackPath, error) {
	if inv.bh == nil {
		return nil, fmt.Errorf("bloodhound: client not configured")
	}

	// Find the domain that contains fromPrincipal.
	domains, err := inv.bh.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: list domains: %w", err)
	}

	var paths []AttackPath
	for _, dom := range domains {
		if !dom.Collected {
			continue
		}

		bhPaths, err := inv.bh.ListAttackPaths(ctx, dom.ID)
		if err != nil {
			continue
		}
		for _, bp := range bhPaths {
			// Fetch the full graph for each finding to extract step labels.
			graph, err := inv.bh.GetAttackPathDetails(ctx, bp.ID)
			if err != nil {
				continue
			}
			ap := inv.graphToAttackPath(bp, graph)
			// Only include paths that originate from or pass through the
			// requested principal.
			if containsPrincipal(ap.Steps, fromPrincipal) {
				paths = append(paths, ap)
			}
		}
	}
	return paths, nil
}

// graphToAttackPath converts a BloodHound PathFinding + AttackPath finding
// into the SOC-layer AttackPath type.
func (inv *BloodHoundInvestigator) graphToAttackPath(bp bloodhound.AttackPath, graph *bloodhound.PathFinding) AttackPath {
	steps := make([]string, 0, len(graph.Nodes))
	for _, n := range graph.Nodes {
		steps = append(steps, n.Label)
	}
	techniques := edgesToTechniques(graph.Edges)

	sev := 1
	if bp.ImpactValue >= 9.0 {
		sev = 4
	} else if bp.ImpactValue >= 7.0 {
		sev = 3
	} else if bp.ImpactValue >= 4.0 {
		sev = 2
	}

	return AttackPath{
		PathID:     bp.ID,
		Steps:      steps,
		Techniques: techniques,
		Severity:   sev,
	}
}

// ---------------------------------------------------------------------------
// Kerberoastable accounts
// ---------------------------------------------------------------------------

// GetKerberoastableAccounts returns all Kerberoastable service accounts for
// the named domain, enriched with SPN and status information.
func (inv *BloodHoundInvestigator) GetKerberoastableAccounts(ctx context.Context, domainName string) ([]ServiceAccount, error) {
	if inv.bh == nil {
		return nil, fmt.Errorf("bloodhound: client not configured")
	}

	domainID, err := inv.resolveDomainID(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: resolve domain: %w", err)
	}

	nodes, err := inv.bh.ListKerberoastable(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: list kerberoastable: %w", err)
	}

	accounts := make([]ServiceAccount, 0, len(nodes))
	for _, n := range nodes {
		sa := ServiceAccount{
			Name:    n.Label,
			Enabled: true, // default; overridden below if property exists
		}
		if v, ok := n.Props["serviceprincipalnames"]; ok {
			if spns, ok := v.([]interface{}); ok && len(spns) > 0 {
				sa.SPN = fmt.Sprint(spns[0])
			}
		}
		if v, ok := n.Props["enabled"]; ok {
			if b, ok := v.(bool); ok {
				sa.Enabled = b
			}
		}
		if v, ok := n.Props["pwdlastset"]; ok {
			if ts, ok := v.(float64); ok && ts > 0 {
				sa.LastChanged = time.Unix(int64(ts), 0).UTC()
			}
		}
		accounts = append(accounts, sa)
	}
	return accounts, nil
}

// ---------------------------------------------------------------------------
// New-path alerting
// ---------------------------------------------------------------------------

// AlertOnNewPaths compares the current attack paths in BloodHound to a
// baseline snapshot and publishes any newly observed paths to the NATS subject:
//
//	kubric.{tenantID}.detection.ad_attack_path.v1
func (inv *BloodHoundInvestigator) AlertOnNewPaths(
	ctx context.Context,
	nc *nats.Conn,
	tenantID string,
	baseline []AttackPath,
) error {
	if inv.bh == nil {
		return fmt.Errorf("bloodhound: client not configured")
	}
	if nc == nil {
		return fmt.Errorf("bloodhound: nats connection is nil")
	}

	// Build a set of known path IDs from the baseline.
	known := make(map[string]struct{}, len(baseline))
	for _, p := range baseline {
		known[p.PathID] = struct{}{}
	}

	// Retrieve current paths from all collected domains.
	domains, err := inv.bh.ListDomains(ctx)
	if err != nil {
		return fmt.Errorf("bloodhound: list domains for alert scan: %w", err)
	}

	var newPaths []AttackPath
	domainName := ""
	for _, dom := range domains {
		if !dom.Collected {
			continue
		}
		if domainName == "" {
			domainName = dom.Name
		}
		bhPaths, err := inv.bh.ListAttackPaths(ctx, dom.ID)
		if err != nil {
			continue
		}
		for _, bp := range bhPaths {
			if _, exists := known[bp.ID]; exists {
				continue
			}
			graph, err := inv.bh.GetAttackPathDetails(ctx, bp.ID)
			if err != nil {
				continue
			}
			ap := inv.graphToAttackPath(bp, graph)
			newPaths = append(newPaths, ap)
		}
	}

	if len(newPaths) == 0 {
		return nil // Nothing new to report.
	}

	msg := adAttackPathNATSMessage{
		TenantID:   tenantID,
		DomainName: domainName,
		NewPaths:   newPaths,
		DetectedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("bloodhound: marshal new paths: %w", err)
	}
	subject := fmt.Sprintf("kubric.%s.detection.ad_attack_path.v1", tenantID)
	if err := nc.Publish(subject, payload); err != nil {
		return fmt.Errorf("bloodhound: publish new paths to %q: %w", subject, err)
	}
	return nc.Flush()
}

// ---------------------------------------------------------------------------
// TheHive case export
// ---------------------------------------------------------------------------

// ExportToTheHive creates a TheHive case summarising the domain risk findings.
// The case includes tasks for the most pressing remediation actions.
func (inv *BloodHoundInvestigator) ExportToTheHive(
	ctx context.Context,
	summary *RiskSummary,
	hive *thehive.Client,
) error {
	if hive == nil {
		return fmt.Errorf("bloodhound: thehive client is nil")
	}
	if summary == nil {
		return fmt.Errorf("bloodhound: risk summary is nil")
	}

	severity := 2 // Medium default
	switch {
	case summary.RiskScore >= 75:
		severity = 4 // Critical
	case summary.RiskScore >= 50:
		severity = 3 // High
	case summary.RiskScore >= 25:
		severity = 2 // Medium
	default:
		severity = 1 // Low
	}

	desc := fmt.Sprintf(`## BloodHound AD Analysis — %s

**Risk Score**: %.0f / 100
**Kerberoastable Accounts**: %d
**DCSync Principals**: %d
**Attack Paths**: %d (Critical: %d)

*Generated by Kubric BloodHoundInvestigator at %s*`,
		summary.DomainName,
		summary.RiskScore,
		summary.KerberoastableCount,
		summary.DCSyncCount,
		summary.AttackPathCount,
		summary.CriticalPathCount,
		summary.CollectedAt.Format(time.RFC3339),
	)

	cas := thehive.Case{
		Title:       fmt.Sprintf("[BloodHound] AD Risk — %s", summary.DomainName),
		Description: desc,
		Severity:    severity,
		StartDate:   summary.CollectedAt.UnixMilli(),
		TLP:         2, // AMBER
		PAP:         2,
		Tags: []string{
			"bloodhound",
			"active-directory",
			"identity",
			fmt.Sprintf("domain:%s", summary.DomainName),
			fmt.Sprintf("risk-score:%.0f", summary.RiskScore),
		},
	}

	created, err := hive.CreateCase(ctx, cas)
	if err != nil {
		return fmt.Errorf("bloodhound: create thehive case: %w", err)
	}

	// Add remediation tasks.
	tasks := []thehive.Task{
		{
			Title:       "Remediate Kerberoastable service accounts",
			Description: fmt.Sprintf("Investigate and rotate passwords for %d Kerberoastable accounts. Use AES256 encryption and enforce managed service accounts (gMSA) where possible.", summary.KerberoastableCount),
			Status:      "Waiting",
			Order:       1,
		},
		{
			Title:       "Review DCSync principals",
			Description: fmt.Sprintf("Audit %d principals with DCSync privileges (GetChanges + GetChangesAll). Remove unnecessary rights and monitor for suspicious replication traffic.", summary.DCSyncCount),
			Status:      "Waiting",
			Order:       2,
		},
		{
			Title:       "Validate and block critical attack paths",
			Description: fmt.Sprintf("Review %d critical attack paths identified by BloodHound. Prioritize by ImpactValue and apply ACL hardening, group policy, and tiered admin model.", summary.CriticalPathCount),
			Status:      "Waiting",
			Order:       3,
		},
	}
	for _, t := range tasks {
		if _, err := hive.CreateTask(ctx, created.ID, t); err != nil {
			// Non-fatal — case was created successfully.
			_ = err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// resolveDomainID looks up the BloodHound domain ID for a given FQDN or
// display name.  Returns an error if the domain is not found or not collected.
func (inv *BloodHoundInvestigator) resolveDomainID(ctx context.Context, domainName string) (string, error) {
	domains, err := inv.bh.ListDomains(ctx)
	if err != nil {
		return "", err
	}
	normalized := strings.ToUpper(strings.TrimSpace(domainName))
	for _, d := range domains {
		if strings.ToUpper(d.Name) == normalized || d.ID == domainName {
			if !d.Collected {
				return "", fmt.Errorf("domain %q exists but has not been collected by SharpHound", domainName)
			}
			return d.ID, nil
		}
	}
	return "", fmt.Errorf("domain %q not found in BloodHound", domainName)
}

// containsPrincipal checks whether name appears in steps (case-insensitive).
func containsPrincipal(steps []string, name string) bool {
	name = strings.ToUpper(name)
	for _, s := range steps {
		if strings.ToUpper(s) == name {
			return true
		}
	}
	return false
}

// edgeKindToTechnique maps BloodHound edge kinds to MITRE ATT&CK technique IDs.
var edgeKindToTechnique = map[string]string{
	"HasSession":       "T1078",   // Valid Accounts
	"AdminTo":          "T1021",   // Remote Services
	"MemberOf":         "T1069",   // Permission Groups Discovery
	"GenericAll":       "T1222",   // File/Directory Permissions Modification
	"GenericWrite":     "T1222",
	"WriteDacl":        "T1222",
	"WriteOwner":       "T1222",
	"DCSync":           "T1003.006", // OS Credential Dumping: DCSync
	"GetChanges":       "T1003.006",
	"GetChangesAll":    "T1003.006",
	"AllExtendedRights": "T1484",  // Domain Policy Modification
	"ForceChangePassword": "T1098", // Account Manipulation
	"Kerberoastable":   "T1558.003", // Steal or Forge Kerberos Tickets: Kerberoasting
}

// edgesToTechniques maps a slice of path edges to deduplicated ATT&CK technique IDs.
func edgesToTechniques(edges []bloodhound.PathEdge) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, e := range edges {
		if tech, ok := edgeKindToTechnique[e.Kind]; ok {
			if _, dup := seen[tech]; !dup {
				seen[tech] = struct{}{}
				out = append(out, tech)
			}
		}
	}
	return out
}
