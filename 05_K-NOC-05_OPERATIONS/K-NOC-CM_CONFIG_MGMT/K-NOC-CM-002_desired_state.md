# K-NOC-CM-002 -- Desired State Configuration

**Role:** Define and enforce desired infrastructure state across managed endpoints using Salt state declarations. The NOC service continuously reconciles actual vs desired state.

---

## 1. Desired State Model

```go
// internal/noc/desired_state.go
package noc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
)

// DesiredState represents the target configuration for a managed endpoint.
type DesiredState struct {
	NodeID       string            `json:"node_id"`
	TenantID     string            `json:"tenant_id"`
	Hostname     string            `json:"hostname"`
	Packages     []PackageState    `json:"packages"`
	Services     []ServiceState    `json:"services"`
	Files        []FileState       `json:"files"`
	FirewallRules []FirewallRule   `json:"firewall_rules"`
	Users        []UserState       `json:"users"`
	Version      int               `json:"version"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Hash         string            `json:"hash"`
}

type PackageState struct {
	Name    string `json:"name"`
	Version string `json:"version"` // "" = latest, "absent" = remove
	Source  string `json:"source,omitempty"` // custom repo URL
}

type ServiceState struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
}

type FileState struct {
	Path     string `json:"path"`
	Content  string `json:"content,omitempty"`
	Source   string `json:"source,omitempty"` // MinIO URL
	Mode     string `json:"mode"`
	Owner    string `json:"owner"`
	Group    string `json:"group"`
	Template bool   `json:"template"` // Jinja2 template
}

type FirewallRule struct {
	Chain    string `json:"chain"` // INPUT, OUTPUT, FORWARD
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Source   string `json:"source,omitempty"`
	Action   string `json:"action"` // ACCEPT, DROP, REJECT
	Comment  string `json:"comment"`
}

type UserState struct {
	Name      string   `json:"name"`
	UID       int      `json:"uid"`
	Groups    []string `json:"groups"`
	Shell     string   `json:"shell"`
	Locked    bool     `json:"locked"`
	SSHKeys   []string `json:"ssh_keys,omitempty"`
}

// ComputeHash generates a deterministic hash of the desired state.
func (ds *DesiredState) ComputeHash() string {
	data, _ := json.Marshal(struct {
		Packages      []PackageState  `json:"packages"`
		Services      []ServiceState  `json:"services"`
		Files         []FileState     `json:"files"`
		FirewallRules []FirewallRule  `json:"firewall_rules"`
		Users         []UserState     `json:"users"`
	}{ds.Packages, ds.Services, ds.Files, ds.FirewallRules, ds.Users})

	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
```

---

## 2. State Store (PostgreSQL)

```sql
-- migrations/012_desired_state.sql
CREATE TABLE IF NOT EXISTS desired_states (
    node_id       UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    hostname      TEXT NOT NULL,
    state_json    JSONB NOT NULL,
    state_hash    TEXT NOT NULL,
    version       INT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_desired_states_tenant ON desired_states(tenant_id);
CREATE INDEX idx_desired_states_hash   ON desired_states(state_hash);

-- Track applied states for drift detection
CREATE TABLE IF NOT EXISTS applied_states (
    id            BIGSERIAL PRIMARY KEY,
    node_id       UUID NOT NULL REFERENCES desired_states(node_id),
    state_hash    TEXT NOT NULL,
    applied_at    TIMESTAMPTZ DEFAULT NOW(),
    success       BOOLEAN NOT NULL,
    error_message TEXT,
    duration_ms   INT,
    changes       JSONB  -- List of what changed
);

CREATE INDEX idx_applied_states_node ON applied_states(node_id, applied_at DESC);
```

---

## 3. Desired State Repository

```go
// internal/noc/desired_state_repo.go
package noc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type DesiredStateRepo struct {
	db *sql.DB
}

func NewDesiredStateRepo(db *sql.DB) *DesiredStateRepo {
	return &DesiredStateRepo{db: db}
}

// Get retrieves the current desired state for a node.
func (r *DesiredStateRepo) Get(ctx context.Context, nodeID string) (*DesiredState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT node_id, tenant_id, hostname, state_json, state_hash, version, updated_at
		FROM desired_states WHERE node_id = $1
	`, nodeID)

	var ds DesiredState
	var stateJSON []byte
	err := row.Scan(&ds.NodeID, &ds.TenantID, &ds.Hostname,
		&stateJSON, &ds.Hash, &ds.Version, &ds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get desired state: %w", err)
	}

	if err := json.Unmarshal(stateJSON, &ds); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &ds, nil
}

// Upsert creates or updates the desired state for a node.
func (r *DesiredStateRepo) Upsert(ctx context.Context, ds *DesiredState) error {
	ds.Hash = ds.ComputeHash()
	ds.UpdatedAt = time.Now().UTC()

	stateJSON, err := json.Marshal(ds)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO desired_states (node_id, tenant_id, hostname, state_json, state_hash, version, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (node_id) DO UPDATE SET
			state_json = EXCLUDED.state_json,
			state_hash = EXCLUDED.state_hash,
			version = desired_states.version + 1,
			updated_at = EXCLUDED.updated_at
	`, ds.NodeID, ds.TenantID, ds.Hostname, stateJSON, ds.Hash, ds.Version, ds.UpdatedAt)

	return err
}

// RecordApplied records an applied state result.
func (r *DesiredStateRepo) RecordApplied(
	ctx context.Context,
	nodeID, stateHash string,
	success bool,
	errMsg string,
	durationMs int,
	changes interface{},
) error {
	changesJSON, _ := json.Marshal(changes)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO applied_states (node_id, state_hash, success, error_message, duration_ms, changes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, nodeID, stateHash, success, errMsg, durationMs, changesJSON)
	return err
}
```

---

## 4. Reconciliation Engine

```go
// internal/noc/reconciler.go
package noc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
)

