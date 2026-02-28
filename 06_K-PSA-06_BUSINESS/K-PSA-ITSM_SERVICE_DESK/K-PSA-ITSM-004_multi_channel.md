# K-PSA-ITSM-004 -- Multi-Channel Ticket Intake

**Role:** Unified ticket intake across email, portal, Microsoft Teams, and API. All channels converge into a single ITSM pipeline with tenant isolation.

---

## 1. Architecture

```
┌──────────┐  IMAP/Graph  ┌─────────────────┐
│  Email   │─────────────►│                 │
└──────────┘              │                 │
                          │  Go Service     │   NATS
┌──────────┐  Webhook     │  (PSA/ITSM)     │──────────►  kubric.itsm.ticket.{tenant_id}
│  Teams   │─────────────►│                 │
│  Bot     │              │  Normalize      │
└──────────┘              │  Classify       │
                          │  Route          │
┌──────────┐  REST API    │  Deduplicate    │   PostgreSQL
│  Portal  │─────────────►│                 │──────────►  tickets table
└──────────┘              │                 │
                          │                 │
┌──────────┐  REST API    │                 │
│  API     │─────────────►│                 │
│  Direct  │              └─────────────────┘
└──────────┘
```

---

## 2. Ticket Data Model

```go
// internal/psa/itsm/ticket.go
package itsm

import (
	"time"
)

// Ticket represents a service desk ticket from any channel.
type Ticket struct {
	ID           string       `json:"id"`
	TenantID     string       `json:"tenant_id"`
	ExternalID   string       `json:"external_id,omitempty"` // source-specific ID
	Channel      string       `json:"channel"`       // email, teams, portal, api
	Status       string       `json:"status"`        // new, open, pending, resolved, closed
	Priority     int          `json:"priority"`      // 1=critical, 2=high, 3=medium, 4=low
	Category     string       `json:"category"`      // incident, service_request, change, problem
	Subject      string       `json:"subject"`
	Description  string       `json:"description"`
	Requester    Contact      `json:"requester"`
	Assignee     *Contact     `json:"assignee,omitempty"`
	Tags         []string     `json:"tags"`
	Attachments  []Attachment `json:"attachments,omitempty"`
	SLADeadline  *time.Time   `json:"sla_deadline,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	ResolvedAt   *time.Time   `json:"resolved_at,omitempty"`
	FirstResponse *time.Time  `json:"first_response_at,omitempty"`
}

type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	StorageKey  string `json:"storage_key"` // MinIO key
}
```

---

## 3. Email Intake (Microsoft Graph)

```go
// internal/psa/itsm/email_intake.go
package itsm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const graphAPIBase = "https://graph.microsoft.com/v1.0"

// EmailIntake polls a shared mailbox via Microsoft Graph API.
type EmailIntake struct {
	httpClient  *http.Client
	accessToken string
	mailbox     string // support@customer.com
	processor   *TicketProcessor
}

func NewEmailIntake(token, mailbox string, proc *TicketProcessor) *EmailIntake {
	return &EmailIntake{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		accessToken: token,
		mailbox:     mailbox,
		processor:   proc,
	}
}

type GraphMessage struct {
	ID                  string        `json:"id"`
	Subject             string        `json:"subject"`
	BodyPreview         string        `json:"bodyPreview"`
	Body                MessageBody   `json:"body"`
	From                EmailAddress  `json:"from"`
	ReceivedDateTime    string        `json:"receivedDateTime"`
	HasAttachments      bool          `json:"hasAttachments"`
	ConversationID      string        `json:"conversationId"`
	IsRead              bool          `json:"isRead"`
}

type MessageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type EmailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// Poll checks for new unread messages and creates tickets.
func (ei *EmailIntake) Poll(ctx context.Context, tenantID string) error {
	url := fmt.Sprintf("%s/users/%s/mailFolders/inbox/messages?$filter=isRead eq false&$top=50&$orderby=receivedDateTime desc",
		graphAPIBase, ei.mailbox)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ei.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := ei.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Value []GraphMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	for _, msg := range result.Value {
		ticket := &Ticket{
			TenantID:    tenantID,
			ExternalID:  msg.ID,
			Channel:     "email",
			Status:      "new",
			Priority:    classifyPriority(msg.Subject, msg.BodyPreview),
			Category:    classifyCategory(msg.Subject, msg.BodyPreview),
			Subject:     msg.Subject,
			Description: stripHTML(msg.Body.Content),
			Requester: Contact{
				Name:  msg.From.EmailAddress.Name,
				Email: msg.From.EmailAddress.Address,
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := ei.processor.Process(ctx, ticket); err != nil {
			continue
		}

		// Mark as read
		ei.markAsRead(ctx, msg.ID)
	}

	return nil
}

func (ei *EmailIntake) markAsRead(ctx context.Context, messageID string) {
	url := fmt.Sprintf("%s/users/%s/messages/%s", graphAPIBase, ei.mailbox, messageID)
	body := strings.NewReader(`{"isRead": true}`)
	req, _ := http.NewRequestWithContext(ctx, "PATCH", url, body)
	req.Header.Set("Authorization", "Bearer "+ei.accessToken)
	req.Header.Set("Content-Type", "application/json")
	ei.httpClient.Do(req)
}

func stripHTML(html string) string {
	// Simple HTML stripping — production would use a proper parser
	result := html
	for strings.Contains(result, "<") {
		start := strings.Index(result, "<")
		end := strings.Index(result[start:], ">")
		if end < 0 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}
```

---

## 4. Teams Bot Webhook

```go
// internal/psa/itsm/teams_intake.go
package itsm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TeamsWebhook handles incoming messages from Microsoft Teams bot.
type TeamsWebhook struct {
	processor *TicketProcessor
}

type TeamsActivity struct {
	Type         string         `json:"type"`
	Text         string         `json:"text"`
	From         TeamsFrom      `json:"from"`
	ChannelData  TeamsChannel   `json:"channelData"`
	Conversation TeamsConv      `json:"conversation"`
}

type TeamsFrom struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TeamsChannel struct {
	Tenant struct {
		ID string `json:"id"`
	} `json:"tenant"`
}

type TeamsConv struct {
	ID string `json:"id"`
}

// HandleWebhook processes Teams bot webhook payloads.
func (tw *TeamsWebhook) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var activity TeamsActivity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if activity.Type != "message" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Map Teams tenant to Kubric tenant
	tenantID := mapTeamsTenant(activity.ChannelData.Tenant.ID)

	ticket := &Ticket{
		TenantID:    tenantID,
		ExternalID:  activity.Conversation.ID,
		Channel:     "teams",
		Status:      "new",
		Priority:    classifyPriority(activity.Text, ""),
		Category:    classifyCategory(activity.Text, ""),
		Subject:     truncate(activity.Text, 100),
		Description: activity.Text,
		Requester: Contact{
			Name: activity.From.Name,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	ctx := r.Context()
	if err := tw.processor.Process(ctx, ticket); err != nil {
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	// Send acknowledgment back to Teams
	resp := map[string]string{
		"type": "message",
		"text": fmt.Sprintf("Ticket created: %s\nPriority: P%d\nWe'll get back to you shortly.",
			ticket.Subject, ticket.Priority),
	}
	json.NewEncoder(w).Encode(resp)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func mapTeamsTenant(teamsID string) string {
	// Production: look up in customer_integrations table
	return teamsID
}
```

---

## 5. Ticket Processor (Normalize/Classify/Route)

```go
// internal/psa/itsm/processor.go
package itsm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
)

// TicketProcessor normalizes, deduplicates, and routes tickets.
type TicketProcessor struct {
	db *sql.DB
	nc *nats.Conn
}

func NewTicketProcessor(db *sql.DB, nc *nats.Conn) *TicketProcessor {
	return &TicketProcessor{db: db, nc: nc}
}

// Process handles a new ticket from any channel.
func (tp *TicketProcessor) Process(ctx context.Context, ticket *Ticket) error {
	// Generate ID
	ticket.ID = uuid.New().String()

	// Deduplicate by external ID + channel
	if ticket.ExternalID != "" {
		exists, err := tp.isDuplicate(ctx, ticket.TenantID, ticket.Channel, ticket.ExternalID)
		if err != nil {
			return err
		}
		if exists {
			return nil // Skip duplicate
		}
	}

	// Compute SLA deadline based on priority
	deadline := tp.computeSLA(ticket.Priority)
	ticket.SLADeadline = &deadline

	// Persist to PostgreSQL
	if err := tp.store(ctx, ticket); err != nil {
		return fmt.Errorf("store ticket: %w", err)
	}

	// Publish to NATS
	return tp.publish(ticket)
}

func (tp *TicketProcessor) isDuplicate(
	ctx context.Context,
	tenantID, channel, externalID string,
) (bool, error) {
	var exists bool
	err := tp.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tickets
			WHERE tenant_id = $1 AND channel = $2 AND external_id = $3
			  AND created_at > NOW() - INTERVAL '24 hours'
		)
	`, tenantID, channel, externalID).Scan(&exists)
	return exists, err
}

func (tp *TicketProcessor) computeSLA(priority int) time.Time {
	now := time.Now().UTC()
	switch priority {
	case 1:
		return now.Add(1 * time.Hour) // P1: 1 hour
	case 2:
		return now.Add(4 * time.Hour) // P2: 4 hours
	case 3:
		return now.Add(8 * time.Hour) // P3: 8 business hours
	default:
		return now.Add(24 * time.Hour) // P4: 24 hours
	}
}

func (tp *TicketProcessor) store(ctx context.Context, t *Ticket) error {
	tags, _ := json.Marshal(t.Tags)
	_, err := tp.db.ExecContext(ctx, `
		INSERT INTO tickets (id, tenant_id, external_id, channel, status, priority,
		                     category, subject, description, requester_name,
		                     requester_email, tags, sla_deadline, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, t.ID, t.TenantID, t.ExternalID, t.Channel, t.Status, t.Priority,
		t.Category, t.Subject, t.Description, t.Requester.Name,
		t.Requester.Email, tags, t.SLADeadline, t.CreatedAt, t.UpdatedAt)
	return err
}

func (tp *TicketProcessor) publish(t *Ticket) error {
	data, _ := json.Marshal(t)
	return tp.nc.Publish(
		fmt.Sprintf("kubric.itsm.ticket.%s", t.TenantID), data,
	)
}

// classifyPriority determines ticket priority from subject/body keywords.
func classifyPriority(subject, body string) int {
	text := strings.ToLower(subject + " " + body)

	criticalWords := []string{"down", "outage", "emergency", "critical", "p1",
		"ransomware", "breach", "compromised", "production down"}
	for _, word := range criticalWords {
		if strings.Contains(text, word) {
			return 1
		}
	}

	highWords := []string{"urgent", "high", "p2", "not working", "broken",
		"security alert", "data loss"}
	for _, word := range highWords {
		if strings.Contains(text, word) {
			return 2
		}
	}

	lowWords := []string{"question", "info", "feature request", "when can",
		"low priority", "p4", "nice to have"}
	for _, word := range lowWords {
		if strings.Contains(text, word) {
			return 4
		}
	}

	return 3 // Default: medium
}

// classifyCategory determines ticket category from content.
func classifyCategory(subject, body string) string {
	text := strings.ToLower(subject + " " + body)

	incidentWords := []string{"down", "outage", "error", "broken", "not working",
		"crash", "alert", "incident", "failure"}
	for _, word := range incidentWords {
		if strings.Contains(text, word) {
			return "incident"
		}
	}

	changeWords := []string{"change", "update", "upgrade", "install", "deploy",
		"migration", "modify"}
	for _, word := range changeWords {
		if strings.Contains(text, word) {
			return "change"
		}
	}

	return "service_request"
}
```

---

## 6. PostgreSQL Schema

```sql
-- migrations/015_tickets.sql
CREATE TABLE IF NOT EXISTS tickets (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    external_id     TEXT,
    channel         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'new',
    priority        INT NOT NULL DEFAULT 3,
    category        TEXT NOT NULL DEFAULT 'service_request',
    subject         TEXT NOT NULL,
    description     TEXT,
    requester_name  TEXT,
    requester_email TEXT,
    assignee_id     UUID,
    tags            JSONB DEFAULT '[]',
    sla_deadline    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    first_response_at TIMESTAMPTZ
);

CREATE INDEX idx_tickets_tenant ON tickets(tenant_id, created_at DESC);
CREATE INDEX idx_tickets_status ON tickets(tenant_id, status);
CREATE INDEX idx_tickets_sla    ON tickets(sla_deadline) WHERE status NOT IN ('resolved', 'closed');
CREATE INDEX idx_tickets_dedup  ON tickets(tenant_id, channel, external_id);
```

---

## 7. REST API Endpoint

```go
// HTTP handler for portal/API ticket creation
func (h *ITSMHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject     string   `json:"subject"`
		Description string   `json:"description"`
		Priority    int      `json:"priority"`
		Category    string   `json:"category"`
		Tags        []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tenantID := r.Context().Value("tenant_id").(string)

	ticket := &Ticket{
		TenantID:    tenantID,
		Channel:     "portal",
		Status:      "new",
		Priority:    req.Priority,
		Category:    req.Category,
		Subject:     req.Subject,
		Description: req.Description,
		Tags:        req.Tags,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := h.processor.Process(r.Context(), ticket); err != nil {
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ticket)
}
```
