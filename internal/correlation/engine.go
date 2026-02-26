// Package correlation is the Kubric detection-response pipeline.
//
// It subscribes to all agent event subjects via NATS JetStream, buffers events
// in a rolling 5-minute window per tenant, evaluates correlation rules, and
// creates incidents when patterns are matched.  Incidents are dispatched to
// TheHive (AGPL-3.0 boundary: HTTP only) and Shuffle for SOAR playbooks.
package correlation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/managekube-hue/Kubric-UiDR/internal/shuffle"
	"github.com/managekube-hue/Kubric-UiDR/internal/thehive"
	nats "github.com/nats-io/nats.go"
)

// ---------------------------------------------------------------------------
// NATS subject constants — must match the agent NATS subject scheme:
//
//	kubric.{tenant_id}.{category}.{class}.v1
//
// The wildcard (*) matches any single tenant_id component.
// ---------------------------------------------------------------------------

const (
	SubjProcessEvent   = "kubric.*.endpoint.process.v1"
	SubjFimEvent       = "kubric.*.endpoint.fim.v1"
	SubjNetworkEvent   = "kubric.*.network.activity.v1"
	SubjNetworkAlert   = "kubric.*.detection.network_ids.v1"
	SubjThresholdAlert = "kubric.*.detection.threshold.v1"
	SubjWazuhAlert     = "kubric.*.wazuh.alert.v1"
	SubjFalcoAlert     = "kubric.*.falco.alert.v1"
	SubjSuricataAlert  = "kubric.*.suricata.alert.v1"
)

// allSubjects enumerates every NATS subject the engine subscribes to.
var allSubjects = []string{
	SubjProcessEvent,
	SubjFimEvent,
	SubjNetworkEvent,
	SubjNetworkAlert,
	SubjThresholdAlert,
	SubjWazuhAlert,
	SubjFalcoAlert,
	SubjSuricataAlert,
}

// bufferWindow is how long events are retained in the per-tenant rolling window.
const bufferWindow = 5 * time.Minute

// maxBufferPerKey caps the number of events stored per (tenant, eventType) key
// to prevent unbounded memory growth under high load.
const maxBufferPerKey = 1000

// ---------------------------------------------------------------------------
// NormalizedEvent — canonical event format ingested from any source
// ---------------------------------------------------------------------------

// NormalizedEvent is the canonical in-memory representation of any event
// ingested from CoreSec, NetGuard, Wazuh, Falco, or Suricata.
type NormalizedEvent struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	AgentID        string                 `json:"agent_id"`
	Source         string                 `json:"source"`         // "coresec","netguard","wazuh","falco","suricata"
	EventType      string                 `json:"event_type"`     // "process","network","fim","alert"
	Severity       int                    `json:"severity"`       // 1=info 2=low 3=medium 4=high 5=critical
	Timestamp      time.Time              `json:"timestamp"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Indicators     []string               `json:"indicators"`     // IPs, hashes, domains, filenames
	MITRETactic    string                 `json:"mitre_tactic,omitempty"`
	MITRETechnique string                 `json:"mitre_technique,omitempty"`
	RawData        map[string]interface{} `json:"raw_data"`
	Fingerprint    string                 `json:"fingerprint"`    // event-level dedup key
}

// ---------------------------------------------------------------------------
// CorrelationRule — declarative pattern definition
// ---------------------------------------------------------------------------

// CorrelationRule defines a pattern that, when matched against the event
// buffer, triggers the creation of an Incident.
type CorrelationRule struct {
	ID             string
	Name           string
	Description    string
	Severity       int           // incident severity when triggered (1–5)
	Window         time.Duration // time window for co-occurrence detection
	Conditions     []RuleCondition
	MITRETactic    string
	MITRETechnique string
	Threshold      int // minimum number of events matching (for quantity-based rules)
}

// RuleCondition is a single predicate within a CorrelationRule.
// A rule fires only when ALL of its conditions are satisfied (AND semantics).
type RuleCondition struct {
	Source     string            // "any", "coresec", "netguard", "wazuh", "falco", "suricata"
	EventType  string            // "process", "network", "fim", "alert", "any"
	FieldMatch map[string]string // field path → required substring (case-insensitive)
	MinCount   int               // minimum events matching this condition; defaults to 1
}

// ---------------------------------------------------------------------------
// Incident — correlated finding
// ---------------------------------------------------------------------------

// Incident is a correlated security finding derived from one or more events.
type Incident struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Severity       int               `json:"severity"`
	Status         string            `json:"status"` // "new","investigating","resolved"
	Events         []NormalizedEvent `json:"events"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Fingerprint    string            `json:"fingerprint"`
	TheHiveID      string            `json:"thehive_id,omitempty"`
	MITRETactic    string            `json:"mitre_tactic,omitempty"`
	MITRETechnique string            `json:"mitre_technique,omitempty"`
	RuleID         string            `json:"rule_id"`
	RuleName       string            `json:"rule_name"`
}

