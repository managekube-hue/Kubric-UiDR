//go:build ignore

// Package psa provides Professional Services Automation tooling.
// K-PSA-BILL-001 — Stripe Webhook Handler: subscription lifecycle, invoice events,
// tenant billing state management, and agent-seat quota enforcement.
package psa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SubscriptionTier represents a billing plan available on the platform.
type SubscriptionTier string

const (
	TierFree       SubscriptionTier = "free"
	TierStarter    SubscriptionTier = "starter"
	TierPro        SubscriptionTier = "pro"
	TierBusiness   SubscriptionTier = "business"
	TierEnterprise SubscriptionTier = "enterprise"
)

// agentSeatLimits maps each tier to its maximum licensed agent seats.
// -1 means unlimited (Enterprise).
var agentSeatLimits = map[SubscriptionTier]int{
	TierFree:       2,
	TierStarter:    10,
	TierPro:        50,
	TierBusiness:   200,
	TierEnterprise: -1,
}

// TierFromProductID maps Stripe product IDs to subscription tiers.
// Populate from your Stripe dashboard product IDs.
var TierFromProductID = map[string]SubscriptionTier{
	"prod_free":       TierFree,
	"prod_starter":    TierStarter,
	"prod_pro":        TierPro,
	"prod_business":   TierBusiness,
	"prod_enterprise": TierEnterprise,
}

