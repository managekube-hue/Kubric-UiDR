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

	// -- Integration endpoints (all optional; empty = disabled) ---------------

	WazuhURL      string // WAZUH_URL         e.g. https://wazuh:55000
	WazuhUser     string // WAZUH_USER
	WazuhPassword string // WAZUH_PASSWORD

	VelociraptorURL    string // VELOCIRAPTOR_URL   e.g. https://velociraptor:8001
	VelociraptorAPIKey string // VELOCIRAPTOR_API_KEY

	TheHiveURL    string // THEHIVE_URL        e.g. http://thehive:9000
	TheHiveAPIKey string // THEHIVE_API_KEY

	CortexURL    string // CORTEX_URL         e.g. http://cortex:9001
	CortexAPIKey string // CORTEX_API_KEY

	FalcoURL string // FALCO_URL          e.g. http://falco:8765

	OsqueryURL    string // OSQUERY_URL        e.g. https://fleet:8080
	OsqueryAPIKey string // OSQUERY_API_KEY

	ShuffleURL    string // SHUFFLE_URL        e.g. https://shuffle:3443
	ShuffleAPIKey string // SHUFFLE_API_KEY

	BloodHoundURL      string // BLOODHOUND_URL         e.g. http://bloodhound:8080
	BloodHoundTokenID  string // BLOODHOUND_TOKEN_ID
	BloodHoundTokenKey string // BLOODHOUND_TOKEN_KEY
}

// LoadConfig reads NOC configuration. Vault-backed in K8s, env vars in dev.
func LoadConfig() Config {
	creds := security.LoadServiceCreds(context.Background(), "noc")
	return Config{
		ListenAddr:  getenv("NOC_LISTEN_ADDR", ":8083"),
		DatabaseURL: creds.DatabaseURL,
		NATSUrl:     creds.NATSUrl,

		// Integration endpoints (all optional)
		WazuhURL:           getenv("WAZUH_URL", ""),
		WazuhUser:          getenv("WAZUH_USER", ""),
		WazuhPassword:      getenv("WAZUH_PASSWORD", ""),
		VelociraptorURL:    getenv("VELOCIRAPTOR_URL", ""),
		VelociraptorAPIKey: getenv("VELOCIRAPTOR_API_KEY", ""),
		TheHiveURL:         getenv("THEHIVE_URL", ""),
		TheHiveAPIKey:      getenv("THEHIVE_API_KEY", ""),
		CortexURL:          getenv("CORTEX_URL", ""),
		CortexAPIKey:       getenv("CORTEX_API_KEY", ""),
		FalcoURL:           getenv("FALCO_URL", ""),
		OsqueryURL:         getenv("OSQUERY_URL", ""),
		OsqueryAPIKey:      getenv("OSQUERY_API_KEY", ""),
		ShuffleURL:         getenv("SHUFFLE_URL", ""),
		ShuffleAPIKey:      getenv("SHUFFLE_API_KEY", ""),
		BloodHoundURL:      getenv("BLOODHOUND_URL", ""),
		BloodHoundTokenID:  getenv("BLOODHOUND_TOKEN_ID", ""),
		BloodHoundTokenKey: getenv("BLOODHOUND_TOKEN_KEY", ""),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