// ---------------------------------------------------------------------------
// EngineMetrics — runtime counters
// ---------------------------------------------------------------------------

// EngineMetrics holds atomic counters for the correlation engine's runtime
// behaviour, exposed via the /detection/health endpoint.
type EngineMetrics struct {
	EventsIngested   atomic.Int64
	RulesEvaluated   atomic.Int64
	IncidentsCreated atomic.Int64
	DedupHits        atomic.Int64
}

// ---------------------------------------------------------------------------
// Engine
// ---------------------------------------------------------------------------

// Engine ingests events from NATS, correlates them into incidents, and
// dispatches those incidents to TheHive and Shuffle.
type Engine struct {
	nc      *nats.Conn
	js      nats.JetStreamContext // nil when JetStream is unavailable
	thehive *thehive.Client       // nil when TheHive is not configured
	shuffle *shuffle.Client       // nil when Shuffle is not configured

	rules []CorrelationRule

	// Event buffer — keyed by "<tenantID>:<eventType>".
	// Protected by mu.
	mu          sync.RWMutex
	eventBuffer map[string][]NormalizedEvent

	// In-memory incident store.
	// Protected by mu (same lock; incidents are low-volume).
	incidentMap map[string]Incident

	// Dedup cache — fingerprint → last seen time.
	// Protected by dedupMu.
	dedupMu    sync.Mutex
	dedupCache map[string]time.Time

	// Internal dispatch channel.
	incidentCh chan Incident

	// Metrics counters.
	metrics EngineMetrics
}

// New creates a new correlation Engine and connects it to NATS.
// thehive and shuffle may both be nil; the engine handles all combinations.
func New(natsURL string, th *thehive.Client, sh *shuffle.Client) (*Engine, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1), // reconnect indefinitely
		nats.ReconnectWait(3*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("correlation: NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Println("correlation: NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("correlation: nats connect %q: %w", natsURL, err)
	}

	js, err := nc.JetStream()
	if err != nil {
		// JetStream is optional — fall back to core NATS subscriptions.
		log.Printf("correlation: JetStream unavailable (%v); using core NATS", err)
		js = nil
	}

	return &Engine{
		nc:          nc,
		js:          js,
		thehive:     th,
		shuffle:     sh,
		rules:       DefaultRules(),
		eventBuffer: make(map[string][]NormalizedEvent),
		incidentMap: make(map[string]Incident),
		dedupCache:  make(map[string]time.Time),
		incidentCh:  make(chan Incident, 512),
	}, nil
}

// SetRules replaces the active rule set (useful for tenant-specific rules or testing).
func (e *Engine) SetRules(rules []CorrelationRule) {
	e.rules = rules
}

