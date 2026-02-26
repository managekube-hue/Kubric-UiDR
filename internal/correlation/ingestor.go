package correlation

// ingestor.go — NATS event parsing and normalization.
//
// parseEvent is the single entry-point.  It extracts the tenant ID from the
// NATS subject, dispatches to the appropriate type-specific parser, and
// returns a NormalizedEvent ready for the buffer.
//
// Each raw struct mirrors the JSON payload that the corresponding Rust agent
// or integration bridge publishes to NATS.  Field names must match the
// serialization output of those agents exactly.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Raw event structs — mirrors of the Rust agent / integration bridge types
// ---------------------------------------------------------------------------

// rawProcessEvent mirrors coresec/src/event.rs ProcessEvent (OCSF 4007).
// Subject: kubric.{tenant_id}.endpoint.process.v1
type rawProcessEvent struct {
	TenantID   string `json:"tenant_id"`
	AgentID    string `json:"agent_id"`
	EventID    string `json:"event_id"`
	Timestamp  string `json:"timestamp"`  // RFC3339
	ClassUID   uint32 `json:"class_uid"`
	SeverityID uint8  `json:"severity_id"`
	ActivityID uint8  `json:"activity_id"`
	PID        uint32 `json:"pid"`
	PPID       uint32 `json:"ppid"`
	Executable string `json:"executable"`
	Cmdline    string `json:"cmdline"`
	User       string `json:"user"`
	Blake3Hash string `json:"blake3_hash"`
}

// rawFimEvent mirrors coresec/src/fim.rs FimEvent (OCSF 4010).
// Subject: kubric.{tenant_id}.endpoint.fim.v1
type rawFimEvent struct {
	TenantID   string `json:"tenant_id"`
	AgentID    string `json:"agent_id"`
	EventID    string `json:"event_id"`
	Timestamp  uint64 `json:"timestamp"`  // Unix ms
	ClassUID   uint32 `json:"class_uid"`
	ActivityID uint8  `json:"activity_id"` // 1=Create 2=Modify 3=Delete
	Path       string `json:"path"`
	OldHash    string `json:"old_hash"`
	NewHash    string `json:"new_hash"`
	Severity   string `json:"severity"` // "info","low","medium","high"
}

// rawNetworkActivity mirrors NetGuard capture.rs OCSF NetworkActivity (4001).
// Subject: kubric.{tenant_id}.network.activity.v1
type rawNetworkActivity struct {
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
	EventID   string `json:"event_id"`
	Timestamp uint64 `json:"timestamp_ms"`
	ClassUID  uint32 `json:"class_uid"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dst_ip"`
	SrcPort   uint32 `json:"src_port"`
	DstPort   uint32 `json:"dst_port"`
	Protocol  string `json:"protocol"` // "TCP","UDP","ICMP"
	Direction string `json:"direction"` // "inbound","outbound","internal"
	BytesIn   uint64 `json:"bytes_in"`
	BytesOut  uint64 `json:"bytes_out"`
	Payload   string `json:"payload_hex,omitempty"` // first 128 bytes, hex-encoded
}

// rawIdsAlert mirrors NetGuard ids.rs IdsAlert.
// Subject: kubric.{tenant_id}.detection.network_ids.v1
type rawIdsAlert struct {
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
	EventID   string `json:"event_id"`
	Timestamp uint64 `json:"timestamp"`
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dst_ip"`
	SrcPort   uint32 `json:"src_port"`
	DstPort   uint32 `json:"dst_port"`
	Protocol  string `json:"protocol"`
	Severity  string `json:"severity"` // "informational","low","medium","high","critical"
	ClassUID  uint32 `json:"class_uid"`
}

// rawThresholdAlert is published by Kubric's threshold-detection service.
// Subject: kubric.{tenant_id}.detection.threshold.v1
type rawThresholdAlert struct {
	TenantID   string  `json:"tenant_id"`
	AgentID    string  `json:"agent_id"`
	AlertID    string  `json:"alert_id"`
	Timestamp  string  `json:"timestamp"` // RFC3339
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Severity   int     `json:"severity"`
	Message    string  `json:"message"`
}

// rawWazuhAlert mirrors the Wazuh alert format forwarded by the
// kubric-wazuh-bridge (or via Wazuh's built-in NATS output).
// Subject: kubric.{tenant_id}.wazuh.alert.v1
type rawWazuhAlert struct {
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
	AlertID   string `json:"alert_id"`
	Timestamp string `json:"timestamp"` // RFC3339
	Rule      struct {
		ID          int      `json:"id"`
		Level       int      `json:"level"` // 0–15; ≥10 = high
		Description string   `json:"description"`
		Groups      []string `json:"groups"`
		MITRE       struct {
			Tactic    []string `json:"tactic"`
			Technique []string `json:"technique"`
		} `json:"mitre"`
	} `json:"rule"`
	Agent struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	} `json:"agent"`
	Data    map[string]interface{} `json:"data"`
	Decoder struct {
		Name string `json:"name"`
	} `json:"decoder"`
	Location string `json:"location"`
}

