# K-SOC-IS-003 -- Multi-Module Incident Correlation / Stitching

**Role:** Correlate events from EDR, NDR, and ITDR streams into unified security incidents using temporal windowing and entity linking.

---

## 1. Architecture

```
  kubric.edr.*  ──┐
  kubric.ndr.*  ──┼──►  Incident Stitcher  ──►  kubric.soc.incident.>
  kubric.itdr.* ──┘     (Go service)        │
                            │                ├──►  ClickHouse (history)
                            │                └──►  NATS JetStream (state)
                            ▼
                    ┌───────────────┐
                    │  5-min sliding │
                    │  window per   │
                    │  entity        │
                    └───────────────┘
                            │
                    ┌───────┴───────┐
                    │ Entity Link   │
                    │ IP → host →   │
                    │ user → proc → │
                    │ file chain    │
                    └───────────────┘
                            │
                    ┌───────┴───────┐
                    │ MITRE ATT&CK  │
                    │ Technique     │
                    │ Assignment    │
                    └───────────────┘
```

---

## 2. NATS Subscription Setup

```go
// internal/stitcher/subscriber.go
package stitcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type EventSubscriber struct {
	js       jetstream.JetStream
	stitcher *IncidentStitcher
}

func NewEventSubscriber(nc *nats.Conn, stitcher *IncidentStitcher) (*EventSubscriber, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	// Create durable consumers for each detection domain
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     "KUBRIC_DETECTIONS",
		Subjects: []string{"kubric.edr.>", "kubric.ndr.>", "kubric.itdr.>"},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

	return &EventSubscriber{js: js, stitcher: stitcher}, nil
}

func (es *EventSubscriber) Start(ctx context.Context) error {
	consumer, err := es.js.CreateOrUpdateConsumer(ctx, "KUBRIC_DETECTIONS",
		jetstream.ConsumerConfig{
			Durable:       "incident-stitcher",
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: "kubric.*.>", // edr, ndr, itdr
			MaxDeliver:    3,
			AckWait:       30 * time.Second,
		},
	)
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	iter, err := consumer.Messages()
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			iter.Stop()
			return ctx.Err()
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			continue
		}

		var event OCSFEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			msg.Nak()
			continue
		}

		event.NATSSubject = msg.Subject()
		es.stitcher.Ingest(event)
		msg.Ack()
	}
}
```

---

## 3. Incident Stitcher Core

```go
// internal/stitcher/engine.go
package stitcher

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	CorrelationWindow = 5 * time.Minute
	MaxEventsPerIncident = 500
)

// OCSFEvent is a minimal OCSF event for stitching purposes.
type OCSFEvent struct {
	ClassUID    int                    `json:"class_uid"`
	ActivityID  int                    `json:"activity_id"`
	SeverityID  int                    `json:"severity_id"`
	Time        string                 `json:"time"`
	SrcEndpoint *Endpoint              `json:"src_endpoint,omitempty"`
	DstEndpoint *Endpoint              `json:"dst_endpoint,omitempty"`
	Process     *Process               `json:"process,omitempty"`
	Actor       map[string]interface{} `json:"actor,omitempty"`
	File        map[string]interface{} `json:"file,omitempty"`
	FindingInfo map[string]interface{} `json:"finding_info,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Unmapped    map[string]interface{} `json:"unmapped,omitempty"`
	NATSSubject string                 `json:"-"`
}

type Endpoint struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
}

type Process struct {
	PID     int    `json:"pid,omitempty"`
	Name    string `json:"name,omitempty"`
	CmdLine string `json:"cmd_line,omitempty"`
}

// Incident aggregates correlated events.
type Incident struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Events      []OCSFEvent `json:"events"`
	Entities    EntityChain `json:"entities"`
	Techniques  []string    `json:"mitre_techniques"`
	Severity    int         `json:"severity"`
	Score       float64     `json:"score"`
	FirstSeen   time.Time   `json:"first_seen"`
	LastSeen    time.Time   `json:"last_seen"`
	Status      string      `json:"status"` // open, investigating, resolved
}

// EntityChain links related entities.
type EntityChain struct {
	IPs       map[string]bool `json:"ips"`
	Hostnames map[string]bool `json:"hostnames"`
	Users     map[string]bool `json:"users"`
	Processes map[string]bool `json:"processes"`
	Files     map[string]bool `json:"files"`
}

type IncidentStitcher struct {
	mu        sync.RWMutex
	incidents map[string]*Incident // keyed by entity hash
	nc        *nats.Conn
}

func NewIncidentStitcher(nc *nats.Conn) *IncidentStitcher {
	s := &IncidentStitcher{
		incidents: make(map[string]*Incident),
		nc:        nc,
	}
	go s.cleanupLoop()
	return s
}

