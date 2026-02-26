// Package noc implements the Network Operations Center API.
// It tracks cluster health, agent heartbeats, and operational status,
// persists state to PostgreSQL, and publishes NOC events to NATS.
package noc

import (
	"context"
	"os"
	"strings"

	"github.com/managekube-hue/Kubric-UiDR/internal/security"
)

// Config holds all NOC runtime configuration.
type Config struct {
	// ListenAddr — NOC_LISTEN_ADDR (default :8083)
	ListenAddr string
	// DatabaseURL — Vault dynamic creds or KUBRIC_DATABASE_URL env
	DatabaseURL string
	// NATSUrl — Vault KV or KUBRIC_NATS_URL env
	NATSUrl string
}

// LoadConfig reads NOC configuration. Vault-backed in K8s, env vars in dev.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "noc")
	return Config{
		ListenAddr:  getenv("NOC_LISTEN_ADDR", ":8083"),
		DatabaseURL: creds.DatabaseURL,
		NATSUrl:     creds.NATSUrl,
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