// Reconciler compares desired vs actual state and triggers enforcement.
type Reconciler struct {
	repo     *DesiredStateRepo
	nc       *nats.Conn
	interval time.Duration
}

func NewReconciler(repo *DesiredStateRepo, nc *nats.Conn) *Reconciler {
	return &Reconciler{
		repo:     repo,
		nc:       nc,
		interval: 15 * time.Minute,
	}
}

// DriftReport describes differences between desired and actual state.
type DriftReport struct {
	NodeID    string      `json:"node_id"`
	TenantID  string      `json:"tenant_id"`
	Hostname  string      `json:"hostname"`
	Drifts    []DriftItem `json:"drifts"`
	CheckedAt time.Time   `json:"checked_at"`
}

type DriftItem struct {
	Category     string `json:"category"` // package, service, file, firewall, user
	Resource     string `json:"resource"`
	DesiredValue string `json:"desired_value"`
	ActualValue  string `json:"actual_value"`
	Action       string `json:"action"` // install, remove, start, stop, update, create
}

// CheckDrift compares desired state against agent-reported actual state.
func (r *Reconciler) CheckDrift(
	ctx context.Context,
	nodeID string,
	actualState map[string]interface{},
) (*DriftReport, error) {
	desired, err := r.repo.Get(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	report := &DriftReport{
		NodeID:    nodeID,
		TenantID:  desired.TenantID,
		Hostname:  desired.Hostname,
		CheckedAt: time.Now().UTC(),
	}

	// Check packages
	if actualPkgs, ok := actualState["packages"].(map[string]interface{}); ok {
		for _, pkg := range desired.Packages {
			actual, exists := actualPkgs[pkg.Name]
			if pkg.Version == "absent" && exists {
				report.Drifts = append(report.Drifts, DriftItem{
					Category: "package", Resource: pkg.Name,
					DesiredValue: "absent", ActualValue: fmt.Sprint(actual),
					Action: "remove",
				})
			} else if pkg.Version != "absent" && !exists {
				report.Drifts = append(report.Drifts, DriftItem{
					Category: "package", Resource: pkg.Name,
					DesiredValue: pkg.Version, ActualValue: "not installed",
					Action: "install",
				})
			} else if pkg.Version != "" && pkg.Version != "absent" && fmt.Sprint(actual) != pkg.Version {
				report.Drifts = append(report.Drifts, DriftItem{
					Category: "package", Resource: pkg.Name,
					DesiredValue: pkg.Version, ActualValue: fmt.Sprint(actual),
					Action: "update",
				})
			}
		}
	}

	// Check services
	if actualSvcs, ok := actualState["services"].(map[string]interface{}); ok {
		for _, svc := range desired.Services {
			actual, exists := actualSvcs[svc.Name]
			if !exists {
				report.Drifts = append(report.Drifts, DriftItem{
					Category: "service", Resource: svc.Name,
					DesiredValue: "present", ActualValue: "not found",
					Action: "install",
				})
				continue
			}
			if svcMap, ok := actual.(map[string]interface{}); ok {
				if running, ok := svcMap["running"].(bool); ok && running != svc.Running {
					action := "start"
					if !svc.Running {
						action = "stop"
					}
					report.Drifts = append(report.Drifts, DriftItem{
						Category: "service", Resource: svc.Name,
						DesiredValue: fmt.Sprint(svc.Running), ActualValue: fmt.Sprint(running),
						Action: action,
					})
				}
			}
		}
	}

	return report, nil
}

// PublishDrift sends drift report to NATS for alerting and auto-remediation.
func (r *Reconciler) PublishDrift(report *DriftReport) error {
	if len(report.Drifts) == 0 {
		return nil
	}

	event := map[string]interface{}{
		"class_uid":    6003, // OCSF ComplianceFinding
		"activity_id":  1,
		"category_uid": 6,
		"severity_id":  3,
		"time":         report.CheckedAt.Format(time.RFC3339),
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("Configuration drift detected on %s: %d items",
				report.Hostname, len(report.Drifts)),
			"uid": fmt.Sprintf("drift-%s-%d", report.NodeID, report.CheckedAt.Unix()),
		},
		"metadata": map[string]interface{}{
			"product":    map[string]string{"name": "Kubric NOC", "vendor_name": "Kubric"},
			"tenant_uid": report.TenantID,
		},
		"unmapped": map[string]interface{}{
			"drifts":   report.Drifts,
			"hostname": report.Hostname,
		},
	}

	data, _ := json.Marshal(event)
	return r.nc.Publish(
		fmt.Sprintf("kubric.noc.drift.%s", report.TenantID), data,
	)
}
```

---

## 5. Salt State Generation

The desired state model is translated to Salt SLS files for enforcement:

```go
// internal/noc/salt_generator.go
package noc

