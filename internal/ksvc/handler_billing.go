package ksvc

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/managekube-hue/Kubric-UiDR/services/k-svc/billing"
)

type billingHandler struct {
	store *TenantStore
	cfg   Config
}

func newBillingHandler(store *TenantStore, cfg Config) *billingHandler {
	return &billingHandler{store: store, cfg: cfg}
}

// createSubscription handles POST /tenants/{tenantID}/subscription.
// Body: { "price_id": "price_xxxx" }
func (h *billingHandler) createSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var body struct {
		PriceID string `json:"price_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.PriceID == "" {
		writeError(w, http.StatusUnprocessableEntity, "price_id is required")
		return
	}

	t, err := h.store.Get(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if t.StripeCustomerID == "" {
		writeError(w, http.StatusConflict, "tenant has no Stripe customer; create tenant first")
		return
	}

	sub, err := billing.CreateSubscription(r.Context(), t.StripeCustomerID, body.PriceID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "stripe error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"subscription_id": sub.ID,
		"status":          string(sub.Status),
	})
}

// getBillingPortal handles GET /tenants/{tenantID}/billing/portal.
func (h *billingHandler) getBillingPortal(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")

	t, err := h.store.Get(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if t.StripeCustomerID == "" {
		writeError(w, http.StatusConflict, "tenant has no Stripe customer")
		return
	}

	returnURL := h.cfg.BillingReturnURL
	if returnURL == "" {
		returnURL = "https://app.kubric.io/billing"
	}

	url, err := billing.GetBillingPortalURL(r.Context(), t.StripeCustomerID, returnURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "stripe error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}
