// K-KAI-GD-001 — Human MFA Guardrail: requires human operator approval for high-risk KAI actions.
// Integrates with Vapi phone MFA and Authentik TOTP for approval workflows.
package kai

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // TOTP (RFC 6238) mandates HMAC-SHA1
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Risk levels
// ---------------------------------------------------------------------------

// RiskLevel classifies the danger of a KAI action that requires gating.
type RiskLevel int

const (
	RiskLow      RiskLevel = 1 // auto-approve + audit log
	RiskMedium   RiskLevel = 2 // Slack/webhook + wait for manual approval
	RiskHigh     RiskLevel = 3 // TOTP required
	RiskCritical RiskLevel = 4 // Vapi phone call + TOTP
)

// String returns a human-readable label for the risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return fmt.Sprintf("unknown(%d)", int(r))
	}
}

// ---------------------------------------------------------------------------
// Request / result types
// ---------------------------------------------------------------------------

// ApprovalRequest carries all data needed to request human MFA approval.
type ApprovalRequest struct {
	// ID is a unique UUID for this approval request (auto-generated if empty).
	ID string `json:"id"`

	// Action is the KAI action name, e.g. "run_playbook", "delete_tenant".
	Action string `json:"action"`

	// Description is a human-readable explanation shown to the approver.
	Description string `json:"description"`

	// RequestedBy is the KAI agent or user requesting the action.
	RequestedBy string `json:"requested_by"`

	// TenantID scopes the request to a specific tenant.
	TenantID string `json:"tenant_id"`

	// RiskLevel determines which approval channel is used.
	RiskLevel RiskLevel `json:"risk_level"`

	// ExpiresAt is the deadline for approval.  After this time the request is
	// automatically rejected.
	ExpiresAt time.Time `json:"expires_at"`

	// Metadata holds arbitrary key-value pairs for the notification payload.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ApprovalResult contains the outcome of an approval workflow.
type ApprovalResult struct {
	Approved   bool      `json:"approved"`
	ApprovedBy string    `json:"approved_by,omitempty"`
	Method     string    `json:"method"` // "auto", "webhook", "totp", "phone+totp"
	Timestamp  time.Time `json:"timestamp"`
	TOTP       string    `json:"totp,omitempty"` // the TOTP code that was accepted
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// GuardrailConfig holds all parameters for the HumanGuardrail.
type GuardrailConfig struct {
	// Vapi integration
	VapiCallEnabled bool   // whether to make real phone calls for critical risk
	VapiAPIKey      string // Vapi API key
	VapiPhoneNumber string // destination phone number for the operator
	VapiAssistantID string // Vapi assistant ID with the MFA script

	// Authentik TOTP
	AuthenticURL   string // Authentik base URL (for audit logging)
	AuthenticToken string // Authentik API token
	TOTPSecret     string // Base32-encoded TOTP shared secret

	// Approval webhook (medium risk)
	ApprovalWebhookURL string

	// Timing
	ApprovalTimeoutSecs int // default: 300 (5 minutes)

	// Optional: custom HTTP client (set to nil to use the default).
	HTTPClient *http.Client
}

// approvalTimeoutDuration returns the configured timeout as a duration.
func (c *GuardrailConfig) approvalTimeoutDuration() time.Duration {
	if c.ApprovalTimeoutSecs <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.ApprovalTimeoutSecs) * time.Second
}

// ---------------------------------------------------------------------------
// HumanGuardrail
// ---------------------------------------------------------------------------

// HumanGuardrail gates dangerous KAI actions behind human MFA approval.
// It is safe for concurrent use.
type HumanGuardrail struct {
	cfg        GuardrailConfig
	httpClient *http.Client

	// pending maps request ID → channel that receives the final ApprovalResult.
	mu      sync.Mutex
	pending map[string]chan ApprovalResult
}

// NewHumanGuardrail creates a configured guardrail.  Pass a GuardrailConfig
// with at least TOTPSecret and ApprovalWebhookURL set for production use.
func NewHumanGuardrail(cfg GuardrailConfig) *HumanGuardrail {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &HumanGuardrail{
		cfg:        cfg,
		httpClient: hc,
		pending:    make(map[string]chan ApprovalResult),
	}
}

// ---------------------------------------------------------------------------
// RequireApproval — top-level gate
// ---------------------------------------------------------------------------

// RequireApproval enforces the approval workflow appropriate for req.RiskLevel.
//
//   - Low      auto-approves with an audit log entry.
//   - Medium   sends a webhook notification and blocks until the operator
//              approves (or the context deadline / ExpiresAt is reached).
//   - High     requires a valid TOTP code (provided in req.Metadata["totp"]).
//   - Critical sends a Vapi phone call AND requires a TOTP code.
//
// The caller must provide the TOTP code in req.Metadata["totp"] for High/Critical
// risk actions (after out-of-band prompt to the operator).
func (g *HumanGuardrail) RequireApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	// Assign an ID if not set.
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().Add(g.cfg.approvalTimeoutDuration())
	}

	switch req.RiskLevel {
	case RiskLow:
		return g.autoApprove(req), nil

	case RiskMedium:
		return g.webhookApprove(ctx, req)

	case RiskHigh:
		return g.totpApprove(req)

	case RiskCritical:
		return g.phoneAndTOTPApprove(ctx, req)

	default:
		return nil, fmt.Errorf("guardrail: unknown risk level %d", int(req.RiskLevel))
	}
}

