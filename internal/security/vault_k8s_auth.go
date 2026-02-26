// K-SEC-006 — Vault K8s Auth
// Kubernetes authentication for Vault — service accounts authenticate
// to Vault using their K8s JWT token. Secrets are fetched at runtime.
//
// No secrets are stored in environment variables or K8s Secrets directly.
// All services fetch secrets from Vault on startup via this client.

package security

import (
	"context"
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

// VaultClient wraps the HashiCorp Vault API client with K8s auth.
type VaultClient struct {
	client *vault.Client
}

// NewVaultClient creates a Vault client authenticated via K8s service account.
// When running outside K8s (dev mode), falls back to VAULT_TOKEN env var.
func NewVaultClient() (*VaultClient, error) {
	config := vault.DefaultConfig()
	config.Address = os.Getenv("VAULT_ADDR")
	if config.Address == "" {
		config.Address = "https://vault.kubric.svc.cluster.local:8200"
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("vault client init: %w", err)
	}

	// K8s auth when running in-cluster
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		role := os.Getenv("VAULT_ROLE")
		if role == "" {
			role = "kubric-default"
		}

		k8sAuth, err := auth.NewKubernetesAuth(
			role,
			auth.WithServiceAccountTokenPath(
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			),
		)
		if err != nil {
			return nil, fmt.Errorf("k8s auth init: %w", err)
		}

		authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
		if err != nil {
			return nil, fmt.Errorf("vault k8s login: %w", err)
		}
		if authInfo == nil {
			return nil, fmt.Errorf("vault k8s login returned nil auth info")
		}
	}
	// else: uses VAULT_TOKEN from env (dev mode)

	return &VaultClient{client: client}, nil
}

// GetSecret reads a KV v2 secret from vault at the given path.
func (v *VaultClient) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := v.client.KVv2("secret").Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read secret %s: %w", path, err)
	}
	return secret.Data, nil
}

// GetDatabaseCreds fetches dynamic database credentials from Vault's database engine.
func (v *VaultClient) GetDatabaseCreds(ctx context.Context, role string) (username, password string, err error) {
	secret, err := v.client.Logical().ReadWithContext(ctx, "database/creds/"+role)
	if err != nil {
		return "", "", fmt.Errorf("read db creds for role %s: %w", role, err)
	}
	if secret == nil {
		return "", "", fmt.Errorf("no db creds returned for role %s", role)
	}

	u, ok := secret.Data["username"].(string)
	if !ok {
		return "", "", fmt.Errorf("username not found in db creds")
	}
	p, ok := secret.Data["password"].(string)
	if !ok {
		return "", "", fmt.Errorf("password not found in db creds")
	}
	return u, p, nil
}

// GetNATSCredentials fetches NATS connection credentials from Vault.
func (v *VaultClient) GetNATSCredentials(ctx context.Context, serviceName string) (url, user, pass string, err error) {
	data, err := v.GetSecret(ctx, "nats/"+serviceName)
	if err != nil {
		return "", "", "", err
	}

	url, _ = data["url"].(string)
	user, _ = data["user"].(string)
	pass, _ = data["password"].(string)
	return url, user, pass, nil
}

// GetStripeKey fetches the Stripe secret key from Vault.
func (v *VaultClient) GetStripeKey(ctx context.Context) (secretKey, webhookSecret string, err error) {
	data, err := v.GetSecret(ctx, "stripe/api")
	if err != nil {
		return "", "", err
	}

	secretKey, _ = data["secret_key"].(string)
	webhookSecret, _ = data["webhook_secret"].(string)
	return secretKey, webhookSecret, nil
}

// GetTIAPIKey fetches a threat intelligence API key from Vault.
func (v *VaultClient) GetTIAPIKey(ctx context.Context, provider string) (string, error) {
	data, err := v.GetSecret(ctx, "ti/"+provider)
	if err != nil {
		return "", err
	}

	key, ok := data["api_key"].(string)
	if !ok {
		return "", fmt.Errorf("api_key not found for provider %s", provider)
	}
	return key, nil
}

// IssueCertificate requests a TLS certificate from Vault PKI.
func (v *VaultClient) IssueCertificate(ctx context.Context, role, commonName string, altNames []string) (cert, key, ca string, err error) {
	data := map[string]interface{}{
		"common_name": commonName,
		"ttl":         "720h",
	}
	if len(altNames) > 0 {
		combined := ""
		for i, n := range altNames {
			if i > 0 {
				combined += ","
			}
			combined += n
		}
		data["alt_names"] = combined
	}

	secret, err := v.client.Logical().WriteWithContext(ctx, "pki_int/issue/"+role, data)
	if err != nil {
		return "", "", "", fmt.Errorf("issue cert: %w", err)
	}

	cert, _ = secret.Data["certificate"].(string)
	key, _ = secret.Data["private_key"].(string)
	ca, _ = secret.Data["issuing_ca"].(string)
	return cert, key, ca, nil
}
