# K-MB-004 — ZeroMQ IPC

## Overview

ZeroMQ provides sub-millisecond intra-host messaging between co-located Kubric agent processes. Used when CoreSec and NetGuard run on the same endpoint — avoids NATS network round-trip for latency-critical eBPF event handoff.

## Messaging Hierarchy

| Layer | Transport | Latency | Use Case |
|-------|-----------|---------|----------|
| ZeroMQ (IPC) | Unix socket / TCP localhost | <1ms | eBPF-to-Go handoff on same host |
| NATS JetStream | TCP (cluster) | ~5ms | Inter-service durable messaging |
| Temporal | HTTP (workflow) | ~50ms | Durable business workflows |
| n8n | HTTP (webhook) | ~100ms | API integration glue |

## Architecture

```
CoreSec (Rust)                    NetGuard (Rust)
    │                                 │
    │ eBPF hooks                      │ pcap capture
    │                                 │
    ▼                                 ▼
ZMQ PUSH ──────────────────────► ZMQ PULL
(tcp://localhost:5555)       (tcp://localhost:5555)
    │                                 │
    └──── Both publish to NATS ───────┘
          (kubric.edr.> / kubric.ndr.>)
```

## Go Integration

```go
import (
    "context"
    "github.com/go-zeromq/zmq4"
)

// Publisher (CoreSec agent)
func startZMQPublisher(ctx context.Context) error {
    sock := zmq4.NewPush(ctx)
    defer sock.Close()

    if err := sock.Dial("tcp://localhost:5555"); err != nil {
        return err
    }

    // Send eBPF process event to local NetGuard for correlation
    msg := zmq4.NewMsgFrom([]byte(ocsf_event_json))
    return sock.Send(msg)
}

// Subscriber (NetGuard agent)
func startZMQSubscriber(ctx context.Context) error {
    sock := zmq4.NewPull(ctx)
    defer sock.Close()

    if err := sock.Listen("tcp://*:5555"); err != nil {
        return err
    }

    for {
        msg, err := sock.Recv()
        if err != nil {
            return err
        }
        // Correlate process event with network flow
        correlateWithFlow(msg.Frames[0])
    }
}
```

## Python Integration (KAI agents)

```python
import zmq

context = zmq.Context()

# Subscriber (KAI-TRIAGE receiving local events)
sock = context.socket(zmq.PULL)
sock.connect("tcp://localhost:5555")

while True:
    event = sock.recv_json()
    triage_event(event)
```

## When to Use ZeroMQ vs NATS

| Scenario | Transport |
|----------|-----------|
| eBPF event → same-host Go process | ZeroMQ |
| Agent → KAI cluster (different host) | NATS |
| KAI → K-SVC (API call) | NATS or HTTP |
| Billing workflow (durable) | Temporal |
| Cloud API polling | n8n |

## Configuration

ZeroMQ IPC is enabled via environment variable:

```bash
KUBRIC_ZMQ_ENABLED=true
KUBRIC_ZMQ_ENDPOINT=tcp://localhost:5555
```

When disabled (default for single-agent deployments), events go directly to NATS.
