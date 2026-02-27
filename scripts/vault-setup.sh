#!/usr/bin/env bash
# scripts/vault-setup.sh
# ---------------------------------------------------------------------------
# Kubric Vault Initialization Script
#
# Run ONCE after Vault is unsealed to configure:
#   1. KV-v2 secrets engine at secret/
#   2. Kubernetes auth method for External Secrets Operator (ESO)
#   3. Kubernetes auth method for Vault Agent Injector (app pods)
#   4. Policies: kubric-read, kubric-admin, kubric-external-secrets
#   5. Placeholder secret paths (ESO + KAI require these to exist)
#
# Usage:
#   export VAULT_ADDR=https://vault.your-domain.com
#   export VAULT_TOKEN=<root-token-or-admin-token>
#   ./scripts/vault-setup.sh
#
#   Optional env vars:
#     KUBRIC_DOMAIN   — your domain (default: kubric.security)
#     VAULT_NAMESPACE — Vault Enterprise namespace (default: empty/root)
#     DRY_RUN=1       — print commands without executing
# ---------------------------------------------------------------------------
set -euo pipefail

# ── Defaults ────────────────────────────────────────────────────────────────
VAULT_ADDR="${VAULT_ADDR:-http://vault.kubric.svc.cluster.local:8200}"
KUBRIC_DOMAIN="${KUBRIC_DOMAIN:-kubric.security}"
K8S_HOST="${K8S_HOST:-https://kubernetes.default.svc}"
DRY_RUN="${DRY_RUN:-0}"

GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m"

log_ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_step() { echo -e "\n${YELLOW}━━━ $* ━━━${NC}"; }
log_err()  { echo -e "${RED}[ERR]${NC}   $*" >&2; }

run() {
    if [[ "$DRY_RUN" == "1" ]]; then
        echo "[DRY-RUN] $*"
    else
        eval "$@"
    fi
}

# ── Preflight ────────────────────────────────────────────────────────────────
log_step "Preflight checks"
if ! command -v vault &>/dev/null; then
    log_err "vault CLI not found. Install from https://developer.hashicorp.com/vault/downloads"
    exit 1
fi
if [[ -z "${VAULT_TOKEN:-}" ]]; then
    log_err "VAULT_TOKEN not set"
    exit 1
fi
if ! vault status &>/dev/null; then
    log_err "Cannot reach Vault at $VAULT_ADDR — is it unsealed?"
    exit 1
fi
log_ok "Vault reachable at $VAULT_ADDR"

# ── 1. Enable KV-v2 ──────────────────────────────────────────────────────────
log_step "1. Enable KV-v2 secrets engine"
if vault secrets list | grep -q "^secret/"; then
    log_warn "secret/ already mounted — skipping"
else
    run vault secrets enable -path=secret kv-v2
    log_ok "KV-v2 mounted at secret/"
fi

# ── 2. Write policies ────────────────────────────────────────────────────────
log_step "2. Write Vault policies"

run vault policy write kubric-read - <<'EOF'
# kubric-read: read-only access to all kubric secrets (app pods)
path "secret/data/kubric/*" {
  capabilities = ["read", "list"]
}
path "secret/metadata/kubric/*" {
  capabilities = ["list"]
}
EOF
log_ok "Policy kubric-read written"

run vault policy write kubric-external-secrets - <<'EOF'
# kubric-external-secrets: External Secrets Operator service account
# Needs list + read on all kubric paths
path "secret/data/kubric/*" {
  capabilities = ["read", "list"]
}
path "secret/metadata/kubric/*" {
  capabilities = ["list"]
}
# ESO also needs to list the KV engine itself
path "secret/" {
  capabilities = ["list"]
}
EOF
log_ok "Policy kubric-external-secrets written"

run vault policy write kubric-admin - <<'EOF'
# kubric-admin: CI/CD pipeline and bootstrap scripts
path "secret/data/kubric/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "secret/metadata/kubric/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "auth/*" {
  capabilities = ["read", "list"]
}
path "sys/policies/*" {
  capabilities = ["read", "list"]
}
EOF
log_ok "Policy kubric-admin written"

