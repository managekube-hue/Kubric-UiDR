// Package billing implements Stripe billing integration for K-SVC.
// Uses github.com/stripe/stripe-go/v76 for customer, subscription,
// usage record, and billing portal operations.
package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/usagerecord"
)

// Init sets the Stripe API key. Call once at service startup.
// Key is fetched from Vault: secret/kubric/stripe/secret_key.
func Init(apiKey string) {
	stripe.Key = apiKey
}

// Tenant represents a Kubric tenant for billing purposes.
type Tenant struct {
	ID    string
	Name  string
	Email string
}

// CreateCustomer creates a Stripe customer for the given tenant.
func CreateCustomer(ctx context.Context, t *Tenant) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Params: stripe.Params{Context: ctx},
		Name:   stripe.String(t.Name),
		Email:  stripe.String(t.Email),
		Metadata: map[string]string{
			"kubric_tenant_id": t.ID,
		},
	}
	return customer.New(params)
}

// CreateSubscription creates a Stripe subscription for a customer.
// tier maps to a Stripe Price ID configured in the dashboard.
func CreateSubscription(ctx context.Context, customerID string, priceID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Params:   stripe.Params{Context: ctx},
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(priceID)},
		},
	}
	return subscription.New(params)
}

// CreateUsageRecord reports metered usage for a subscription item.
func CreateUsageRecord(ctx context.Context, subscriptionItemID string, quantity int64) (*stripe.UsageRecord, error) {
	params := &stripe.UsageRecordParams{
		Params:           stripe.Params{Context: ctx},
		SubscriptionItem: stripe.String(subscriptionItemID),
		Quantity:         stripe.Int64(quantity),
		Action:           stripe.String("increment"),
	}
	return usagerecord.New(params)
}

// GetBillingPortalURL returns a Stripe billing portal URL for customer self-service.
func GetBillingPortalURL(ctx context.Context, customerID string, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Params:    stripe.Params{Context: ctx},
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
	s, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("billing portal: %w", err)
	}
	return s.URL, nil
}

// CreateCheckoutSession creates a Stripe Checkout session for new subscriptions.
func CreateCheckoutSession(ctx context.Context, customerID string, priceID string, successURL string, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Params:     stripe.Params{Context: ctx},
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
	}
	s, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("checkout session: %w", err)
	}
	return s.URL, nil
}
