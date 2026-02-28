# K-NOC-CM-004 -- Rudder Drift Detection

**License:** GPL 3.0 — runs as separate service.  
**Role:** Continuous compliance and drift detection for managed endpoints. Rudder provides an alternative/complementary CM path to SaltStack, with a focus on compliance reporting.

---

## 1. Architecture

```
┌──────────────┐   Agent       ┌──────────────┐   REST API    ┌──────────────┐
│  Managed     │──────────────►│  Rudder       │◄────────────►│  Go Service  │
│  Endpoints   │  Inventory   │  Server       │              │  (NOC)       │
│  (Rudder     │  + Reports   │              │              │              │
│   Agent)     │              │  Policy       │              │  Query drift │
└──────────────┘              │  Server       │              │  Map OCSF    │
                              │  Compliance   │              │  Publish NATS│
                              └──────────────┘              └──────────────┘
```

---

## 2. Rudder Docker Service

```yaml
# docker-compose.yml (snippet)
services:
  rudder:
    image: rudder/rudder-server:8.1
    restart: unless-stopped
    ports:
      - "443:443"     # Web UI + agent communication
      - "5309:5309"   # CFEngine agent port
    volumes:
      - rudder-data:/var/rudder
      - rudder-config:/opt/rudder/etc
      - rudder-db:/var/lib/postgresql
    environment:
      RUDDER_ADMIN_PASSWORD: ${RUDDER_ADMIN_PASSWORD}
      SERVER_ROLES: rudder-server-root
    deploy:
      resources:
        limits:
          memory: 4G
          cpus: "2.0"

volumes:
  rudder-data:
  rudder-config:
  rudder-db:
```

---

## 3. Go REST API Client

```go
// internal/noc/rudder.go
package noc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	nats "github.com/nats-io/nats.go"
)

const rudderAPIBase = "https://rudder.internal/rudder/api/latest"

// RudderClient communicates with Rudder REST API.
type RudderClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
	nc         *nats.Conn
	tenantID   string
}

func NewRudderClient(apiToken string, nc *nats.Conn, tenantID string) *RudderClient {
	return &RudderClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    rudderAPIBase,
		apiToken:   apiToken,
		nc:         nc,
		tenantID:   tenantID,
	}
}

func (rc *RudderClient) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method,
		fmt.Sprintf("%s%s", rc.baseURL, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Token", rc.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rudder api %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// ── Data Types ──────────────────────────────────────────────

// Node represents a Rudder-managed node.
type RudderNode struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Status        string `json:"status"`
	OS            string `json:"os"`
	PolicyServer  string `json:"policyServerId"`
	LastReport    string `json:"lastReport"`
	AgentVersion  string `json:"agentReportingContext,omitempty"`
	CompliancePercent float64 `json:"compliancePercent,omitempty"`
}

// ComplianceReport from Rudder.
type ComplianceReport struct {
	GlobalCompliance float64              `json:"globalCompliance"`
	Rules            []RuleCompliance     `json:"rules"`
}

type RuleCompliance struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Compliance  float64             `json:"compliance"`
	Mode        string              `json:"mode"`
	Directives  []DirectiveStatus   `json:"directives"`
}

type DirectiveStatus struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Compliance float64            `json:"compliance"`
	Components []ComponentStatus  `json:"components"`
}

type ComponentStatus struct {
	Name       string  `json:"name"`
	Compliance float64 `json:"compliance"`
	Status     string  `json:"status"` // compliant, non-compliant, error, repaired
}
```

---

## 4. Compliance Polling

```go
// internal/noc/rudder_compliance.go
package noc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetNodes returns all managed nodes.
func (rc *RudderClient) GetNodes(ctx context.Context) ([]RudderNode, error) {
	body, err := rc.doRequest(ctx, "GET", "/nodes")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Nodes []RudderNode `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Data.Nodes, nil
}

