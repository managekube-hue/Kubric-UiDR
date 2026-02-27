"""
K-KAI-RAG-002: OSCAL document embedder using OpenAI text-embedding-3-small.
Stores vectors in PostgreSQL pgvector (kai_embeddings table).
"""

import json
import logging
import os
import uuid
from typing import Any

import asyncpg
from openai import AsyncOpenAI

logger = logging.getLogger("kai.rag.oscal")

_TABLE_DDL = """
CREATE TABLE IF NOT EXISTS kai_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    doc_id      TEXT NOT NULL,
    doc_type    TEXT NOT NULL DEFAULT 'oscal',
    content     TEXT NOT NULL,
    embedding   vector(1536) NOT NULL,
    model       TEXT NOT NULL DEFAULT 'openai-text-embedding-3-small',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, doc_id, model)
);
"""


class OSCALEmbedder:
    """
    Embeds OSCAL JSON documents using OpenAI text-embedding-3-small and
    stores resulting vectors in PostgreSQL with pgvector.
    """

    MODEL = "text-embedding-3-small"
    DIMENSIONS = 1536

    def __init__(self, tenant_id: str = "global") -> None:
        self._tenant_id = tenant_id
        api_key = os.environ.get("OPENAI_API_KEY")
        if not api_key:
            raise RuntimeError("OPENAI_API_KEY environment variable not set")
        self._openai = AsyncOpenAI(api_key=api_key)
        self._pool: asyncpg.Pool | None = None

    # ------------------------------------------------------------------
    # DB setup
    # ------------------------------------------------------------------
    async def _ensure_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            dsn = os.environ.get("DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai")
            self._pool = await asyncpg.create_pool(dsn, min_size=1, max_size=5)
            # Ensure pgvector extension and table exist
            async with self._pool.acquire() as conn:
                await conn.execute("CREATE EXTENSION IF NOT EXISTS vector;")
                await conn.execute(_TABLE_DDL)
        return self._pool

    # ------------------------------------------------------------------
    # Embedding
    # ------------------------------------------------------------------
    async def embed_document(self, oscal_json: dict) -> list[float]:
        """
        Flatten a partial OSCAL document to text and return its embedding.
        Concatenates catalog/profile title + control IDs + descriptions.
        """
        text_parts: list[str] = []

        # Catalog / profile metadata
        for key in ("catalog", "profile", "assessment-plan", "plan-of-action-and-milestones"):
            node = oscal_json.get(key, {})
            if node:
                meta = node.get("metadata", {})
                text_parts.append(meta.get("title", ""))
                for rev in meta.get("revisions", []):
                    text_parts.append(rev.get("title", ""))

        # Extract controls recursively
        def _extract_controls(node: Any) -> None:
            if isinstance(node, dict):
                if "id" in node and "title" in node:
                    text_parts.append(f"Control {node['id']}: {node.get('title','')}")
                    for part in node.get("parts", []):
                        text_parts.append(part.get("prose", ""))
                for v in node.values():
                    _extract_controls(v)
            elif isinstance(node, list):
                for item in node:
                    _extract_controls(item)

        _extract_controls(oscal_json)

        text = " ".join(t.strip() for t in text_parts if t.strip())[:8000]  # stay in context
        return await self._embed_text(text)

    async def _embed_text(self, text: str) -> list[float]:
        """Call the OpenAI Embeddings API and return the vector."""
        response = await self._openai.embeddings.create(
            input=text,
            model=self.MODEL,
            dimensions=self.DIMENSIONS,
        )
        return response.data[0].embedding

    # ------------------------------------------------------------------
    # Storage
    # ------------------------------------------------------------------
    async def store_embedding(
        self,
        doc_id: str,
        content: str,
        embedding: list[float],
        doc_type: str = "oscal",
        metadata: dict | None = None,
    ) -> None:
        """Upsert (tenant_id, doc_id, model) with the provided embedding."""
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
                self.MODEL,
                json.dumps(metadata or {}),
            )
        logger.info("Stored embedding tenant=%s doc_id=%s", self._tenant_id, doc_id)

    # ------------------------------------------------------------------
    # Search
    # ------------------------------------------------------------------
    async def search_similar(
        self,
        query: str,
        tenant_id: str | None = None,
        top_k: int = 5,
    ) -> list[dict]:
        """
        Return the top_k most similar documents to *query*
        for the specified *tenant_id* (defaults to self._tenant_id).
        """
        tenant = tenant_id or self._tenant_id
        query_embedding = await self._embed_text(query)
        vec_str = "[" + ",".join(str(v) for v in query_embedding) + "]"
        pool = await self._ensure_pool()

        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT doc_id, content, metadata,
                       1 - (embedding <=> $1::vector) AS similarity
                FROM kai_embeddings
                WHERE tenant_id = $2
                ORDER BY embedding <=> $1::vector
                LIMIT $3
                """,
                vec_str,
                tenant,
                top_k,
            )

        results = []
        for row in rows:
            results.append(
                {
                    "doc_id": row["doc_id"],
                    "content": row["content"],
                    "similarity": float(row["similarity"]),
                    "metadata": json.loads(row["metadata"]) if isinstance(row["metadata"], str) else row["metadata"],
                }
            )
        return results

    # ------------------------------------------------------------------
    # Convenience: embed + store in one call
    # ------------------------------------------------------------------
    async def embed_and_store(
        self,
        doc_id: str,
        oscal_json: dict,
        doc_type: str = "oscal",
        metadata: dict | None = None,
    ) -> list[float]:
        """Embed an OSCAL document and persist it. Returns the embedding vector."""
        embedding = await self.embed_document(oscal_json)
        # Use a summary snippet as *content* for retrieval display
        content = json.dumps(oscal_json, separators=(",", ":"))[:2000]
        await self.store_embedding(doc_id, content, embedding, doc_type=doc_type, metadata=metadata)
        return embedding
