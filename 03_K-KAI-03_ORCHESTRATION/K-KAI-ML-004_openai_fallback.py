"""
K-KAI-ML-004_openai_fallback.py
OpenAI API fallback client with retry, cost tracking, and model selection.
"""

import logging
import os
import time
from typing import Optional

logger = logging.getLogger(__name__)

try:
    import openai
    from openai import OpenAI, RateLimitError
    _OPENAI_AVAILABLE = True
except ImportError:
    _OPENAI_AVAILABLE = False
    RateLimitError = Exception  # type: ignore


# ---- Pricing table (USD per 1K tokens, combined input+output average) ----
MODEL_PRICING: dict = {
    "gpt-4o": 0.005,
    "gpt-4o-mini": 0.00015,
    "gpt-4-turbo": 0.01,
    "gpt-4": 0.03,
    "gpt-3.5-turbo": 0.0005,
    "text-embedding-3-small": 0.00002,
    "text-embedding-3-large": 0.00013,
    "text-embedding-ada-002": 0.0001,
}

MAX_RETRIES = 5
BASE_DELAY = 1.0  # seconds


class OpenAIFallback:
    """
    Production-grade OpenAI client with:
    - Exponential back-off on RateLimitError (up to MAX_RETRIES attempts)
    - Per-request and cumulative cost tracking
    - Helper methods for chat completion and embeddings
    """

    def __init__(
        self,
        api_key: Optional[str] = None,
        default_model: str = "gpt-4o-mini",
    ) -> None:
        if not _OPENAI_AVAILABLE:
            raise RuntimeError("openai package not installed — pip install openai")

        self._client = OpenAI(
            api_key=api_key or os.environ["OPENAI_API_KEY"]
        )
        self.default_model = default_model
        self.total_tokens_used: int = 0
        self.total_cost_usd: float = 0.0

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _cost_for_tokens(self, model: str, tokens: int) -> float:
        rate = MODEL_PRICING.get(model, 0.005)
        return tokens / 1000.0 * rate

    def _call_with_retry(self, fn, *args, **kwargs):
        """Execute fn(*args, **kwargs) with exponential back-off."""
        delay = BASE_DELAY
        for attempt in range(1, MAX_RETRIES + 1):
            try:
                return fn(*args, **kwargs)
            except RateLimitError as exc:
                if attempt == MAX_RETRIES:
                    logger.error("Rate limit — max retries exhausted")
                    raise
                logger.warning(
                    "Rate limit (attempt %d/%d) — sleeping %.1fs: %s",
                    attempt,
                    MAX_RETRIES,
                    delay,
                    exc,
                )
                time.sleep(delay)
                delay = min(delay * 2, 60.0)
            except Exception:
                raise

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def complete(
        self,
        prompt: str,
        system: str = "",
        model: str = "gpt-4o-mini",
        max_tokens: int = 1000,
        temperature: float = 0.2,
    ) -> str:
        """Chat completion with automatic retry and cost tracking."""
        messages = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        def _do():
            return self._client.chat.completions.create(
                model=model,
                messages=messages,
                max_tokens=max_tokens,
                temperature=temperature,
            )

        response = self._call_with_retry(_do)
        usage = response.usage
        tokens = usage.total_tokens if usage else 0
        cost = self._cost_for_tokens(model, tokens)
        self.total_tokens_used += tokens
        self.total_cost_usd += cost
        logger.debug(
            "complete — model=%s tokens=%d cost=$%.6f", model, tokens, cost
        )
        return response.choices[0].message.content or ""

    def embed(self, text: str, model: str = "text-embedding-3-small") -> list:
        """Return a float embedding vector for a single text string."""
        def _do():
            return self._client.embeddings.create(input=[text], model=model)

        response = self._call_with_retry(_do)
        usage = response.usage
        tokens = usage.total_tokens if usage else 0
        cost = self._cost_for_tokens(model, tokens)
        self.total_tokens_used += tokens
        self.total_cost_usd += cost
        return response.data[0].embedding

    def embed_batch(
        self, texts: list, model: str = "text-embedding-3-small"
    ) -> list:
        """Return embedding vectors for a batch of text strings."""
        if not texts:
            return []

        def _do():
            return self._client.embeddings.create(input=texts, model=model)

        response = self._call_with_retry(_do)
        usage = response.usage
        tokens = usage.total_tokens if usage else 0
        cost = self._cost_for_tokens(model, tokens)
        self.total_tokens_used += tokens
        self.total_cost_usd += cost
        # Sort by index to guarantee order matches input
        sorted_data = sorted(response.data, key=lambda d: d.index)
        return [d.embedding for d in sorted_data]

    # ------------------------------------------------------------------
    # Cost reporting
    # ------------------------------------------------------------------

    def cost_summary(self) -> dict:
        return {
            "total_tokens_used": self.total_tokens_used,
            "total_cost_usd": round(self.total_cost_usd, 6),
        }

    def reset_cost_counters(self) -> None:
        self.total_tokens_used = 0
        self.total_cost_usd = 0.0


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    client = OpenAIFallback()
    result = client.complete(
        prompt="Summarise the MITRE ATT&CK technique T1055 in one sentence.",
        system="You are a concise cybersecurity expert.",
    )
    print(result)
    print(client.cost_summary())