// GetNodeCompliance returns compliance for a specific node.
func (rc *RudderClient) GetNodeCompliance(ctx context.Context, nodeID string) (*ComplianceReport, error) {
	body, err := rc.doRequest(ctx, "GET", fmt.Sprintf("/compliance/nodes/%s", nodeID))
	if err != nil {
		return nil, err
	}

	var result struct {
		Data ComplianceReport `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetGlobalCompliance returns the overall compliance status.
func (rc *RudderClient) GetGlobalCompliance(ctx context.Context) (float64, error) {
	body, err := rc.doRequest(ctx, "GET", "/compliance")
	if err != nil {
		return 0, err
	}

	var result struct {
		Data struct {
			GlobalCompliance float64 `json:"globalCompliance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return result.Data.GlobalCompliance, nil
}

// PollCompliance runs a periodic compliance check and publishes drift events.
func (rc *RudderClient) PollCompliance(ctx context.Context) error {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			nodes, err := rc.GetNodes(ctx)
			if err != nil {
				continue
			}

			for _, node := range nodes {
				compliance, err := rc.GetNodeCompliance(ctx, node.ID)
				if err != nil {
					continue
				}

				// Find non-compliant rules (drift)
				var drifts []DriftItem
				for _, rule := range compliance.Rules {
					if rule.Compliance < 100.0 {
						for _, dir := range rule.Directives {
							for _, comp := range dir.Components {
								if comp.Status == "non-compliant" || comp.Status == "error" {
									drifts = append(drifts, DriftItem{
										Category:     "rudder-rule",
										Resource:     fmt.Sprintf("%s/%s/%s", rule.Name, dir.Name, comp.Name),
										DesiredValue: "compliant",
										ActualValue:  comp.Status,
										Action:       "remediate",
									})
								}
							}
						}
					}
				}

				if len(drifts) > 0 {
					rc.publishDriftEvent(node, drifts, compliance.GlobalCompliance)
				}
			}
		}
	}
}

func (rc *RudderClient) publishDriftEvent(node RudderNode, drifts []DriftItem, compliance float64) {
	severity := 2 // Low
	switch {
	case compliance < 50:
		severity = 5 // Critical
	case compliance < 75:
		severity = 4 // High
	case compliance < 90:
		severity = 3 // Medium
	}

	event := map[string]interface{}{
		"class_uid":    6003, // OCSF ComplianceFinding
		"activity_id":  1,
		"category_uid": 6,
		"severity_id":  severity,
		"time":         time.Now().UTC().Format(time.RFC3339),
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("Rudder drift: %s at %.1f%% compliance (%d issues)",
				node.Hostname, compliance, len(drifts)),
			"uid": fmt.Sprintf("rudder-drift-%s-%d", node.ID, time.Now().Unix()),
		},
		"compliance": map[string]interface{}{
			"status":     "non-compliant",
			"status_detail": fmt.Sprintf("%.1f%% compliant", compliance),
		},
		"resource": map[string]interface{}{
			"type": "Host",
			"name": node.Hostname,
			"uid":  node.ID,
		},
		"metadata": map[string]interface{}{
			"product":    map[string]string{"name": "Rudder", "vendor_name": "Rudder"},
			"tenant_uid": rc.tenantID,
		},
		"unmapped": map[string]interface{}{
			"drifts":        drifts,
			"compliance_pct": compliance,
			"node_os":       node.OS,
			"agent_version": node.AgentVersion,
		},
	}

	data, _ := json.Marshal(event)
	_ = rc.nc.Publish(
		fmt.Sprintf("kubric.noc.drift.%s", rc.tenantID), data,
	)
}
```

---

## 5. Rudder Techniques for Kubric Baseline

```json
// Rudder technique definition — kubric baseline security
{
  "name": "Kubric Security Baseline",
  "version": "1.0",
  "description": "Baseline security configuration for Kubric-managed endpoints",
  "category": "Security",
  "methods": [
    {
      "name": "Ensure SSH key-only auth",
      "method": "file_ensure_line_present_in_file",
      "params": {
        "path": "/etc/ssh/sshd_config",
        "line": "PasswordAuthentication no"
      }
    },
    {
      "name": "Ensure firewall enabled",
      "method": "service_started",
      "params": {
        "name": "firewalld"
      }
    },
    {
      "name": "Ensure NTP sync",
      "method": "service_started",
      "params": {
        "name": "chronyd"
      }
    },
    {
      "name": "Ensure Kubric agent running",
      "method": "service_started",
      "params": {
        "name": "kubric-agent"
      }
    },
    {
      "name": "Ensure audit logging",
      "method": "service_started",
      "params": {
        "name": "auditd"
      }
    }
  ]
}
```

---

## 6. Comparison: Salt vs Rudder

| Feature | SaltStack | Rudder |
|---------|-----------|--------|
| Agent | salt-minion | rudder-agent (CFEngine) |
| Compliance UI | Third party | Built-in |
| Drift detection | Custom (beacon + reactor) | Native compliance reports |
| Remediation | State apply (immediate) | Policy enforcement (scheduled) |
| API | salt-api REST | REST API v13+ |
| License | Apache 2.0 | GPL 3.0 |
| Use case | Real-time remediation | Continuous compliance |
| Kubric role | Primary CM | Compliance overlay |