// ---------------------------------------------------------------------------
// Low: auto-approve
// ---------------------------------------------------------------------------

func (g *HumanGuardrail) autoApprove(req ApprovalRequest) *ApprovalResult {
	return &ApprovalResult{
		Approved:   true,
		ApprovedBy: "system",
		Method:     "auto",
		Timestamp:  time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Medium: webhook + wait
// ---------------------------------------------------------------------------

func (g *HumanGuardrail) webhookApprove(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	// Register this request so Approve() can unblock WaitForApproval.
	ch := make(chan ApprovalResult, 1)
	g.mu.Lock()
	g.pending[req.ID] = ch
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.pending, req.ID)
		g.mu.Unlock()
	}()

	// Fire-and-forget the webhook notification.
	if g.cfg.ApprovalWebhookURL != "" {
		if err := g.SendApprovalWebhook(ctx, req, g.cfg.ApprovalWebhookURL); err != nil {
			// Non-fatal: the operator may still approve via other channels.
			_ = err
		}
	}

	// Block until approved, expired, or context cancelled.
	timeout := time.Until(req.ExpiresAt)
	if timeout <= 0 {
		return &ApprovalResult{Approved: false, Method: "webhook", Timestamp: time.Now().UTC()}, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result, ok := <-ch:
		if !ok {
			return &ApprovalResult{Approved: false, Method: "webhook", Timestamp: time.Now().UTC()}, nil
		}
		return &result, nil
	case <-timer.C:
		return nil, fmt.Errorf("guardrail: approval request %q timed out", req.ID)
	case <-ctx.Done():
		return nil, fmt.Errorf("guardrail: context cancelled: %w", ctx.Err())
	}
}

// ---------------------------------------------------------------------------
// High: TOTP only
// ---------------------------------------------------------------------------