// Start runs the entire event processing pipeline.
// It blocks until ctx is cancelled, then drains cleanly.
func (e *Engine) Start(ctx context.Context) error {
	subs := make([]*nats.Subscription, 0, len(allSubjects))
	for _, subj := range allSubjects {
		sub, err := e.nc.Subscribe(subj, func(msg *nats.Msg) {
			e.processEvent(msg.Subject, msg.Data)
		})
		if err != nil {
			return fmt.Errorf("correlation: subscribe %q: %w", subj, err)
		}
		subs = append(subs, sub)
	}
	log.Printf("correlation: engine started; subscribed to %d subjects", len(subs))

	// Correlation evaluator — fires every 10 seconds.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, inc := range e.evaluateRules() {
					select {
					case e.incidentCh <- inc:
					default:
						log.Printf("correlation: incident channel full; dropping incident %s", inc.ID)
					}
				}
			}
		}
	}()

	// Dedup cache cleanup — removes entries older than 1 h every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.cleanDedupCache()
			}
		}
	}()

	// Event buffer pruner — removes events beyond bufferWindow every minute.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.pruneBuffer()
			}
		}
	}()

	// Incident dispatcher — sends incidents to TheHive and Shuffle.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case inc := <-e.incidentCh:
				dispCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				e.dispatchIncident(dispCtx, inc)
				cancel()
			}
		}
	}()

	<-ctx.Done()

	for _, sub := range subs {
		_ = sub.Unsubscribe()
	}
	_ = e.nc.Drain()
	log.Println("correlation: engine stopped")
	return nil
}

// ---------------------------------------------------------------------------
// Event ingestion
// ---------------------------------------------------------------------------

