// Package ksvc implements the K-SVC tenant/customer REST API service.
// It exposes tenant CRUD over HTTP (Chi router), persists to PostgreSQL (pgx),
// and publishes lifecycle events to NATS.
package ksvc

import (
	"os"
	"strings"
)

// Config holds all runtime configuration for K-SVC, read from environment variables.
type Config struct {
	// ListenAddr is the TCP address the HTTP server binds to.
	// Env: KSVC_LISTEN_ADDR (default: :8080)
	ListenAddr string

	// DatabaseURL is a PostgreSQL connection string (pgx/pgxpool DSN).
	// Compatible with Supabase connection strings.
	// Env: KUBRIC_DATABASE_URL
	DatabaseURL string

	// NATSUrl is the NATS server URL for publishing tenant lifecycle events.
	// Env: KUBRIC_NATS_URL (default: nats://127.0.0.1:4222)
	NATSUrl string
}

// LoadConfig reads K-SVC configuration from environment variables.
func LoadConfig() Config {
	return Config{
		ListenAddr:  getenv("KSVC_LISTEN_ADDR", ":8080"),
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
