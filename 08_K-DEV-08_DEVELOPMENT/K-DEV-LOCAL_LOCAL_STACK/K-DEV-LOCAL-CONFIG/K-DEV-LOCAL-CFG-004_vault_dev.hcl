# ─────────────────────────────────────────────────────────────────────
# HashiCorp Vault – Development Configuration for Kubric UiDR
# ─────────────────────────────────────────────────────────────────────
# This configuration runs Vault in dev-friendly mode (TLS disabled)
# and pre-configures the secrets engines, auth methods and policies
# that the Kubric platform requires.
#
# Usage:
#   vault server -config=K-DEV-LOCAL-CFG-004_vault_dev.hcl
# ─────────────────────────────────────────────────────────────────────

# ── Listener ─────────────────────────────────────────────────────────
listener "tcp" {
  address         = "0.0.0.0:8200"
  tls_disable     = true           # Dev only – production uses mTLS
  cluster_address = "0.0.0.0:8201"
}

# ── Storage backend ─────────────────────────────────────────────────
storage "file" {
  path = "/vault/data"
}

# ── API / Cluster addresses ─────────────────────────────────────────
api_addr     = "http://127.0.0.1:8200"
cluster_addr = "http://127.0.0.1:8201"

# ── General settings ────────────────────────────────────────────────
ui            = true
log_level     = "info"
disable_mlock = true   # Required for containers / dev laptops

# ── Telemetry ────────────────────────────────────────────────────────
telemetry {
  prometheus_retention_time = "30s"
  disable_hostname          = true
}

# ─────────────────────────────────────────────────────────────────────
# BOOTSTRAP SCRIPT  (run once after `vault operator init`)
# ─────────────────────────────────────────────────────────────────────
# The sections below document the exact Vault CLI commands that the
# bootstrap script (scripts/bootstrap/vault-init.sh) executes.
# They are kept here as a declarative reference.
#
# ── 1. KV v2 Secrets Engine ─────────────────────────────────────────
# Stores application secrets (API keys, DB creds, signing keys).
#
#   vault secrets enable -path=secret -version=2 kv
#
#   Pre-seeded paths:
#     secret/data/kubric/nats          – NATS credentials
#     secret/data/kubric/clickhouse    – ClickHouse users
#     secret/data/kubric/postgres      – PostgreSQL superuser
#     secret/data/kubric/minio         – MinIO root credentials
#     secret/data/kubric/blake3        – BLAKE3 signing key material
#     secret/data/kubric/openai        – OpenAI / LLM API keys
#     secret/data/kubric/anthropic     – Anthropic API key
#
# ── 2. Transit Secrets Engine (BLAKE3 Signing) ───────────────────────
# Used for content-addressable hashing of audit log records.
#
#   vault secrets enable transit
#   vault write -f transit/keys/kubric-audit type=aes256-gcm96
#   vault write -f transit/keys/kubric-blake3 type=aes256-gcm96
#
# ── 3. PKI Secrets Engine (mTLS Certificates) ───────────────────────
# Issues short-lived mTLS certs for inter-service communication.
#
#   # Root CA
#   vault secrets enable -path=pki pki
#   vault secrets tune -max-lease-ttl=87600h pki
#   vault write pki/root/generate/internal \
#     common_name="Kubric UiDR Root CA" \
#     ttl=87600h
#
#   # Intermediate CA for agent certs
#   vault secrets enable -path=pki_int pki
#   vault secrets tune -max-lease-ttl=43800h pki_int
#   vault write pki_int/intermediate/generate/internal \
#     common_name="Kubric UiDR Intermediate CA" \
#     ttl=43800h
#
#   # Role for agent certificates
#   vault write pki_int/roles/kubric-agent \
#     allowed_domains="kubric.local,kubric.svc.cluster.local" \
#     allow_subdomains=true \
#     max_ttl=720h \
#     generate_lease=true
#
#   # Role for service mesh certificates
#   vault write pki_int/roles/kubric-service \
#     allowed_domains="kubric.svc.cluster.local" \
#     allow_subdomains=true \
#     max_ttl=24h \
#     generate_lease=true
#
# ── 4. Kubernetes Auth Method ────────────────────────────────────────
# Allows Kubernetes pods to authenticate with their ServiceAccount JWT.
#
#   vault auth enable kubernetes
#   vault write auth/kubernetes/config \
#     kubernetes_host="https://kubernetes.default.svc:443"
#
#   # Bind agent service accounts to Vault roles
#   vault write auth/kubernetes/role/kubric-coresec \
#     bound_service_account_names=kubric-coresec \
#     bound_service_account_namespaces=kubric \
#     policies=kubric-agent \
#     ttl=1h
#
#   vault write auth/kubernetes/role/kubric-netguard \
#     bound_service_account_names=kubric-netguard \
#     bound_service_account_namespaces=kubric \
#     policies=kubric-agent \
#     ttl=1h
#
#   vault write auth/kubernetes/role/kubric-perftrace \
#     bound_service_account_names=kubric-perftrace \
#     bound_service_account_namespaces=kubric \
#     policies=kubric-agent \
#     ttl=1h
#
#   vault write auth/kubernetes/role/kubric-watchdog \
#     bound_service_account_names=kubric-watchdog \
#     bound_service_account_namespaces=kubric \
#     policies=kubric-agent \
#     ttl=1h
#
#   vault write auth/kubernetes/role/kubric-api \
#     bound_service_account_names=kubric-api \
#     bound_service_account_namespaces=kubric \
#     policies=kubric-api \
#     ttl=1h
#
# ── 5. Database Secrets Engine (PostgreSQL Dynamic Creds) ────────────
# Generates short-lived PostgreSQL credentials per service request.
#
#   vault secrets enable database
#
#   vault write database/config/kubric-postgres \
#     plugin_name=postgresql-database-plugin \
#     allowed_roles="kubric-api-rw,kubric-readonly" \
#     connection_url="postgresql://{{username}}:{{password}}@postgres:5432/kubric?sslmode=disable" \
#     username="vault_admin" \
#     password="vault_admin_password"
#
#   # Read-write role for API service
#   vault write database/roles/kubric-api-rw \
#     db_name=kubric-postgres \
#     creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
#       GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA kubric TO \"{{name}}\"; \
#       ALTER ROLE \"{{name}}\" SET search_path = kubric;" \
#     default_ttl=1h \
#     max_ttl=24h
#
#   # Read-only role for dashboards / analytics
#   vault write database/roles/kubric-readonly \
#     db_name=kubric-postgres \
#     creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
#       GRANT SELECT ON ALL TABLES IN SCHEMA kubric TO \"{{name}}\"; \
#       ALTER ROLE \"{{name}}\" SET search_path = kubric;" \
#     default_ttl=1h \
#     max_ttl=8h
# ─────────────────────────────────────────────────────────────────────

