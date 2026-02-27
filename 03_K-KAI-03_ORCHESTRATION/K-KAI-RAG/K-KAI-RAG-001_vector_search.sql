-- K-KAI-RAG-001: pgvector DDL for KAI semantic search
-- Stores embeddings from OpenAI, Cohere, or other models.
-- HNSW index for fast cosine similarity search.
-- Includes kai_search_similar() plpgsql function.

-- ---------------------------------------------------------------------------
-- Prerequisites
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS vector;

-- ---------------------------------------------------------------------------
-- Table: kai_embeddings
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kai_embeddings (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id   TEXT        NOT NULL,
    doc_id      TEXT        NOT NULL,
    doc_type    TEXT        NOT NULL DEFAULT 'oscal'
                    CHECK (doc_type IN (
                        'oscal',
                        'policy',
                        'runbook',
                        'threat_intel',
                        'cve'
                    )),
    content     TEXT        NOT NULL,
    embedding   vector(1536) NOT NULL,
    model       TEXT        NOT NULL DEFAULT 'openai-text-embedding-3-small',
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT  kai_embeddings_tenant_doc_model_uq
        UNIQUE (tenant_id, doc_id, model)
);

COMMENT ON TABLE kai_embeddings IS
    'Semantic embedding store for KAI RAG. '
    'Stores vectors produced by OpenAI text-embedding-3-small (1536-d) or '
    'Cohere embed-english-v3 (stored as 1536-d via zero-padding if needed). '
    'Uses pgvector HNSW for sub-millisecond ANN search.';

COMMENT ON COLUMN kai_embeddings.doc_type IS
    'OSCAL control catalog, security policy, SOP runbook, threat intel report, or CVE advisory.';

COMMENT ON COLUMN kai_embeddings.model IS
    'Embedding model identifier, e.g. openai-text-embedding-3-small or cohere-v3.';

COMMENT ON COLUMN kai_embeddings.metadata IS
    'Arbitrary JSON metadata: control_id, framework, severity, source_url, etc.';

-- ---------------------------------------------------------------------------
-- HNSW index for cosine similarity (fastest for retrieval workloads)
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_kai_embeddings_hnsw
    ON kai_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

COMMENT ON INDEX idx_kai_embeddings_hnsw IS
    'HNSW index on embedding column using cosine distance. '
    'm=16 and ef_construction=64 balances index build time with recall@10.';

-- ---------------------------------------------------------------------------
-- Supporting indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_kai_embeddings_tenant
    ON kai_embeddings (tenant_id);

CREATE INDEX IF NOT EXISTS idx_kai_embeddings_doc_type
    ON kai_embeddings (tenant_id, doc_type);

CREATE INDEX IF NOT EXISTS idx_kai_embeddings_model
    ON kai_embeddings (model);

CREATE INDEX IF NOT EXISTS idx_kai_embeddings_created_at
    ON kai_embeddings (created_at DESC);

-- ---------------------------------------------------------------------------
-- Function: kai_search_similar
-- Returns the k most similar documents to query_embedding for a given tenant.
-- Optionally filtered by doc_type.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION kai_search_similar(
    query_embedding     vector,
    tenant              TEXT,
    k                   INT         DEFAULT 5,
    doc_type_filter     TEXT        DEFAULT NULL
)
RETURNS TABLE (
    doc_id      TEXT,
    content     TEXT,
    similarity  FLOAT,
    metadata    JSONB
)
LANGUAGE sql
STABLE
PARALLEL SAFE
AS $$
    SELECT
        e.doc_id,
        e.content,
        1 - (e.embedding <=> query_embedding)   AS similarity,
        e.metadata
    FROM
        kai_embeddings AS e
    WHERE
        e.tenant_id = tenant
        AND (doc_type_filter IS NULL OR e.doc_type = doc_type_filter)
    ORDER BY
        e.embedding <=> query_embedding   -- ascending distance = descending similarity
    LIMIT
        k;
$$;

COMMENT ON FUNCTION kai_search_similar(vector, text, int, text) IS
    'Returns the k most similar embeddings for a given tenant using cosine similarity. '
    'query_embedding must be the same dimensionality as the stored vectors (1536). '
    'similarity = 1 - cosine_distance; ranges from -1 (opposite) to 1 (identical). '
    'Filter by doc_type (oscal | policy | runbook | threat_intel | cve) when provided.';

-- ---------------------------------------------------------------------------
-- Example usage:
--
-- SELECT * FROM kai_search_similar(
--     '[0.12, -0.34, ...]'::vector,   -- 1536-d query embedding
--     'acme-corp',                     -- tenant_id
--     5,                               -- top-k
--     'oscal'                          -- doc_type filter (optional)
-- );
-- ---------------------------------------------------------------------------
