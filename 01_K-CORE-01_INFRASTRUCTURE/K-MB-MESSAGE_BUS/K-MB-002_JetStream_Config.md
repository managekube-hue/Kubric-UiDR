# K-MB-002  JetStream Configuration Reference

**Component**: NATS 2.10 JetStream
**Namespace**: kubric
**Last Updated**: 2026-02-26
**Owner**: Platform Engineering

---

## Table of Contents

1. [Stream Inventory](#1-stream-inventory)
2. [CLI Provisioning Commands](#2-cli-provisioning-commands)
3. [Consumer Configuration](#3-consumer-configuration)
4. [Account Limits](#4-account-limits)
5. [Storage Types](#5-storage-types)
6. [Deduplication Strategy](#6-deduplication-strategy)
7. [Dead-Letter Queue Pattern](#7-dead-letter-queue-pattern)
8. [Cross-DC Replication](#8-cross-dc-replication)
9. [Flow Control and Back-Pressure](#9-flow-control-and-back-pressure)
10. [Observability](#10-observability)

---

## 1. Stream Inventory

All 15 production streams provisioned in the kubric NATS account.
max_age is expressed both in nanoseconds (NATS native) and human-readable form.

| # | Stream Name | Subject Filter | Retention | Discard | max_age | max_bytes | Replicas | Dup Window |
|---|-------------|----------------|-----------|---------|---------|-----------|----------|------------|
| 1 | EDR-PROCESS | edr.process.v1.* | limits | old | 7d / 604800000000000 ns | 50 GiB | 3 | 2m |
| 2 | EDR-FILE | edr.file.v1.* | limits | old | 7d / 604800000000000 ns | 30 GiB | 3 | 2m |
| 3 | NDR-FLOW | ndr.flow.v1.* | limits | old | 3d / 259200000000000 ns | 100 GiB | 3 | 1m |
| 4 | NDR-BEACON | ndr.beacon.v1.* | limits | old | 30d / 2592000000000000 ns | unlimited | 3 | 5m |
| 5 | ITDR-AUTH | itdr.auth.v1.* | limits | old | 90d / 7776000000000000 ns | 20 GiB | 3 | 2m |
| 6 | VDR-VULN | vdr.vuln.v1.* | limits | old | 180d / 15552000000000000 ns | 10 GiB | 3 | 10m |
| 7 | GRC-DRIFT | grc.drift.v1.* | limits | old | 365d / 31536000000000000 ns | 5 GiB | 3 | 10m |
| 8 | SVC-TICKET | svc.ticket.v1.* | limits | old | 365d / 31536000000000000 ns | 10 GiB | 3 | 5m |
| 9 | BILLING-USAGE | billing.usage.v1.* | limits | old | 395d / 34128000000000000 ns | 10 GiB | 3 | 10m |
| 10 | HEALTH-SCORE | health.score.v1.* | limits | old | 90d / 7776000000000000 ns | 5 GiB | 3 | 5m |
| 11 | TI-IOC | ti.ioc.v1.* | limits | old | 30d / 2592000000000000 ns | 20 GiB | 3 | 5m |
| 12 | COMM-ALERT | comm.alert.v1.* | limits | old | 90d / 7776000000000000 ns | 5 GiB | 3 | 2m |
| 13 | SECURITY-ALERT | security.alert.v1.* | limits | old | 365d / 31536000000000000 ns | 100 GiB | 3 | 2m |
| 14 | REMEDIATION-TASK | remediation.task.v1.* | limits | old | 365d / 31536000000000000 ns | 10 GiB | 3 | 5m |
| 15 | ASSET-EVENTS | asset.provisioned.v1.* | limits | old | unlimited | unlimited | 3 | 10m |

Notes:
- NDR-BEACON: max_msgs = 1,000,000 (no byte cap)
- ASSET-EVENTS: max_msgs = 10,000,000 (no age or byte cap -- append-only ledger)

---

## 2. CLI Provisioning Commands

Connect to the NATS cluster before running any stream commands:

    nats context add kubric       --server nats://nats-cluster.kubric.svc.cluster.local:4222       --tlscert /etc/nats-tls/tls.crt       --tlskey  /etc/nats-tls/tls.key       --tlsca   /etc/nats-tls/ca.crt

    nats context select kubric

### 2.1 Create All 15 Streams (nats stream add)

    # 1. EDR-PROCESS
    nats stream add EDR-PROCESS       --subjects edr.process.v1.*       --retention limits --discard old       --max-age 604800s --max-bytes 53687091200       --replicas 3 --dupe-window 2m --storage file

    # 2. EDR-FILE
    nats stream add EDR-FILE       --subjects edr.file.v1.*       --retention limits --discard old       --max-age 604800s --max-bytes 32212254720       --replicas 3 --dupe-window 2m --storage file

    # 3. NDR-FLOW
    nats stream add NDR-FLOW       --subjects ndr.flow.v1.*       --retention limits --discard old       --max-age 259200s --max-bytes 107374182400       --replicas 3 --dupe-window 1m --storage file

    # 4. NDR-BEACON (no byte cap, 1M msg cap)
    nats stream add NDR-BEACON       --subjects ndr.beacon.v1.*       --retention limits --discard old       --max-age 2592000s --max-msgs 1000000       --replicas 3 --dupe-window 5m --storage file

    # 5. ITDR-AUTH
    nats stream add ITDR-AUTH       --subjects itdr.auth.v1.*       --retention limits --discard old       --max-age 7776000s --max-bytes 21474836480       --replicas 3 --dupe-window 2m --storage file
    # 6. VDR-VULN
    nats stream add VDR-VULN       --subjects "vdr.vuln.v1.*"       --retention limits --discard old       --max-age 15552000s --max-bytes 10737418240       --replicas 3 --dupe-window 10m --storage file

    # 7. GRC-DRIFT
    nats stream add GRC-DRIFT       --subjects "grc.drift.v1.*"       --retention limits --discard old       --max-age 31536000s --max-bytes 5368709120       --replicas 3 --dupe-window 10m --storage file

    # 8. SVC-TICKET
    nats stream add SVC-TICKET       --subjects "svc.ticket.v1.*"       --retention limits --discard old       --max-age 31536000s --max-bytes 10737418240       --replicas 3 --dupe-window 5m --storage file

    # 9. BILLING-USAGE
    nats stream add BILLING-USAGE       --subjects "billing.usage.v1.*"       --retention limits --discard old       --max-age 34128000s --max-bytes 10737418240       --replicas 3 --dupe-window 10m --storage file

    # 10. HEALTH-SCORE
    nats stream add HEALTH-SCORE       --subjects "health.score.v1.*"       --retention limits --discard old       --max-age 7776000s --max-bytes 5368709120       --replicas 3 --dupe-window 5m --storage file

    # 11. TI-IOC
    nats stream add TI-IOC       --subjects "ti.ioc.v1.*"       --retention limits --discard old       --max-age 2592000s --max-bytes 21474836480       --replicas 3 --dupe-window 5m --storage file

    # 12. COMM-ALERT
    nats stream add COMM-ALERT       --subjects "comm.alert.v1.*"       --retention limits --discard old       --max-age 7776000s --max-bytes 5368709120       --replicas 3 --dupe-window 2m --storage file

    # 13. SECURITY-ALERT
    nats stream add SECURITY-ALERT       --subjects "security.alert.v1.*"       --retention limits --discard old       --max-age 31536000s --max-bytes 107374182400       --replicas 3 --dupe-window 2m --storage file

    # 14. REMEDIATION-TASK
    nats stream add REMEDIATION-TASK       --subjects "remediation.task.v1.*"       --retention limits --discard old       --max-age 31536000s --max-bytes 10737418240       --replicas 3 --dupe-window 5m --storage file

    # 15. ASSET-EVENTS (no age limit, 10M msg cap)
    nats stream add ASSET-EVENTS       --subjects "asset.provisioned.v1.*"       --retention limits --discard old       --max-msgs 10000000       --replicas 3 --dupe-window 10m --storage file

### 2.2 Verify All Streams

    nats stream ls
    nats stream report

    # Per-stream detail
    nats stream info EDR-PROCESS

---

## 3. Consumer Configuration

JetStream supports push and pull consumers. Kubric services use pull consumers
exclusively for back-pressure control.

### 3.1 Pull Consumer (recommended for all services)

    # Create a durable pull consumer on EDR-PROCESS
    nats consumer add EDR-PROCESS edr-process-worker       --pull       --deliver all       --ack explicit       --max-deliver 5       --ack-wait 30s       --max-ack-pending 1000       --filter edr.process.v1.*

    # Fetch messages (batch of 100, wait 5s)
    nats consumer next EDR-PROCESS edr-process-worker       --count 100 --wait 5s

### 3.2 Push Consumer (monitoring / audit replay only)

    nats consumer add SECURITY-ALERT security-alert-audit       --push       --deliver all       --target security.alert.replay       --ack none       --replay original

### 3.3 Consumer per Tenant (subject-filtered)

Each tenant receives a filtered consumer scoped to their tenant ID:

    # Tenant consumer for edr.process.v1.<tenant_id>
    nats consumer add EDR-PROCESS edr-process-tenant-abc123       --pull       --deliver new       --ack explicit       --max-deliver 5       --ack-wait 30s       --max-ack-pending 500       --filter edr.process.v1.abc123

---

## 4. Account Limits

Configured in the NATS operator JWT resolver under the kubric account.

| Parameter | Value | Notes |
|-----------|-------|-------|
| max_connections | 10,000 | Total NATS connections per pod |
| max_subscriptions | 500,000 | Total subscriptions |
| max_payload | 8,388,608 (8 MiB) | Matches server max_payload |
| max_imports | 100 | Cross-account imports |
| max_exports | 100 | Cross-account exports |
| jetstream | enabled | Account-level JetStream flag |
| max_memory_store | 2,147,483,648 (2 GiB) | JetStream in-memory quota |
| max_file_store | 107,374,182,400 (100 GiB) | JetStream file quota per pod |
| max_streams | 50 | Upper bound for stream count |
| max_consumers | 5,000 | Upper bound for consumer count |

Limits are enforced by the server. Operators can tighten limits per sub-account
for individual tenants using NATS operator tooling:

    nsc add account --name tenant-abc123
    nsc edit account --name tenant-abc123       --js-mem-storage 536870912       --js-disk-storage 10737418240       --js-streams 10       --js-consumers 200
---

## 5. Storage Types

### 5.1 File Storage (all 15 streams)

All production streams use file storage backed by the StatefulSet PVC
(20 GiB per pod, storageClassName: standard).

Characteristics:
- Survives pod restarts
- Raft-replicated across 3 pods
- Write path: fsync on acknowledgement (safety over throughput)
- Compaction occurs automatically as messages age out or limits are hit
- Directory layout: /data/jetstream/<account>/<stream>/

Tuning the JetStream file store (nats.conf excerpt):

    jetstream {
      store_dir: /data/jetstream
      max_memory_store: 2147483648    # 2 GiB in-memory cache
      max_file_store:   107374182400  # 100 GiB on-disk cap
    }

### 5.2 Memory Storage (cache streams only)

Memory storage is reserved for low-latency ephemeral caches.
No production streams currently use memory storage.
To add a memory-backed stream:

    nats stream add <STREAM_NAME>       --storage memory       --retention workqueue       --max-age 300s

Caveat: memory-backed streams are lost on pod restart. Only use for
idempotent or reproducible workloads.

### 5.3 Storage Sizing Reference

    EDR-PROCESS  :  50 GiB  /  3 replicas  =  150 GiB raw disk
    EDR-FILE     :  30 GiB  /  3 replicas  =   90 GiB raw disk
    NDR-FLOW     : 100 GiB  /  3 replicas  =  300 GiB raw disk
    SECURITY-ALERT: 100 GiB / 3 replicas  =  300 GiB raw disk
    (all others) :  5-20 GiB / stream @ 3x = 45-180 GiB raw disk

Total estimated raw disk: ~1.5 TiB across the 3-node cluster.
Each PVC is 20 GiB. Scale storageRequest up to 512 GiB for production.

---

## 6. Deduplication Strategy

JetStream uses a per-stream duplicate detection window based on the
Nats-Msg-Id message header. Publishers MUST set this header.

### 6.1 How It Works

1. Publisher sets header: Nats-Msg-Id: <unique-id>
2. NATS server stores the ID in a rolling dedup cache for the configured
   duplicate_window duration.
3. Any message published with the same Nats-Msg-Id within the window
   is silently dropped (200 OK returned to publisher, no second write).
4. After the window expires the ID is evicted. Re-publishing then succeeds.

### 6.2 Recommended ID Format per Stream

| Stream | Nats-Msg-Id Format | Example |
|--------|--------------------|---------|
| EDR-PROCESS | proc:<tenant>:<pid>:<boot_id>:<seq> | proc:abc123:1234:boot7:00042 |
| EDR-FILE | file:<tenant>:<inode>:<mtime_ns> | file:abc123:2097154:1709000001 |
| NDR-FLOW | flow:<tenant>:<src_ip>:<dst_ip>:<ts_ms> | flow:abc123:10.0.0.1:1.2.3.4:1709000001000 |
| NDR-BEACON | beacon:<tenant>:<fqdn>:<ts_s> | beacon:abc123:evil.example.com:1709000001 |
| ITDR-AUTH | auth:<tenant>:<user>:<ts_ms> | auth:abc123:jdoe@corp:1709000001000 |
| VDR-VULN | vuln:<tenant>:<cve>:<asset_id> | vuln:abc123:CVE-2024-1234:host-007 |
| SECURITY-ALERT | alert:<tenant>:<alert_id> | alert:abc123:uuid-v4-here |
| ASSET-EVENTS | asset:<tenant>:<asset_id>:<event_type> | asset:abc123:host-007:provisioned |
| (others) | <stream_lower>:<tenant>:<uuid4> | ticket:abc123:uuid-v4-here |

### 6.3 Python Publisher Example

    import nats, asyncio

    async def publish_with_dedup(tenant_id: str, pid: int, seq: int, payload: bytes):
        nc = await nats.connect(
            servers=["nats://nats-cluster.kubric.svc.cluster.local:4222"],
            tls=tls_ctx
        )
        js = nc.jetstream()
        msg_id = f"proc:{tenant_id}:{pid}:{seq}"
        ack = await js.publish(
            f"edr.process.v1.{tenant_id}",
            payload,
            headers={"Nats-Msg-Id": msg_id},
        )
        await nc.close()

### 6.4 Dedup Window per Stream

| Stream | Dup Window | Rationale |
|--------|-----------|-----------|
| EDR-PROCESS | 2m | High-frequency; 2m covers agent retry window |
| EDR-FILE | 2m | Same agent retry budget |
| NDR-FLOW | 1m | Very high volume; shorter window to save memory |
| NDR-BEACON | 5m | Beacon intervals may be slow |
| ITDR-AUTH | 2m | Auth events are low-volume; short window sufficient |
| VDR-VULN | 10m | Scan results republished on scan restart |
| GRC-DRIFT | 10m | Drift polls run every 15m; 10m overlap safe |
| all others | 2m - 10m | See stream table in section 1 |
---

## 7. Dead-Letter Queue Pattern

### 7.1 Convention

When a consumer exhausts max-deliver retries, the message is not automatically
moved. Services MUST implement the DLQ pattern at the application layer:

Subject convention:  kubric.<tenant_id>.dlq.<original_stream_lower>

Examples:
  kubric.abc123.dlq.edr-process
  kubric.abc123.dlq.security-alert

### 7.2 Provision DLQ Streams

One DLQ stream per high-value domain (EDR, NDR, ITDR, SECURITY).
DLQ streams are single-replica, short-retention, small-cap.

    # EDR DLQ
    nats stream add EDR-DLQ       --subjects "kubric.*.dlq.edr-*"       --retention limits --discard old       --max-age 604800s --max-bytes 1073741824       --replicas 1 --storage file

    # NDR DLQ
    nats stream add NDR-DLQ       --subjects "kubric.*.dlq.ndr-*"       --retention limits --discard old       --max-age 604800s --max-bytes 1073741824       --replicas 1 --storage file

    # SECURITY DLQ
    nats stream add SECURITY-DLQ       --subjects "kubric.*.dlq.security-alert"       --retention limits --discard old       --max-age 1209600s --max-bytes 2147483648       --replicas 1 --storage file

    # Generic DLQ catch-all
    nats stream add GENERIC-DLQ       --subjects "kubric.*.dlq.*"       --retention limits --discard old       --max-age 604800s --max-bytes 1073741824       --replicas 1 --storage file

### 7.3 Service-Side DLQ Logic (Python example)

    MAX_DELIVER = 5

    async def process_message(msg):
        try:
            handle(msg.data)
            await msg.ack()
        except Exception as e:
            if msg.metadata.num_delivered >= MAX_DELIVER:
                dlq_subject = f"kubric.{tenant_id}.dlq.edr-process"
                await js.publish(dlq_subject, msg.data, headers={
                    "X-DLQ-Reason": str(e),
                    "X-Original-Subject": msg.subject,
                    "Nats-Msg-Id": f"dlq:{msg.metadata.sequence.stream}",
                })
                await msg.ack()  # ack to remove from main stream
            else:
                await msg.nak(delay=5)  # exponential back-off handled by caller

---

## 8. Cross-DC Replication

### 8.1 Strategy

JetStream sources pull messages from a remote stream into a local stream.
The remote cluster connects via NATS LeafNode on port 7422.

Topology:
  DC-PRIMARY (us-east-1)  <-->  DC-SECONDARY (us-west-2)
  LeafNode outbound to: nats://nats-cluster-headless.kubric.svc.cluster.local:7422

### 8.2 Source Configuration (NATS stream source)

    # On DC-SECONDARY: create mirror of SECURITY-ALERT from DC-PRIMARY
    nats stream add SECURITY-ALERT-REPLICA       --source SECURITY-ALERT       --source-domain dc-primary       --storage file       --replicas 3

Equivalent stream JSON config for GitOps:

    {
      "name": "SECURITY-ALERT-REPLICA",
      "storage": "file",
      "num_replicas": 3,
      "sources": [
        {
          "name": "SECURITY-ALERT",
          "domain": "dc-primary",
          "start_seq": 1
        }
      ]
    }

### 8.3 Streams Recommended for Cross-DC Replication

| Stream | Justification |
|--------|--------------|
| SECURITY-ALERT | SOC visibility in both DCs |
| TI-IOC | Threat intel must be globally consistent |
| ASSET-EVENTS | Asset ledger must be globally consistent |
| REMEDIATION-TASK | Tasks dispatched from either DC |
| BILLING-USAGE | Billing requires full global audit trail |

Streams NOT replicated cross-DC (high volume, local consumption only):
  EDR-PROCESS, EDR-FILE, NDR-FLOW, NDR-BEACON, ITDR-AUTH

### 8.4 LeafNode Server Config

    leafnodes {
      port: 7422
      remotes [
        {
          urls: ["nats://nats-cluster-headless.kubric-dc-primary.svc.cluster.local:7422"]
          credentials: /etc/nats-creds/dc-primary-leaf.creds
        }
      ]
    }
---

## 9. Flow Control and Back-Pressure

### 9.1 Pull Consumer Back-Pressure

Pull consumers inherently provide back-pressure because the client controls
fetch rate. Configure max_ack_pending to limit in-flight messages:

    max_ack_pending: 1000   # stop delivering if >1000 unacked messages exist

If the processing service is slow, pending ack count rises. NATS stops
delivering new batches once the limit is reached.

Recommended max_ack_pending values by stream:

| Stream | max_ack_pending | Rationale |
|--------|----------------|-----------|
| EDR-PROCESS | 1000 | High-throughput; worker pool of 10 |
| EDR-FILE | 500 | Each event triggers file hash I/O |
| NDR-FLOW | 5000 | Volume is extremely high; wide window |
| NDR-BEACON | 200 | Enrichment calls external threat intel API |
| SECURITY-ALERT | 100 | Each alert triggers downstream notification |
| (others) | 500 | Default baseline |

### 9.2 Publisher Flow Control

For very high-rate publishers (NDR-FLOW, EDR-PROCESS), enable flow_control
on push consumers and heartbeat on idle connections:

    nats consumer add NDR-FLOW ndr-flow-monitor       --push       --flow-control       --heartbeat 5s       --target ndr.flow.monitor       --ack none

### 9.3 Ack-Wait and Redelivery

    ack_wait: 30s           # service must ack within 30s or message is redelivered
    max_deliver: 5          # after 5 failed deliveries, stop redelivery
    backoff: [1s, 5s, 15s, 30s, 60s]   # exponential back-off schedule

Set via nats consumer add flags:
    --ack-wait 30s --max-deliver 5

### 9.4 Rate Limiting per Consumer

    nats consumer add EDR-PROCESS edr-ratelimited       --pull       --rate-limit 10485760    # 10 MiB/s per consumer

---

## 10. Observability

### 10.1 JetStream Advisory Subjects

JetStream publishes internal advisory events to these built-in subjects:

| Advisory | Subject | Description |
|----------|---------|-------------|
| Stream created | $JS.EVENT.ADVISORY.STREAM.CREATED.> | Stream provisioned |
| Stream deleted | $JS.EVENT.ADVISORY.STREAM.DELETED.> | Stream removed |
| Consumer created | $JS.EVENT.ADVISORY.CONSUMER.CREATED.> | Consumer added |
| Message ACKed | $JS.EVENT.ADVISORY.CONSUMER.MSG_NAKED.> | Explicit ack received |
| Max deliveries reached | $JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.> | DLQ trigger point |
| Stream leader change | $JS.EVENT.ADVISORY.STREAM.LEADER_ELECTED.> | Raft leader elected |
| API audit | $JS.EVENT.AUDIT.> | All API calls |

Subscribe to advisories for alerting:

    nats sub "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>"

### 10.2 Prometheus Metrics via nats-exporter Sidecar

The prometheus-nats-exporter sidecar (port 7777) exposes:

| Metric | Description |
|--------|-------------|
| gnatsd_varz_connections | Active connections |
| gnatsd_varz_in_msgs | Total messages in |
| gnatsd_varz_out_msgs | Total messages out |
| gnatsd_varz_in_bytes | Total bytes in |
| gnatsd_varz_out_bytes | Total bytes out |
| gnatsd_jsz_memory | JetStream memory used |
| gnatsd_jsz_storage | JetStream file storage used |
| gnatsd_jsz_streams | Number of streams |
| gnatsd_jsz_consumers | Number of consumers |
| gnatsd_connz_subscriptions | Active subscriptions |

Scrape endpoint: http://<pod-ip>:7777/metrics

ServiceMonitor for Prometheus Operator:

    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: nats-exporter
      namespace: kubric
    spec:
      selector:
        matchLabels:
          app: nats
      endpoints:
        - port: metrics
          interval: 15s
          path: /metrics

### 10.3 Stream Health CLI Checks

    # View stream state (msg count, bytes, consumer lag)
    nats stream report

    # Per-stream consumer lag
    nats consumer report EDR-PROCESS

    # Server-level JetStream stats
    nats server report jetstream

### 10.4 Recommended Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| StreamStorageHigh | jetstream_storage_used > 0.85 * limit | warning |
| ConsumerLagHigh | consumer num_pending > 100000 | warning |
| MaxDeliveriesReached | advisory MAX_DELIVERIES rate > 10/min | critical |
| StreamLeaderFlapping | leader_elected events > 3/5min | critical |
| NATSConnectionDrop | connections < expected_min | critical |
