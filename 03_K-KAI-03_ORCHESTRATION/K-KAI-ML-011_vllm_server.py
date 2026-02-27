"""
K-KAI-ML-011_vllm_server.py
vLLM local inference server launcher and OpenAI-compatible client.
"""

import logging
import os
import subprocess
import time
from typing import Optional

logger = logging.getLogger(__name__)

try:
    from openai import OpenAI
    _OPENAI_AVAILABLE = True
except ImportError:
    _OPENAI_AVAILABLE = False

DEFAULT_MODEL = "mistralai/Mistral-7B-Instruct-v0.3"
VLLM_BASE_URL = os.environ.get("VLLM_BASE_URL", "http://localhost:8000/v1")
VLLM_API_KEY = os.environ.get("VLLM_API_KEY", "EMPTY")  # vLLM accepts any key


class VLLMServer:
    """
    Launcher and manager for a local vLLM inference process.

    The server is started as a background subprocess and exposes an
    OpenAI-compatible REST API at localhost:8000/v1.
    """

    def start(
        self,
        model_id: str = DEFAULT_MODEL,
        port: int = 8000,
        host: str = "0.0.0.0",
        gpu_memory_utilization: float = 0.9,
        max_model_len: int = 4096,
        tensor_parallel_size: int = 1,
        extra_args: Optional[list] = None,
    ) -> subprocess.Popen:
        """
        Launch the vLLM server process.

        Returns the Popen handle so the caller can manage the lifecycle.
        """
        cmd = [
            "python", "-m", "vllm.entrypoints.openai.api_server",
            "--model", model_id,
            "--host", host,
            "--port", str(port),
            "--gpu-memory-utilization", str(gpu_memory_utilization),
            "--max-model-len", str(max_model_len),
            "--tensor-parallel-size", str(tensor_parallel_size),
        ]
        if extra_args:
            cmd.extend(extra_args)

        logger.info("Starting vLLM server — model=%s port=%d", model_id, port)
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        # Give the server up to 120 seconds to start
        client = VLLMClient(base_url=f"http://{host}:{port}/v1")
        for attempt in range(24):
            time.sleep(5)
            if proc.poll() is not None:
                raise RuntimeError(f"vLLM process exited early — code={proc.returncode}")
            if client.health_check():
                logger.info("vLLM server ready after %d seconds", (attempt + 1) * 5)
                return proc
        raise TimeoutError("vLLM server did not become ready within 120 seconds")

    def stop(self, proc: subprocess.Popen) -> None:
        """Terminate the vLLM server process."""
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=15)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
            logger.info("vLLM server stopped")


class VLLMClient:
    """
    OpenAI-SDK client that talks to a vLLM server.

    vLLM exposes a fully OpenAI-compatible API, so we reuse the openai
    Python library pointed at the local server.
    """

    def __init__(
        self,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        default_model: str = DEFAULT_MODEL,
    ) -> None:
        if not _OPENAI_AVAILABLE:
            raise RuntimeError("openai not installed — pip install openai")
        self._base_url = base_url or VLLM_BASE_URL
        self._api_key = api_key or VLLM_API_KEY
        self._default_model = default_model
        self._client = OpenAI(
            base_url=self._base_url,
            api_key=self._api_key,
        )

    def health_check(self) -> bool:
        """Return True if the server is reachable and responding."""
        import urllib.request
        import urllib.error
        try:
            # vLLM exposes /health endpoint
            health_url = self._base_url.replace("/v1", "") + "/health"
            with urllib.request.urlopen(health_url, timeout=3) as resp:
                return resp.status == 200
        except Exception:
            return False

    def complete(
        self,
        prompt: str,
        model: Optional[str] = None,
        max_tokens: int = 512,
        temperature: float = 0.1,
        system: str = "",
    ) -> str:
        """Single-prompt completion via the OpenAI chat completions API."""
        messages = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        response = self._client.chat.completions.create(
            model=model or self._default_model,
            messages=messages,
            max_tokens=max_tokens,
            temperature=temperature,
        )
        return response.choices[0].message.content or ""

    def complete_batch(
        self,
        prompts: list,
        model: Optional[str] = None,
        max_tokens: int = 512,
        temperature: float = 0.1,
    ) -> list:
        """
        Complete a batch of prompts.

        vLLM does not have a native batch endpoint in the OpenAI format,
        so we submit each request sequentially (async optimisation can be
        added by the caller as needed).
        """
        results = []
        for prompt in prompts:
            try:
                text = self.complete(
                    prompt=prompt,
                    model=model,
                    max_tokens=max_tokens,
                    temperature=temperature,
                )
            except Exception as exc:
                logger.error("batch complete failed for prompt: %s", exc)
                text = ""
            results.append(text)
        return results

    def list_models(self) -> list:
        """List models available on the vLLM server."""
        try:
            response = self._client.models.list()
            return [m.id for m in response.data]
        except Exception as exc:
            logger.error("list_models failed: %s", exc)
            return []


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    client = VLLMClient()
    if client.health_check():
        answer = client.complete("Explain CVE-2021-44228 in one sentence.")
        print(answer)
    else:
        print("vLLM server not reachable — start it with VLLMServer().start()")
