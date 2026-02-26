# K-SEC-002 — TPM Root of Trust

## Overview

Agent identity is anchored to hardware via a TPM 2.0 (Trusted Platform Module)
or software-emulated root of trust. This prevents agent cloning and ensures
each endpoint has a unique, non-forgeable identity.

## Hardware Flow (TPM 2.0)

1. Agent reads TPM Endorsement Key (EK) public portion
2. `Blake3Hash(EK_pub || hostname || OS || kernel || MACs || CPU)` → agent fingerprint
3. Fingerprint submitted during provisioning registration
4. Server stores fingerprint; re-enrollment detects hardware changes

## Software Fallback (no TPM)

When no TPM is available (VMs, containers), the agent falls back to:

```
Blake3Hash(hostname || OS || kernel || MAC_addresses || CPU_model)
```

Implemented in `internal/security/blake3_fingerprint.go`:

```go
func AgentFingerprint(hostname, os, kernel string, macs []string, cpuModel string) string
```

## Provisioning Flow

```
┌─────────────┐    RegistrationRequest     ┌──────────────────┐
│  New Agent  │  ──────────────────────▶  │  Provisioning     │
│             │    (agent_type, hash,     │  Agent            │
│             │     hostname, OS, arch)   │                    │
│             │  ◀──────────────────────  │  Validates hash   │
│             │    (NATS token,           │  against known     │
│             │     Vault AppRole)        │  binary fingerprints│
└─────────────┘                           └──────────────────┘
```

## Implementation References

- **Fingerprint generation**: `internal/security/blake3_fingerprint.go`
- **Binary validation**: `agents/provisioning/src/fingerprint.rs`
- **Registration flow**: `agents/provisioning/src/registration.rs`
- **Install scripts**: `agents/provisioning/src/install_script.rs` (Linux/Windows/macOS)
- **OTA updates**: `agents/watchdog/src/tuf_updater.rs` (TUF + blake3 verification)
