// Package vdr implements the Vulnerability Detection & Response intake API.
// It receives normalized findings from Nuclei, Trivy, and Grype scanners,
// persists them to PostgreSQL, and publishes vuln events to NATS.
package vdr

import (
	"os"
	"strings"
)

// Config holds all VDR runtime configuration read from environment variables.
type Config struct {
	// ListenAddr — VDR_LISTEN_ADDR (default :8081)
	ListenAddr string
	// DatabaseURL — KUBRIC_DATABASE_URL (Supabase or local Postgres)
	DatabaseURL string
	// NATSUrl — KUBRIC_NATS_URL (default nats://127.0.0.1:4222)
	NATSUrl string
}

// LoadConfig reads VDR configuration from environment variables.
func LoadConfig() Config {
	return Config{
		ListenAddr:  getenv("VDR_LISTEN_ADDR", ":8081"),
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