// processEvent normalizes a raw NATS message and adds it to the event buffer.
func (e *Engine) processEvent(subject string, data []byte) {
	event, err := parseEvent(subject, data)
	if err != nil {
		log.Printf("correlation: parse error on subject %q: %v", subject, err)
		return
	}
	e.metrics.EventsIngested.Add(1)

	key := event.TenantID + ":" + event.EventType
	e.mu.Lock()
	buf := e.eventBuffer[key]
	if len(buf) >= maxBufferPerKey {
		buf = buf[1:] // drop the oldest element
	}
	e.eventBuffer[key] = append(buf, event)
	e.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Rule evaluation
// ---------------------------------------------------------------------------

// evaluateRules iterates all rules against the current event buffer and
// returns a list of new, deduplicated Incidents.
func (e *Engine) evaluateRules() []Incident {
	now := time.Now()
	e.metrics.RulesEvaluated.Add(int64(len(e.rules)))

	// Snapshot the buffer under read lock so evaluation does not hold the lock.
	e.mu.RLock()
	snapshot := make(map[string][]NormalizedEvent, len(e.eventBuffer))
	for k, v := range e.eventBuffer {
		cp := make([]NormalizedEvent, len(v))
		copy(cp, v)
		snapshot[k] = cp
	}
	e.mu.RUnlock()

	// Group all within-window events by tenant.
	type byTenant map[string][]NormalizedEvent

	tenantSnapshots := make(byTenant)
	for key, events := range snapshot {
		colonIdx := strings.Index(key, ":")
		if colonIdx < 0 {
			continue
		}
		tenantID := key[:colonIdx]
		cutoffDefault := now.Add(-bufferWindow)
		for _, ev := range events {
			if ev.Timestamp.After(cutoffDefault) {
				tenantSnapshots[tenantID] = append(tenantSnapshots[tenantID], ev)
			}
		}
	}

	var triggered []Incident

	for _, rule := range e.rules {
		cutoff := now.Add(-rule.Window)

		for tenantID, allEvents := range tenantSnapshots {
			// Filter to events within this rule's specific window.
			var windowEvents []NormalizedEvent
			for _, ev := range allEvents {
				if ev.Timestamp.After(cutoff) {
					windowEvents = append(windowEvents, ev)
				}
			}
			if len(windowEvents) == 0 {
				continue
			}

			matchedEvents, matched := matchRule(rule, windowEvents)
			if !matched {
				continue
			}

			fp := e.fingerprint(matchedEvents, rule, tenantID)

			// Dedup check — suppress if seen within 2× the rule window.
			e.dedupMu.Lock()
			lastSeen, seen := e.dedupCache[fp]
			if seen && now.Sub(lastSeen) < rule.Window*2 {
				e.dedupMu.Unlock()
				e.metrics.DedupHits.Add(1)
				continue
			}
			e.dedupCache[fp] = now
			e.dedupMu.Unlock()

			incID := uuid.NewString()
			inc := Incident{
				ID:             incID,
				TenantID:       tenantID,
				Title:          rule.Name,
				Description:    rule.Description,
				Severity:       rule.Severity,
				Status:         "new",
				Events:         matchedEvents,
				CreatedAt:      now,
				UpdatedAt:      now,
				Fingerprint:    fp,
				MITRETactic:    rule.MITRETactic,
				MITRETechnique: rule.MITRETechnique,
				RuleID:         rule.ID,
				RuleName:       rule.Name,
			}

			e.mu.Lock()
			e.incidentMap[incID] = inc
			e.mu.Unlock()

			e.metrics.IncidentsCreated.Add(1)
			triggered = append(triggered, inc)
		}
	}

	return triggered
}

// matchRule tests whether the given events satisfy all conditions of a rule.
// It returns the contributing events and true when the rule fires.
func matchRule(rule CorrelationRule, events []NormalizedEvent) ([]NormalizedEvent, bool) {
	var contributing []NormalizedEvent

	for _, cond := range rule.Conditions {
		var hits []NormalizedEvent
		for _, ev := range events {
			if matchCondition(cond, ev) {
				hits = append(hits, ev)
			}
		}
		required := cond.MinCount
		if required <= 0 {
			required = 1
		}
		if rule.Threshold > 0 && required < rule.Threshold {
			required = rule.Threshold
		}
		if len(hits) < required {
			return nil, false
		}
		contributing = append(contributing, hits...)
	}

	if len(contributing) == 0 {
		return nil, false
	}

	// Deduplicate contributing events by ID while preserving order.
	seen := make(map[string]struct{}, len(contributing))
	unique := contributing[:0]
	for _, ev := range contributing {
		if _, ok := seen[ev.ID]; !ok {
			seen[ev.ID] = struct{}{}
			unique = append(unique, ev)
		}
	}
	return unique, true
}

// matchCondition returns true when a NormalizedEvent satisfies a single RuleCondition.
func matchCondition(cond RuleCondition, ev NormalizedEvent) bool {
	if cond.Source != "" && cond.Source != "any" && ev.Source != cond.Source {
		return false
	}
	if cond.EventType != "" && cond.EventType != "any" && ev.EventType != cond.EventType {
		return false
	}
	for field, pattern := range cond.FieldMatch {
		val := extractFieldValue(ev, field)
		if !strings.Contains(strings.ToLower(val), strings.ToLower(pattern)) {
			return false
		}
	}
	return true
}

// extractFieldValue pulls a string value from a NormalizedEvent or its RawData map.
func extractFieldValue(ev NormalizedEvent, field string) string {
	switch field {
	case "title":
		return ev.Title
	case "description":
		return ev.Description
	case "source":
		return ev.Source
	case "event_type":
		return ev.EventType
	case "mitre_tactic":
		return ev.MITRETactic
	case "mitre_technique":
		return ev.MITRETechnique
	case "indicators":
		return strings.Join(ev.Indicators, " ")
	}
	if ev.RawData != nil {
		if v, ok := ev.RawData[field]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Fingerprinting
// ---------------------------------------------------------------------------

// fingerprint computes a stable SHA-256 dedup key for an incident.
// Key components: ruleID + tenantID + sorted (agentID:eventType:title) tuples.
func (e *Engine) fingerprint(events []NormalizedEvent, rule CorrelationRule, tenantID string) string {
	parts := []string{rule.ID, tenantID}

	seen := make(map[string]struct{})
	for _, ev := range events {
		k := ev.AgentID + ":" + ev.EventType + ":" + ev.Title
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			parts = append(parts, k)
		}
	}
	// Sort only the event-derived parts for determinism.
	sort.Strings(parts[2:])

	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:16]) // 16 bytes = 32 hex chars is sufficient
}

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

// dispatchIncident forwards an incident to TheHive (as an alert) and to
// Shuffle (as a workflow execution payload).
func (e *Engine) dispatchIncident(ctx context.Context, incident Incident) {
	if e.thehive != nil {
		e.dispatchToTheHive(ctx, incident)
	}
	if e.shuffle != nil {
		e.dispatchToShuffle(ctx, incident)
	}
}

