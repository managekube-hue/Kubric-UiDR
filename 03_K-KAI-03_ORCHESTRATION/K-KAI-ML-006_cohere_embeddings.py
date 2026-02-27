"""
K-KAI-ML-006_cohere_embeddings.py
Cohere Command R+ client for security use-case RAG.
"""

import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)

try:
    import cohere
    from cohere import Client as CohereClient
    _COHERE_AVAILABLE = True
except ImportError:
    _COHERE_AVAILABLE = False
    CohereClient = None  # type: ignore

EMBED_MODEL = "embed-english-v3.0"
RERANK_MODEL = "rerank-english-v3.0"
CHAT_MODEL = "command-r-plus-08-2024"


class CohereRAGClient:
    """
    Cohere-backed RAG helper for Kubric KAI.

    Provides:
    - Dense embedding generation (embed-english-v3.0)
    - Neural reranking for retrieved documents (rerank-english-v3.0)
    - Grounded chat with document citations (command-r-plus-08-2024)
    """

    def __init__(self, api_key: Optional[str] = None) -> None:
        if not _COHERE_AVAILABLE:
            raise RuntimeError(
                "cohere package not installed — pip install cohere"
            )
        self._client = CohereClient(
            api_key=api_key or os.environ["COHERE_API_KEY"]
        )

    # ------------------------------------------------------------------
    # Embedding
    # ------------------------------------------------------------------

    def embed(
        self,
        texts: list,
        input_type: str = "search_document",
    ) -> list:
        """
        Generate dense embeddings for a list of texts.

        input_type: "search_document" for indexing, "search_query" for queries.
        Returns list of float vectors.
        """
        if not texts:
            return []

        response = self._client.embed(
            texts=texts,
            model=EMBED_MODEL,
            input_type=input_type,
        )
        logger.debug(
            "Embedded %d texts — model=%s input_type=%s",
            len(texts),
            EMBED_MODEL,
            input_type,
        )
        return [list(v) for v in response.embeddings]

    # ------------------------------------------------------------------
    # Reranking
    # ------------------------------------------------------------------

    def rerank(
        self,
        query: str,
        documents: list,
        top_n: int = 5,
    ) -> list:
        """
        Rerank a list of document strings for relevance to query.

        Returns list of dicts: {index, document, relevance_score}
        sorted by relevance_score descending.
        """
        if not documents:
            return []

        response = self._client.rerank(
            query=query,
            documents=documents,
            top_n=min(top_n, len(documents)),
            model=RERANK_MODEL,
        )
        results = []
        for hit in response.results:
            results.append(
                {
                    "index": hit.index,
                    "document": documents[hit.index],
                    "relevance_score": hit.relevance_score,
                }
            )
        logger.debug(
            "Reranked %d docs → top %d — model=%s",
            len(documents),
            top_n,
            RERANK_MODEL,
        )
        return results

    # ------------------------------------------------------------------
    # Grounded chat
    # ------------------------------------------------------------------

    def chat_with_documents(
        self,
        message: str,
        documents: list,
        temperature: float = 0.2,
    ) -> str:
        """
        Chat with Command R+ using grounded document citations.

        documents: list of dicts with at least a "text" key (Cohere RAG format).
        Returns the assistant's response text with inline citations.
        """
        if not documents:
            # Fall back to plain chat if no documents supplied
            response = self._client.chat(
                message=message,
                model=CHAT_MODEL,
                temperature=temperature,
            )
            return response.text

        # Ensure each document has required "title" and "snippet" / "text" fields
        cohere_docs = []
        for i, doc in enumerate(documents):
            if isinstance(doc, str):
                cohere_docs.append({"title": f"Document {i}", "snippet": doc})
            elif isinstance(doc, dict):
                cohere_docs.append(
                    {
                        "title": doc.get("title", f"Document {i}"),
                        "snippet": doc.get("text", doc.get("snippet", "")),
                    }
                )

        response = self._client.chat(
            message=message,
            model=CHAT_MODEL,
            documents=cohere_docs,
            temperature=temperature,
        )

        text = response.text
        # Append citation summary if available
        if hasattr(response, "citations") and response.citations:
            citation_lines = []
            for c in response.citations:
                sources = ", ".join(
                    doc_src.get("title", "?")
                    for doc_src in getattr(c, "document_ids", [])
                )
                citation_lines.append(f"[{c.start}:{c.end}] → {sources}")
            if citation_lines:
                text += "\n\nCitations:\n" + "\n".join(citation_lines)

        logger.debug("chat_with_documents — model=%s docs=%d", CHAT_MODEL, len(documents))
        return text

    # ------------------------------------------------------------------
    # Security-specific helper
    # ------------------------------------------------------------------

    def query_ioc_knowledge_base(
        self,
        ioc: str,
        kb_documents: list,
        top_n: int = 5,
    ) -> str:
        """
        Full RAG pipeline: embed query → rerank → answer with citations.
        """
        reranked = self.rerank(ioc, kb_documents, top_n=top_n)
        if not reranked:
            return "No relevant documents found in the knowledge base."

        top_docs = [{"text": r["document"], "title": f"Rank {i+1}"} for i, r in enumerate(reranked)]
        return self.chat_with_documents(
            message=f"What do we know about this IOC or threat indicator: {ioc}?",
            documents=top_docs,
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    client = CohereRAGClient()
    vecs = client.embed(["Cobalt Strike beacon", "Mimikatz credential dump"], input_type="search_document")
    print(f"Embedding dimensions: {len(vecs[0])}")