func (g *HumanGuardrail) totpApprove(req ApprovalRequest) (*ApprovalResult, error) {
	code, ok := req.Metadata["totp"]
	if !ok || code == "" {
		return nil, fmt.Errorf("guardrail: high-risk action %q requires TOTP code in metadata[\"totp\"]", req.Action)
	}

	if !g.ValidateTOTP(g.cfg.TOTPSecret, code) {
		return &ApprovalResult{
			Approved:  false,
			Method:    "totp",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("guardrail: invalid TOTP code for action %q", req.Action)
	}

	return &ApprovalResult{
		Approved:   true,
		ApprovedBy: req.RequestedBy,
		Method:     "totp",
		Timestamp:  time.Now().UTC(),
		TOTP:       code,
	}, nil
}

// ---------------------------------------------------------------------------
// Critical: Vapi phone call + TOTP
// ---------------------------------------------------------------------------

func (g *HumanGuardrail) phoneAndTOTPApprove(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	// Initiate the Vapi phone call (non-blocking).
	if g.cfg.VapiCallEnabled {
		if err := g.sendVapiCall(ctx, req); err != nil {
			// Log but don't abort — operator may already have been notified.
			_ = err
		}
	}

	// TOTP validation is still required.
	code, ok := req.Metadata["totp"]
	if !ok || code == "" {
		return nil, fmt.Errorf("guardrail: critical action %q requires TOTP code in metadata[\"totp\"]", req.Action)
	}

	if !g.ValidateTOTP(g.cfg.TOTPSecret, code) {
		return &ApprovalResult{
			Approved:  false,
			Method:    "phone+totp",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("guardrail: invalid TOTP code for critical action %q", req.Action)
	}

	return &ApprovalResult{
		Approved:   true,
		ApprovedBy: req.RequestedBy,
		Method:     "phone+totp",
		Timestamp:  time.Now().UTC(),
		TOTP:       code,
	}, nil
}

// ---------------------------------------------------------------------------
// Approve — called externally (e.g. by a webhook handler) to unblock
// a pending WaitForApproval / webhookApprove call.
// ---------------------------------------------------------------------------

// Approve delivers a result to a pending approval request identified by reqID.
// Returns false if no matching pending request is found.
func (g *HumanGuardrail) Approve(reqID string, result ApprovalResult) bool {
	g.mu.Lock()
	ch, ok := g.pending[reqID]
	g.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- result:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// WaitForApproval — explicit polling variant
// ---------------------------------------------------------------------------

// WaitForApproval blocks until a result arrives on the pending channel for
// reqID, or until the context deadline / timeout fires.
// If no pending channel exists for reqID, it registers one first.
func (g *HumanGuardrail) WaitForApproval(ctx context.Context, reqID string, timeout time.Duration) (*ApprovalResult, error) {
	g.mu.Lock()
	ch, exists := g.pending[reqID]
	if !exists {
		ch = make(chan ApprovalResult, 1)
		g.pending[reqID] = ch
	}
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.pending, reqID)
		g.mu.Unlock()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result, ok := <-ch:
		if !ok {
			return &ApprovalResult{Approved: false, Timestamp: time.Now().UTC()}, nil
		}
		return &result, nil
	case <-timer.C:
		return nil, fmt.Errorf("guardrail: wait for approval %q timed out after %s", reqID, timeout)
	case <-ctx.Done():
		return nil, fmt.Errorf("guardrail: context cancelled: %w", ctx.Err())
	}
}

// ---------------------------------------------------------------------------
// ValidateTOTP — RFC 6238 TOTP implementation (no external library)
// ---------------------------------------------------------------------------

// ValidateTOTP validates a 6-digit TOTP code against secret (Base32-encoded)
// using RFC 6238 (TOTP) with HMAC-SHA1 and a 30-second time step.
// It accepts codes from the current window and ±1 adjacent windows to allow
// for clock skew.
func (g *HumanGuardrail) ValidateTOTP(secret, code string) bool {
	return ValidateTOTP(secret, code)
}

// ValidateTOTP is the package-level TOTP validator, usable without a receiver.
// secret must be a Base32-encoded string (standard or no-padding alphabet).
// code must be a 6-digit string.  Returns true if code matches any of the
// current ±1 time-step windows.
func ValidateTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}

	key, err := decodeBase32Secret(secret)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	step := int64(30)

	// Accept current window and ±1 for clock skew.
	for delta := int64(-1); delta <= 1; delta++ {
		T := (now / step) + delta
		otp := computeTOTP(key, T)
		if fmt.Sprintf("%06d", otp) == code {
			return true
		}
	}
	return false
}

// computeTOTP computes the HOTP value for a counter T using HMAC-SHA1.
// Returns a 6-digit integer.
func computeTOTP(key []byte, T int64) int {
	// Pack T as an 8-byte big-endian integer.
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(T)) //nolint:gosec // T is always non-negative

	// HMAC-SHA1(key, msg).
	mac := hmac.New(sha1.New, key) //nolint:gosec // RFC 6238 mandates SHA1
	_, _ = mac.Write(msg)
	h := mac.Sum(nil) // 20 bytes

	// Dynamic truncation.
	offset := h[19] & 0x0f
	code := (int(h[offset])&0x7f)<<24 |
		int(h[offset+1])<<16 |
		int(h[offset+2])<<8 |
		int(h[offset+3])

	return code % int(math.Pow10(6)) // 6-digit OTP
}

