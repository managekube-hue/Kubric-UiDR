"""
K-KAI-TR-002: LLaMA3 Reasoning Wrapper
Calls Ollama HTTP API for local LLaMA3 inference.
Falls back to Anthropic claude-3-haiku-20240307 if Ollama is unavailable.
Includes retry logic and timeouts.
"""

import json
import logging
import os
import time
from typing import Any

import anthropic
import httpx

logger = logging.getLogger("K-KAI-TR-002")

OLLAMA_URL: str = os.getenv("OLLAMA_URL", "http://localhost:11434")
ANTHROPIC_API_KEY: str = os.getenv("ANTHROPIC_API_KEY", "")
OLLAMA_MODEL: str = os.getenv("OLLAMA_MODEL", "llama3")
OLLAMA_TIMEOUT: float = float(os.getenv("OLLAMA_TIMEOUT_SECONDS", "30"))
MAX_RETRIES: int = 3
RETRY_BACKOFF: float = 1.5  # seconds, exponential base


class LlamaReasoner:
    """
    LLM reasoning facade.

    Primary:  Ollama /api/generate  (local LLaMA3)
    Fallback: Anthropic claude-3-haiku-20240307
    """

    def __init__(self) -> None:
        self._anthropic_client: anthropic.Anthropic | None = None
        if ANTHROPIC_API_KEY:
            self._anthropic_client = anthropic.Anthropic(api_key=ANTHROPIC_API_KEY)
        else:
            logger.warning("ANTHROPIC_API_KEY not set; Anthropic fallback unavailable.")

        # Validate Ollama connectivity at construction time (best-effort)
        self._ollama_available: bool = self._ping_ollama()

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def reason(self, prompt: str, context: dict | None = None) -> str:
        """
        Run the prompt through LLaMA3 (Ollama) or fall back to Anthropic.

        Args:
            prompt:  Natural-language prompt.
            context: Optional extra key/value context serialised into the
                     system message.

        Returns:
            LLM response text.
        """
        if context:
            system_msg = (
                "You are a security AI assistant embedded in the Kubric-UiDR "
                "platform. Use the following context to inform your answer:\n"
                + json.dumps(context, default=str)[:2000]
            )
        else:
            system_msg = (
                "You are a security AI assistant embedded in the Kubric-UiDR "
                "platform. Provide concise, actionable security analysis."
            )

        if self._ollama_available:
            result = self._try_ollama(system_msg, prompt)
            if result is not None:
                return result
            logger.warning("Ollama call failed; switching to Anthropic fallback.")
            self._ollama_available = False

        if self._anthropic_client:
            return self._anthropic_reason(system_msg, prompt)

        raise RuntimeError(
            "No LLM backend available. Configure OLLAMA_URL or ANTHROPIC_API_KEY."
        )

    # ------------------------------------------------------------------
    # Ollama backend
    # ------------------------------------------------------------------

    def _ping_ollama(self) -> bool:
        """Return True if Ollama /api/tags responds within 5 s."""
        try:
            resp = httpx.get(f"{OLLAMA_URL}/api/tags", timeout=5.0)
            return resp.status_code == 200
        except Exception:  # noqa: BLE001
            logger.debug("Ollama not reachable at %s", OLLAMA_URL)
            return False

    def _try_ollama(self, system_msg: str, prompt: str) -> str | None:
        """
        Call Ollama /api/generate with retry.
        Returns response text or None on total failure.
        """
        payload: dict[str, Any] = {
            "model": OLLAMA_MODEL,
            "prompt": f"[SYSTEM]\n{system_msg}\n\n[USER]\n{prompt}",
            "stream": False,
            "options": {
                "temperature": 0.2,
                "num_predict": 512,
            },
        }

        for attempt in range(1, MAX_RETRIES + 1):
            try:
                resp = httpx.post(
                    f"{OLLAMA_URL}/api/generate",
                    json=payload,
                    timeout=OLLAMA_TIMEOUT,
                )
                resp.raise_for_status()
                data = resp.json()
                return data.get("response", "").strip()
            except httpx.TimeoutException:
                logger.warning("Ollama timeout on attempt %d/%d", attempt, MAX_RETRIES)
            except httpx.HTTPStatusError as exc:
                logger.warning(
                    "Ollama HTTP error %s on attempt %d/%d",
                    exc.response.status_code,
                    attempt,
                    MAX_RETRIES,
                )
            except Exception as exc:  # noqa: BLE001
                logger.warning("Ollama error on attempt %d/%d: %s", attempt, MAX_RETRIES, exc)

            if attempt < MAX_RETRIES:
                sleep_time = RETRY_BACKOFF ** attempt
                logger.debug("Retrying Ollama in %.1f s", sleep_time)
                time.sleep(sleep_time)

        return None

    # ------------------------------------------------------------------
    # Anthropic fallback
    # ------------------------------------------------------------------

    def _anthropic_reason(self, system_msg: str, prompt: str) -> str:
        """Call Anthropic claude-3-haiku with retry."""
        assert self._anthropic_client is not None  # mypy

        for attempt in range(1, MAX_RETRIES + 1):
            try:
                message = self._anthropic_client.messages.create(
                    model="claude-3-haiku-20240307",
                    max_tokens=512,
                    temperature=0.2,
                    system=system_msg,
                    messages=[{"role": "user", "content": prompt}],
                )
                return message.content[0].text.strip()
            except anthropic.RateLimitError:
                logger.warning(
                    "Anthropic rate limit on attempt %d/%d", attempt, MAX_RETRIES
                )
            except anthropic.APITimeoutError:
                logger.warning(
                    "Anthropic timeout on attempt %d/%d", attempt, MAX_RETRIES
                )
            except anthropic.APIError as exc:
                logger.warning(
                    "Anthropic API error on attempt %d/%d: %s", attempt, MAX_RETRIES, exc
                )

            if attempt < MAX_RETRIES:
                time.sleep(RETRY_BACKOFF ** attempt)

        raise RuntimeError("Anthropic API failed after all retries.")
