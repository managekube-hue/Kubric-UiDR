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

	// -- ITDR identity intelligence (optional) ---------------------------------
	ITDRSigmaSecurityDir      string // ITDR_SIGMA_SECURITY_DIR
	ITDRSigmaPrivEscDir       string // ITDR_SIGMA_PRIVESC_DIR
	ITDRWazuhRulesDir         string // ITDR_WAZUH_RULES_DIR
	ITDRMispTaxonomiesDir     string // ITDR_MISP_TAXONOMIES_DIR
	ITDRBloodHoundCypherDir   string // ITDR_BLOODHOUND_CYPHER_DIR
	ITDRIdentityRespondersDir string // ITDR_IDENTITY_RESPONDERS_DIR
	OTXBaseURL                string // OTX_BASE_URL
	OTXAPIKey                 string // OTX_API_KEY

	// -- ZMQ high-throughput event fanout (optional) --------------------------
	ZMQPublishAddr string // ZMQ_PUBLISH_ADDR  e.g. tcp://0.0.0.0:5555

	// -- Twilio phone-based critical alert escalation (optional) ---------------
	TwilioAccountSID string // TWILIO_ACCOUNT_SID
	TwilioAuthToken  string // TWILIO_AUTH_TOKEN
	TwilioFromNumber string // TWILIO_FROM_NUMBER   E.164 e.g. +15005550006

	// -- Neo4j graph store (optional) ─────────────────────────────────────────
	Neo4jURI      string // NEO4J_URI          e.g. bolt://neo4j:7687
	Neo4jUser     string // NEO4J_USER         default: neo4j
	Neo4jPassword string // NEO4J_PASSWORD     default: kubric-neo4j

	// -- MinIO object store (optional) ────────────────────────────────────────
	MinIOEndpoint  string // MINIO_ENDPOINT     e.g. minio:9000
	MinIOAccessKey string // MINIO_ACCESS_KEY
	MinIOSecretKey string // MINIO_SECRET_KEY
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

		// ITDR identity intelligence
		ITDRSigmaSecurityDir:      getenv("ITDR_SIGMA_SECURITY_DIR", "vendor/sigma/rules/windows/builtin/security"),
		ITDRSigmaPrivEscDir:       getenv("ITDR_SIGMA_PRIVESC_DIR", "vendor/sigma/rules/windows/builtin/security/privesc"),
		ITDRWazuhRulesDir:         getenv("ITDR_WAZUH_RULES_DIR", "vendor/wazuh-rules"),
		ITDRMispTaxonomiesDir:     getenv("ITDR_MISP_TAXONOMIES_DIR", "vendor/misp/taxonomies"),
		ITDRBloodHoundCypherDir:   getenv("ITDR_BLOODHOUND_CYPHER_DIR", "vendor/bloodhound/cypher"),
		ITDRIdentityRespondersDir: getenv("ITDR_IDENTITY_RESPONDERS_DIR", "vendor/cortex/responders/identity"),
		OTXBaseURL:                getenv("OTX_BASE_URL", "https://otx.alienvault.com/api/v1/indicators"),
		OTXAPIKey:                 getenv("OTX_API_KEY", ""),

		// ZMQ fanout
		ZMQPublishAddr: getenv("ZMQ_PUBLISH_ADDR", ""),

		// Twilio escalation
		TwilioAccountSID: getenv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getenv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getenv("TWILIO_FROM_NUMBER", ""),

		// Neo4j graph
		Neo4jURI:      getenv("NEO4J_URI", "bolt://neo4j:7687"),
		Neo4jUser:     getenv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getenv("NEO4J_PASSWORD", "kubric-neo4j"),

		// MinIO object store
		MinIOEndpoint:  getenv("MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey: getenv("MINIO_ACCESS_KEY", "kubric"),
		MinIOSecretKey: getenv("MINIO_SECRET_KEY", "kubric-minio-secret"),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
