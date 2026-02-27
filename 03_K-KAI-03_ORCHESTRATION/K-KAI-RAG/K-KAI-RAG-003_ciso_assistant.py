"""
K-KAI-RAG-003: CISO AI assistant answering security policy queries via RAG.
Uses OpenAI GPT-4o with context retrieved from pgvector.
Returns structured response: answer, sources (OSCAL control IDs), confidence.
"""

import logging
import os
import re
from typing import Any

from openai import AsyncOpenAI

from K_KAI_RAG_002_oscal_embeddings import OSCALEmbedder

logger = logging.getLogger("kai.rag.ciso")

_SYSTEM_PROMPT = """You are CISO-AI, a senior security advisor for the Kubric platform.
You answer security policy and compliance questions strictly grounded in the retrieved
context from OSCAL documents.

FRAMEWORKS YOU UNDERSTAND:
- SOC 2 Type II (Trust Services Criteria)
- ISO 27001:2022 (Information Security Management)
- NIST SP 800-53 Rev 5 (Security and Privacy Controls)
- PCI DSS v4.0 (Payment Card Industry Data Security Standard)
- NIST Cybersecurity Framework 2.0

RULES:
1. Base your answer ONLY on the retrieved OSCAL/policy context provided.
2. Cite specific control IDs (e.g., AC-2, CC6.1, A.9.2.1) when relevant.
3. If the context does not contain enough information, say so explicitly.
4. Never fabricate control requirements; cite the source.
5. Express confidence as a float between 0.0 and 1.0.
6. Structure your final output as JSON with keys: answer, sources, confidence.
"""


class CISOAssistant:
    """
    CISO AI assistant with RAG over OSCAL embeddings.
    Retrieves relevant policy context before calling GPT-4o.
    """

    MODEL = "gpt-4o"
    TOP_K = 5

    def __init__(self) -> None:
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            raise RuntimeError("OPENAI_API_KEY environment variable not set")
        self._openai = AsyncOpenAI(api_key=api_key)

    # ------------------------------------------------------------------
    # Main entry point
    # ------------------------------------------------------------------
    async def answer(
        self,
        question: str,
        tenant_id: str,
        top_k: int | None = None,
    ) -> dict:
        """
        Answer a security policy question using RAG.

        Returns::
            {
                "answer": str,
                "sources": list[str],   # OSCAL control IDs
                "confidence": float,
                "retrieved_docs": list[str],
            }
        """
        k = top_k or self.TOP_K
        embedder = OSCALEmbedder(tenant_id=tenant_id)
        docs = await embedder.search_similar(question, tenant_id=tenant_id, top_k=k)

        if not docs:
            logger.warning("No OSCAL docs found for tenant=%s query=%r", tenant_id, question[:80])
            return {
                "answer": "No relevant policy documents found for your query.",
                "sources": [],
                "confidence": 0.0,
                "retrieved_docs": [],
            }

        # Build context block
        context_parts: list[str] = []
        for i, doc in enumerate(docs, 1):
            snippet = doc["content"][:600]
            sim = doc["similarity"]
            context_parts.append(f"[DOC {i} | sim={sim:.3f} | id={doc['doc_id']}]\n{snippet}")
        context_text = "\n\n".join(context_parts)

        user_message = (
            f"RETRIEVED CONTEXT:\n{context_text}\n\n"
            f"QUESTION: {question}\n\n"
            "Respond as JSON with keys: answer (string), sources (list of control IDs), "
            "confidence (float 0.0–1.0)."
        )

        # Call GPT-4o
        response = await self._openai.chat.completions.create(
            model=self.MODEL,
            messages=[
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": user_message},
            ],
            response_format={"type": "json_object"},
            temperature=0.1,
            max_tokens=1500,
        )

        raw = response.choices[0].message.content or "{}"
        parsed = self._parse_response(raw, docs)
        parsed["retrieved_docs"] = [d["doc_id"] for d in docs]
        return parsed

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------
    def _parse_response(self, raw: str, docs: list[dict]) -> dict:
        """Parse the GPT-4o JSON response, falling back gracefully on error."""
        import json as _json
        try:
            data = _json.loads(raw)
            return {
                "answer": str(data.get("answer", "")),
                "sources": list(data.get("sources", [])),
                "confidence": float(data.get("confidence", 0.5)),
            }
        except (_json.JSONDecodeError, ValueError, TypeError) as exc:
            logger.warning("Failed to parse GPT-4o JSON response: %s", exc)
            # Extract control IDs heuristically from raw text
            control_ids = re.findall(
                r"\b(?:[A-Z]{2,5}-\d+(?:\.\d+)?|CC\d\.\d|A\.\d+\.\d+\.\d+)\b",
                raw,
            )
            return {
                "answer": raw,
                "sources": list(set(control_ids)),
                "confidence": 0.3,
            }
