# K-SEC-005 — Vault Policies
# HashiCorp Vault HCL policies for all Kubric service identities
# Apply: vault policy write <name> <file>

# ──────────────────────────────────────────────────────────────
# Agent secrets — each Rust agent gets its own credential path
# ──────────────────────────────────────────────────────────────
path "secret/data/agents/*" {
  capabilities = ["read"]
}

# ──────────────────────────────────────────────────────────────
# NATS credentials — connection strings, NKeys, JWT tokens
# ──────────────────────────────────────────────────────────────
path "secret/data/nats/*" {
  capabilities = ["read"]
}

# ──────────────────────────────────────────────────────────────
# Database credentials — dynamic secrets per service identity
# Each service gets its own database role via Vault database engine
# ──────────────────────────────────────────────────────────────
path "database/creds/kubric-{{identity.entity.name}}" {
  capabilities = ["read"]
}

# K-SVC needs PostgreSQL read/write
path "database/creds/kubric-ksvc" {
  capabilities = ["read"]
}

# KAI needs ClickHouse read + PostgreSQL read
path "database/creds/kubric-kai-clickhouse" {
  capabilities = ["read"]
}

path "database/creds/kubric-kai-postgres" {
  capabilities = ["read"]
}

# VDR needs ClickHouse write for scan results
path "database/creds/kubric-vdr" {
  capabilities = ["read"]
}

# ──────────────────────────────────────────────────────────────
# PKI — mTLS certificate issuance for service-to-service TLS
# ──────────────────────────────────────────────────────────────
path "pki_int/issue/kubric-service" {
  capabilities = ["create", "update"]
}

path "pki_int/issue/kubric-agent" {
  capabilities = ["create", "update"]
}

path "pki_int/issue/kubric-nats" {
  capabilities = ["create", "update"]
}

# ──────────────────────────────────────────────────────────────
# Transit engine — Blake3 signing via HSM-backed keys
# ──────────────────────────────────────────────────────────────
path "transit/encrypt/blake3-audit" {
  capabilities = ["create", "update"]
}

path "transit/decrypt/blake3-audit" {
  capabilities = ["create", "update"]
}

path "transit/sign/kubric-agent-signing" {
  capabilities = ["create", "update"]
}

path "transit/verify/kubric-agent-signing" {
  capabilities = ["read"]
}

# ──────────────────────────────────────────────────────────────
# K8s auth — service accounts authenticate to Vault
# ──────────────────────────────────────────────────────────────
path "auth/kubernetes/login" {
  capabilities = ["create", "update"]
}

# ──────────────────────────────────────────────────────────────
# API keys and third-party secrets
# ──────────────────────────────────────────────────────────────

# Stripe API keys (KAI-CLERK / K-SVC billing)
path "secret/data/stripe/*" {
  capabilities = ["read"]
}

# LLM API keys (Ollama, vLLM, OpenAI fallback, Anthropic fallback)
path "secret/data/llm/*" {
  capabilities = ["read"]
}

# Threat intelligence API keys (OTX, AbuseIPDB, MISP, VirusTotal)
path "secret/data/ti/*" {
  capabilities = ["read"]
}

# PSA integrations (Zammad, n8n)
path "secret/data/psa/*" {
  capabilities = ["read"]
}

# Communication (Vapi, Twilio)
path "secret/data/comm/*" {
  capabilities = ["read"]
}

# MinIO / S3 object storage
path "secret/data/minio/*" {
  capabilities = ["read"]
}

# ──────────────────────────────────────────────────────────────
# Admin policy (NOC operators only — MFA required)
# ──────────────────────────────────────────────────────────────
# path "secret/*" {
#   capabilities = ["create", "read", "update", "delete", "list"]
#   required_parameters = ["mfa_validated"]
# }

# ──────────────────────────────────────────────────────────────
# Deny rules — prevent accidental secret deletion
# ──────────────────────────────────────────────────────────────
path "secret/data/agents/*" {
  capabilities = ["read"]
  # Explicitly deny delete for non-admin
  denied_parameters = {}
}

path "secret/metadata/*" {
  capabilities = ["list", "read"]
}

# Prevent any identity from reading the root token path
path "auth/token/create-orphan" {
  capabilities = ["deny"]
}
