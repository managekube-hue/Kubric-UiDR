// Package suricata parses Suricata Eve JSON alerts and provides an HTTP
// webhook receiver for log forwarders.
//
// Suricata is a GPL-2.0-licensed network IDS/IPS that outputs events in its
// Eve JSON format (file, unix socket, or HTTP).  This package contains no
// imported Suricata code -- it only parses the public Eve JSON schema.
//
// Integration modes:
//   - WebhookReceiver: mount as an http.Handler to receive Eve JSON POSTs
//     from Filebeat, Fluentd, Vector, or a custom HTTP output plugin.
//   - ParseEveFile / ParseEveStream: read Eve JSON from a log file, pipe,
//     or unix socket line-by-line.
package suricata

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Eve JSON types
// ---------------------------------------------------------------------------

// EveEvent is the unified Suricata Eve JSON format.
type EveEvent struct {
	Timestamp string                 `json:"timestamp"`
	FlowID    int64                  `json:"flow_id"`
	InIface   string                 `json:"in_iface,omitempty"`
	EventType string                 `json:"event_type"` // alert, dns, http, tls, flow, fileinfo, stats
	SrcIP     string                 `json:"src_ip"`
	SrcPort   int                    `json:"src_port"`
	DstIP     string                 `json:"dest_ip"`
	DstPort   int                    `json:"dest_port"`
	Proto     string                 `json:"proto"` // TCP, UDP, ICMP
	Community string                 `json:"community_id,omitempty"`
	Alert     *EveAlert              `json:"alert,omitempty"`
	DNS       *EveDNS                `json:"dns,omitempty"`
	HTTP      *EveHTTP               `json:"http,omitempty"`
	TLS       *EveTLS                `json:"tls,omitempty"`
	Flow      *EveFlow               `json:"flow,omitempty"`
	FileInfo  *EveFileInfo           `json:"fileinfo,omitempty"`
	AppProto  string                 `json:"app_proto,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EveAlert holds the alert-specific fields inside an Eve event.
type EveAlert struct {
	Action      string              `json:"action"`      // allowed, blocked
	GID         int                 `json:"gid"`         // generator ID
	SignatureID int                 `json:"signature_id"` // rule SID
	Rev         int                 `json:"rev"`
	Signature   string              `json:"signature"`
	Category    string              `json:"category"`
	Severity    int                 `json:"severity"` // 1=high, 2=medium, 3=low
	Metadata    map[string][]string `json:"metadata,omitempty"`
}

// EveDNS holds DNS transaction fields.
type EveDNS struct {
	Type   string `json:"type"` // query, answer
	ID     int    `json:"id"`
	RRName string `json:"rrname"`
	RRType string `json:"rrtype"`
	RCode  string `json:"rcode,omitempty"`
	RData  string `json:"rdata,omitempty"`
	TTL    int    `json:"ttl,omitempty"`
}

// EveHTTP holds HTTP transaction fields.
type EveHTTP struct {
	Hostname       string `json:"hostname"`
	URL            string `json:"url"`
	UserAgent      string `json:"http_user_agent"`
	ContentType    string `json:"http_content_type,omitempty"`
	Method         string `json:"http_method"`
	Protocol       string `json:"protocol"`
	Status         int    `json:"status"`
	Length         int64  `json:"length"`
	ReferrerHeader string `json:"http_refer,omitempty"`
}

// EveTLS holds TLS handshake fields.
type EveTLS struct {
	Subject     string `json:"subject"`
	Issuer      string `json:"issuerdn"`
	Serial      string `json:"serial"`
	Fingerprint string `json:"fingerprint"`
	SNI         string `json:"sni"`
	Version     string `json:"version"`
	NotBefore   string `json:"notbefore"`
	NotAfter    string `json:"notafter"`
	JA3         string `json:"ja3,omitempty"`
	JA3S        string `json:"ja3s,omitempty"`
}

// EveFlow holds flow summary fields emitted when a flow closes.
type EveFlow struct {
	PktsToServer  int64  `json:"pkts_toserver"`
	PktsToClient  int64  `json:"pkts_toclient"`
	BytesToServer int64  `json:"bytes_toserver"`
	BytesToClient int64  `json:"bytes_toclient"`
	Start         string `json:"start"`
	End           string `json:"end"`
	State         string `json:"state"`
	Reason        string `json:"reason"`
}

// EveFileInfo holds file extraction metadata.
type EveFileInfo struct {
	Filename string `json:"filename"`
	Magic    string `json:"magic"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5,omitempty"`
	SHA1     string `json:"sha1,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Stored   bool   `json:"stored"`
}

// EventHandler is a callback invoked for every parsed Eve event.
// Implementations must be safe for concurrent use when used with WebhookReceiver.
type EventHandler func(event EveEvent)

// ---------------------------------------------------------------------------
// WebhookReceiver
// ---------------------------------------------------------------------------

// WebhookReceiver is an http.Handler that accepts Eve JSON via HTTP POST from
// log shippers (Filebeat, Fluentd, Vector, etc.) and dispatches events to
// registered EventHandler callbacks.
//
// Mount it at a route, e.g.:
//
//	r.Post("/webhooks/suricata", receiver.ServeHTTP)
type WebhookReceiver struct {
	mu       sync.Mutex
	handlers []EventHandler
}

// NewWebhookReceiver returns a new WebhookReceiver with no handlers registered.
func NewWebhookReceiver() *WebhookReceiver {
	return &WebhookReceiver{}
}

// OnEvent registers fn to be called for every successfully-parsed Eve event.
// Handlers are invoked synchronously in registration order; if you need
// non-blocking dispatch, launch a goroutine inside the handler.
func (wr *WebhookReceiver) OnEvent(fn EventHandler) {
	if wr == nil || fn == nil {
		return
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.handlers = append(wr.handlers, fn)
}

// ServeHTTP implements http.Handler.  It accepts POST requests containing
// either a single Eve JSON object, a JSON array of events, or newline-
// delimited JSON (NDJSON) -- all three forms are common with log shippers.
func (wr *WebhookReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if wr == nil {
		http.Error(w, "suricata: receiver not initialised", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "suricata: only POST accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MiB cap
	if err != nil {
		http.Error(w, fmt.Sprintf("suricata: read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		http.Error(w, "suricata: empty body", http.StatusBadRequest)
		return
	}

	var events []EveEvent

	switch {
	case trimmed[0] == '[':
		// JSON array of events
		if err := json.Unmarshal(body, &events); err != nil {
			http.Error(w, fmt.Sprintf("suricata: unmarshal array: %v", err), http.StatusBadRequest)
			return
		}

	case strings.Contains(trimmed, "\n"):
		// Newline-delimited JSON (NDJSON) -- one Eve event per line
		scanner := bufio.NewScanner(strings.NewReader(trimmed))
		scanner.Buffer(make([]byte, 0, 1<<20), 1<<20) // 1 MiB line buffer
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var ev EveEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				http.Error(w, fmt.Sprintf("suricata: unmarshal NDJSON line: %v", err), http.StatusBadRequest)
				return
			}
			events = append(events, ev)
		}
		if err := scanner.Err(); err != nil {
			http.Error(w, fmt.Sprintf("suricata: scan NDJSON: %v", err), http.StatusBadRequest)
			return
		}

	default:
		// Single JSON object
		var ev EveEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			http.Error(w, fmt.Sprintf("suricata: unmarshal event: %v", err), http.StatusBadRequest)
			return
		}
		events = append(events, ev)
	}

	// Snapshot handlers under lock, then dispatch without holding the lock.
	wr.mu.Lock()
	snapshot := make([]EventHandler, len(wr.handlers))
	copy(snapshot, wr.handlers)
	wr.mu.Unlock()

	for i := range events {
		for _, fn := range snapshot {
			fn(events[i])
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"accepted":%d}`, len(events))))
}

