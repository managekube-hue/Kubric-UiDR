# K-SOC-TI-014 -- OpenCTI Connector Integration

**License:** Apache 2.0  
**Execution model:** Python subprocess — isolated process, not embedded.  
**Role:** Import STIX2 threat intelligence from OpenCTI and publish to Kubric's detection pipeline via NATS.

---

## 1. Architecture

```
┌──────────────┐     GraphQL      ┌──────────────┐
│  OpenCTI     │◄────────────────►│  Python      │
│  Platform    │     API          │  Connector   │
│              │                  │  (subprocess)│
└──────────────┘                  └──────┬───────┘
                                         │ STIX2 bundles
                                         ▼
                                  ┌──────────────┐
                                  │  Go Service  │
                                  │  (SOC/TI)    │
                                  │              │
                                  │  Parse STIX2 │
                                  │  Dedup IOCs  │
                                  │  Publish NATS│
                                  └──────────────┘
                                         │
                              kubric.ti.ioc.{tenant_id}
```

---

## 2. OpenCTI Connector Configuration

```yaml
# config/opencti/connector.yaml
opencti:
  url: "https://opencti.internal:8080"
  token: "${OPENCTI_API_TOKEN}"
  ssl_verify: true

connector:
  id: "kubric-threat-intel-ingester"
  type: "EXTERNAL_IMPORT"
  name: "Kubric TI Ingester"
  scope: "indicator,malware,attack-pattern,tool,vulnerability"
  confidence_level: 80
  update_existing_data: true
  log_level: "info"

# Connector types used:
# - EXTERNAL_IMPORT: Pull indicators, malware, attack patterns
# - INTERNAL_ENRICHMENT: Enrich existing entities with additional context
```

---

## 3. Python Connector Script

