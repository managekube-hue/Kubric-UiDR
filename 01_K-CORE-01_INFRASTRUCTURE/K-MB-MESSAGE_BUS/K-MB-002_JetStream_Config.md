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

All 15 production streams provisioned in the  NATS account.  
 is expressed both in nanoseconds (NATS native) and human-readable form.

| # | Stream Name | Subject Filter | Retention | Discard | max_age | max_bytes | Replicas | Dup Window |
|---|-------------|---------------|-----------|---------|---------|-----------|----------|------------|
| 1 | EDR-PROCESS |  | limits | old | 7d (604800000000000 ns) | 50 GiB | 3 | 2m |
| 2 | EDR-FILE |  | limits | old | 7d (604800000000000 ns) | 30 GiB | 3 | 2m |
| 3 | NDR-FLOW |  | limits | old | 3d (259200000000000 ns) | 100 GiB | 3 | 1m |
| 4 | NDR-BEACON |  | limits | old | 30d (2592000000000000 ns) | unlimited | 3 | 5m |
| 5 | ITDR-AUTH |  | limits | old | 90d (7776000000000000 ns) | 20 GiB | 3 | 2m |
| 6 | VDR-VULN |  | limits | old | 180d (15552000000000000 ns) | 10 GiB | 3 | 10m |
| 7 | GRC-DRIFT |  | limits | old | 365d (31536000000000000 ns) | 5 GiB | 3 | 10m |
| 8 | SVC-TICKET |  | limits | old | 365d (31536000000000000 ns) | 10 GiB | 3 | 5m |
| 9 | BILLING-USAGE |  | limits | old | 395d (34128000000000000 ns) | 10 GiB | 3 | 10m |
| 10 | HEALTH-SCORE |  | limits | old | 90d (7776000000000000 ns) | 5 GiB | 3 | 5m |
| 11 | TI-IOC |  | limits | old | 30d (2592000000000000 ns) | 20 GiB | 3 | 5m |
| 12 | COMM-ALERT |  | limits | old | 90d (7776000000000000 ns) | 5 GiB | 3 | 2m |
| 13 | SECURITY-ALERT |  | limits | old | 365d (31536000000000000 ns) | 100 GiB | 3 | 2m |
| 14 | REMEDIATION-TASK |  | limits | old | 365d (31536000000000000 ns) | 10 GiB | 3 | 5m |
| 15 | ASSET-EVENTS |  | limits | old | unlimited | unlimited | 3 | 10m |

> NDR-BEACON:  (no byte cap).  
> ASSET-EVENTS:  (no age or byte cap — append-only ledger).

---

## 2. CLI Provisioning Commands

Connect to the NATS cluster via the kubric context before running any commands:



### 2.1 Create All 15 Streams


