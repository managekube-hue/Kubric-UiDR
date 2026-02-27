// Package kai implements the KAI (Kubric AI) orchestration gateway service.
// The Go gateway validates JWT auth, enforces tenant context, and reverse-proxies
// all /kai/* requests to the Python FastAPI process that hosts the CrewAI agents.
package kai

import (
	"context"
	"os"
	"strings"

	"github.com/managekube-hue/Kubric-UiDR/internal/security"
)

// Config holds all KAI gateway runtime configuration.
type Config struct {
	// ListenAddr — KAI_LISTEN_ADDR (default :8100)
	ListenAddr string
	// DatabaseURL — Vault dynamic creds or KUBRIC_DATABASE_URL env
	DatabaseURL string
	// NATSUrl — Vault KV or KUBRIC_NATS_URL env
	NATSUrl string
	// KAIServiceURL — URL of the Python KAI FastAPI process to proxy to.
	// KAI_SERVICE_URL (default http://kai-python:8101)
	KAIServiceURL string
	// AnthropicAPIKey — ANTHROPIC_API_KEY (forwarded to Python service via env; stored here for health checks)
	AnthropicAPIKey string
	// OpenAIAPIKey — OPENAI_API_KEY
	OpenAIAPIKey string
}

// LoadConfig reads KAI configuration. Vault-backed in K8s, env vars in local dev.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "kai")
	return Config{
		ListenAddr:      getenv("KAI_LISTEN_ADDR", ":8100"),
		DatabaseURL:     creds.DatabaseURL,
		NATSUrl:         creds.NATSUrl,
		KAIServiceURL:   getenv("KAI_SERVICE_URL", "http://kai-python:8101"),
		AnthropicAPIKey: getenv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:    getenv("OPENAI_API_KEY", ""),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
