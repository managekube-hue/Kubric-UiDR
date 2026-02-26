"""
Kubric KAI — AI orchestration layer.

Layer 2 responsibilities:
  - CrewAI personas (TRIAGE, SENTINEL, KEEPER, COMM, FORESIGHT)
  - LangChain chains for alert enrichment
  - vLLM / Ollama model serving
  - NATS subscriber → enrich → publish back
  - FastAPI endpoints for Go services to call synchronously

Wiring order:
  Phase 1 (this file): package skeleton only.
  Phase 2 (Layer 2):   NATS subscriber + FastAPI + CrewAI agents.
  Phase 3 (Layer 3):   ML models + training pipelines.
"""

__version__ = "0.1.0"