// rawFalcoAlert mirrors the Falco JSON output forwarded by the
// kubric-falco-bridge.
// Subject: kubric.{tenant_id}.falco.alert.v1
type rawFalcoAlert struct {
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
	AlertID   string `json:"alert_id"`
	Timestamp string `json:"time"` // RFC3339 from Falco
	Priority  string `json:"priority"` // "Emergency","Alert","Critical","Error","Warning","Notice","Informational","Debug"
	Rule      string `json:"rule"`
	Output    string `json:"output"`
	Source    string `json:"source"` // "syscall","k8s_audit","aws_cloudtrail"
	Tags      []string `json:"tags"`
	OutputFields map[string]interface{} `json:"output_fields"`
}

// rawSuricataAlert mirrors the EVE JSON format from Suricata, forwarded by
// the kubric-suricata-bridge.
// Subject: kubric.{tenant_id}.suricata.alert.v1
type rawSuricataAlert struct {
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
	Timestamp string `json:"timestamp"` // ISO 8601
	FlowID    uint64 `json:"flow_id,omitempty"`
	EventType string `json:"event_type"` // "alert"
	SrcIP     string `json:"src_ip"`
	SrcPort   int    `json:"src_port"`
	DstIP     string `json:"dest_ip"`
	DstPort   int    `json:"dest_port"`
	Protocol  string `json:"proto"`
	Alert     struct {
		Action    string `json:"action"` // "allowed","blocked"
		GID       int    `json:"gid"`
		SignatureID uint64 `json:"signature_id"`
		Rev       int    `json:"rev"`
		Signature string `json:"signature"`
		Category  string `json:"category"`
		Severity  int    `json:"severity"` // 1=high, 2=medium, 3=low
	} `json:"alert"`
	HTTP struct {
		Hostname string `json:"hostname"`
		URL      string `json:"url"`
	} `json:"http,omitempty"`
	DNS struct {
		Query []struct {
			RRName string `json:"rrname"`
			RRType string `json:"rrtype"`
		} `json:"query"`
	} `json:"dns,omitempty"`
}

// ---------------------------------------------------------------------------
// parseEvent — main subject router
// ---------------------------------------------------------------------------

// parseEvent dispatches a raw NATS message to the appropriate type-specific
// parser and returns a NormalizedEvent ready for the correlation buffer.
//
// Subject format: kubric.{tenantID}.{category}.{class}.v1
// The tenantID is extracted from the second dot-delimited segment.
func parseEvent(subject string, data []byte) (NormalizedEvent, error) {
	tenantID, err := extractTenantID(subject)
	if err != nil {
		return NormalizedEvent{}, err
	}

	switch {
	case matchesSubject(subject, SubjProcessEvent):
		return parseProcessEvent(tenantID, data)
	case matchesSubject(subject, SubjFimEvent):
		return parseFimEvent(tenantID, data)
	case matchesSubject(subject, SubjNetworkEvent):
		return parseNetworkEvent(tenantID, data)
	case matchesSubject(subject, SubjNetworkAlert):
		return parseNetworkAlertEvent(tenantID, data)
	case matchesSubject(subject, SubjThresholdAlert):
		return parseThresholdAlert(tenantID, data)
	case matchesSubject(subject, SubjWazuhAlert):
		return parseWazuhAlert(tenantID, data)
	case matchesSubject(subject, SubjFalcoAlert):
		return parseFalcoAlert(tenantID, data)
	case matchesSubject(subject, SubjSuricataAlert):
		return parseSuricataAlert(tenantID, data)
	default:
		return NormalizedEvent{}, fmt.Errorf("unrecognised subject %q", subject)
	}
}

// extractTenantID returns the tenant_id from a NATS subject in the form
// kubric.{tenantID}.{...}
func extractTenantID(subject string) (string, error) {
	parts := strings.SplitN(subject, ".", 3)
	if len(parts) < 3 || parts[0] != "kubric" || parts[1] == "" {
		return "", fmt.Errorf("cannot extract tenant_id from subject %q", subject)
	}
	return parts[1], nil
}