// dispatchToTheHive creates a TheHive alert for the incident.
func (e *Engine) dispatchToTheHive(ctx context.Context, incident Incident) {
	tags := []string{
		"kubric",
		"tenant:" + incident.TenantID,
		"rule:" + incident.RuleID,
	}
	if incident.MITRETactic != "" {
		tags = append(tags, "mitre:"+incident.MITRETactic)
	}
	if incident.MITRETechnique != "" {
		tags = append(tags, "technique:"+incident.MITRETechnique)
	}

	alert := thehive.Alert{
		Title:       fmt.Sprintf("[%s] %s", incident.TenantID, incident.Title),
		Description: buildMarkdownDescription(incident),
		Severity:    mapSeverityToTheHive(incident.Severity),
		Date:        incident.CreatedAt.UnixMilli(),
		TLP:         2, // AMBER
		PAP:         2, // AMBER
		Type:        "kubric-correlation",
		Source:      "kubric",
		SourceRef:   incident.Fingerprint, // unique per dedup window → natural dedup
		Tags:        tags,
		Observables: extractObservables(incident.Events),
	}

	created, err := e.thehive.CreateAlert(ctx, alert)
	if err != nil {
		log.Printf("correlation: thehive: create alert for incident %s: %v", incident.ID, err)
		return
	}

	// Persist the TheHive alert ID back into the incident map.
	e.mu.Lock()
	if inc, ok := e.incidentMap[incident.ID]; ok {
		inc.TheHiveID = created.ID
		inc.UpdatedAt = time.Now()
		e.incidentMap[incident.ID] = inc
	}
	e.mu.Unlock()

	log.Printf("correlation: incident %s → TheHive alert %s", incident.ID, created.ID)

	// Auto-promote high/critical incidents to cases.
	if incident.Severity >= 4 {
		if _, err := e.thehive.PromoteAlert(ctx, created.ID); err != nil {
			log.Printf("correlation: thehive: promote alert %s: %v", created.ID, err)
		} else {
			log.Printf("correlation: TheHive alert %s promoted to case (severity %d)",
				created.ID, incident.Severity)
		}
	}
}

// dispatchToShuffle triggers a Shuffle workflow for SOAR automation.
func (e *Engine) dispatchToShuffle(ctx context.Context, incident Incident) {
	workflows, err := e.shuffle.ListWorkflows(ctx)
	if err != nil {
		log.Printf("correlation: shuffle: list workflows for incident %s: %v", incident.ID, err)
		return
	}
	workflowID := selectShuffleWorkflow(workflows, incident.Severity, incident.MITRETactic)
	if workflowID == "" {
		// No matching workflow — not an error; SOAR is optional.
		return
	}

	payload := buildShufflePayload(incident)
	exec, err := e.shuffle.ExecuteWorkflow(ctx, workflowID, payload)
	if err != nil {
		log.Printf("correlation: shuffle: execute workflow %s for incident %s: %v",
			workflowID, incident.ID, err)
		return
	}
	log.Printf("correlation: incident %s → Shuffle execution %s (workflow %s)",
		incident.ID, exec.ID, workflowID)
}

// DispatchByID re-dispatches a known incident on demand (for the /dispatch API endpoint).
func (e *Engine) DispatchByID(ctx context.Context, incidentID string) error {
	e.mu.RLock()
	inc, ok := e.incidentMap[incidentID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("incident %s not found", incidentID)
	}
	e.dispatchIncident(ctx, inc)
	return nil
}

// ---------------------------------------------------------------------------
// Maintenance goroutine helpers
// ---------------------------------------------------------------------------

// cleanDedupCache removes entries older than 1 hour from the dedup cache.
func (e *Engine) cleanDedupCache() {
	cutoff := time.Now().Add(-1 * time.Hour)
	e.dedupMu.Lock()
	defer e.dedupMu.Unlock()
	for fp, t := range e.dedupCache {
		if t.Before(cutoff) {
			delete(e.dedupCache, fp)
		}
	}
}

// pruneBuffer removes events outside the bufferWindow from the event buffer.
func (e *Engine) pruneBuffer() {
	cutoff := time.Now().Add(-bufferWindow)
	e.mu.Lock()
	defer e.mu.Unlock()
	for key, events := range e.eventBuffer {
		fresh := events[:0]
		for _, ev := range events {
			if ev.Timestamp.After(cutoff) {
				fresh = append(fresh, ev)
			}
		}
		if len(fresh) == 0 {
			delete(e.eventBuffer, key)
		} else {
			e.eventBuffer[key] = fresh
		}
	}
}

