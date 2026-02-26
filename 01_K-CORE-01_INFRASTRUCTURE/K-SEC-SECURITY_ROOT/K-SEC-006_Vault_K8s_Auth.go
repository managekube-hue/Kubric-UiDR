//go:build ignore

// K-SEC-006 — Vault K8s Auth (reference stub)
//
// Production implementation: internal/security/vault_k8s_auth.go
//
// Types:
//   VaultClient — wraps hashicorp/vault/api with K8s ServiceAccount JWT auth
//
// Functions:
//   NewVaultClient() (*VaultClient, error)
//     K8s in-cluster: reads /var/run/secrets/kubernetes.io/serviceaccount/token
//     Dev fallback: uses VAULT_TOKEN env var
//
//   GetSecret(ctx, path) (map[string]interface{}, error)
//   GetDatabaseCreds(ctx, role) (username, password string, error)
//   GetNATSCredentials(ctx, serviceName) (url, user, pass string, error)
//   GetStripeKey(ctx) (secretKey, webhookSecret string, error)
//   GetTIAPIKey(ctx, provider) (apiKey string, error)
//   IssueCertificate(ctx, role, cn string, altNames []string) (cert, key, ca string, error)
//
// Credential loading orchestrated by:
//   internal/security/config.go → LoadServiceCreds()
//   All Go services call LoadServiceCreds() in their config.go

package stub
