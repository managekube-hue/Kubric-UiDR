//go:build ignore

// Package psa provides Professional Services Automation tooling.
// K-PSA-ITSM-001 — Zammad Bridge: full CRUD client, incident-to-ticket conversion,
// and inbound Zammad webhook handler for bidirectional state synchronisation.
package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---- Domain types -----------------------------------------------------------

// ZammadTicket models a Zammad ticket object returned by the API.
type ZammadTicket struct {
	ID         int               `json:"id,omitempty"`
	Number     string            `json:"number,omitempty"`
	TitleText  string            `json:"title"`
	GroupID    int               `json:"group_id"`
	CustomerID int               `json:"customer_id,omitempty"`
	OwnerID    int               `json:"owner_id,omitempty"`
	StateID    int               `json:"state_id"`
	PriorityID int               `json:"priority_id"`
	Tags       []string          `json:"tags,omitempty"`
	Note       string            `json:"note,omitempty"`
	CreatedAt  time.Time         `json:"created_at,omitempty"`
	UpdatedAt  time.Time         `json:"updated_at,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// ZammadArticle models a message/note attached to a ticket.
type ZammadArticle struct {
	ID         int       `json:"id,omitempty"`
	TicketID   int       `json:"ticket_id"`
	Subject    string    `json:"subject,omitempty"`
	Body       string    `json:"body"`
	ContentType string   `json:"content_type"` // "text/plain" or "text/html"
	Type       string    `json:"type"`         // "note", "email", "phone"
	Internal   bool      `json:"internal"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// ZammadState constants — numeric IDs as configured in a default Zammad instance.
const (
	StateNew        = 1
	StateOpen       = 2
	StatePending    = 3
	StateClosedSucc = 4
	StateClosedFail = 5
)

// ZammadPriority constants.
const (
	PriorityLow    = 1
	PriorityNormal = 2
	PriorityHigh   = 3
)

// Incident is a normalised internal incident record that can be converted to a
// Zammad ticket. It is intentionally agnostic of the originating system
// (Wazuh alert, SIEM hit, customer report, etc.).
type Incident struct {
	ID          string
	TenantID    string
	Title       string
	Description string
	Severity    string // critical, high, medium, low, info
	Source      string // e.g. "wazuh", "user_report", "synthetic_monitor"
	AssetID     string
	Tags        []string
	OccurredAt  time.Time
	Properties  map[string]string
}

// ---- Zammad HTTP client -----------------------------------------------------

// ZammadClient wraps the Zammad REST API v1.
type ZammadClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewZammadClient constructs a client for the Zammad instance at baseURL.
// token is a Zammad API token.  baseURL should be e.g. "https://helpdesk.example.com".
func NewZammadClient(baseURL, token string) *ZammadClient {
	return &ZammadClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// CreateTicket creates a new ticket in Zammad and returns the created object.
func (z *ZammadClient) CreateTicket(ctx context.Context, ticket ZammadTicket) (*ZammadTicket, error) {
	var created ZammadTicket
	if err := z.do(ctx, http.MethodPost, "/api/v1/tickets", ticket, &created); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	return &created, nil
}

// GetTicket retrieves a ticket by its numeric ID.
func (z *ZammadClient) GetTicket(ctx context.Context, id int) (*ZammadTicket, error) {
	var ticket ZammadTicket
	if err := z.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", id), nil, &ticket); err != nil {
		return nil, fmt.Errorf("get ticket %d: %w", id, err)
	}
	return &ticket, nil
}

// UpdateTicket partially updates a ticket.  Only non-zero fields in patch are
// sent to the API.
func (z *ZammadClient) UpdateTicket(ctx context.Context, id int, patch ZammadTicket) (*ZammadTicket, error) {
	var updated ZammadTicket
	if err := z.do(ctx, http.MethodPut, fmt.Sprintf("/api/v1/tickets/%d", id), patch, &updated); err != nil {
		return nil, fmt.Errorf("update ticket %d: %w", id, err)
	}
	return &updated, nil
}

// AddArticle appends an article (note or reply) to an existing ticket.
func (z *ZammadClient) AddArticle(ctx context.Context, article ZammadArticle) (*ZammadArticle, error) {
	if article.ContentType == "" {
		article.ContentType = "text/plain"
	}
	if article.Type == "" {
		article.Type = "note"
	}
	var created ZammadArticle
	if err := z.do(ctx, http.MethodPost, "/api/v1/ticket_articles", article, &created); err != nil {
		return nil, fmt.Errorf("add article to ticket %d: %w", article.TicketID, err)
	}
	return &created, nil
}

// ListOpenTickets fetches all tickets in state Open (state_id=2) for the given
// group, paginated by perPage entries starting at page.
func (z *ZammadClient) ListOpenTickets(ctx context.Context, groupID, page, perPage int) ([]ZammadTicket, error) {
	path := fmt.Sprintf(
		"/api/v1/tickets/search?query=state_id:%d+group_id:%d&page=%d&per_page=%d",
		StateOpen, groupID, page, perPage,
	)
	var result struct {
		Assets struct {
			Ticket map[string]ZammadTicket `json:"Ticket"`
		} `json:"assets"`
	}
	if err := z.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("list open tickets: %w", err)
	}
	tickets := make([]ZammadTicket, 0, len(result.Assets.Ticket))
	for _, t := range result.Assets.Ticket {
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// SearchTickets performs a full-text search across all tickets and returns
// at most limit results.
func (z *ZammadClient) SearchTickets(ctx context.Context, query string, limit int) ([]ZammadTicket, error) {
	encoded := url.QueryEscape(query)
	path := fmt.Sprintf("/api/v1/tickets/search?query=%s&limit=%d", encoded, limit)
	var result struct {
		Assets struct {
			Ticket map[string]ZammadTicket `json:"Ticket"`
		} `json:"assets"`
	}
	if err := z.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("search tickets %q: %w", query, err)
	}
	tickets := make([]ZammadTicket, 0, len(result.Assets.Ticket))
	for _, t := range result.Assets.Ticket {
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// do is the shared HTTP helper.  It serialises body to JSON, attaches the auth
// header, sends the request, and deserialises the response into out.
func (z *ZammadClient) do(ctx context.Context, method, path string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, z.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token token="+z.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB cap
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("zammad %s %s → %d: %s", method, path, resp.StatusCode, respBody)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// ---- IncidentToTicket conversion --------------------------------------------

// incidentSeverityToState maps incident severity strings to Zammad state IDs.
var incidentSeverityToState = map[string]int{
	"critical": StateNew,
	"high":     StateNew,
	"medium":   StateOpen,
	"low":      StateOpen,
	"info":     StateOpen,
}

// incidentSeverityToPriority maps incident severity strings to Zammad priority IDs.
var incidentSeverityToPriority = map[string]int{
	"critical": PriorityHigh,
	"high":     PriorityHigh,
	"medium":   PriorityNormal,
	"low":      PriorityLow,
	"info":     PriorityLow,
}

// IncidentToTicket converts a platform Incident into a ZammadTicket ready for
// creation.  groupID is the Zammad group that should own the ticket.
func IncidentToTicket(inc Incident, groupID int) ZammadTicket {
	sev := strings.ToLower(inc.Severity)
	stateID, ok := incidentSeverityToState[sev]
	if !ok {
		stateID = StateOpen
	}
	priorityID, ok := incidentSeverityToPriority[sev]
	if !ok {
		priorityID = PriorityNormal
	}

	title := inc.Title
	if title == "" {
		title = fmt.Sprintf("[%s] Incident %s", strings.ToUpper(inc.Severity), inc.ID)
	}

	custom := map[string]string{
		"kubric_incident_id": inc.ID,
		"kubric_tenant_id":   inc.TenantID,
		"kubric_asset_id":    inc.AssetID,
		"kubric_source":      inc.Source,
	}
	for k, v := range inc.Properties {
		custom["kubric_"+k] = v
	}

	tags := append([]string{"kubric", inc.Source}, inc.Tags...)

	body := fmt.Sprintf(
		"Severity: %s\nSource: %s\nAsset: %s\nOccurred: %s\n\n%s",
		inc.Severity,
		inc.Source,
		inc.AssetID,
		inc.OccurredAt.UTC().Format(time.RFC3339),
		inc.Description,
	)
	_ = body // used in first article; ticket note field holds summary

	return ZammadTicket{
		TitleText:    title,
		GroupID:      groupID,
		StateID:      stateID,
		PriorityID:   priorityID,
		Tags:         tags,
		Note:         body,
		CustomFields: custom,
	}
}

// CreateIncidentTicket is a convenience method that converts an Incident to a
// ZammadTicket, creates it, and attaches the full description as the first article.
func (z *ZammadClient) CreateIncidentTicket(
	ctx context.Context, inc Incident, groupID int,
) (*ZammadTicket, error) {
	tkt := IncidentToTicket(inc, groupID)
	created, err := z.CreateTicket(ctx, tkt)
	if err != nil {
		return nil, err
	}

	// Attach the full description as the initial internal note.
	article := ZammadArticle{
		TicketID:    created.ID,
		Subject:     fmt.Sprintf("Initial report — %s", inc.ID),
		Body:        tkt.Note,
		ContentType: "text/plain",
		Type:        "note",
		Internal:    true,
	}
	if _, err := z.AddArticle(ctx, article); err != nil {
		// Non-fatal: the ticket exists; just log the failure in production.
		_ = err
	}
	return created, nil
}

// ---- Inbound Zammad webhook handler -----------------------------------------

// ZammadWebhookPayload is the envelope Zammad sends for ticket.updated events.
type ZammadWebhookPayload struct {
	Event  string       `json:"event"`
	Ticket ZammadTicket `json:"ticket"`
}

// TicketUpdateHandler is called when a ticket is updated via Zammad's webhook.
// Implementations should synchronise the update back into the platform's state.
type TicketUpdateHandler interface {
	OnTicketUpdated(ctx context.Context, payload ZammadWebhookPayload) error
}

// ZammadWebhookServer receives POST events from Zammad and dispatches them to
// the registered handler.
type ZammadWebhookServer struct {
	handler    TicketUpdateHandler
	sharedToken string // pre-shared token in Zammad trigger HTTP header
}

// NewZammadWebhookServer constructs the server.  sharedToken is validated
// against the X-Zammad-Token header for basic origin checking.
func NewZammadWebhookServer(handler TicketUpdateHandler, sharedToken string) *ZammadWebhookServer {
	return &ZammadWebhookServer{handler: handler, sharedToken: sharedToken}
}

// ServeHTTP implements http.Handler and accepts POST /webhooks/zammad.
func (s *ZammadWebhookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate shared token if configured.
	if s.sharedToken != "" {
		tok := r.Header.Get("X-Zammad-Token")
		if tok != s.sharedToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	var payload ZammadWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.handler.OnTicketUpdated(r.Context(), payload); err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ---- utility ----------------------------------------------------------------

// TicketIDFromString converts a string ticket ID to int for API calls.
func TicketIDFromString(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid ticket id %q: %w", s, err)
	}
	return id, nil
}