# ── 3. Enable Kubernetes auth ────────────────────────────────────────────────
log_step "3. Enable Kubernetes auth method"
if vault auth list | grep -q "^kubernetes/"; then
    log_warn "kubernetes/ auth already enabled — skipping enable"
else
    run vault auth enable kubernetes
    log_ok "Kubernetes auth enabled"
fi

# Configure with in-cluster credentials
# When running inside the cluster, TokenReviewerJWT and CA are auto-detected
run vault write auth/kubernetes/config \
    kubernetes_host="$K8S_HOST" \
    issuer="https://kubernetes.default.svc.cluster.local"
log_ok "Kubernetes auth configured (host: $K8S_HOST)"

# ── 4. Create Kubernetes auth roles ──────────────────────────────────────────
log_step "4. Create Kubernetes auth roles"

# Role for External Secrets Operator
run vault write auth/kubernetes/role/kubric-external-secrets \
    bound_service_account_names=external-secrets \
    bound_service_account_namespaces=external-secrets \
    policies=kubric-external-secrets \
    ttl=1h
log_ok "Role kubric-external-secrets (ESO SA)"

# Role for KAI Python pods
run vault write auth/kubernetes/role/kubric-kai \
    bound_service_account_names=kubric-kai \
    bound_service_account_namespaces=kubric \
    policies=kubric-read \
    ttl=1h
log_ok "Role kubric-kai (KAI pods)"

# Role for Go service pods (k-svc, vdr, kic, noc)
run vault write auth/kubernetes/role/kubric-services \
    bound_service_account_names=kubric-services \
    bound_service_account_namespaces=kubric \
    policies=kubric-read \
    ttl=1h
log_ok "Role kubric-services (Go service pods)"

# Role for Rust XRO agents (registered via Vault AppRole, not K8s SA)
run vault auth enable approle 2>/dev/null || log_warn "AppRole auth already enabled"
run vault write auth/approle/role/kubric-xro \
    token_policies=kubric-read \
    token_ttl=24h \
    token_max_ttl=72h \
    secret_id_ttl=0 \
    bind_secret_id=true
log_ok "AppRole role kubric-xro (Rust XRO agents)"

# ── 5. Create placeholder secrets (ESO requires paths to exist) ──────────────
log_step "5. Create placeholder secrets in Vault"
log_warn "Replace ALL placeholder values with real credentials before production!"

# JWT signing key
run vault kv put secret/kubric/jwt \
    signing_key="REPLACE_ME_32_BYTE_HEX_JWT_SIGNING_KEY"
log_ok "secret/kubric/jwt"

# Stripe
run vault kv put secret/kubric/stripe \
    api_key="sk_live_REPLACE_ME" \
    webhook_secret="whsec_REPLACE_ME"
log_ok "secret/kubric/stripe"

# PostgreSQL
run vault kv put secret/kubric/postgres \
    url="postgresql://kubric:REPLACE_ME@postgres.kubric.svc.cluster.local:5432/kubric"
log_ok "secret/kubric/postgres"

# ClickHouse
run vault kv put secret/kubric/clickhouse \
    url="clickhouse://kubric:REPLACE_ME@clickhouse.kubric.svc.cluster.local:9000/kubric"
log_ok "secret/kubric/clickhouse"

# Threat Intelligence API keys
run vault kv put secret/kubric/ti/otx \
    api_key="REPLACE_ME_OTX_API_KEY"
log_ok "secret/kubric/ti/otx"

run vault kv put secret/kubric/ti/abuseipdb \
    api_key="REPLACE_ME_ABUSEIPDB_API_KEY"
log_ok "secret/kubric/ti/abuseipdb"

run vault kv put secret/kubric/ti/nvd \
    api_key="REPLACE_ME_NVD_API_KEY"