// Ingest processes an incoming OCSF event.
func (s *IncidentStitcher) Ingest(event OCSFEvent) {
	entities := extractEntities(event)
	tenantID := extractTenantID(event)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find existing incident by entity overlap
	var matched *Incident
	for _, inc := range s.incidents {
		if inc.TenantID != tenantID {
			continue
		}
		if time.Since(inc.LastSeen) > CorrelationWindow {
			continue
		}
		if entitiesOverlap(inc.Entities, entities) {
			matched = inc
			break
		}
	}

	if matched == nil {
		// Create new incident
		incID := fmt.Sprintf("INC-%s-%d", tenantID[:8], time.Now().UnixMilli())
		matched = &Incident{
			ID:        incID,
			TenantID:  tenantID,
			Entities:  entities,
			FirstSeen: time.Now(),
			Status:    "open",
		}
		s.incidents[incID] = matched
	}

	// Add event to incident
	if len(matched.Events) < MaxEventsPerIncident {
		matched.Events = append(matched.Events, event)
	}
	matched.LastSeen = time.Now()
	mergeEntities(&matched.Entities, entities)

	// Assign MITRE techniques
	techniques := mapToMITRE(event)
	for _, t := range techniques {
		if !contains(matched.Techniques, t) {
			matched.Techniques = append(matched.Techniques, t)
		}
	}

	// Recalculate severity
	matched.Severity = calculateSeverity(matched)
	matched.Score = calculateScore(matched)

	// Publish updated incident
	s.publishIncident(matched)
}

func (s *IncidentStitcher) publishIncident(inc *Incident) {
	data, _ := json.Marshal(inc)
	subject := fmt.Sprintf("kubric.soc.incident.%s", inc.TenantID)
	_ = s.nc.Publish(subject, data)
}

func (s *IncidentStitcher) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for id, inc := range s.incidents {
			if time.Since(inc.LastSeen) > 30*time.Minute {
				delete(s.incidents, id)
			}
		}
		s.mu.Unlock()
	}
}
```

---

## 4. Entity Extraction and Linking

```go
// internal/stitcher/entities.go
package stitcher

func extractEntities(event OCSFEvent) EntityChain {
	ec := EntityChain{
		IPs:       make(map[string]bool),
		Hostnames: make(map[string]bool),
		Users:     make(map[string]bool),
		Processes: make(map[string]bool),
		Files:     make(map[string]bool),
	}

	if event.SrcEndpoint != nil {
		if event.SrcEndpoint.IP != "" {
			ec.IPs[event.SrcEndpoint.IP] = true
		}
		if event.SrcEndpoint.Hostname != "" {
			ec.Hostnames[event.SrcEndpoint.Hostname] = true
		}
	}

	if event.DstEndpoint != nil {
		if event.DstEndpoint.IP != "" {
			ec.IPs[event.DstEndpoint.IP] = true
		}
		if event.DstEndpoint.Hostname != "" {
			ec.Hostnames[event.DstEndpoint.Hostname] = true
		}
	}

	if event.Process != nil && event.Process.Name != "" {
		ec.Processes[event.Process.Name] = true
	}

	if actor, ok := event.Actor["user"].(map[string]interface{}); ok {
		if name, ok := actor["name"].(string); ok {
			ec.Users[name] = true
		}
	}

	if name, ok := event.File["name"].(string); ok {
		ec.Files[name] = true
	}

	return ec
}

func entitiesOverlap(a, b EntityChain) bool {
	return mapOverlap(a.IPs, b.IPs) ||
		mapOverlap(a.Hostnames, b.Hostnames) ||
		mapOverlap(a.Users, b.Users) ||
		mapOverlap(a.Processes, b.Processes) ||
		mapOverlap(a.Files, b.Files)
}

func mapOverlap(a, b map[string]bool) bool {
	for k := range a {
		if b[k] {
			return true
		}
	}
	return false
}

func mergeEntities(dst *EntityChain, src EntityChain) {
	for k := range src.IPs       { dst.IPs[k] = true }
	for k := range src.Hostnames { dst.Hostnames[k] = true }
	for k := range src.Users     { dst.Users[k] = true }
	for k := range src.Processes { dst.Processes[k] = true }
	for k := range src.Files     { dst.Files[k] = true }
}

