"""
K-KAI-KP-003: HashiCorp Vault Secret Fetcher
Uses hvac library with AppRole auth. Supports automatic token renewal.
"""

import logging
import os
import threading
import time
from typing import Any

import hvac

logger = logging.getLogger("K-KAI-KP-003")

VAULT_ADDR: str = os.getenv("VAULT_ADDR", "http://vault:8200")
VAULT_TOKEN: str = os.getenv("VAULT_TOKEN", "")
VAULT_ROLE_ID: str = os.getenv("VAULT_ROLE_ID", "")
VAULT_SECRET_ID: str = os.getenv("VAULT_SECRET_ID", "")

TOKEN_TTL_SECONDS: int = 768 * 3600      # 768 hours
RENEW_AT_FRACTION: float = 0.80          # renew when 80% of TTL elapsed
RENEW_CHECK_INTERVAL: int = 300          # check every 5 minutes


class VaultSecretFetcher:
    """
    Thread-safe Vault client with AppRole auth and automatic token renewal.

    Constructor prefers VAULT_ROLE_ID / VAULT_SECRET_ID (AppRole) over
    VAULT_TOKEN for auth.  Falls back to VAULT_TOKEN if AppRole is not set.
    """

    def __init__(self) -> None:
        self._client: hvac.Client = hvac.Client(url=VAULT_ADDR)
        self._token_issued_at: float = 0.0
        self._lock: threading.Lock = threading.Lock()
        self._renewal_thread: threading.Thread | None = None

        self._authenticate()
        self._start_renewal_thread()

    # ------------------------------------------------------------------
    # Auth
    # ------------------------------------------------------------------

    def _authenticate(self) -> None:
        """Authenticate via AppRole if credentials are present, else use VAULT_TOKEN."""
        if VAULT_ROLE_ID and VAULT_SECRET_ID:
            logger.info("Authenticating to Vault via AppRole …")
            result = self._client.auth.approle.login(
                role_id=VAULT_ROLE_ID,
                secret_id=VAULT_SECRET_ID,
            )
            self._client.token = result["auth"]["client_token"]
            ttl = result["auth"].get("lease_duration", TOKEN_TTL_SECONDS)
            self._token_ttl: int = int(ttl)
            self._token_issued_at = time.monotonic()
            logger.info("Vault AppRole auth successful (TTL %d s).", ttl)
        elif VAULT_TOKEN:
            logger.info("Authenticating to Vault via static VAULT_TOKEN.")
            self._client.token = VAULT_TOKEN
            self._token_ttl = TOKEN_TTL_SECONDS
            self._token_issued_at = time.monotonic()
        else:
            raise RuntimeError("No Vault credentials found. Set VAULT_ROLE_ID+VAULT_SECRET_ID or VAULT_TOKEN.")

        if not self._client.is_authenticated():
            raise RuntimeError("Vault authentication failed.")

    def _start_renewal_thread(self) -> None:
        """Start background thread that renews the Vault token before expiry."""
        self._renewal_thread = threading.Thread(
            target=self._renewal_loop, daemon=True, name="vault-token-renewer"
        )
        self._renewal_thread.start()

    def _renewal_loop(self) -> None:
        while True:
            time.sleep(RENEW_CHECK_INTERVAL)
            elapsed = time.monotonic() - self._token_issued_at
            if elapsed >= self._token_ttl * RENEW_AT_FRACTION:
                try:
                    self.rotate_token()
                except Exception as exc:  # noqa: BLE001
                    logger.error("Token renewal failed: %s", exc)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def get_secret(self, path: str) -> dict:
        """
        Read a KV v2 secret from Vault.

        Args:
            path: Secret path relative to the KV v2 mount, e.g.
                  "kubric/database/prod" (without the "secret/data/" prefix).

        Returns:
            The 'data' dict from the KV v2 response.
        """
        with self._lock:
            try:
                # Try KV v2 first
                response = self._client.secrets.kv.v2.read_secret_version(
                    path=path, raise_on_deleted_version=True
                )
                return response["data"]["data"]
            except Exception:  # noqa: BLE001
                pass

            try:
                # Fallback to KV v1
                response = self._client.secrets.kv.v1.read_secret(path=path)
                return response["data"]
            except Exception as exc:
                logger.error("Failed to read Vault secret at path '%s': %s", path, exc)
                raise

    def get_db_creds(self, service: str) -> tuple[str, str]:
        """
        Retrieve PostgreSQL credentials for a service from Vault.

        Convention: secrets are stored at "kubric/{service}/db"
        with keys "username" and "password".

        Returns:
            (username, password) tuple.
        """
        secret = self.get_secret(f"kubric/{service}/db")
        username: str = secret.get("username") or secret.get("user", "")
        password: str = secret.get("password") or secret.get("pass", "")
        if not username or not password:
            raise ValueError(
                f"Vault secret for service '{service}' missing username or password."
            )
        return username, password

    def rotate_token(self) -> str:
        """
        Renew (or re-authenticate) the current Vault token.

        Returns the new token string.
        """
        with self._lock:
            if VAULT_ROLE_ID and VAULT_SECRET_ID:
                logger.info("Rotating Vault token via AppRole re-auth …")
                result = self._client.auth.approle.login(
                    role_id=VAULT_ROLE_ID, secret_id=VAULT_SECRET_ID
                )
                self._client.token = result["auth"]["client_token"]
                self._token_ttl = int(result["auth"].get("lease_duration", TOKEN_TTL_SECONDS))
                self._token_issued_at = time.monotonic()
                logger.info("Vault token rotated. New TTL: %d s.", self._token_ttl)
            else:
                # Renew existing token
                result = self._client.auth.token.renew_self(increment=TOKEN_TTL_SECONDS)
                self._token_ttl = int(result["auth"].get("lease_duration", TOKEN_TTL_SECONDS))
                self._token_issued_at = time.monotonic()
                logger.info("Vault token renewed. New TTL: %d s.", self._token_ttl)

            return self._client.token
