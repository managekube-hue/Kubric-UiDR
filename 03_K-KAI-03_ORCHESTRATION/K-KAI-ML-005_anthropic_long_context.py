"""
K-KAI-ML-005_anthropic_long_context.py
Anthropic long-context client for analyzing large security reports (up to 200K tokens).
Streams responses for documents > 50 KB.
"""

import json
import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)

try:
    import anthropic
    from anthropic import Anthropic
    _ANTHROPIC_AVAILABLE = True
except ImportError:
    _ANTHROPIC_AVAILABLE = False
    Anthropic = None  # type: ignore

STREAM_THRESHOLD_BYTES = 50 * 1024  # 50 KB
DEFAULT_MODEL = "claude-3-5-sonnet-20241022"
MAX_TOKENS = 4096


class AnthropicLongContext:
    """
    Production Anthropic client tuned for long security documents.

    - Streams responses for documents larger than 50 KB to avoid gateway
      time-outs and improve first-token latency.
    - Supports structured extraction via JSON-mode system prompts.
    """

    def __init__(self, api_key: Optional[str] = None) -> None:
        if not _ANTHROPIC_AVAILABLE:
            raise RuntimeError(
                "anthropic package not installed — pip install anthropic"
            )
        self._client = Anthropic(
            api_key=api_key or os.environ["ANTHROPIC_API_KEY"]
        )

    # ------------------------------------------------------------------
    # Core method
    # ------------------------------------------------------------------

    def analyze(
        self,
        document: str,
        task: str,
        model: str = DEFAULT_MODEL,
        max_tokens: int = MAX_TOKENS,
    ) -> str:
        """
        Analyze an arbitrary document with a task instruction.

        Automatically streams when document > STREAM_THRESHOLD_BYTES.
        """
        should_stream = len(document.encode()) > STREAM_THRESHOLD_BYTES
        messages = [
            {
                "role": "user",
                "content": f"<document>\n{document}\n</document>\n\n{task}",
            }
        ]

        if should_stream:
            logger.debug(
                "Document exceeds %d bytes — using streaming", STREAM_THRESHOLD_BYTES
            )
            return self._stream(model, messages, max_tokens)
        else:
            return self._complete(model, messages, max_tokens)

    # ------------------------------------------------------------------
    # Structured helpers
    # ------------------------------------------------------------------

    def analyze_incident(self, incident: dict, model: str = DEFAULT_MODEL) -> str:
        """
        Structured prompt for OCSF incident analysis.

        Expects incident keys: incident_id, severity, title, raw_events,
        affected_assets, tenant_id.
        """
        system = (
            "You are a senior security analyst. Provide a structured incident "
            "analysis with: root_cause, attack_vector, affected_assets, "
            "recommended_immediate_actions, and escalation_required (yes/no)."
        )
        doc = json.dumps(incident, indent=2, default=str)
        task = (
            "Analyse the above security incident JSON. "
            "Structure your response as numbered sections."
        )
        messages = [
            {"role": "user", "content": f"{system}\n\n<incident>\n{doc}\n</incident>\n\n{task}"}
        ]
        should_stream = len(doc.encode()) > STREAM_THRESHOLD_BYTES
        if should_stream:
            return self._stream(model, messages)
        return self._complete(model, messages)

    def summarize_threat_report(
        self, pdf_text: str, model: str = DEFAULT_MODEL
    ) -> dict:
        """
        Extract structured IOC/TTP/actor data from a threat intelligence report.

        Returns dict with keys:
          iocs, ttps, threat_actors, recommended_actions
        """
        system = (
            "You are a CTI analyst. Extract threat intelligence from the report "
            "and return ONLY valid JSON with keys: "
            "iocs (list of strings), ttps (list of MITRE ATT&CK IDs), "
            "threat_actors (list of names), recommended_actions (list of strings). "
            "No markdown, no explanation, pure JSON."
        )
        task = "Extract the threat intelligence as instructed."
        messages = [
            {
                "role": "user",
                "content": (
                    f"{system}\n\n<report>\n{pdf_text}\n</report>\n\n{task}"
                ),
            }
        ]
        should_stream = len(pdf_text.encode()) > STREAM_THRESHOLD_BYTES
        if should_stream:
            raw = self._stream(model, messages)
        else:
            raw = self._complete(model, messages)

        # Strip fenced code blocks if model included them
        cleaned = raw.strip().lstrip("```json").lstrip("```").rstrip("```").strip()
        try:
            return json.loads(cleaned)
        except json.JSONDecodeError:
            logger.warning("summarize_threat_report — JSON parse failed, returning raw")
            return {
                "iocs": [],
                "ttps": [],
                "threat_actors": [],
                "recommended_actions": [raw],
            }

    # ------------------------------------------------------------------
    # Low-level transport
    # ------------------------------------------------------------------

    def _complete(
        self,
        model: str,
        messages: list,
        max_tokens: int = MAX_TOKENS,
    ) -> str:
        response = self._client.messages.create(
            model=model,
            max_tokens=max_tokens,
            messages=messages,
        )
        return response.content[0].text

    def _stream(
        self,
        model: str,
        messages: list,
        max_tokens: int = MAX_TOKENS,
    ) -> str:
        """Collect a streaming response into a single string."""
        collected: list = []
        with self._client.messages.stream(
            model=model,
            max_tokens=max_tokens,
            messages=messages,
        ) as stream:
            for text in stream.text_stream:
                collected.append(text)
        return "".join(collected)


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    client = AnthropicLongContext()
    sample_incident = {
        "incident_id": "INC-001",
        "severity": 5,
        "title": "Suspected APT lateral movement",
        "raw_events": ["logon from 192.168.1.55", "mimikatz detected on DC01"],
        "affected_assets": ["DC01", "WEB03"],
        "tenant_id": "acme",
    }
    result = client.analyze_incident(sample_incident)
    print(result[:500])
