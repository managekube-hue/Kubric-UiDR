"""
kai/core/vault.py — HashiCorp Vault secret injection.

Called once at KAI startup (lifespan) before any other core components are
initialised.  On failure the startup continues — secrets may already be
injected by the Kubernetes external-secrets operator or set directly in the
container env.

Supports two auth methods:
  - AppRole (production): VAULT_ROLE_ID + VAULT_SECRET_ID
  - Token   (dev / CI):   VAULT_TOKEN

Secrets are read from the KV v2 mount at ``secret/kubric/kai`` and written
directly into ``os.environ`` so that all downstream imports (nats_client, llm,
ti_feeds, billing) pick them up transparently.
"""

from __future__ import annotations

import logging
import os

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Vault path → env var mapping
# Format: "kv_path|field_name" → "ENV_VAR_NAME"
# ---------------------------------------------------------------------------
_VAULT_SECRETS: dict[str, str] = {
    "kubric/kai|stripe_api_key":         "KUBRIC_STRIPE_API_KEY",
    "kubric/kai|stripe_webhook_secret":  "KUBRIC_STRIPE_WEBHOOK_SECRET",
    "kubric/kai|nats_creds":             "KUBRIC_NATS_CREDS",
    "kubric/kai|clickhouse_url":         "KUBRIC_CLICKHOUSE_URL",
    "kubric/kai|nvd_api_key":            "KUBRIC_NVD_API_KEY",
    "kubric/kai|otx_api_key":            "KUBRIC_OTX_API_KEY",
    "kubric/kai|misp_url":               "KUBRIC_MISP_URL",
    "kubric/kai|misp_api_key":           "KUBRIC_MISP_API_KEY",
    "kubric/kai|abuseipdb_api_key":      "KUBRIC_ABUSEIPDB_API_KEY",
    "kubric/kai|database_url":           "KUBRIC_DATABASE_URL",
    "kubric/kai|jwt_secret":             "KUBRIC_JWT_SECRET",
}

# KV mount point (default for Vault dev server and most productions setups)
_KV_MOUNT = "secret"


def inject_vault_secrets() -> bool:
    """
    Connect to Vault and inject missing secrets into ``os.environ``.

    Returns True if at least one secret was injected.  Returns False if Vault
    is unreachable or not configured — the caller should log a warning and
    continue (secrets may already be present from Kubernetes secrets / env).
    """
    vault_addr = os.getenv("VAULT_ADDR", "").strip()
    if not vault_addr:
        logger.info("vault: VAULT_ADDR not set — skipping secret injection")
        return False

    try:
        import hvac  # type: ignore[import]
    except ImportError:
        logger.warning(
            "vault: hvac not installed (pip install hvac) — skipping; "
            "set secrets via env vars or K8s external-secrets instead"
        )
        return False

    client = hvac.Client(url=vault_addr)

    # ── Authentication ────────────────────────────────────────────────────────
    role_id   = os.getenv("VAULT_ROLE_ID", "").strip()
    secret_id = os.getenv("VAULT_SECRET_ID", "").strip()
    vault_token = os.getenv("VAULT_TOKEN", "").strip()

    if role_id and secret_id:
        try:
            resp = client.auth.approle.login(role_id=role_id, secret_id=secret_id)
            client.token = resp["auth"]["client_token"]
            logger.info("vault: authenticated via AppRole")
        except Exception as exc:  # noqa: BLE001
            logger.error("vault: AppRole auth failed: %s", exc)
            return False
    elif vault_token:
        client.token = vault_token
        logger.info("vault: authenticated via token")
    else:
        logger.warning(
            "vault: no credentials — set VAULT_TOKEN or VAULT_ROLE_ID+VAULT_SECRET_ID"
        )
        return False

    if not client.is_authenticated():
        logger.error("vault: token not authenticated")
        return False

    # ── Secret injection ──────────────────────────────────────────────────────
    # Cache reads per KV path to avoid duplicate API calls
    _cache: dict[str, dict[str, str]] = {}
    injected = 0

    for path_field, env_name in _VAULT_SECRETS.items():
        # Skip env vars that are already set (explicit env always wins)
        if os.environ.get(env_name):
            continue

        kv_path, _, field = path_field.partition("|")
        if kv_path not in _cache:
            try:
                secret = client.secrets.kv.v2.read_secret_version(
                    path=kv_path,
                    mount_point=_KV_MOUNT,
                    raise_on_deleted_version=True,
                )
                _cache[kv_path] = secret.get("data", {}).get("data", {})
            except Exception as exc:  # noqa: BLE001
                logger.warning("vault: could not read secret/%s: %s", kv_path, exc)
                _cache[kv_path] = {}

        value = _cache[kv_path].get(field, "")
        if value:
            os.environ[env_name] = str(value)
            injected += 1
            logger.debug("vault: injected %s from %s|%s", env_name, kv_path, field)

    logger.info("vault: injected %d secret(s) from Vault at %s", injected, vault_addr)
    return injected > 0