import (
	"fmt"
	"strings"
)

// GenerateSLS converts a DesiredState to a Salt SLS YAML string.
func GenerateSLS(ds *DesiredState) string {
	var b strings.Builder

	b.WriteString("# Auto-generated by Kubric NOC — DO NOT EDIT\n")
	b.WriteString(fmt.Sprintf("# Node: %s  Hash: %s\n\n", ds.Hostname, ds.Hash))

	// Packages
	for _, pkg := range ds.Packages {
		if pkg.Version == "absent" {
			b.WriteString(fmt.Sprintf("%s:\n  pkg.removed\n\n", pkg.Name))
		} else {
			b.WriteString(fmt.Sprintf("%s:\n  pkg.installed:\n", pkg.Name))
			if pkg.Version != "" {
				b.WriteString(fmt.Sprintf("    - version: '%s'\n", pkg.Version))
			}
			b.WriteString("\n")
		}
	}

	// Services
	for _, svc := range ds.Services {
		action := "running"
		if !svc.Running {
			action = "dead"
		}
		b.WriteString(fmt.Sprintf("%s:\n  service.%s:\n", svc.Name, action))
		b.WriteString(fmt.Sprintf("    - enable: %t\n\n", svc.Enabled))
	}

	// Files
	for _, f := range ds.Files {
		b.WriteString(fmt.Sprintf("%s:\n  file.managed:\n", f.Path))
		if f.Source != "" {
			b.WriteString(fmt.Sprintf("    - source: %s\n", f.Source))
		}
		b.WriteString(fmt.Sprintf("    - mode: '%s'\n", f.Mode))
		b.WriteString(fmt.Sprintf("    - user: %s\n", f.Owner))
		b.WriteString(fmt.Sprintf("    - group: %s\n", f.Group))
		if f.Template {
			b.WriteString("    - template: jinja\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}
```
