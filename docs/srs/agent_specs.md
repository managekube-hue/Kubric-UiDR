# Agent Specifications

## K-XRO-02: Next-Generation Security Agents

### Agent Types Overview

| Agent | Language | Purpose | Interval |
|-------|----------|---------|----------|
| **CoreSec** | Rust + eBPF | Kernel security monitoring | Real-time |
| **NetGuard** | Rust + Pcap | Network detection & response | Real-time |
| **PerfTrace** | Rust + perf | Performance profiling | 30 seconds |
| **Watchdog** | Rust + Go | Orchestration & patching | 5 minutes |

### CoreSec (eBPF Security Agent)

**Purpose**: Monitor system calls, file access, and privilege escalation in kernel.

**Capabilities**:
- Fork/exec tracing with full command-line capture
- File open/write operations with DACLs
- Network socket operations (bind, connect, listen)
- Privilege escalation detection
- Container-aware context

**Data Output**: NATS JetStream `agent.coresec.events`

**Performance**: <2% CPU overhead at 10k events/second

### NetGuard (Network Detection & Response)

**Purpose**: Monitor network flows, detect anomalies, and support active blocking.

**Capabilities**:
- Flow metadata: src/dst IPs, ports, protocols, bytes, packets
- Protocol analysis: DNS, HTTP, TLS, SSH signatures
- Anomaly scoring based on baselines
- Automatic response triggers (block, isolate, alert)
- PCAP integration for packet capture

**Data Output**: NATS JetStream `agent.netguard.flows`

**Performance**: Wire-speed analysis on 10Gbps interfaces

### PerfTrace (Performance Profiling)

**Purpose**: Continuous hardware performance monitoring and flame graphs.

**Capabilities**:
- CPU sampling at adjustable frequencies (100Hz-1kHz)
- Memory allocation tracking
- I/O latency histograms
- Context switching analysis
- Container-integrated metrics

**Data Output**: NATS JetStream `agent.perftrace.metrics`

### Update Mechanism

**Framework**: go-tuf (The Update Framework)

**Process**:
1. Agent checks for update metadata from secure endpoint
2. Validates signatures against trusted root
3. Downloads binary delta using Zstd
4. Applies patch with integrity verification
5. Executes restart with health checks

**Rollback**: Automatic to previous version if health check fails

### Agent Registration

**Process**:
1. Agent generates Blake3 hardware fingerprint
2. Requests JWT install token from API (valid 24 hours)
3. Submits registration CSR with hardware identity
4. Registry generates unique agent ID
5. Agent stores ID in TPM or secure enclave

**Security**: mTLS required; install token single-use

---

Generated: 2026-02-12
