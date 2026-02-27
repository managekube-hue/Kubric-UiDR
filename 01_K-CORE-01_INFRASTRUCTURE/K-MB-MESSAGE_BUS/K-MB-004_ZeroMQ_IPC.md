# K-MB-004  ZeroMQ IPC Design Document

**Component**: ZeroMQ Intra-Node Communication  
**Namespace**: kubric  
**Last Updated**: 2026-02-26  
**Owner**: Platform Engineering  

---

## Table of Contents

1. [Overview](#1-overview)
2. [Socket Patterns and Addresses](#2-socket-patterns-and-addresses)
3. [Kubernetes emptyDir Volume Configuration](#3-kubernetes-emptydir-volume-configuration)
4. [Message Format (msgpack)](#4-message-format-msgpack)
5. [Python Implementation (pyzmq)](#5-python-implementation-pyzmq)
6. [Rust Implementation (zeromq crate)](#6-rust-implementation-zeromq-crate)
7. [ZMQ vs NATS Decision Guide](#7-zmq-vs-nats-decision-guide)
8. [Security](#8-security)

---

## 1. Overview

ZeroMQ IPC sockets provide sub-millisecond latency communication between
containers sharing a single Kubernetes Pod. NATS JetStream is used for
inter-pod and inter-service messaging; ZMQ IPC fills the gap for tight
intra-node pipelines where NATS network overhead is unacceptable.

Primary use cases:
- EDR agent: kernel eBPF probe -> userspace collector -> NATS publisher
- NDR agent: AF_PACKET capture -> flow parser -> NATS publisher
- KAI orchestrator: sub-agent spawner -> result aggregator -> response emitter

ZMQ transport: ipc:// (Unix domain socket over a shared emptyDir volume).
This avoids all TCP stack overhead while remaining language/runtime-agnostic.

---

## 2. Socket Patterns and Addresses

### 2.1 PUSH/PULL (pipeline, fan-out to workers)

    Producer  PUSH --> ipc:///tmp/kubric/edr.sock --> PULL  Consumer

Used for: EDR process events, file events, NDR flow records.

- PUSH socket: producer (eBPF probe / packet capture) binds the socket.
- PULL socket: consumer (encoder / NATS publisher) connects.
- Load-balanced across multiple PULL workers if multiple consumers connect.
- One-way; no reply socket needed.

Socket addresses:
  ipc:///tmp/kubric/edr.sock          -- EDR process and file events
  ipc:///tmp/kubric/ndr.sock          -- NDR flow and beacon events
  ipc:///tmp/kubric/itdr.sock         -- ITDR auth events
  ipc:///tmp/kubric/remediation.sock  -- Remediation task dispatch

### 2.2 PUB/SUB (broadcast to all subscribers)

    Publisher  PUB --> ipc:///tmp/kubric/broadcast.sock --> SUB  Subscriber(s)

Used for: configuration updates, policy pushes, shutdown signals.

- PUB socket: control plane (policy agent) binds.
- SUB socket: each worker subscribes with topic filters.
- No message buffering; late-joining subscribers miss past messages.

Socket addresses:
  ipc:///tmp/kubric/broadcast.sock    -- Policy and config broadcast
  ipc:///tmp/kubric/control.sock      -- Lifecycle signals (pause/resume/shutdown)

### 2.3 REQ/REP (synchronous query, internal use only)

    Client  REQ --> ipc:///tmp/kubric/query.sock --> REP  Server

Used for: health probes between sidecar containers, capability queries.
Avoid for high-throughput paths; deadlock risk if server dies.

---

## 3. Kubernetes emptyDir Volume Configuration

All ZMQ IPC socket files reside under /tmp/kubric/ on a shared emptyDir
volume. All containers in the Pod must mount the same volume.

    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: edr-agent
      namespace: kubric
    spec:
      template:
        spec:
          volumes:
            - name: zmq-ipc
              emptyDir:
                medium: Memory        # tmpfs: kernel-managed, no disk I/O
                sizeLimit: 128Mi      # cap memory usage; sockets use ~0 bytes

          initContainers:
            - name: ipc-dir-init
              image: busybox:1.36
              command: ["sh", "-c", "mkdir -p /tmp/kubric && chmod 770 /tmp/kubric"]
              volumeMounts:
                - name: zmq-ipc
                  mountPath: /tmp/kubric

          containers:
            - name: ebpf-probe
              image: kubric/edr-probe:latest
              volumeMounts:
                - name: zmq-ipc
                  mountPath: /tmp/kubric

            - name: nats-publisher
              image: kubric/edr-publisher:latest
              volumeMounts:
                - name: zmq-ipc
                  mountPath: /tmp/kubric

Notes:
- medium: Memory uses tmpfs. Socket files themselves have no data,
  but ZMQ kernel buffers are backed by RAM. 128Mi is sufficient for
  up to 10 concurrent high-throughput sockets.
- The initContainer ensures the directory exists before ZMQ processes start.
  Without this, the PUSH socket bind() will fail with ENOENT.
- All containers must mount the volume at the same path (/tmp/kubric).
---

## 4. Message Format (msgpack)

All ZMQ messages are serialized with msgpack (binary, compact, schema-less).
The envelope wraps every domain event:

### 4.1 Envelope Schema

    {
      "v":   1,             # schema version (uint8)
      "ts":  1709000001,    # UNIX timestamp seconds (uint32)
      "tid": "abc123",      # tenant ID (str, max 32 chars)
      "t":   "ProcessEvent", # event type discriminator (str)
      "d":   { ... }        # domain payload (map)
    }

### 4.2 ProcessEvent Payload

    {
      "pid":    1234,          # process ID (uint32)
      "ppid":   1000,          # parent PID (uint32)
      "comm":   "bash",         # process name (str, max 16 bytes)
      "exe":    "/bin/bash",    # full path (str)
      "args":   ["-c", "id"],   # argv (array of str)
      "uid":    1000,           # UID (uint32)
      "gid":    1000,           # GID (uint32)
      "net_ns": 4026531840,     # network namespace inode (uint64)
      "cgroup": "kubepods-...", # cgroup v2 path fragment (str)
      "boot_id": "abc-def-...", # /proc/sys/kernel/random/boot_id (str)
      "seq":    42              # monotonic per-agent sequence (uint64)
    }

### 4.3 FileEvent Payload

    {
      "path":   "/etc/passwd",  # file path (str)
      "op":     "open",          # operation: open, read, write, exec, unlink (str)
      "inode":  2097154,          # inode number (uint64)
      "dev":    66306,            # device number (uint64)
      "size":   2971,             # file size bytes (uint64)
      "mode":   33188,            # file mode (uint32, octal as int)
      "uid":    0,                # owning UID (uint32)
      "mtime":  1709000001        # mtime UNIX seconds (uint32)
    }

### 4.4 Python msgpack Example

    import msgpack

    def encode_process_event(tenant_id: str, pid: int, comm: str) -> bytes:
        envelope = {
            "v":   1,
            "ts":  int(time.time()),
            "tid": tenant_id,
            "t":   "ProcessEvent",
            "d":   {
                "pid":  pid,
                "comm": comm,
                "exe":  f"/proc/{pid}/exe",
                "seq":  next_seq(),
            },
        }
        return msgpack.packb(envelope, use_bin_type=True)

    def decode_envelope(raw: bytes) -> dict:
        return msgpack.unpackb(raw, raw=False)

---

## 5. Python Implementation (pyzmq)

Dependency: pyzmq >= 25.0.0, msgpack >= 1.0.0

### 5.1 PUSH Producer (eBPF probe side)

    import zmq
    import msgpack
    import time
    import itertools

    class EDRProducer:
        def __init__(self):
            self._ctx = zmq.Context.instance()
            self._sock = self._ctx.socket(zmq.PUSH)
            # SNDHWM: drop messages if consumer is too slow (avoid OOM)
            self._sock.setsockopt(zmq.SNDHWM, 10000)
            self._sock.bind("ipc:///tmp/kubric/edr.sock")
            self._seq = itertools.count()

        def emit_process_event(self, tenant_id: str, pid: int, comm: str) -> None:
            envelope = {
                "v": 1, "ts": int(time.time()), "tid": tenant_id,
                "t": "ProcessEvent",
                "d": {"pid": pid, "comm": comm, "seq": next(self._seq)},
            }
            self._sock.send(msgpack.packb(envelope, use_bin_type=True),
                           zmq.NOBLOCK)

        def close(self) -> None:
            self._sock.close()
            self._ctx.term()

### 5.2 PULL Consumer (NATS publisher side)

    import zmq
    import msgpack
    import nats
    import asyncio

    class EDRConsumer:
        def __init__(self):
            self._ctx = zmq.Context.instance()
            self._sock = self._ctx.socket(zmq.PULL)
            self._sock.setsockopt(zmq.RCVHWM, 10000)
            self._sock.connect("ipc:///tmp/kubric/edr.sock")

        def recv_batch(self, max_count: int = 100) -> list:
            items = []
            for _ in range(max_count):
                try:
                    raw = self._sock.recv(zmq.NOBLOCK)
                    items.append(msgpack.unpackb(raw, raw=False))
                except zmq.Again:
                    break
            return items

        def close(self) -> None:
            self._sock.close()

### 5.3 PUB Broadcaster (policy agent)

    class PolicyBroadcaster:
        def __init__(self):
            self._ctx = zmq.Context.instance()
            self._sock = self._ctx.socket(zmq.PUB)
            self._sock.bind("ipc:///tmp/kubric/broadcast.sock")

        def broadcast_policy(self, topic: str, payload: dict) -> None:
            msg = topic.encode() + b" " + msgpack.packb(payload, use_bin_type=True)
            self._sock.send(msg)

### 5.4 SUB Subscriber (worker side)

    class PolicySubscriber:
        def __init__(self, topic: str = "policy"):
            self._ctx = zmq.Context.instance()
            self._sock = self._ctx.socket(zmq.SUB)
            self._sock.setsockopt_string(zmq.SUBSCRIBE, topic)
            self._sock.connect("ipc:///tmp/kubric/broadcast.sock")

        def recv_policy(self) -> tuple[str, dict]:
            raw = self._sock.recv()
            topic, data = raw.split(b" ", 1)
            return topic.decode(), msgpack.unpackb(data, raw=False)
---

## 6. Rust Implementation (zeromq crate)

Dependency: zeromq = "0.4", rmp-serde = "1.3", serde = { features = ["derive"] }

### 6.1 ProcessEvent Struct

    use serde::{Deserialize, Serialize};

    #[derive(Debug, Serialize, Deserialize)]
    pub struct Envelope {
        pub v:   u8,
        pub ts:  u32,
        pub tid: String,
        pub t:   String,
        pub d:   serde_json::Value,
    }

    #[derive(Debug, Serialize, Deserialize)]
    pub struct ProcessEvent {
        pub pid:  u32,
        pub ppid: u32,
        pub comm: String,
        pub exe:  String,
        pub uid:  u32,
        pub gid:  u32,
        pub seq:  u64,
    }

### 6.2 PUSH Producer (Rust)

    use zeromq::{PushSocket, Socket, SocketSend};
    use rmp_serde::to_vec_named;
    use std::time::{SystemTime, UNIX_EPOCH};

    pub struct EDRProducerRust {
        socket: PushSocket,
    }

    impl EDRProducerRust {
        pub async fn new() -> anyhow::Result<Self> {
            let mut socket = PushSocket::new();
            socket.bind("ipc:///tmp/kubric/edr.sock").await?;
            Ok(Self { socket })
        }

        pub async fn emit(&mut self, tid: &str, event: ProcessEvent) -> anyhow::Result<()> {
            let ts = SystemTime::now()
                .duration_since(UNIX_EPOCH)?.as_secs() as u32;
            let env = Envelope {
                v: 1, ts, tid: tid.to_string(),
                t: "ProcessEvent".to_string(),
                d: serde_json::to_value(&event)?,
            };
            let bytes = to_vec_named(&env)?;
            self.socket.send(bytes.into()).await?;
            Ok(())
        }
    }

### 6.3 PULL Consumer (Rust)

    use zeromq::{PullSocket, Socket, SocketRecv};
    use rmp_serde::from_slice;

    pub struct EDRConsumerRust {
        socket: PullSocket,
    }

    impl EDRConsumerRust {
        pub async fn new() -> anyhow::Result<Self> {
            let mut socket = PullSocket::new();
            socket.connect("ipc:///tmp/kubric/edr.sock").await?;
            Ok(Self { socket })
        }

        pub async fn recv(&mut self) -> anyhow::Result<Envelope> {
            let msg = self.socket.recv().await?;
            let bytes = msg.get(0)
                .ok_or_else(|| anyhow::anyhow!("empty zmq msg"))?;
            Ok(from_slice::<Envelope>(bytes)?)
        }
    }

---

## 7. ZMQ vs NATS Decision Guide

| Criterion | ZMQ IPC | NATS JetStream |
|-----------|---------|---------------|
| Latency | sub-100 us (kernel bypass) | 1-5 ms (network + TLS) |
| Throughput | 2-10 M msg/s per socket | 100-500 K msg/s per subject |
| Persistence | NONE (in-memory ring only) | YES (file-backed, Raft) |
| Replay / rewind | No | Yes (consumer sequence) |
| Multi-pod fan-out | No (same pod only) | Yes (any pod, any DC) |
| Back-pressure | HWM drop (configurable) | max_ack_pending |
| Observability | Manual metrics only | Built-in advisory + Prometheus |
| Deduplication | No | Yes (Nats-Msg-Id, dup window) |
| Authentication | File permissions only | mTLS + JWT operator model |
| Cross-language | Yes (C, Python, Go, Rust, ...) | Yes (same) |
| Operational cost | Low (no server) | Medium (StatefulSet + PVC) |

### 7.1 Use ZMQ IPC When

- Sender and receiver are co-located in the same Pod.
- Latency budget is < 1 ms (e.g., eBPF probe -> userspace encoder).
- Event loss is acceptable if the consumer falls behind (sensor self-protection).
- No cross-restart durability is required.

### 7.2 Use NATS JetStream When

- Events must survive producer or consumer restarts.
- Multiple independent consumers must receive the same event.
- Events originate from, or must be consumed by, different Pods or DCs.
- Deduplication, replay, or audit trail is required.
- Back-pressure must propagate upstream (consumer-driven flow control).

### 7.3 Hybrid Pattern (recommended for EDR/NDR agents)

    [kernel eBPF probe]
         |  ZMQ PUSH  (zero-copy, intra-pod, lossy)
         v
    [userspace encoder / enricher]     <- same Pod
         |  NATS publish (mTLS, durable, exactly-once with Nats-Msg-Id)
         v
    [JetStream EDR-PROCESS stream]     <- survives pod restart, replicated x3

The ZMQ hop absorbs bursts from the kernel probe without back-pressuring
the eBPF ring buffer. The NATS hop provides durability and multi-consumer
fan-out to downstream analytics, alerting, and billing pipelines.

---

## 8. Security

### 8.1 File Permission Model

The /tmp/kubric directory and all socket files must be:

    chmod 770 /tmp/kubric               # rwxrwx--- group access
    chown 1000:1000 /tmp/kubric          # match runAsUser/fsGroup

Socket files are created by ZMQ bind() with default umask permissions.
Set umask 0007 in the container entrypoint before starting ZMQ processes:

    #!/bin/sh
    umask 0007
    exec /app/edr-probe

Only containers in the same Pod can access ipc:// sockets by filesystem
isolation (each Pod has its own mount namespace and emptyDir).

### 8.2 SELinux / AppArmor

When SELinux is enforced:
- Label the emptyDir volume with type svirt_sandbox_file_t.
- All containers in the Pod share the same MCS label pair.
  No additional policy changes needed for intra-Pod IPC.

When AppArmor is enforced:
- Ensure the profile allows ipc operations and write to /tmp/kubric/:
    /tmp/kubric/ rw,
    /tmp/kubric/** rw,

### 8.3 tmpfs Rationale (medium: Memory)

Using emptyDir with medium: Memory (tmpfs):
- Socket data never touches disk (avoids data-at-rest risk for sensitive events).
- Kernel manages the memory; it is released immediately on Pod deletion.
- Swap is disabled in Kubernetes by default so tmpfs data never swaps.

Do NOT use emptyDir without medium: Memory for security-sensitive event
pipelines. The default emptyDir uses the node ephemeral storage (hostPath-
backed), which may persist data beyond Pod lifetime in some configurations.

### 8.4 securityContext for ZMQ Containers

    securityContext:
      runAsNonRoot: true
      runAsUser: 1000
      runAsGroup: 1000
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
          - ALL
        add:
          - NET_BIND_SERVICE   # only if binding to privileged ports (none for ZMQ IPC)

Note: readOnlyRootFilesystem: true is compatible with ZMQ IPC because the
socket files live on the emptyDir volume (/tmp/kubric/), not on the
container root filesystem.