// ---------------------------------------------------------------------------
// File / stream parsers
// ---------------------------------------------------------------------------

// ParseEveFile reads an Eve JSON log file line-by-line and calls handler for
// each successfully-parsed event.  It returns the number of events parsed and
// the first non-EOF error encountered (if any).
func ParseEveFile(path string, handler EventHandler) (int, error) {
	if handler == nil {
		return 0, fmt.Errorf("suricata: handler must not be nil")
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("suricata: open eve file %s: %w", path, err)
	}
	defer f.Close()

	return ParseEveStream(f, handler)
}

// ParseEveStream reads Eve JSON from any io.Reader (file, socket, stdin, pipe)
// line-by-line and calls handler for each event.  It returns the number of
// events parsed and the first non-EOF error encountered (if any).
func ParseEveStream(reader io.Reader, handler EventHandler) (int, error) {
	if reader == nil {
		return 0, fmt.Errorf("suricata: reader must not be nil")
	}
	if handler == nil {
		return 0, fmt.Errorf("suricata: handler must not be nil")
	}

	scanner := bufio.NewScanner(reader)
	// Eve JSON lines can be large (e.g. fileinfo with full metadata); allow up to 1 MiB.
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Suricata eve.json may include comment lines starting with '#' in
		// some configurations; skip them.
		if line[0] == '#' {
			continue
		}

		var ev EveEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return count, fmt.Errorf("suricata: unmarshal line %d: %w", count+1, err)
		}
		handler(ev)
		count++
	}

	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("suricata: scan stream: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Filter helpers
// ---------------------------------------------------------------------------

// FilterAlerts returns only events whose EventType is "alert".
func FilterAlerts(events []EveEvent) []EveEvent {
	out := make([]EveEvent, 0, len(events)/2)
	for i := range events {
		if events[i].EventType == "alert" {
			out = append(out, events[i])
		}
	}
	return out
}

// FilterByEventType returns events matching the given event_type (e.g.
// "dns", "http", "tls", "flow", "fileinfo", "stats").
func FilterByEventType(events []EveEvent, eventType string) []EveEvent {
	out := make([]EveEvent, 0, len(events)/4)
	for i := range events {
		if events[i].EventType == eventType {
			out = append(out, events[i])
		}
	}
	return out
}