// matchesSubject returns true when the concrete subject matches the wildcard
// pattern (e.g. "kubric.*.endpoint.process.v1").
// We do simple segment-by-segment matching where "*" matches any single segment.
func matchesSubject(subject, pattern string) bool {
	sp := strings.Split(subject, ".")
	pp := strings.Split(pattern, ".")
	if len(sp) != len(pp) {
		return false
	}
	for i, seg := range pp {
		if seg == "*" {
			continue
		}
		if seg != sp[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Type-specific parsers
// ---------------------------------------------------------------------------

// parseProcessEvent normalizes a CoreSec OCSF-4007 process event.
func parseProcessEvent(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawProcessEvent
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("process event: %w", err)
	}

	ts, err := parseTimestamp(raw.Timestamp, 0)
	if err != nil {
		ts = time.Now()
	}

	// Build a human-readable title from the executable basename.
	execBase := baseName(raw.Executable)
	title := fmt.Sprintf("Process: %s (pid %d)", execBase, raw.PID)

	evID := raw.EventID
	if evID == "" {
		evID = uuid.NewString()
	}

	// Standard BLAKE3 hash as an indicator when present.
	var indicators []string
	if raw.Blake3Hash != "" {
		indicators = append(indicators, raw.Blake3Hash)
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "coresec",
		EventType:   "process",
		Severity:    int(raw.SeverityID),
		Timestamp:   ts,
		Title:       title,
		Description: fmt.Sprintf("Process %q (pid %d, ppid %d) spawned by user %q; cmdline: %s", raw.Executable, raw.PID, raw.PPID, raw.User, raw.Cmdline),
		Indicators:  indicators,
		RawData:     marshalRaw(raw),
		Fingerprint: raw.Blake3Hash,
	}, nil
}

// parseFimEvent normalizes a CoreSec OCSF-4010 file-integrity event.
func parseFimEvent(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawFimEvent
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("fim event: %w", err)
	}

	ts, _ := parseTimestamp("", raw.Timestamp)

	activity := fimActivityName(raw.ActivityID)
	title := fmt.Sprintf("FIM %s: %s", activity, raw.Path)

	evID := raw.EventID
	if evID == "" {
		evID = uuid.NewString()
	}

	var indicators []string
	if raw.NewHash != "" {
		indicators = append(indicators, raw.NewHash)
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "coresec",
		EventType:   "fim",
		Severity:    parseSeverityString(raw.Severity),
		Timestamp:   ts,
		Title:       title,
		Description: fmt.Sprintf("File %s: path=%q old_hash=%s new_hash=%s", activity, raw.Path, raw.OldHash, raw.NewHash),
		Indicators:  indicators,
		RawData:     marshalRaw(raw),
		Fingerprint: raw.NewHash,
	}, nil
}

// parseNetworkEvent normalizes a NetGuard OCSF-4001 network activity event.
func parseNetworkEvent(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawNetworkActivity
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("network event: %w", err)
	}

	ts, _ := parseTimestamp("", raw.Timestamp)

	evID := raw.EventID
	if evID == "" {
		evID = uuid.NewString()
	}

	title := fmt.Sprintf("Network %s: %s:%d → %s:%d (%s)",
		raw.Direction, raw.SrcIP, raw.SrcPort, raw.DstIP, raw.DstPort, raw.Protocol)

	// Both src and dst IPs as indicators when they are non-RFC1918 addresses.
	var indicators []string
	if raw.DstIP != "" && !isPrivateIP(raw.DstIP) {
		indicators = append(indicators, raw.DstIP)
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "netguard",
		EventType:   "network",
		Severity:    1, // raw network events are informational; alerts carry severity
		Timestamp:   ts,
		Title:       title,
		Description: fmt.Sprintf("%s %s→%s proto=%s bytes_in=%d bytes_out=%d",
			raw.Direction, fmt.Sprintf("%s:%d", raw.SrcIP, raw.SrcPort),
			fmt.Sprintf("%s:%d", raw.DstIP, raw.DstPort), raw.Protocol,
			raw.BytesIn, raw.BytesOut),
		Indicators: indicators,
		RawData:    marshalRaw(raw),
	}, nil
}