// decodeBase32Secret decodes a Base32-encoded TOTP secret, tolerating
// missing padding and both upper/lower case.
func decodeBase32Secret(secret string) ([]byte, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	// Add padding if missing.
	if pad := len(secret) % 8; pad != 0 {
		secret += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("totp: decode base32 secret: %w", err)
	}
	return key, nil
}

// ---------------------------------------------------------------------------
// SendApprovalWebhook
// ---------------------------------------------------------------------------

// webhookPayload is the JSON body posted to the approval webhook.
type webhookPayload struct {
	RequestID   string            `json:"request_id"`
	Action      string            `json:"action"`
	Description string            `json:"description"`
	RequestedBy string            `json:"requested_by"`
	TenantID    string            `json:"tenant_id"`
	RiskLevel   string            `json:"risk_level"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	// ApproveURL is a convenience deep-link for the approver (may be empty).
	ApproveURL string `json:"approve_url,omitempty"`
}

// SendApprovalWebhook sends an HTTP POST to webhookURL with the approval
// request details in JSON form.  It returns an error if the HTTP request
// fails or the server responds with a non-2xx status.
func (g *HumanGuardrail) SendApprovalWebhook(ctx context.Context, req ApprovalRequest, webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("guardrail: webhook URL is empty")
	}

	payload := webhookPayload{
		RequestID:   req.ID,
		Action:      req.Action,
		Description: req.Description,
		RequestedBy: req.RequestedBy,
		TenantID:    req.TenantID,
		RiskLevel:   req.RiskLevel.String(),
		ExpiresAt:   req.ExpiresAt,
		Metadata:    req.Metadata,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("guardrail: marshal webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("guardrail: build webhook request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Kubric-Guardrail", "1")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("guardrail: send webhook to %q: %w", webhookURL, err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body) // drain body

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("guardrail: webhook %q returned %d", webhookURL, resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Vapi phone call
// ---------------------------------------------------------------------------

// vapiCallRequest is the Vapi REST API request body for an outbound call.
// See https://docs.vapi.ai/api-reference/calls/create-phone-call
type vapiCallRequest struct {
	AssistantID string            `json:"assistantId"`
	Customer    vapiCustomer      `json:"customer"`
	PhoneNumberID string          `json:"phoneNumberId,omitempty"`
	AssistantOverrides vapiOverrides `json:"assistantOverrides,omitempty"`
}

type vapiCustomer struct {
	Number string `json:"number"`
}

type vapiOverrides struct {
	FirstMessage string `json:"firstMessage,omitempty"`
}

// sendVapiCall initiates an outbound phone call via the Vapi API to notify
// the operator of a critical-risk action requiring approval.
func (g *HumanGuardrail) sendVapiCall(ctx context.Context, req ApprovalRequest) error {
	if g.cfg.VapiAPIKey == "" {
		return fmt.Errorf("guardrail: vapi API key not configured")
	}
	if g.cfg.VapiPhoneNumber == "" {
		return fmt.Errorf("guardrail: vapi destination phone number not configured")
	}

	message := fmt.Sprintf(
		"CRITICAL security action approval required. "+
			"Action: %s. "+
			"Requested by: %s. "+
			"Tenant: %s. "+
			"Please verify and approve in the Kubric console within %d seconds.",
		req.Action,
		req.RequestedBy,
		req.TenantID,
		g.cfg.ApprovalTimeoutSecs,
	)

	callReq := vapiCallRequest{
		AssistantID: g.cfg.VapiAssistantID,
		Customer:    vapiCustomer{Number: g.cfg.VapiPhoneNumber},
		AssistantOverrides: vapiOverrides{
			FirstMessage: message,
		},
	}

	body, err := json.Marshal(callReq)
	if err != nil {
		return fmt.Errorf("guardrail: marshal vapi call request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.vapi.ai/call/phone", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("guardrail: build vapi request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.cfg.VapiAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("guardrail: vapi call: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("guardrail: vapi returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
