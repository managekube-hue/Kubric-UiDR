// Package falco integrates with the Falco runtime-security engine (Apache 2.0).
//
// Falco pushes alerts to us via its HTTP webhook output.  This package provides:
//   - WebhookReceiver: an http.Handler that accepts Falco alert POSTs and fans
//     them out to registered AlertHandler callbacks.
//   - Client: a thin HTTP client that talks to the Falco webserver (default
//     port 8765) for health-check and version introspection.
package falco

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Client is a lightweight HTTP client for the Falco webserver (GET /healthz,
// GET /version).  It is NOT used for receiving alerts -- see WebhookReceiver.
type Client struct {
	baseURL string
	hc      *http.Client
}

// Alert represents a single Falco alert as delivered through the HTTP webhook
// output channel.
type Alert struct {
	Output       string                 `json:"output"`
	Priority     string                 `json:"priority"` // Emergency, Alert, Critical, Error, Warning, Notice, Informational, Debug
	Rule         string                 `json:"rule"`
	Source       string                 `json:"source"`   // syscall, k8s_audit, aws_cloudtrail, etc.
	Hostname     string                 `json:"hostname"`
	Time         time.Time              `json:"time"`
	OutputFields map[string]interface{} `json:"output_fields"`
	Tags         []string               `json:"tags,omitempty"`
}

// AlertHandler is a callback invoked for every alert received by the
// WebhookReceiver.  Implementations must be safe for concurrent use.
type AlertHandler func(alert Alert)

// WebhookReceiver is an http.Handler that parses incoming Falco alert POSTs
// and dispatches them to registered AlertHandler callbacks.
//
// Mount it at a route, e.g.:
//
//	r.Post("/webhooks/falco", receiver.ServeHTTP)
type WebhookReceiver struct {
	mu       sync.Mutex
	handlers []AlertHandler
}

// ---------------------------------------------------------------------------
// WebhookReceiver
// ---------------------------------------------------------------------------

// NewWebhookReceiver returns a new WebhookReceiver with no handlers registered.
func NewWebhookReceiver() *WebhookReceiver {
	return &WebhookReceiver{}
}

// OnAlert registers fn to be called for every successfully-parsed alert.
// Handlers are invoked synchronously in registration order; if you need
// non-blocking dispatch, launch a goroutine inside the handler.
func (wr *WebhookReceiver) OnAlert(fn AlertHandler) {
	if wr == nil || fn == nil {
		return
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.handlers = append(wr.handlers, fn)
}

// ServeHTTP implements http.Handler.  It reads the request body, unmarshals it
// as a Falco Alert JSON object, and dispatches to all registered handlers.
//
// Falco may send a single JSON object per POST or (with some output plugins)
// a JSON array of alerts.  We handle both forms.
func (wr *WebhookReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if wr == nil {
		http.Error(w, "falco: receiver not initialised", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "falco: only POST accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2 MiB cap
	if err != nil {
		http.Error(w, fmt.Sprintf("falco: read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		http.Error(w, "falco: empty body", http.StatusBadRequest)
		return
	}

	var alerts []Alert
	if trimmed[0] == '[' {
		// Array of alerts
		if err := json.Unmarshal(body, &alerts); err != nil {
			http.Error(w, fmt.Sprintf("falco: unmarshal array: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Single alert object
		var a Alert
		if err := json.Unmarshal(body, &a); err != nil {
			http.Error(w, fmt.Sprintf("falco: unmarshal alert: %v", err), http.StatusBadRequest)
			return
		}
		alerts = append(alerts, a)
	}

	wr.mu.Lock()
	snapshot := make([]AlertHandler, len(wr.handlers))
	copy(snapshot, wr.handlers)
	wr.mu.Unlock()

	for _, a := range alerts {
		for _, fn := range snapshot {
			fn(a)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"accepted":true}`))
}

// ---------------------------------------------------------------------------
// Client – Falco webserver health / version
// ---------------------------------------------------------------------------

// New creates a Client that points at the Falco webserver (e.g.
// "http://falco:8765").
// Returns nil, nil when baseURL is empty (Falco integration is optional).
func New(baseURL string) (*Client, error) {
	if baseURL == "" {
		return nil, nil
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // internal cluster traffic
			},
		},
	}, nil
}

// Health calls GET /healthz.  Returns nil when Falco responds 200 OK.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("falco: build health request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("falco: health request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falco: health check returned %d", resp.StatusCode)
	}
	return nil
}

// versionResp is the JSON envelope returned by GET /version.
type versionResp struct {
	Version string `json:"version"`
}

// Version queries GET /version and returns the Falco engine version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	if c == nil {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		return "", fmt.Errorf("falco: build version request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("falco: version request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("falco: version returned %d", resp.StatusCode)
	}

	var v versionResp
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", fmt.Errorf("falco: decode version response: %w", err)
	}
	return v.Version, nil
}

// Close releases any idle HTTP connections held by the client.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.hc.CloseIdleConnections()
}