// parseNetworkAlertEvent normalizes a NetGuard YARA IDS alert.
func parseNetworkAlertEvent(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawIdsAlert
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("network ids alert: %w", err)
	}

	ts, _ := parseTimestamp("", raw.Timestamp)

	evID := raw.EventID
	if evID == "" {
		evID = uuid.NewString()
	}

	var indicators []string
	if raw.SrcIP != "" && !isPrivateIP(raw.SrcIP) {
		indicators = append(indicators, raw.SrcIP)
	}
	if raw.DstIP != "" && !isPrivateIP(raw.DstIP) {
		indicators = append(indicators, raw.DstIP)
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "netguard",
		EventType:   "alert",
		Severity:    parseSeverityString(raw.Severity),
		Timestamp:   ts,
		Title:       fmt.Sprintf("IDS: %s", raw.RuleName),
		Description: fmt.Sprintf("Network IDS rule %q matched: %s:%d → %s:%d (%s)",
			raw.RuleID, raw.SrcIP, raw.SrcPort, raw.DstIP, raw.DstPort, raw.Protocol),
		Indicators: indicators,
		RawData:    marshalRaw(raw),
	}, nil
}

// parseThresholdAlert normalizes a Kubric threshold-detection alert.
func parseThresholdAlert(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawThresholdAlert
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("threshold alert: %w", err)
	}

	ts, err := parseTimestamp(raw.Timestamp, 0)
	if err != nil {
		ts = time.Now()
	}

	evID := raw.AlertID
	if evID == "" {
		evID = uuid.NewString()
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "coresec",
		EventType:   "alert",
		Severity:    raw.Severity,
		Timestamp:   ts,
		Title:       fmt.Sprintf("Threshold: %s exceeded (%.2f > %.2f)", raw.MetricName, raw.Value, raw.Threshold),
		Description: raw.Message,
		RawData:     marshalRaw(raw),
	}, nil
}

// parseWazuhAlert normalizes a Wazuh HIDS alert.
func parseWazuhAlert(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawWazuhAlert
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("wazuh alert: %w", err)
	}

	ts, err := parseTimestamp(raw.Timestamp, 0)
	if err != nil {
		ts = time.Now()
	}

	evID := raw.AlertID
	if evID == "" {
		evID = uuid.NewString()
	}

	// Wazuh rule level 0–15: map to Kubric 1–5.
	sev := wazuhLevelToSeverity(raw.Rule.Level)

	// Build MITRE fields from the first entry if present.
	var tactic, technique string
	if len(raw.Rule.MITRE.Tactic) > 0 {
		tactic = raw.Rule.MITRE.Tactic[0]
	}
	if len(raw.Rule.MITRE.Technique) > 0 {
		technique = raw.Rule.MITRE.Technique[0]
	}

	// Wazuh agent IP as an indicator.
	var indicators []string
	if raw.Agent.IP != "" {
		indicators = append(indicators, raw.Agent.IP)
	}

	// Rule group as a field in RawData for condition matching.
	rd := marshalRaw(raw)
	if len(raw.Rule.Groups) > 0 {
		rd["rule_group"] = strings.Join(raw.Rule.Groups, ",")
	}

	return NormalizedEvent{
		ID:             evID,
		TenantID:       tenantID,
		AgentID:        raw.AgentID,
		Source:         "wazuh",
		EventType:      "alert",
		Severity:       sev,
		Timestamp:      ts,
		Title:          fmt.Sprintf("Wazuh[%d]: %s", raw.Rule.ID, raw.Rule.Description),
		Description:    fmt.Sprintf("Wazuh rule %d (level %d) on agent %s (%s): %s", raw.Rule.ID, raw.Rule.Level, raw.Agent.Name, raw.Agent.IP, raw.Rule.Description),
		Indicators:     indicators,
		MITRETactic:    tactic,
		MITRETechnique: technique,
		RawData:        rd,
	}, nil
}

// parseFalcoAlert normalizes a Falco runtime security alert.
func parseFalcoAlert(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawFalcoAlert
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("falco alert: %w", err)
	}

	ts, err := parseTimestamp(raw.Timestamp, 0)
	if err != nil {
		ts = time.Now()
	}

	evID := raw.AlertID
	if evID == "" {
		evID = uuid.NewString()
	}

	sev := falcoPriorityToSeverity(raw.Priority)

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "falco",
		EventType:   "alert",
		Severity:    sev,
		Timestamp:   ts,
		Title:       fmt.Sprintf("Falco: %s", raw.Rule),
		Description: raw.Output,
		RawData:     marshalRaw(raw),
	}, nil
}

