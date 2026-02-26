// Package noc implements the Network Operations Center API.
// It tracks cluster health, agent heartbeats, and operational status,
// persists state to PostgreSQL, and publishes NOC events to NATS.
package noc

import (
	"os"
	"strings"
)

// Config holds all NOC runtime configuration read from environment variables.
type Config struct {
	// ListenAddr — NOC_LISTEN_ADDR (default :8083)
	ListenAddr string
	// DatabaseURL — KUBRIC_DATABASE_URL (Supabase or local Postgres)
	DatabaseURL string
	// NATSUrl — KUBRIC_NATS_URL (default nats://127.0.0.1:4222)
	NATSUrl string
}

// LoadConfig reads NOC configuration from environment variables.
func LoadConfig() Config {
	return Config{
		ListenAddr:  getenv("NOC_LISTEN_ADDR", ":8083"),
		DatabaseURL: getenv("KUBRIC_DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/kubric"),
		NATSUrl:     getenv("KUBRIC_NATS_URL", "nats://127.0.0.1:4222"),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
