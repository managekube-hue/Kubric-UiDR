// Package security provides Vault-backed secret management for all Kubric services.
//
// LoadServiceConfig attempts to fetch credentials from Vault when running in
// Kubernetes (detected via KUBERNETES_SERVICE_HOST env var). Falls back to
// environment variables for local development.
package security

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// ServiceCreds holds the credentials a Go service needs to operate.
type ServiceCreds struct {
	DatabaseURL   string
	NATSUrl       string
	ClickHouseURL string
	JWTSecret     string
}

// LoadServiceCreds fetches credentials from Vault when in-cluster,
// falls back to environment variables for local development.
func LoadServiceCreds(ctx context.Context, serviceName string) ServiceCreds {
	// When running inside K8s, use Vault
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		creds, err := loadFromVault(ctx, serviceName)
		if err != nil {
			log.Printf("security: vault fetch failed for %s, falling back to env: %v", serviceName, err)
			return loadFromEnv()
		}
		return creds
	}
	return loadFromEnv()
}

func loadFromVault(ctx context.Context, serviceName string) (ServiceCreds, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	vc, err := NewVaultClient()
	if err != nil {
		return ServiceCreds{}, fmt.Errorf("vault client: %w", err)
	}

	creds := ServiceCreds{}

	// Database credentials — dynamic secrets from Vault database engine
	dbUser, dbPass, err := vc.GetDatabaseCreds(ctx, "kubric-"+serviceName)
	if err != nil {
		log.Printf("security: vault db creds for %s: %v (falling back to env)", serviceName, err)
		creds.DatabaseURL = envOrDefault("KUBRIC_DATABASE_URL", "")
	} else {
		dbHost := envOrDefault("KUBRIC_DB_HOST", "postgresql:5432")
		dbName := envOrDefault("KUBRIC_DB_NAME", "kubric")
		creds.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=require", dbUser, dbPass, dbHost, dbName)
	}

	// NATS credentials from Vault KV
	natsURL, natsUser, natsPass, err := vc.GetNATSCredentials(ctx, serviceName)
	if err != nil {
		log.Printf("security: vault nats creds for %s: %v (falling back to env)", serviceName, err)
		creds.NATSUrl = envOrDefault("KUBRIC_NATS_URL", "nats://127.0.0.1:4222")
	} else {
		if natsUser != "" && natsPass != "" {
			// Inject creds into URL: nats://user:pass@host:port
			natsURL = strings.Replace(natsURL, "nats://", fmt.Sprintf("nats://%s:%s@", natsUser, natsPass), 1)
		}
		creds.NATSUrl = natsURL
	}

	// JWT secret from Vault KV
	jwtData, err := vc.GetSecret(ctx, "auth/jwt")
	if err != nil {
		log.Printf("security: vault jwt secret: %v (falling back to env)", err)
		creds.JWTSecret = envOrDefault("KUBRIC_JWT_SECRET", "")
	} else {
		if s, ok := jwtData["secret"].(string); ok {
			creds.JWTSecret = s
		}
	}

	return creds, nil
}

func loadFromEnv() ServiceCreds {
	return ServiceCreds{
		DatabaseURL:   envOrDefault("KUBRIC_DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/kubric"),
		NATSUrl:       envOrDefault("KUBRIC_NATS_URL", "nats://127.0.0.1:4222"),
		ClickHouseURL: envOrDefault("KUBRIC_CLICKHOUSE_URL", ""),
		JWTSecret:     envOrDefault("KUBRIC_JWT_SECRET", ""),
	}
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
