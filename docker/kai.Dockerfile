# Kubric KAI — Python AI orchestration layer
# Multi-stage build: keeps the final image lean.
#
# Usage:
#   kai serve      → start FastAPI server
#   kai-worker     → start Temporal activity worker
#
# Build:
#   docker build -f docker/kai.Dockerfile -t kubric-kai:latest .

FROM python:3.11-slim AS builder

WORKDIR /build

# Install build deps
RUN pip install --no-cache-dir hatchling

# Copy only package manifest first for layer caching
COPY kai/pyproject.toml .
COPY kai/ ./kai/

# Install the package with Layer 2 extras
RUN pip install --no-cache-dir --prefix=/install ".[layer2]" -f .

# ─── final stage ──────────────────────────────────────────────────────────────
FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local

# Copy source
COPY kai/ ./kai/

# Non-root user
RUN useradd --no-create-home --shell /bin/false kai
USER kai

EXPOSE 8100

# Default: start the API server
CMD ["kai", "serve"]