// TenantBillingState holds the canonical billing record for a tenant.
type TenantBillingState struct {
	TenantID             string           `json:"tenant_id"`
	StripeCustomerID     string           `json:"stripe_customer_id"`
	StripeSubscriptionID string           `json:"stripe_subscription_id"`
	Tier                 SubscriptionTier `json:"tier"`
	AgentSeatsUsed       int              `json:"agent_seats_used"`
	Status               string           `json:"status"` // active, past_due, canceled, trialing
	CurrentPeriodStart   time.Time        `json:"current_period_start"`
	CurrentPeriodEnd     time.Time        `json:"current_period_end"`
	CanceledAt           *time.Time       `json:"canceled_at,omitempty"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// BillingStore persists and retrieves TenantBillingState records.
type BillingStore interface {
	Upsert(ctx context.Context, state TenantBillingState) error
	GetByCustomer(ctx context.Context, stripeCustomerID string) (*TenantBillingState, error)
	GetBySubscription(ctx context.Context, stripeSubscriptionID string) (*TenantBillingState, error)
	GetByTenant(ctx context.Context, tenantID string) (*TenantBillingState, error)
	IncrementSeats(ctx context.Context, tenantID string, delta int) error
}

// StripeWebhookHandler validates Stripe webhook signatures and dispatches events
// to the appropriate billing state handlers.
type StripeWebhookHandler struct {
	store          BillingStore
	webhookSecret  string
	onPaymentFail  func(ctx context.Context, state *TenantBillingState) // optional alert hook
	toleranceWindow time.Duration
}

// NewStripeWebhookHandler constructs the handler. webhookSecret is the Stripe
// dashboard signing secret (whsec_…). toleranceWindow is the maximum age of an
// event that will be accepted (Stripe recommends 300 s; pass 0 for default).
func NewStripeWebhookHandler(
	store BillingStore,
	webhookSecret string,
	onPaymentFail func(ctx context.Context, state *TenantBillingState),
) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		store:          store,
		webhookSecret:  webhookSecret,
		onPaymentFail:  onPaymentFail,
		toleranceWindow: 300 * time.Second,
	}
}

// ServeHTTP implements http.Handler and acts as the POST /webhooks/stripe endpoint.
func (h *StripeWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	if err := validateStripeSignature(sig, body, h.webhookSecret, h.toleranceWindow); err != nil {
		http.Error(w, fmt.Sprintf("bad signature: %v", err), http.StatusUnauthorized)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.dispatch(ctx, event); err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *StripeWebhookHandler) dispatch(ctx context.Context, event stripeEvent) error {
	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		return h.handleSubscriptionUpsert(ctx, event)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(ctx, event)
	case "invoice.payment_succeeded":
		return h.handleInvoiceSucceeded(ctx, event)
	case "invoice.payment_failed":
		return h.handleInvoiceFailed(ctx, event)
	default:
		// Unhandled event types are silently acknowledged per Stripe recommendation.
		return nil
	}
}

// handleSubscriptionUpsert processes subscription.created and .updated events.
func (h *StripeWebhookHandler) handleSubscriptionUpsert(ctx context.Context, event stripeEvent) error {
	var sub stripeSubscription
	if err := json.Unmarshal(event.Data.Object, &sub); err != nil {
		return fmt.Errorf("decode subscription: %w", err)
	}

	tier := TierFree
	if len(sub.Items.Data) > 0 {
		productID := sub.Items.Data[0].Price.Product
		if t, ok := TierFromProductID[productID]; ok {
			tier = t
		}
	}

	existing, _ := h.store.GetBySubscription(ctx, sub.ID)
	tenantID := ""
	if existing != nil {
		tenantID = existing.TenantID
	}
	if tenantID == "" {
		tenantID = sub.Metadata["tenant_id"]
	}

	state := TenantBillingState{
		TenantID:             tenantID,
		StripeCustomerID:     sub.Customer,
		StripeSubscriptionID: sub.ID,
		Tier:                 tier,
		Status:               sub.Status,
		CurrentPeriodStart:   time.Unix(sub.CurrentPeriodStart, 0).UTC(),
		CurrentPeriodEnd:     time.Unix(sub.CurrentPeriodEnd, 0).UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	if existing != nil {
		state.AgentSeatsUsed = existing.AgentSeatsUsed
	}

	return h.store.Upsert(ctx, state)
}

// handleSubscriptionDeleted marks the subscription as canceled.
func (h *StripeWebhookHandler) handleSubscriptionDeleted(ctx context.Context, event stripeEvent) error {
	var sub stripeSubscription
	if err := json.Unmarshal(event.Data.Object, &sub); err != nil {
		return fmt.Errorf("decode subscription: %w", err)
	}

	existing, err := h.store.GetBySubscription(ctx, sub.ID)
	if err != nil || existing == nil {
		return nil // nothing to cancel; idempotent
	}

	now := time.Now().UTC()
	existing.Status = "canceled"
	existing.CanceledAt = &now
	existing.UpdatedAt = now

	return h.store.Upsert(ctx, *existing)
}

// handleInvoiceSucceeded re-activates a tenant whose subscription was past_due.
func (h *StripeWebhookHandler) handleInvoiceSucceeded(ctx context.Context, event stripeEvent) error {
	var inv stripeInvoice
	if err := json.Unmarshal(event.Data.Object, &inv); err != nil {
		return fmt.Errorf("decode invoice: %w", err)
	}
	if inv.Subscription == "" {
		return nil
	}

	existing, err := h.store.GetBySubscription(ctx, inv.Subscription)
	if err != nil || existing == nil {
		return nil
	}

	if existing.Status == "past_due" {
		existing.Status = "active"
		existing.UpdatedAt = time.Now().UTC()
		return h.store.Upsert(ctx, *existing)
	}
	return nil
}

// handleInvoiceFailed marks a tenant as past_due and fires the optional alert hook.
func (h *StripeWebhookHandler) handleInvoiceFailed(ctx context.Context, event stripeEvent) error {
	var inv stripeInvoice
	if err := json.Unmarshal(event.Data.Object, &inv); err != nil {
		return fmt.Errorf("decode invoice: %w", err)
	}
	if inv.Subscription == "" {
		return nil
	}

	existing, err := h.store.GetBySubscription(ctx, inv.Subscription)
	if err != nil || existing == nil {
		return nil
	}

	existing.Status = "past_due"
	existing.UpdatedAt = time.Now().UTC()
	if upsertErr := h.store.Upsert(ctx, *existing); upsertErr != nil {
		return upsertErr
	}

	if h.onPaymentFail != nil {
		h.onPaymentFail(ctx, existing)
	}
	return nil
}

// GetAgentQuota returns (limit, used, ok). ok is false if the tenant would
// exceed their tier's seat limit by adding delta more agents.
func (h *StripeWebhookHandler) GetAgentQuota(
	ctx context.Context, tenantID string, delta int,
) (limit, used int, ok bool, err error) {
	state, err := h.store.GetByTenant(ctx, tenantID)
	if err != nil {
		return 0, 0, false, fmt.Errorf("get billing state for %s: %w", tenantID, err)
	}
	if state == nil {
		return agentSeatLimits[TierFree], 0, true, nil
	}

	limit = agentSeatLimits[state.Tier]
	used = state.AgentSeatsUsed
	if limit < 0 { // unlimited tier
		return limit, used, true, nil
	}
	ok = (used + delta) <= limit
	return limit, used, ok, nil
}

// ---- validateStripeSignature -----------------------------------------------
// Implements the Stripe webhook signature verification described at
// https://stripe.com/docs/webhooks/signatures using HMAC-SHA256.

func validateStripeSignature(header string, payload []byte, secret string, tolerance time.Duration) error {
	if header == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	// Parse t= and v1= components from the header.
	var timestamp int64
	var signatures []string

	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp in signature header")
			}
			timestamp = ts
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == 0 {
		return fmt.Errorf("no timestamp found in Stripe-Signature header")
	}
	if len(signatures) == 0 {
		return fmt.Errorf("no v1 signature found in Stripe-Signature header")
	}

	// Reject stale events.
	if tolerance > 0 {
		eventTime := time.Unix(timestamp, 0)
		if time.Since(eventTime) > tolerance {
			return fmt.Errorf("webhook timestamp too old: %v", eventTime)
		}
	}

	// Compute expected signature: HMAC-SHA256(<timestamp>.<payload>, secret).
	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return fmt.Errorf("no matching v1 signature found")
}

// ---- minimal Stripe event structs (avoids pulling the SDK into referenced code) ----

type stripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripeSubscription struct {
	ID                 string            `json:"id"`
	Customer           string            `json:"customer"`
	Status             string            `json:"status"`
	CurrentPeriodStart int64             `json:"current_period_start"`
	CurrentPeriodEnd   int64             `json:"current_period_end"`
	Metadata           map[string]string `json:"metadata"`
	Items              stripeSubItems    `json:"items"`
}

type stripeSubItems struct {
	Data []stripeSubItem `json:"data"`
}

type stripeSubItem struct {
	Price stripePrice `json:"price"`
}

type stripePrice struct {
	Product string `json:"product"`
}

type stripeInvoice struct {
	ID           string `json:"id"`
	Customer     string `json:"customer"`
	Subscription string `json:"subscription"`
	Paid         bool   `json:"paid"`
	AmountDue    int64  `json:"amount_due"`
}
