// Package kic implements the Kubernetes Infrastructure Compliance intake API.
// It receives OSCAL/Lula/kube-bench assessment results, persists them to
// PostgreSQL, and publishes compliance events to NATS.
package kic

import (
	"context"
	"os"
	"strings"

	"github.com/managekube-hue/Kubric-UiDR/internal/security"
)

// Config holds all KIC runtime configuration.
type Config struct {
	// ListenAddr — KIC_LISTEN_ADDR (default :8082)
	ListenAddr string
	// DatabaseURL — Vault dynamic creds or KUBRIC_DATABASE_URL env
	DatabaseURL string
	// NATSUrl — Vault KV or KUBRIC_NATS_URL env
	NATSUrl string
	// RAGServiceURL — base URL of the KAI RAG CISO-Assistant Python service
	RAGServiceURL string
	// GitHubToken — PAT for OpenSSF Scorecard API (optional)
	GitHubToken string
	// DuckDBPath — path to DuckDB file or ":memory:" (default :memory:)
	DuckDBPath string
	// CosignPubKeyPath — PEM path for Sigstore image verification (optional)
	CosignPubKeyPath string
}

// LoadConfig reads KIC configuration. Vault-backed in K8s, env vars in dev.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "kic")
	return Config{
		ListenAddr:       getenv("KIC_LISTEN_ADDR", ":8082"),
		DatabaseURL:      creds.DatabaseURL,
		NATSUrl:          creds.NATSUrl,
		RAGServiceURL:    getenv("KAI_RAG_URL", "http://kai-rag:8090"),
		GitHubToken:      getenv("GITHUB_AUTH_TOKEN", ""),
		DuckDBPath:       getenv("DUCKDB_PATH", ":memory:"),
		CosignPubKeyPath: getenv("COSIGN_PUB_KEY", ""),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
