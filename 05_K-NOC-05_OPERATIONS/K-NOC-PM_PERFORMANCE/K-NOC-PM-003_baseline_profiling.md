# K-NOC-PM-003 -- Kubric Performance Baseline Profiling Runbook

**Scope:** Defines CPU/memory baselines per Kubric service, profiling procedures,
ClickHouse query targets, NATS throughput baselines, and alert thresholds.

---

## 1. Service CPU/Memory Baselines

These baselines reflect steady-state under 50-tenant load (1000 events/min total).
Capture measurements during a 15-minute window excluding backup windows (02:00-04:00 UTC).

| Service     | Port | Baseline CPU (avg) | Baseline CPU (p99) | Baseline RAM (RSS) | Max RAM (OOM threshold) |
|-------------|------|--------------------|--------------------|--------------------|-------------------------|
| noc         | 8083 | 15% (1 vCPU)       | 60%                | 256 MiB             | 512 MiB                 |
| vdr         | 8081 | 8% (1 vCPU)        | 35%                | 192 MiB             | 384 MiB                 |
| ksvc        | 8080 | 5% (1 vCPU)        | 25%                | 128 MiB             | 256 MiB                 |
| kic         | 8082 | 10% (1 vCPU)       | 40%                | 192 MiB             | 384 MiB                 |
| correlation | 9000 | 20% (2 vCPU)       | 75%                | 512 MiB             | 1 GiB                   |
| temporal    | 7233 | 12% (2 vCPU)       | 50%                | 384 MiB             | 768 MiB                 |
| clickhouse  | 9000 | 25% (4 vCPU)       | 80%                | 2 GiB               | 4 GiB                   |
| nats        | 4222 | 5% (1 vCPU)        | 20%                | 128 MiB             | 256 MiB                 |

**Measurement method:**
```bash
# CPU (5-minute average for a single PID)
pidstat -p $(pgrep -f kubric-noc) 5 60

# RSS (resident set size)
ps -o pid,rss,vsz,comm -p $(pgrep -f kubric-noc)

# Or via cgroup (Kubernetes):
kubectl top pod -n kubric --containers
```

---

## 2. Go Service pprof Profiling

All Kubric Go services expose pprof endpoints on their HTTP port at `/debug/pprof/`.
Ensure `KUBRIC_PPROF_ENABLED=true` is set in the service environment.

### 2.1 CPU Profile (30s sample)
```bash
# Replace <service> and <port> per the table above
go tool pprof -http=:8080 http://localhost:<port>/debug/pprof/profile?seconds=30
```

### 2.2 Memory (Heap) Profile
```bash
go tool pprof -http=:8081 http://localhost:<port>/debug/pprof/heap
```

### 2.3 Goroutine Dump (check for leaks)
```bash
curl -s http://localhost:<port>/debug/pprof/goroutine?debug=1 | head -100
```

### 2.4 Trace (1s window)
```bash
curl -o /tmp/trace.out http://localhost:<port>/debug/pprof/trace?seconds=1
go tool trace /tmp/trace.out
```

### 2.5 CPU Flame Graph (requires graphviz)
```bash
go tool pprof -png http://localhost:<port>/debug/pprof/profile?seconds=15 > /tmp/cpu.png
```

---

## 3. ClickHouse Query Performance Targets

| Query Type                        | P50 Target | P99 Target | Alert Threshold |
|-----------------------------------|------------|------------|-----------------|
| SELECT from events (1h window)    | < 50ms     | < 200ms    | > 500ms         |
| SELECT from events (24h window)   | < 200ms    | < 1s       | > 3s            |
| INSERT batch (1000 rows)          | < 20ms     | < 100ms    | > 300ms         |
| Aggregation (GROUP BY tenant, 1d) | < 500ms    | < 3s       | > 10s           |
| Full-table scan (analytics)       | < 5s       | < 30s      | > 60s           |

