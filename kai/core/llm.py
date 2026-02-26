"""
LLM inference client for KAI.

Priority:
  1. Ollama  (local — zero cost, data stays on-prem)
  2. vLLM    (local R740 GPU node)
  3. OpenAI  (cloud fallback — only if KUBRIC_OPENAI_API_KEY set)
  4. Anthropic (cloud fallback — only if KUBRIC_ANTHROPIC_API_KEY set)

All methods return plain-text strings.  Structured JSON extraction is done
by callers using orjson.  This module never sends PII or customer raw telemetry
to cloud providers — callers must strip sensitive fields before passing context.
"""

from __future__ import annotations

import json

import httpx
import structlog

from kai.config import settings

log = structlog.get_logger(__name__)

_DEFAULT_OLLAMA_MODEL = "llama3.2"
_DEFAULT_VLLM_MODEL = "meta-llama/Llama-3.1-8B-Instruct"
_TIMEOUT = 30.0  # seconds


async def complete(
    prompt: str,
    system: str = "You are a cyber-security analyst assistant.",
    model: str | None = None,
    max_tokens: int = 512,
) -> str:
    """
    Run a completion against the best available LLM.

    Returns the response text, or a placeholder string if all backends fail.
    """
    # 1. Ollama (local)
    result = await _ollama(prompt, system=system, model=model or _DEFAULT_OLLAMA_MODEL, max_tokens=max_tokens)
    if result:
        return result

    # 2. vLLM (local GPU node)
    result = await _vllm(prompt, system=system, model=model or _DEFAULT_VLLM_MODEL, max_tokens=max_tokens)
    if result:
        return result

    # 3. OpenAI cloud fallback
    if settings.openai_api_key:
        result = await _openai(prompt, system=system, max_tokens=max_tokens)
        if result:
            return result

    # 4. Anthropic cloud fallback
    if settings.anthropic_api_key:
        result = await _anthropic(prompt, system=system, max_tokens=max_tokens)
        if result:
            return result

    log.warning("llm.all_backends_failed")
    return "[LLM unavailable — manual review required]"


async def _ollama(prompt: str, *, system: str, model: str, max_tokens: int) -> str:
    url = f"{settings.ollama_url.rstrip('/')}/api/generate"
    payload = {
        "model": model,
        "system": system,
        "prompt": prompt,
        "stream": False,
        "options": {"num_predict": max_tokens},
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
            resp = await client.post(url, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return str(data.get("response", "")).strip()
    except Exception as exc:
        log.debug("llm.ollama_failed", error=str(exc))
        return ""


async def _vllm(prompt: str, *, system: str, model: str, max_tokens: int) -> str:
    url = f"{settings.vllm_url.rstrip('/')}/v1/chat/completions"
    payload = {
        "model": model,
        "messages": [{"role": "system", "content": system}, {"role": "user", "content": prompt}],
        "max_tokens": max_tokens,
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
            resp = await client.post(url, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return str(data["choices"][0]["message"]["content"]).strip()
    except Exception as exc:
        log.debug("llm.vllm_failed", error=str(exc))
        return ""


async def _openai(prompt: str, *, system: str, max_tokens: int) -> str:
    url = "https://api.openai.com/v1/chat/completions"
    headers = {"Authorization": f"Bearer {settings.openai_api_key}"}
    payload = {
        "model": "gpt-4o",
        "messages": [{"role": "system", "content": system}, {"role": "user", "content": prompt}],
        "max_tokens": max_tokens,
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
            resp = await client.post(url, headers=headers, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return str(data["choices"][0]["message"]["content"]).strip()
    except Exception as exc:
        log.debug("llm.openai_failed", error=str(exc))
        return ""


async def _anthropic(prompt: str, *, system: str, max_tokens: int) -> str:
    url = "https://api.anthropic.com/v1/messages"
    headers = {
        "x-api-key": settings.anthropic_api_key,
        "anthropic-version": "2023-06-01",
    }
    payload = {
        "model": "claude-3-5-sonnet-20241022",
        "system": system,
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": max_tokens,
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
            resp = await client.post(url, headers=headers, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return str(data["content"][0]["text"]).strip()
    except Exception as exc:
        log.debug("llm.anthropic_failed", error=str(exc))
        return ""


async def complete_json(
    prompt: str,
    system: str = "You are a cyber-security analyst assistant. Respond ONLY with valid JSON.",
    model: str | None = None,
    max_tokens: int = 512,
) -> dict:  # type: ignore[type-arg]
    """Run a completion expected to return a JSON object. Falls back to empty dict."""
    raw = await complete(prompt, system=system, model=model, max_tokens=max_tokens)
    # strip markdown code fences if present
    raw = raw.strip()
    if raw.startswith("```"):
        lines = raw.splitlines()
        raw = "\n".join(lines[1:-1]) if len(lines) > 2 else raw
    try:
        return json.loads(raw)  # type: ignore[no-any-return]
    except Exception:
        return {"raw": raw}
