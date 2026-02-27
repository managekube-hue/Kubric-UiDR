# K-KAI ML Model Tiering

## Overview

Kubric-KAI uses a three-tier ML inference model hierarchy to balance latency,
cost, and accuracy across detection, risk, and prediction workloads.

---

## Tier 1 — Lightweight Edge Models (On-Agent, < 10ms)

Run directly inside the CoreSec Rust agent using `candle` or ONNX Runtime.
No network round-trip. Suitable for high-frequency event filtering.

| Model | File | Purpose | Latency |
|---|---|---|---|
| AnomalyNet (12→64→32→1 MLP) | `K-XRO-CS-ML-001` | Process event anomaly score | < 2ms |
| TinyLlama-1.1B (quantised) | `K-XRO-CS-ML-002` | Local LLM reasoning, offline fallback | 50–200ms |

**Trigger:** Every process/network event streamed via eBPF.
**Threshold:** anomaly_score >= 0.70 triggers escalation to Tier 2.

---

## Tier 2 — Platform ML Service (vLLM + XGBoost, < 500ms)

Hosted on the platform control plane. Invoked on Tier-1 escalations
and scheduled enrichment jobs.

| Model | File | Dataset | Framework | Purpose |
|---|---|---|---|---|
| EMBER XGBoost | `K-KAI-ML-008` | EMBER 2018 (1M PE files) | XGBoost 2.x | Malware classification |
| UNSW-NB15 Random Forest | `K-KAI-ML-009` | UNSW-NB15 network | scikit-learn | Network intrusion detection |
| Mordor LSTM | `K-KAI-ML-010` | Mordor APT event logs | PyTorch | APT lateral movement detection |
| Threat Forecast LSTM | `K-KAI-FS-001` | Historical alert timeseries | PyTorch | Next-hour alert volume forecast |
| Churn Risk LR | `K-KAI-SEN-002` | Tenant usage features | scikit-learn | 30-day churn probability |
| LTV GBT | `K-KAI-SIM-001` | Tenant billing + health | scikit-learn | 12-month LTV prediction |

**Trigger:** NATS subject `kai.ml.infer.<model_id>`.
**SLA:** p99 < 500ms for synchronous inference; async via aiokafka for batch.

---

## Tier 3 — Large Language Models (API / vLLM, < 10s)

Used for complex reasoning: incident summarisation, compliance gap analysis,
remediation narrative generation, and CISO assistant queries.

| Model | File | Provider | Context | Purpose |
|---|---|---|---|---|
| GPT-4o | `K-KAI-ML-004` | OpenAI API | 128K tokens | Incident narrative, fallback LLM |
| Claude 3.5 Sonnet | `K-KAI-ML-005` | Anthropic API | 200K tokens | Long-context log analysis, RAG |
| Command R+ | `K-KAI-ML-006` / `K-KAI-RAG-004` | Cohere API | 128K tokens | RAG retrieval + rerank |
| Llama-3-70B (self-hosted) | `K-KAI-ML-011` | vLLM (OpenAI-compat) | 8K tokens | On-prem inference, air-gapped |

**Trigger:** Direct HTTP call from CrewAI persona agents or RAG pipeline.
**Cost control:** Model router (`openai_fallback`) falls back Tier-3 to vLLM if
API quota exceeded or latency > 8s.

---

## Model Lifecycle

```
Training                  Evaluation           Serving
-----------------------  ------------------   ----------------------
ClearML experiment        Cross-val (k=5)     vLLM (OpenAI compat)
PySpark distributed       ROC-AUC / MSE       Hikari PG pool
TensorBoard logging       Drift detection     NATS publish results
Model registry (S3)       A/B shadow mode     ClearML monitoring
```

### Retraining Schedule

| Model | Trigger | Schedule |
|---|---|---|
| EMBER XGBoost | New malware family detected | Monthly |
| UNSW-NB15 RF | Network baseline drift | Bi-weekly |
| Mordor LSTM | New APT campaign tagged | On-demand |
| Churn LR | > 5% prediction error | Weekly |
| Threat LSTM | Alert volume regime change | Daily (incremental) |

---

## ML Feature Store

All features are computed by `K-KAI-ML-007_hikari_preprocessor.py` and
persisted to:

- **Polars DataFrames** in memory during training (`K-KAI-LIBS-001`)
- **Parquet files** (zstd compressed) for offline training sets (`K-KAI-LIBS-002`)
- **PostgreSQL** `kai_ml_features` table for online inference lookups (Hikari pool)
- **ClickHouse** `kubric_ml_feature_store` for analytics and audit

---

## Privacy & Security

- No raw log data sent to external LLM APIs — only structured feature vectors
  or anonymised summaries
- PII fields (email, IP) redacted before Tier-3 API calls
- All model artefacts signed with Ed25519 (same key as GRC evidence bundles)
- vLLM self-hosted instance operates in tenant-isolated namespace