**Profiling ClickHouse slow queries:**
```sql
-- Enable query log
SET log_queries = 1;

-- Review recent slow queries (> 1s)
SELECT
    query_start_time,
    query_duration_ms,
    read_rows,
    memory_usage,
    substring(query, 1, 120) AS query_preview
FROM system.query_log
WHERE query_duration_ms > 1000
  AND type = 'QueryFinish'
  AND event_date = today()
ORDER BY query_duration_ms DESC
LIMIT 20;

-- Check table sizes
SELECT
    table,
    formatReadableSize(sum(bytes_on_disk)) AS disk_size,
    sum(rows) AS total_rows
FROM system.parts
WHERE database = 'kubric_events' AND active
GROUP BY table
ORDER BY sum(bytes_on_disk) DESC;
```

---

## 4. NATS Throughput Baselines

**Streams:** `kubric.events`, `kubric.incidents`, `kubric.heartbeats`, `kubric.audit`

| Stream               | Baseline msg/sec | P99 burst | Max sustained | Alert threshold  |
|----------------------|------------------|-----------|---------------|------------------|
| kubric.events        | 500 msg/s        | 2000 msg/s| 1500 msg/s    | > 4000 msg/s     |
| kubric.incidents     | 50 msg/s         | 200 msg/s | 150 msg/s     | > 500 msg/s      |
| kubric.heartbeats    | 100 msg/s        | 300 msg/s | 200 msg/s     | > 600 msg/s      |
| kubric.audit         | 200 msg/s        | 600 msg/s | 400 msg/s     | > 1000 msg/s     |

**Measure NATS throughput (requires nats CLI):**
```bash
# Install: go install github.com/nats-io/natscli/nats@latest
nats --server=nats://nats:4222 stream info kubric.events
nats --server=nats://nats:4222 server report jetstream

# Real-time message rate
watch -n2 "nats --server=nats://nats:4222 stream info kubric.events | grep Messages"
```

**JetStream consumer lag check:**
```bash
nats --server=nats://nats:4222 consumer report kubric.events
```
Consumer lag > 10,000 messages is an alert condition.

---

## 5. Alert Thresholds vs Baseline

These thresholds correspond to the Prometheus recording rules in PM-004.

| Metric                              | Warning             | Critical             | Action                         |
|-------------------------------------|---------------------|----------------------|--------------------------------|
| CPU usage (per service)             | > 2x baseline avg   | > p99 threshold      | Scale horizontally; profile    |
| Memory RSS (per service)            | > 75% of max        | > 90% of max         | Heap dump; check for leaks     |
| ClickHouse query P99                | > 2x target         | > alert threshold    | EXPLAIN ANALYZE; add index     |
| NATS consumer lag                   | > 5,000 msgs        | > 10,000 msgs        | Scale consumers; check backpressure |
| kubric:noc:incident_rate5m          | > 100 incidents/min | > 500 incidents/min  | Investigate event storm        |
| kubric:vdr:critical_vuln_count      | > 10 unpatched      | > 50 unpatched       | Trigger K-NOC-CM-ANS-002      |
| kubric:noc:agent_availability       | < 95% agents live   | < 80% agents live    | Check network; re-deploy agents|
| Disk usage (ClickHouse/Postgres)    | > 70% full          | > 85% full           | Trigger S3 cold lifecycle      |

---

## 6. Capturing a Baseline (Procedure)

1. Ensure no maintenance activity is running (no backup jobs, no patch windows).
2. Let the system settle under normal load for 15 minutes.
3. Run the metric capture script:
   ```bash
   # From NOC ops host, outputs CSV for each service
   for svc in noc vdr ksvc kic; do
     echo "$svc,$(date -Is),$(kubectl top pod -n kubric -l app=$svc --no-headers | awk '{print $2,$3}')"
   done >> /var/log/kubric/baselines.csv
   ```
4. Record results in the Kubric performance matrix spreadsheet (NOC > Baselines).
5. Update this document if baselines have shifted > 20%.
