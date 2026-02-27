package psa

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	stripe "github.com/stripe/stripe-go/v76"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripePayments handles Stripe checkout sessions, payment intents, and webhooks
// for the Kubric MSP billing flow.
type StripePayments struct {
	webhookSecret string
	pgPool        *pgxpool.Pool
}

// NewStripePayments creates a StripePayments handler from environment variables.
// Sets stripe.Key globally. Required env: STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET, DATABASE_URL.
func NewStripePayments() (*StripePayments, error) {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY not set")
	}
	stripe.Key = apiKey

	whs := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if whs == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pgPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	return &StripePayments{webhookSecret: whs, pgPool: pgPool}, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for a subscription.
// Returns the Session URL the customer should be redirected to.
func (s *StripePayments) CreateCheckoutSession(
	ctx context.Context,
	tenantID, email, priceID string,
	quantity int64,
) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Params:        stripe.Params{Context: ctx},
		CustomerEmail: stripe.String(email),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(quantity),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String("https://app.kubric.io/billing/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://app.kubric.io/billing/cancel"),
	}
	params.AddMetadata("tenant_id", tenantID)

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe checkout session: %w", err)
	}
	return sess.URL, nil
}

// CreatePaymentIntent creates a one-off Stripe PaymentIntent.
// amountCents is the charge in the smallest currency unit (e.g. cents for USD).
// Returns the PaymentIntent client_secret for frontend confirmation.
func (s *StripePayments) CreatePaymentIntent(
	ctx context.Context,
	amountCents int64,
	tenantID, description string,
) (string, error) {
	params := &stripe.PaymentIntentParams{
		Params:      stripe.Params{Context: ctx},
		Amount:      stripe.Int64(amountCents),
		Currency:    stripe.String("usd"),
		Description: stripe.String(description),
	}
	params.AddMetadata("tenant_id", tenantID)

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe payment intent: %w", err)
	}
	return pi.ClientSecret, nil
}

// HandleWebhook verifies a Stripe webhook signature and returns the parsed stripe.Event.
// webhookSecret overrides the instance-level secret when non-empty.
func (s *StripePayments) HandleWebhook(payload []byte, signature, webhookSecret string) (*stripe.Event, error) {
	secret := webhookSecret
	if secret == "" {
		secret = s.webhookSecret
	}
	event, err := webhook.ConstructEvent(payload, signature, secret)
	if err != nil {
		return nil, fmt.Errorf("webhook construct event: %w", err)
	}

	// Handle known event types as a side-effect.
	switch event.Type {
	case "invoice.paid":
		if err := s.ProcessInvoicePaid(context.Background(), &event, s.pgPool); err != nil {
			log.Printf("stripe_payments: process invoice.paid: %v", err)
		}
	case "customer.subscription.updated":
		if err := s.processSubscriptionUpdated(context.Background(), &event); err != nil {
			log.Printf("stripe_payments: process subscription.updated: %v", err)
		}
	}
	return &event, nil
}

// ProcessInvoicePaid records a successful payment in the stripe_payments table.
// pool may be nil, in which case the receiver's pool is used.
func (s *StripePayments) ProcessInvoicePaid(ctx context.Context, event *stripe.Event, pool *pgxpool.Pool) error {
	if pool == nil {
		pool = s.pgPool
	}
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}
	tenantID := ""
	if inv.Metadata != nil {
		tenantID = inv.Metadata["tenant_id"]
	}
	amountUSD := float64(inv.AmountPaid) / 100.0

	_, err := pool.Exec(ctx, `
		INSERT INTO stripe_payments
			(stripe_invoice_id, tenant_id, amount_usd, status, stripe_event_id, paid_at)
		VALUES ($1,$2,$3,'paid',$4,NOW())
		ON CONFLICT (stripe_invoice_id) DO UPDATE SET
			status   = 'paid',
			paid_at  = NOW()`,
		inv.ID, tenantID, amountUSD, event.ID)
	if err != nil {
		return fmt.Errorf("insert stripe_payments: %w", err)
	}
	log.Printf("stripe_payments: recorded invoice.paid invoice=%s tenant=%s amount=%.2f", inv.ID, tenantID, amountUSD)
	return nil
}

// processSubscriptionUpdated persists Stripe subscription state changes.
func (s *StripePayments) processSubscriptionUpdated(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}
	tenantID := ""
	if sub.Metadata != nil {
		tenantID = sub.Metadata["tenant_id"]
	}
	_, err := s.pgPool.Exec(ctx, `
		INSERT INTO stripe_subscriptions
			(stripe_subscription_id, tenant_id, status, current_period_end, updated_at)
		VALUES ($1,$2,$3,to_timestamp($4),NOW())
		ON CONFLICT (stripe_subscription_id) DO UPDATE SET
			status             = EXCLUDED.status,
			current_period_end = EXCLUDED.current_period_end,
			updated_at         = NOW()`,
		sub.ID, tenantID, string(sub.Status), sub.CurrentPeriodEnd)
	return err
}