func extractTenantID(event OCSFEvent) string {
	if meta, ok := event.Metadata["tenant_uid"].(string); ok {
		return meta
	}
	return "default"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

---

## 5. MITRE ATT&CK Technique Assignment

```go
// internal/stitcher/mitre.go
package stitcher

// mapToMITRE assigns MITRE ATT&CK technique IDs based on event characteristics.
func mapToMITRE(event OCSFEvent) []string {
	var techniques []string

	// Check finding_info for pre-assigned techniques
	if fi, ok := event.FindingInfo["analytic"].(map[string]interface{}); ok {
		if cat, ok := fi["category"].(string); ok {
			if t := lookupTechniqueFromCategory(cat); t != "" {
				techniques = append(techniques, t)
			}
		}
	}

	// Heuristic technique assignment by event class
	switch event.ClassUID {
	case 1007: // ProcessActivity
		if proc := event.Process; proc != nil {
			techniques = append(techniques, classifyProcess(proc)...)
		}
	case 4001: // NetworkActivity
		if dst := event.DstEndpoint; dst != nil {
			techniques = append(techniques, classifyNetwork(dst)...)
		}
	case 3002: // AuthenticationActivity
		techniques = append(techniques, "T1078") // Valid Accounts
	case 4003: // DNSActivity
		techniques = append(techniques, "T1071.004") // DNS C2
	}

	return techniques
}

func classifyProcess(proc *Process) []string {
	var t []string
	switch {
	case proc.Name == "mimikatz.exe" || proc.Name == "sekurlsa":
		t = append(t, "T1003.001") // LSASS Memory
	case proc.Name == "net.exe":
		t = append(t, "T1087") // Account Discovery
	case proc.Name == "powershell.exe":
		t = append(t, "T1059.001") // PowerShell
	case proc.Name == "cmd.exe":
		t = append(t, "T1059.003") // Windows Command Shell
	case proc.Name == "psexec.exe" || proc.Name == "PSEXESVC.exe":
		t = append(t, "T1570") // Lateral Tool Transfer
	case proc.Name == "schtasks.exe":
		t = append(t, "T1053.005") // Scheduled Task
	case proc.Name == "reg.exe":
		t = append(t, "T1112") // Modify Registry
	}
	return t
}

func classifyNetwork(dst *Endpoint) []string {
	var t []string
	switch dst.Port {
	case 4444, 5555:
		t = append(t, "T1571") // Non-Standard Port
	case 53:
		t = append(t, "T1071.004") // DNS
	case 443, 8443:
		t = append(t, "T1071.001") // Web Protocols
	case 22:
		t = append(t, "T1021.004") // SSH
	case 3389:
		t = append(t, "T1021.001") // RDP
	}
	return t
}

func lookupTechniqueFromCategory(category string) string {
	mapping := map[string]string{
		"trojan-activity":         "T1105",
		"web-application-attack":  "T1190",
		"attempted-admin":         "T1068",
		"credential_access":       "T1003",
		"lateral_movement":        "T1021",
	}
	return mapping[category]
}
```

---

## 6. Severity Scoring Algorithm

```go
// internal/stitcher/scoring.go
package stitcher

import "math"

// calculateSeverity returns the highest severity in the incident.
func calculateSeverity(inc *Incident) int {
	max := 0
	for _, e := range inc.Events {
		if e.SeverityID > max {
			max = e.SeverityID
		}
	}
	return max
}

// calculateScore computes a 0-100 incident score.
// Factors: event count, severity, entity breadth, technique count, time span.
func calculateScore(inc *Incident) float64 {
	eventScore := math.Min(float64(len(inc.Events))/10.0, 1.0) * 25.0
	severityScore := float64(inc.Severity) / 5.0 * 30.0

	entityCount := len(inc.Entities.IPs) + len(inc.Entities.Hostnames) +
		len(inc.Entities.Users) + len(inc.Entities.Processes) +
		len(inc.Entities.Files)
	entityScore := math.Min(float64(entityCount)/10.0, 1.0) * 20.0

	techniqueScore := math.Min(float64(len(inc.Techniques))/5.0, 1.0) * 15.0

	timeSpan := inc.LastSeen.Sub(inc.FirstSeen).Minutes()
	timeScore := 10.0
	if timeSpan < 1 {
		timeScore = 10.0 // Very fast = likely automated attack
	} else if timeSpan > 30 {
		timeScore = 5.0
	}

	return math.Min(eventScore+severityScore+entityScore+techniqueScore+timeScore, 100.0)
}
```

---

## 7. ClickHouse Historical Correlation

```sql
-- ClickHouse query for correlating historical events around an incident.
SELECT
    event_time,
    ocsf_class_uid,
    tenant_id,
    src_ip,
    dst_ip,
    hostname,
    user_name,
    process_name,
    finding_title,
    severity_id,
    nats_subject
FROM kubric.security_events
WHERE tenant_id = {tenant_id:String}
  AND event_time BETWEEN {start_time:DateTime64} AND {end_time:DateTime64}
  AND (
      src_ip IN ({ips:Array(String)})
      OR dst_ip IN ({ips:Array(String)})
      OR hostname IN ({hostnames:Array(String)})
      OR user_name IN ({users:Array(String)})
  )
ORDER BY event_time ASC
LIMIT 1000
```
