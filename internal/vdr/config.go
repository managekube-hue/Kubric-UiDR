// Package vdr implements the Vulnerability Detection & Response intake API.
// It receives normalized findings from Nuclei, Trivy, and Grype scanners,
// persists them to PostgreSQL, and publishes vuln events to NATS.
package vdr

import (
	"context"
	"os"
	"strings"

	"github.com/managekube-hue/Kubric-UiDR/internal/security"
)

// Config holds all VDR runtime configuration.
type Config struct {
	// ListenAddr — VDR_LISTEN_ADDR (default :8081)
	ListenAddr string
	// DatabaseURL — Vault dynamic creds or KUBRIC_DATABASE_URL env
	DatabaseURL string
	// NATSUrl — Vault KV or KUBRIC_NATS_URL env
	NATSUrl string
	// ClickHouseURL — KUBRIC_CLICKHOUSE_URL (optional; enables EPSS enrichment)
	ClickHouseURL string
}

// LoadConfig reads VDR configuration. Vault-backed in K8s, env vars in dev.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "vdr")
	return Config{
		ListenAddr:    getenv("VDR_LISTEN_ADDR", ":8081"),
		DatabaseURL:   creds.DatabaseURL,
		NATSUrl:       creds.NATSUrl,
		ClickHouseURL: getenv("KUBRIC_CLICKHOUSE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
