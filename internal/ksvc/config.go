// Package ksvc implements the K-SVC tenant/customer REST API service.
// It exposes tenant CRUD over HTTP (Chi router), persists to PostgreSQL (pgx),
// and publishes lifecycle events to NATS.
package ksvc

import (
	"context"
	"os"
	"strings"

	"github.com/managekube-hue/Kubric-UiDR/internal/security"
)

// Config holds all runtime configuration for K-SVC.
type Config struct {
	// ListenAddr is the TCP address the HTTP server binds to.
	ListenAddr string

	// DatabaseURL is a PostgreSQL connection string (pgx/pgxpool DSN).
	DatabaseURL string

	// NATSUrl is the NATS server URL for publishing tenant lifecycle events.
	NATSUrl string

	// StripeAPIKey is the Stripe secret key used to authenticate Stripe API calls.
	// Read from STRIPE_SECRET_KEY; loaded from Vault in production.
	StripeAPIKey string

	// BillingReturnURL is the URL Stripe redirects customers to after leaving
	// the billing portal.  Defaults to "https://app.kubric.io/billing" when empty.
	BillingReturnURL string
}

// LoadConfig reads K-SVC configuration. When running in Kubernetes, credentials
// are fetched from Vault via service account authentication. Falls back to
// environment variables for local development.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "ksvc")
	return Config{
		ListenAddr:       getenv("KSVC_LISTEN_ADDR", ":8080"),
		DatabaseURL:      creds.DatabaseURL,
		NATSUrl:          creds.NATSUrl,
		StripeAPIKey:     getenv("STRIPE_SECRET_KEY", ""),
		BillingReturnURL: getenv("BILLING_RETURN_URL", ""),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