// ---------------------------------------------------------------------------
// Query methods — used by the NOC detection HTTP handler
// ---------------------------------------------------------------------------

// ListIncidents returns incidents filtered and paginated.
// An empty tenantID, zero severity, or empty status disables the respective filter.
func (e *Engine) ListIncidents(tenantID string, severity int, status string, limit, offset int) []Incident {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []Incident
	for _, inc := range e.incidentMap {
		if tenantID != "" && inc.TenantID != tenantID {
			continue
		}
		if severity > 0 && inc.Severity != severity {
			continue
		}
		if status != "" && inc.Status != status {
			continue
		}
		result = append(result, inc)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if offset >= len(result) {
		return []Incident{}
	}
	result = result[offset:]
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

// GetIncident returns a single incident by ID, or false if not found.
func (e *Engine) GetIncident(id string) (Incident, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	inc, ok := e.incidentMap[id]
	return inc, ok
}

// UpdateIncidentStatus atomically sets the status of an incident.
func (e *Engine) UpdateIncidentStatus(id, status string) (Incident, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	inc, ok := e.incidentMap[id]
	if !ok {
		return Incident{}, false
	}
	inc.Status = status
	inc.UpdatedAt = time.Now()
	e.incidentMap[id] = inc
	return inc, true
}

// ListTimeline returns raw NormalizedEvents for a tenant within the time range.
// since/until zero values are treated as open bounds.
func (e *Engine) ListTimeline(tenantID string, since, until time.Time, limit int) []NormalizedEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []NormalizedEvent
	for key, events := range e.eventBuffer {
		colonIdx := strings.Index(key, ":")
		if colonIdx < 0 || key[:colonIdx] != tenantID {
			continue
		}
		for _, ev := range events {
			afterSince := since.IsZero() || ev.Timestamp.After(since)
			beforeUntil := until.IsZero() || ev.Timestamp.Before(until)
			if afterSince && beforeUntil {
				result = append(result, ev)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

// Metrics returns the engine's current runtime counters as a plain map.
func (e *Engine) Metrics() map[string]int64 {
	e.mu.RLock()
	bufKeys := len(e.eventBuffer)
	incTotal := len(e.incidentMap)
	e.mu.RUnlock()

	return map[string]int64{
		"events_ingested":   e.metrics.EventsIngested.Load(),
		"rules_evaluated":   e.metrics.RulesEvaluated.Load(),
		"incidents_created": e.metrics.IncidentsCreated.Load(),
		"dedup_hits":        e.metrics.DedupHits.Load(),
		"buffer_key_count":  int64(bufKeys),
		"incident_total":    int64(incTotal),
		"rule_count":        int64(len(e.rules)),
	}
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// mapSeverityToTheHive translates Kubric severity (1–5) → TheHive severity (1–4).
func mapSeverityToTheHive(s int) int {
	switch {
	case s <= 2:
		return 1
	case s == 3:
		return 2
	case s == 4:
		return 3
	default: // 5
		return 4
	}
}

// extractObservables builds a deduplicated list of TheHive observables from event indicators.
func extractObservables(events []NormalizedEvent) []thehive.Observable {
	seen := make(map[string]struct{})
	var obs []thehive.Observable
	for _, ev := range events {
		for _, ind := range ev.Indicators {
			if _, ok := seen[ind]; ok {
				continue
			}
			seen[ind] = struct{}{}
			obs = append(obs, thehive.Observable{
				DataType: classifyIndicator(ind),
				Data:     ind,
				TLP:      2,
				IOC:      true,
				Sighted:  true,
			})
		}
	}
	return obs
}

// classifyIndicator heuristically determines the TheHive dataType for an IOC string.
func classifyIndicator(ind string) string {
	// IPv4: four dot-separated decimal octets.
	parts := strings.Split(ind, ".")
	if len(parts) == 4 {
		allNums := true
		for _, p := range parts {
			for _, c := range p {
				if c < '0' || c > '9' {
					allNums = false
					break
				}
			}
		}
		if allNums {
			return "ip"
		}
		return "domain"
	}
	// Hashes by length.
	switch len(ind) {
	case 32: // MD5
		return "hash"
	case 40: // SHA-1
		return "hash"
	case 64: // SHA-256
		return "hash"
	case 128: // BLAKE3 / SHA-512
		return "hash"
	}
	if strings.HasPrefix(ind, "http://") || strings.HasPrefix(ind, "https://") {
		return "url"
	}
	return "other"
}

// buildMarkdownDescription creates a formatted Markdown description for a TheHive alert.
func buildMarkdownDescription(inc Incident) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Correlated Incident: %s\n\n", inc.Title))
	sb.WriteString(fmt.Sprintf("**Tenant:** `%s`  \n", inc.TenantID))
	sb.WriteString(fmt.Sprintf("**Rule:** `%s` (%s)  \n", inc.RuleName, inc.RuleID))
	sb.WriteString(fmt.Sprintf("**Severity:** %d  \n", inc.Severity))
	if inc.MITRETactic != "" {
		sb.WriteString(fmt.Sprintf("**MITRE Tactic:** %s  \n", inc.MITRETactic))
	}
	if inc.MITRETechnique != "" {
		sb.WriteString(fmt.Sprintf("**MITRE Technique:** %s  \n", inc.MITRETechnique))
	}
	sb.WriteString("\n### Description\n\n")
	sb.WriteString(inc.Description)
	sb.WriteString("\n\n### Contributing Events\n\n")
	for _, ev := range inc.Events {
		sb.WriteString(fmt.Sprintf("- **[%s]** `%s` — %s (%s)\n",
			ev.Source, ev.EventType, ev.Title, ev.Timestamp.Format(time.RFC3339)))
	}
	return sb.String()
}

// buildShufflePayload constructs the execution argument map sent to Shuffle.
func buildShufflePayload(inc Incident) map[string]interface{} {
	eventIDs := make([]string, 0, len(inc.Events))
	for _, ev := range inc.Events {
		eventIDs = append(eventIDs, ev.ID)
	}

	// Collect unique indicators across all contributing events.
	seen := make(map[string]struct{})
	var indicators []string
	for _, ev := range inc.Events {
		for _, ind := range ev.Indicators {
			if _, ok := seen[ind]; !ok {
				seen[ind] = struct{}{}
				indicators = append(indicators, ind)
			}
		}
	}

	return map[string]interface{}{
		"incident_id":        inc.ID,
		"tenant_id":          inc.TenantID,
		"title":              inc.Title,
		"description":        inc.Description,
		"severity":           inc.Severity,
		"status":             inc.Status,
		"mitre_tactic":       inc.MITRETactic,
		"mitre_technique":    inc.MITRETechnique,
		"rule_id":            inc.RuleID,
		"rule_name":          inc.RuleName,
		"thehive_alert_id":   inc.TheHiveID,
		"fingerprint":        inc.Fingerprint,
		"event_ids":          eventIDs,
		"indicators":         indicators,
		"created_at":         inc.CreatedAt.Format(time.RFC3339),
	}
}

// selectShuffleWorkflow picks the best Shuffle workflow for the incident by
// matching tags.  Priority: exact severity tag > MITRE tactic tag > generic
// "kubric-correlation" tag.
func selectShuffleWorkflow(workflows []shuffle.Workflow, severity int, tactic string) string {
	severityTag := fmt.Sprintf("severity-%d", severity)
	tacticTag := "mitre:" + strings.ToLower(tactic)

	var genericID, tacticID, severityID string
	for _, wf := range workflows {
		if !wf.IsValid {
			continue
		}
		for _, tag := range wf.Tags {
			switch {
			case tag == severityTag:
				severityID = wf.ID
			case tactic != "" && strings.EqualFold(tag, tacticTag):
				tacticID = wf.ID
			case tag == "kubric-correlation":
				genericID = wf.ID
			}
		}
	}

	switch {
	case severityID != "":
		return severityID
	case tacticID != "":
		return tacticID
	default:
		return genericID
	}
}

// ---------------------------------------------------------------------------
// JSON helpers (not part of the exported API but used across the package)
// ---------------------------------------------------------------------------

// marshalRaw converts any value into a map[string]interface{} for RawData.
func marshalRaw(v interface{}) map[string]interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{"_error": err.Error()}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]interface{}{"_raw": string(b)}
	}
	return m
}
