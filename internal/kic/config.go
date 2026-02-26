// Package kic implements the Kubernetes Infrastructure Compliance intake API.
// It receives OSCAL/Lula/kube-bench assessment results, persists them to
// PostgreSQL, and publishes compliance events to NATS.
package kic

import (
	"os"
	"strings"
)

// Config holds all KIC runtime configuration read from environment variables.
type Config struct {
	// ListenAddr — KIC_LISTEN_ADDR (default :8082)
	ListenAddr string
	// DatabaseURL — KUBRIC_DATABASE_URL (Supabase or local Postgres)
	DatabaseURL string
	// NATSUrl — KUBRIC_NATS_URL (default nats://127.0.0.1:4222)
	NATSUrl string
}

// LoadConfig reads KIC configuration from environment variables.
func LoadConfig() Config {
	return Config{
		ListenAddr:  getenv("KIC_LISTEN_ADDR", ":8082"),
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
