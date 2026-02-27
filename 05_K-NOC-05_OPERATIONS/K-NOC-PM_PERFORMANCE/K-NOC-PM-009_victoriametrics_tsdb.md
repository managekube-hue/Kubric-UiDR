# K-NOC-PM-009 — VictoriaMetrics Long-Term TSDB Runbook

## Motivation: When to Switch from Thanos to VictoriaMetrics

Switch when any of the following thresholds are breached:
- `prometheus_tsdb_head_series > 2_000_000` (active series cardinality)
- Prometheus disk grows faster than 100 GiB/month
- Ingestion rate peaks above 300k samples/sec regularly
- Thanos compaction lag exceeds 2h consistently

VictoriaMetrics is a drop-in Prometheus-compatible replacement with 4x better
compression (~0.4 bytes/sample vs 1.25), single-node capacity up to ~3M samples/sec,
and native long-term retention without a separate object-store compactor.

---

## Single-Node Quickstart (Docker Compose)

```yaml
services:
  victoriametrics:
    image: victoriametrics/victoria-metrics:v1.101.0
    command:
      - -storageDataPath=/var/vm/data
      - -retentionPeriod=12
      - -httpListenAddr=:8428
      - -maxHourlySeries=2000000
      - -search.maxSamplesPerQuery=1000000000
    environment:
      - VM_retentionPeriod=12
      - VM_maxHourlySeries=2000000
    volumes:
      - vm_data:/var/vm/data
    ports:
      - "8428:8428"
volumes:
  vm_data:
```

---

## Prometheus remote_write to VictoriaMetrics

```yaml
remote_write:
  - url: http://victoriametrics:8428/api/v1/write
    queue_config:
      max_samples_per_send: 10000
      capacity: 50000
      max_shards: 4
```

For cluster mode, point to vminsert instead:
`url: http://vminsert:8480/insert/0/prometheus/api/v1/write`

---

## Grafana Datasource (Prometheus-compatible)

```yaml
- name: VictoriaMetrics
  type: prometheus
  uid: victoriametrics
  url: http://victoriametrics:8428
  access: proxy
  isDefault: false
  jsonData:
    httpMethod: POST
    prometheusType: VictoriaMetrics
    prometheusVersion: "1.101.0"
    timeInterval: "15s"
```

---

## Key Environment Variables

| Variable | Description | Default |
|---|---|---|
| `VM_retentionPeriod` | Data retention in months (or suffix h/d/w/y) | `1` (1 month) |
| `VM_maxHourlySeries` | Max unique time series per hour before throttle | unlimited |