```python
#!/usr/bin/env python3
# scripts/opencti_connector.py
"""
OpenCTI connector for Kubric threat intelligence pipeline.
Runs as a subprocess, outputs STIX2 bundles to stdout as JSON lines.
"""

import json
import os
import sys
import time
from datetime import datetime, timedelta, timezone

import requests

OPENCTI_URL = os.getenv("OPENCTI_URL", "https://opencti.internal:8080")
OPENCTI_TOKEN = os.getenv("OPENCTI_API_TOKEN", "")
POLL_INTERVAL = int(os.getenv("OPENCTI_POLL_INTERVAL", "3600"))  # 1 hour
OUTPUT_FILE = os.getenv("OPENCTI_OUTPUT", "/tmp/opencti-stix-bundles.jsonl")


def graphql_query(query: str, variables: dict = None) -> dict:
    """Execute a GraphQL query against OpenCTI."""
    headers = {
        "Authorization": f"Bearer {OPENCTI_TOKEN}",
        "Content-Type": "application/json",
    }
    payload = {"query": query}
    if variables:
        payload["variables"] = variables

    resp = requests.post(
        f"{OPENCTI_URL}/graphql",
        json=payload,
        headers=headers,
        timeout=60,
        verify=True,
    )
    resp.raise_for_status()
    return resp.json()


def fetch_indicators(after_date: str = None) -> list[dict]:
    """Fetch STIX2 indicators from OpenCTI."""
    date_filter = ""
    if after_date:
        date_filter = f', filters: {{ mode: and, filters: [{{ key: "created_at", values: ["{after_date}"], operator: gt }}], filterGroups: [] }}'

    query = f"""
    query GetIndicators($first: Int, $after: ID) {{
        indicators(first: $first, after: $after{date_filter}) {{
            edges {{
                node {{
                    id
                    standard_id
                    name
                    description
                    pattern
                    pattern_type
                    valid_from
                    valid_until
                    confidence
                    created
                    modified
                    x_opencti_score
                    killChainPhases {{
                        edges {{
                            node {{
                                kill_chain_name
                                phase_name
                            }}
                        }}
                    }}
                    objectLabel {{
                        edges {{
                            node {{
                                value
                                color
                            }}
                        }}
                    }}
                }}
            }}
            pageInfo {{
                hasNextPage
                endCursor
            }}
        }}
    }}
    """

    all_indicators = []
    cursor = None

    while True:
        variables = {"first": 100}
        if cursor:
            variables["after"] = cursor

        result = graphql_query(query, variables)
        data = result.get("data", {}).get("indicators", {})
        edges = data.get("edges", [])

        for edge in edges:
            node = edge["node"]
            # Convert to STIX2 indicator format
            stix_indicator = {
                "type": "indicator",
                "spec_version": "2.1",
                "id": node["standard_id"],
                "created": node["created"],
                "modified": node["modified"],
                "name": node["name"],
                "description": node.get("description", ""),
                "pattern": node["pattern"],
                "pattern_type": node["pattern_type"],
                "valid_from": node["valid_from"],
                "valid_until": node.get("valid_until"),
                "confidence": node.get("confidence", 50),
                "x_kubric_score": node.get("x_opencti_score", 50),
                "kill_chain_phases": [
                    {
                        "kill_chain_name": kc["node"]["kill_chain_name"],
                        "phase_name": kc["node"]["phase_name"],
                    }
                    for kc in node.get("killChainPhases", {}).get("edges", [])
                ],
                "labels": [
                    label["node"]["value"]
                    for label in node.get("objectLabel", {}).get("edges", [])
                ],
            }
            all_indicators.append(stix_indicator)

        page_info = data.get("pageInfo", {})
        if not page_info.get("hasNextPage"):
            break
        cursor = page_info["endCursor"]

    return all_indicators


def fetch_malware() -> list[dict]:
    """Fetch malware entities as STIX2 objects."""
    query = """
    query GetMalware($first: Int) {
        malwares(first: $first) {
            edges {
                node {
                    id
                    standard_id
                    name
                    description
                    malware_types
                    is_family
                    first_seen
                    last_seen
                    confidence
                }
            }
        }
    }
    """
    result = graphql_query(query, {"first": 500})
    edges = result.get("data", {}).get("malwares", {}).get("edges", [])

    return [
        {
            "type": "malware",
            "spec_version": "2.1",
            "id": edge["node"]["standard_id"],
            "name": edge["node"]["name"],
            "description": edge["node"].get("description", ""),
            "malware_types": edge["node"].get("malware_types", []),
            "is_family": edge["node"].get("is_family", False),
            "first_seen": edge["node"].get("first_seen"),
            "last_seen": edge["node"].get("last_seen"),
        }
        for edge in edges
    ]


def main():
    """Main polling loop — outputs STIX2 bundles as JSON lines."""
    last_poll = (datetime.now(timezone.utc) - timedelta(hours=24)).isoformat()

    while True:
        try:
            indicators = fetch_indicators(after_date=last_poll)
            malware = fetch_malware()

            if indicators or malware:
                bundle = {
                    "type": "bundle",
                    "id": f"bundle--kubric-{int(time.time())}",
                    "objects": indicators + malware,
                }

                # Write to output file for Go service consumption
                with open(OUTPUT_FILE, "a") as f:
                    f.write(json.dumps(bundle) + "\n")

                # Also write to stdout for piped consumption
                print(json.dumps(bundle), flush=True)

                sys.stderr.write(
                    f"[OpenCTI] Fetched {len(indicators)} indicators, "
                    f"{len(malware)} malware entities\n"
                )

            last_poll = datetime.now(timezone.utc).isoformat()

        except Exception as e:
            sys.stderr.write(f"[OpenCTI] Error: {e}\n")

        time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
```

---

## 4. Go Subprocess Manager

