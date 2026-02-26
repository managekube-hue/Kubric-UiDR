"""
Kubric KAI — AI orchestration layer.

Layer 2 (this version):
  - KAI-TRIAGE   alert enrichment via Ollama (local LLM)
  - KAI-SENTINEL KiSS health score computation
  - KAI-KEEPER   remediation plan generation + Temporal workflow trigger
  - KAI-COMM     escalation routing (Vapi voice, Twilio SMS, webhook)
  - KAI-FORESIGHT predictive risk modelling (heuristic; LSTM in Layer 3)
  - NATS subscriber routes kubric.* events to agents
  - Temporal activities: RemediationWorkflow, BillingWorkflow
  - FastAPI endpoints: /v1/triage, /v1/score, /v1/insights, /v1/remediate, /v1/billing/run

Layer 3 (next):
  - HIKARI-2021 LSTM network baseline training
  - EPSS integration for vulnerability prioritisation
  - CrewAI multi-agent collaboration (agents delegate tasks between each other)
  - Vector store (Chroma/Qdrant) for RAG-based alert correlation
"""

__version__ = "0.2.0"
