"""
K-KAI-RAG-004: Cohere embed-english-v3.0 embedding alternative.
Used in air-gapped or cost-sensitive deployments (COHERE_API_KEY env).
Stores in kai_embeddings with model='cohere-v3'.
Max 96 texts per call, 10 calls/min.
"""

import asyncio
import json
import logging
import os
import time
from typing import Literal

import cohere
import asyncpg

logger = logging.getLogger("kai.rag.cohere")

_COHERE_MODEL = "embed-english-v3.0"
_COHERE_DIMENSIONS = 1024          # embed-english-v3 output size
_MAX_TEXTS_PER_CALL = 96
_MAX_CALLS_PER_MIN = 10
_MIN_CALL_INTERVAL = 60.0 / _MAX_CALLS_PER_MIN   # 6 seconds between calls


class CohereEmbedder:
    """
    Cohere-based embedding client compatible with the kai_embeddings schema.
    Writes model='cohere-v3' so rows are distinguishable from OpenAI embeddings.
    """

    MODEL_TAG = "cohere-v3"

    def __init__(self, tenant_id: str = "global") -> None:
        api_key = os.environ.get("COHERE_API_KEY")
        if not api_key:
            raise RuntimeError("COHERE_API_KEY environment variable not set")
        self._co = cohere.AsyncClientV2(api_key=api_key)
        self._tenant_id = tenant_id
        self._pool: asyncpg.Pool | None = None
        self._last_call_time: float = 0.0

    # ------------------------------------------------------------------
    # Rate limiting helper
    # ------------------------------------------------------------------
    async def _rate_limit(self) -> None:
        """Enforce max 10 calls/min by sleeping when needed."""
        elapsed = time.monotonic() - self._last_call_time
        if elapsed < _MIN_CALL_INTERVAL:
            wait = _MIN_CALL_INTERVAL - elapsed
            logger.debug("Cohere rate-limit: sleeping %.2fs", wait)
            await asyncio.sleep(wait)
        self._last_call_time = time.monotonic()

    # ------------------------------------------------------------------
    # DB pool
    # ------------------------------------------------------------------
    async def _ensure_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            dsn = os.environ.get("DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai")
            self._pool = await asyncpg.create_pool(dsn, min_size=1, max_size=5)
        return self._pool

    # ------------------------------------------------------------------
    # Embedding
    # ------------------------------------------------------------------
    async def embed(
        self,
        texts: list[str],
        input_type: Literal[
            "search_document", "search_query", "classification", "clustering"
        ] = "search_document",
    ) -> list[list[float]]:
        """
        Embed a list of texts using Cohere embed-english-v3.0.
        Automatically batches into chunks of 96 with rate limiting.
        """
        if not texts:
            return []

        all_embeddings: list[list[float]] = []
        for i in range(0, len(texts), _MAX_TEXTS_PER_CALL):
            batch = texts[i : i + _MAX_TEXTS_PER_CALL]
            await self._rate_limit()
            response = await self._co.embed(
                texts=batch,
                model=_COHERE_MODEL,
                input_type=input_type,
                embedding_types=["float"],
            )
            batch_embeddings = response.embeddings.float_
            all_embeddings.extend(batch_embeddings)
            logger.debug(
                "Cohere embed batch %d-%d returned %d vectors",
                i,
                i + len(batch),
                len(batch_embeddings),
            )

        return all_embeddings

    async def embed_query(self, query: str) -> list[float]:
        """Embed a single query string (uses search_query input type)."""
        results = await self.embed([query], input_type="search_query")
        if not results:
            raise RuntimeError("Cohere returned empty embeddings for query")
        return results[0]

    # ------------------------------------------------------------------
    # pgvector storage
    # ------------------------------------------------------------------
    async def store_embedding(
        self,
        doc_id: str,
        content: str,
        embedding: list[float],
        doc_type: str = "policy",
        metadata: dict | None = None,
    ) -> None:
        """Upsert embedding into kai_embeddings with model='cohere-v3'."""
        pool = await self._ensure_pool()
        vec_str = "[" + ",".join(str(v) for v in embedding) + "]"
        async with pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO kai_embeddings
                    (tenant_id, doc_id, doc_type, content, embedding, model, metadata)
                VALUES ($1, $2, $3, $4, $5::vector, $6, $7)
                ON CONFLICT (tenant_id, doc_id, model)
                DO UPDATE SET content=EXCLUDED.content,
                              embedding=EXCLUDED.embedding,
                              metadata=EXCLUDED.metadata,
                              created_at=NOW()
                """,
                self._tenant_id,
                doc_id,
                doc_type,
                content,
                vec_str,
                self.MODEL_TAG,
                json.dumps(metadata or {}),
            )
        logger.info(
            "Cohere embedding stored tenant=%s doc_id=%s model=%s",
            self._tenant_id,
            doc_id,
            self.MODEL_TAG,
        )

    # ------------------------------------------------------------------
    # Similarity search
    # ------------------------------------------------------------------
    async def search_similar(
        self,
        query: str,
        tenant_id: str | None = None,
        top_k: int = 5,
    ) -> list[dict]:
        """Return top_k similar docs using cosine similarity via pgvector."""
        tenant = tenant_id or self._tenant_id
        query_embedding = await self.embed_query(query)
        vec_str = "[" + ",".join(str(v) for v in query_embedding) + "]"
        pool = await self._ensure_pool()

        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT doc_id, content, metadata,
                       1 - (embedding <=> $1::vector) AS similarity
                FROM kai_embeddings
                WHERE tenant_id = $2 AND model = $3
                ORDER BY embedding <=> $1::vector
                LIMIT $4
                """,
                vec_str,
                tenant,
                self.MODEL_TAG,
                top_k,
            )
        return [
            {
                "doc_id": row["doc_id"],
                "content": row["content"],
                "similarity": float(row["similarity"]),
                "metadata": row["metadata"],
            }
            for row in rows
        ]

    # ------------------------------------------------------------------
    # Batch embed and store
    # ------------------------------------------------------------------
    async def embed_and_store_batch(
        self,
        documents: list[dict],
        input_type: str = "search_document",
    ) -> None:
        """
        Embed and store a batch of documents.
        Each document dict must have: doc_id, content.
        Optional: doc_type, metadata.
        """
        texts = [d["content"] for d in documents]
        embeddings = await self.embed(texts, input_type=input_type)  # type: ignore[arg-type]
        for doc, emb in zip(documents, embeddings):
            await self.store_embedding(
                doc_id=doc["doc_id"],
                content=doc["content"],
                embedding=emb,
                doc_type=doc.get("doc_type", "policy"),
                metadata=doc.get("metadata"),
            )
        logger.info(
            "Cohere batch embed+store complete: %d documents for tenant=%s",
            len(documents),
            self._tenant_id,
        )