// parseSuricataAlert normalizes a Suricata EVE JSON alert event.
func parseSuricataAlert(tenantID string, data []byte) (NormalizedEvent, error) {
	var raw rawSuricataAlert
	if err := json.Unmarshal(data, &raw); err != nil {
		return NormalizedEvent{}, fmt.Errorf("suricata alert: %w", err)
	}

	ts, err := parseTimestamp(raw.Timestamp, 0)
	if err != nil {
		ts = time.Now()
	}

	evID := fmt.Sprintf("suricata-%d-%d", raw.FlowID, ts.UnixNano())

	// Suricata severity: 1=high, 2=medium, 3=low → invert to Kubric 1–5.
	sev := suricataSeverityToKubric(raw.Alert.Severity)

	var indicators []string
	if raw.SrcIP != "" && !isPrivateIP(raw.SrcIP) {
		indicators = append(indicators, raw.SrcIP)
	}
	if raw.DstIP != "" && !isPrivateIP(raw.DstIP) {
		indicators = append(indicators, raw.DstIP)
	}
	if raw.HTTP.Hostname != "" {
		indicators = append(indicators, raw.HTTP.Hostname)
	}
	for _, q := range raw.DNS.Query {
		if q.RRName != "" {
			indicators = append(indicators, q.RRName)
		}
	}

	return NormalizedEvent{
		ID:          evID,
		TenantID:    tenantID,
		AgentID:     raw.AgentID,
		Source:      "suricata",
		EventType:   "alert",
		Severity:    sev,
		Timestamp:   ts,
		Title:       fmt.Sprintf("Suricata[%d]: %s", raw.Alert.SignatureID, raw.Alert.Signature),
		Description: fmt.Sprintf("Suricata alert: sig=%q category=%q action=%s %s:%d→%s:%d (%s)",
			raw.Alert.Signature, raw.Alert.Category, raw.Alert.Action,
			raw.SrcIP, raw.SrcPort, raw.DstIP, raw.DstPort, raw.Protocol),
		Indicators: indicators,
		RawData:    marshalRaw(raw),
	}, nil
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// parseTimestamp parses a string timestamp (RFC3339) or a Unix millisecond
// integer into a time.Time.  When rfc is empty and unixMS is 0, returns now.
func parseTimestamp(rfc string, unixMS uint64) (time.Time, error) {
	if rfc != "" {
		// Try RFC3339 first, then RFC3339Nano.
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, rfc); err == nil {
				return t.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse timestamp %q", rfc)
	}
	if unixMS > 0 {
		return time.UnixMilli(int64(unixMS)).UTC(), nil
	}
	return time.Now().UTC(), nil
}

// parseSeverityString maps human-readable severity strings to Kubric 1–5 scale.
func parseSeverityString(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "informational", "info", "debug", "notice":
		return 1
	case "low":
		return 2
	case "medium", "warning":
		return 3
	case "high", "error":
		return 4
	case "critical", "emergency", "alert":
		return 5
	default:
		return 2
	}
}

// wazuhLevelToSeverity maps Wazuh rule level (0–15) to Kubric severity (1–5).
func wazuhLevelToSeverity(level int) int {
	switch {
	case level <= 4:
		return 1
	case level <= 7:
		return 2
	case level <= 10:
		return 3
	case level <= 13:
		return 4
	default: // 14–15
		return 5
	}
}

// falcoPriorityToSeverity maps Falco priority strings to Kubric severity (1–5).
func falcoPriorityToSeverity(p string) int {
	switch strings.ToLower(p) {
	case "debug", "informational":
		return 1
	case "notice":
		return 2
	case "warning":
		return 3
	case "error":
		return 4
	case "critical", "alert", "emergency":
		return 5
	default:
		return 2
	}
}

// suricataSeverityToKubric inverts Suricata severity (1=high → 5, 3=low → 2).
func suricataSeverityToKubric(s int) int {
	switch s {
	case 1:
		return 4
	case 2:
		return 3
	case 3:
		return 2
	default:
		return 1
	}
}

// fimActivityName returns a human-readable activity name for OCSF activity_id.
func fimActivityName(id uint8) string {
	switch id {
	case 1:
		return "Create"
	case 2:
		return "Modify"
	case 3:
		return "Delete"
	case 4:
		return "Rename"
	default:
		return "Unknown"
	}
}

// baseName returns the last path segment for a Unix-style or Windows path.
func baseName(path string) string {
	// Try Unix path separator.
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	// Try Windows path separator.
	if idx := strings.LastIndex(path, "\\"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// isPrivateIP returns true for known RFC1918 / loopback / link-local ranges.
// This is a fast heuristic; full CIDR matching is intentionally not used here.
func isPrivateIP(ip string) bool {
	return strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "172.16.") ||
		strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") ||
		strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.2") ||
		strings.HasPrefix(ip, "172.30.") ||
		strings.HasPrefix(ip, "172.31.") ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "127.") ||
		strings.HasPrefix(ip, "169.254.") ||
		strings.HasPrefix(ip, "fc") ||
		strings.HasPrefix(ip, "fd") ||
		ip == "::1"
}