log_ok "secret/kubric/ti/nvd"

run vault kv put secret/kubric/ti/misp \
    url="https://misp.${KUBRIC_DOMAIN}" \
    api_key="REPLACE_ME_MISP_API_KEY"
log_ok "secret/kubric/ti/misp"

# n8n
run vault kv put secret/kubric/n8n \
    encryption_key="REPLACE_ME_32_CHAR_N8N_ENCRYPTION_KEY"
log_ok "secret/kubric/n8n"

# Zammad ITSM
run vault kv put secret/kubric/zammad \
    url="https://helpdesk.${KUBRIC_DOMAIN}" \
    token="REPLACE_ME_ZAMMAD_API_TOKEN"
log_ok "secret/kubric/zammad"

# Twilio (KAI-COMM voice alerts)
run vault kv put secret/kubric/twilio \
    account_sid="REPLACE_ME_TWILIO_SID" \
    auth_token="REPLACE_ME_TWILIO_AUTH_TOKEN" \
    phone_number="+1REPLACE0000"
log_ok "secret/kubric/twilio"

# Vapi.ai (AI phone calls)
run vault kv put secret/kubric/vapi \
    api_key="REPLACE_ME_VAPI_API_KEY" \
    phone_number_id="REPLACE_ME_VAPI_PHONE_ID"
log_ok "secret/kubric/vapi"

# Authentik secret key
run vault kv put secret/kubric/authentik \
    secret_key="REPLACE_ME_50_CHAR_AUTHENTIK_SECRET_KEY"
log_ok "secret/kubric/authentik"

# Vault Transit key for backup encryption (K-BAK)
if vault secrets list | grep -q "^transit/"; then
    log_warn "transit/ already mounted — skipping"
else
    run vault secrets enable transit
    log_ok "Transit secrets engine enabled"
fi
run vault write -f transit/keys/kubric-backup type=aes256-gcm96
log_ok "Transit key kubric-backup (AES-256-GCM96 for backup encryption)"

# ── 6. Fetch AppRole credentials for XRO agent bootstrap ────────────────────
log_step "6. Fetch XRO AppRole credentials (save these securely)"
if [[ "$DRY_RUN" != "1" ]]; then
    ROLE_ID=$(vault read -field=role_id auth/approle/role/kubric-xro/role-id)
    SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/kubric-xro/secret-id)
    echo ""
    echo "  ┌─────────────────────────────────────────────────────────────────┐"
    echo "  │  XRO Agent Registration Credentials (SAVE THESE NOW)            │"
    echo "  │  Copy to: agents/coresec/config/vault.toml                      │"
    echo "  ├─────────────────────────────────────────────────────────────────┤"
    echo "  │  VAULT_ROLE_ID    = $ROLE_ID"
    echo "  │  VAULT_SECRET_ID  = $SECRET_ID"
    echo "  └─────────────────────────────────────────────────────────────────┘"
    echo ""
    log_warn "SECRET_ID is single-use. Store it in the XRO agent's secure config NOW."
fi

# ── Summary ───────────────────────────────────────────────────────────────────
log_step "Setup Complete"
echo ""
echo "  Vault is configured for Kubric production deployment."
echo ""
echo "  NEXT STEPS:"
echo "  1. Replace ALL 'REPLACE_ME' values in Vault with real credentials:"
echo "       vault kv patch secret/kubric/jwt signing_key=<real-key>"
echo "       vault kv patch secret/kubric/stripe api_key=<real-sk_live_key>"
echo "       ... (repeat for all paths above)"
echo ""
echo "  2. Apply External Secrets Operator:"
echo "       kubectl apply -f infra/external-secrets/"
echo ""
echo "  3. Verify ESO syncing:"
echo "       kubectl get externalsecret -n kubric"
echo ""
echo "  4. Deploy services:"
echo "       kubectl apply -k infra/k8s/overlays/prod"
echo ""
echo "  Vault UI: $VAULT_ADDR/ui"
echo ""
