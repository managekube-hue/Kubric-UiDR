# kai/Dockerfile — KAI Python AI orchestration service (L2)
# Multi-stage: builds wheel, installs into slim runtime image.

# ── Stage 1: builder ─────────────────────────────────────────────────────────
FROM python:3.12-slim AS builder
WORKDIR /app

# System deps for native extensions (psycopg2, cryptography, etc.)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    libffi-dev \
    libssl-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy and install Python package (kai/) in editable mode
COPY pyproject.toml setup.cfg* setup.py* ./
COPY kai/ ./kai/

# Install KAI with all layer2 extras, excluding heavy GPU-only packages
RUN pip install --upgrade pip && \
    pip install --no-cache-dir \
        ".[layer2]" \
    || pip install --no-cache-dir \
        fastapi>=0.115.0 \
        uvicorn[standard]>=0.32.0 \
        nats-py>=2.9.0 \
        orjson>=3.10.0 \
        httpx>=0.28.0 \
        pydantic>=2.9.0 \
        pydantic-settings>=2.6.0 \
        structlog>=24.4.0 \
        crewai>=0.80.0 \
        langchain>=0.3.7 \
        langchain-community>=0.3.7 \
        openai>=1.57.0 \
        anthropic>=0.40.0 \
        temporalio>=1.7.0 \
        asyncpg>=0.30.0 \
        clickhouse-connect>=0.8.0

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM python:3.12-slim AS final

RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq5 \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY --from=builder /app /app

# Create non-root user
RUN useradd -m -u 1000 kubric && chown -R kubric:kubric /app
USER kubric

EXPOSE 8101

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8101/healthz || exit 1

CMD ["uvicorn", "kai.api.main:app", "--host", "0.0.0.0", "--port", "8101", "--workers", "2"]