# ─────────────────────────────────────────────────────────────────────
# POLICIES  (written via `vault policy write <name> - <<EOF ... EOF`)
# ─────────────────────────────────────────────────────────────────────

# ── kubric-agent policy ─────────────────────────────────────────────
# Attached to all XRO agent Kubernetes roles.
#
# path "secret/data/kubric/nats" {
#   capabilities = ["read"]
# }
#
# path "secret/data/kubric/blake3" {
#   capabilities = ["read"]
# }
#
# path "transit/encrypt/kubric-audit" {
#   capabilities = ["update"]
# }
#
# path "transit/encrypt/kubric-blake3" {
#   capabilities = ["update"]
# }
#
# path "pki_int/issue/kubric-agent" {
#   capabilities = ["create", "update"]
# }
#
# path "auth/token/renew-self" {
#   capabilities = ["update"]
# }
#
# path "auth/token/lookup-self" {
#   capabilities = ["read"]
# }

# ── kubric-api policy ───────────────────────────────────────────────
# Attached to the Kubric API service role.
#
# path "secret/data/kubric/*" {
#   capabilities = ["read", "list"]
# }
#
# path "database/creds/kubric-api-rw" {
#   capabilities = ["read"]
# }
#
# path "transit/encrypt/kubric-audit" {
#   capabilities = ["update"]
# }
#
# path "transit/decrypt/kubric-audit" {
#   capabilities = ["update"]
# }
#
# path "transit/encrypt/kubric-blake3" {
#   capabilities = ["update"]
# }
#
# path "pki_int/issue/kubric-service" {
#   capabilities = ["create", "update"]
# }
#
# path "auth/token/renew-self" {
#   capabilities = ["update"]
# }

# ── kubric-dashboard policy ─────────────────────────────────────────
# Read-only access for Grafana / dashboard services.
#
# path "database/creds/kubric-readonly" {
#   capabilities = ["read"]
# }
#
# path "secret/data/kubric/clickhouse" {
#   capabilities = ["read"]
# }
#
# path "auth/token/renew-self" {
#   capabilities = ["update"]
# }