```go
// internal/threatintel/opencti.go
package threatintel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	nats "github.com/nats-io/nats.go"
)

type OpenCTIConnector struct {
	pythonBin    string
	scriptPath   string
	nc           *nats.Conn
	tenantID     string
	deduplicator *IOCDeduplicator
}

func NewOpenCTIConnector(nc *nats.Conn, tenantID string) *OpenCTIConnector {
	return &OpenCTIConnector{
		pythonBin:    "python3",
		scriptPath:   "scripts/opencti_connector.py",
		nc:           nc,
		tenantID:     tenantID,
		deduplicator: NewIOCDeduplicator(),
	}
}

type STIXBundle struct {
	Type    string        `json:"type"`
	ID      string        `json:"id"`
	Objects []STIXObject  `json:"objects"`
}

type STIXObject struct {
	Type         string   `json:"type"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	PatternType  string   `json:"pattern_type,omitempty"`
	ValidFrom    string   `json:"valid_from,omitempty"`
	Confidence   int      `json:"confidence,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	KubricScore  int      `json:"x_kubric_score,omitempty"`
}

// Start runs the Python connector as a subprocess and processes its output.
func (oc *OpenCTIConnector) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, oc.pythonBin, oc.scriptPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start opencti connector: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024) // 10MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		var bundle STIXBundle
		if err := json.Unmarshal([]byte(line), &bundle); err != nil {
			continue
		}

		for _, obj := range bundle.Objects {
			if oc.deduplicator.IsDuplicate(obj.ID) {
				continue
			}

			if err := oc.publishSTIXObject(obj); err != nil {
				continue
			}
		}
	}

	return cmd.Wait()
}

func (oc *OpenCTIConnector) publishSTIXObject(obj STIXObject) error {
	severity := 3 // Medium
	if obj.Confidence > 80 || obj.KubricScore > 80 {
		severity = 4 // High
	}

	event := map[string]interface{}{
		"class_uid":    2001,
		"activity_id":  1,
		"category_uid": 2,
		"severity_id":  severity,
		"time":         time.Now().UTC().Format(time.RFC3339),
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("OpenCTI %s: %s", obj.Type, obj.Name),
			"uid":   obj.ID,
			"types": []string{"Threat Intelligence", obj.Type},
			"analytic": map[string]string{
				"name": "OpenCTI Connector",
				"type": "STIX2 Import",
			},
		},
		"metadata": map[string]interface{}{
			"product":    map[string]string{"name": "OpenCTI", "vendor_name": "Filigran"},
			"tenant_uid": oc.tenantID,
		},
		"unmapped": map[string]interface{}{
			"stix_type":    obj.Type,
			"stix_id":      obj.ID,
			"pattern":      obj.Pattern,
			"pattern_type": obj.PatternType,
			"confidence":   obj.Confidence,
			"labels":       obj.Labels,
		},
	}

	data, _ := json.Marshal(event)
	return oc.nc.Publish(
		fmt.Sprintf("kubric.ti.ioc.%s", oc.tenantID), data,
	)
}
```

---

## 5. IOC Deduplication

```go
// internal/threatintel/dedup.go
package threatintel

import (
	"sync"
	"time"
)

// IOCDeduplicator tracks seen STIX IDs to avoid duplicate processing.
type IOCDeduplicator struct {
	mu   sync.RWMutex
	seen map[string]time.Time
}

func NewIOCDeduplicator() *IOCDeduplicator {
	d := &IOCDeduplicator{seen: make(map[string]time.Time)}
	go d.cleanupLoop()
	return d
}

func (d *IOCDeduplicator) IsDuplicate(stixID string) bool {
	d.mu.RLock()
	_, exists := d.seen[stixID]
	d.mu.RUnlock()

	if exists {
		return true
	}

	d.mu.Lock()
	d.seen[stixID] = time.Now()
	d.mu.Unlock()
	return false
}

func (d *IOCDeduplicator) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		d.mu.Lock()
		cutoff := time.Now().Add(-24 * time.Hour)
		for id, ts := range d.seen {
			if ts.Before(cutoff) {
				delete(d.seen, id)
			}
		}
		d.mu.Unlock()
	}
}
```

---

## 6. Docker Compose Service

```yaml
# docker-compose.yml (snippet)
services:
  opencti-connector:
    build:
      context: .
      dockerfile: docker/opencti-connector/Dockerfile
    environment:
      OPENCTI_URL: https://opencti.internal:8080
      OPENCTI_API_TOKEN: ${OPENCTI_API_TOKEN}
      OPENCTI_POLL_INTERVAL: "3600"
      OPENCTI_OUTPUT: /data/stix-bundles.jsonl
    volumes:
      - opencti-data:/data
    restart: unless-stopped
    depends_on:
      - nats

volumes:
  opencti-data:
```
