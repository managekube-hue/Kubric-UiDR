# K-SEC-002 — TPM Root of Trust

## Overview

Kubric uses Trusted Platform Module (TPM 2.0) hardware to establish a hardware root of trust for agent attestation, sealed secrets, and tamper-evident audit chains. On endpoints without TPM (VMs, containers), a software-based attestation fallback is used.

## Architecture

```
┌──────────────────────────────────────┐
│         TPM 2.0 Hardware             │
│  ┌──────────┐  ┌──────────────────┐  │
│  │ EK (cert) │  │ SRK (Storage)   │  │
│  └──────────┘  └──────────────────┘  │
│  ┌──────────┐  ┌──────────────────┐  │
│  │ AIK (att)│  │ PCR Registers    │  │
│  └──────────┘  └──────────────────┘  │
└──────────────┬───────────────────────┘
               │
               ▼
┌──────────────────────────────────────┐
│     CoreSec Agent (Rust)             │
│  • TPM2_Quote() → signed PCR values │
│  • TPM2_Seal() → sealed agent key   │
│  • TPM2_Unseal() → on valid boot    │
│  • Blake3 chain anchor in TPM PCR   │
└──────────────┬───────────────────────┘
               │ NATS (kubric.health.attestation.>)
               ▼
┌──────────────────────────────────────┐
│     NOC / Vault (Verification)       │
│  • Verify TPM2_Quote signature       │
│  • Check PCR measurements vs golden  │
│  • Issue short-lived agent certs     │
└──────────────────────────────────────┘
```

## TPM Operations

### 1. Agent Key Sealing

The agent's NATS client key is sealed to TPM PCR values. If the boot chain is tampered with (bootloader, kernel, agent binary), the key cannot be unsealed and the agent refuses to connect.

```
Boot sequence:
  PCR[0] = BIOS/UEFI measurement
  PCR[4] = bootloader measurement
  PCR[7] = kernel + initrd measurement
  PCR[14] = kubric agent binary measurement

TPM2_Seal(agent_nats_key, policy=PCR[0,4,7,14])
```

### 2. Remote Attestation

```
Agent                                     NOC Server
  │                                          │
  │──── TPM2_Quote(PCR[0,4,7,14], nonce) ──▶│
  │                                          │ Verify AIK signature
  │                                          │ Compare PCRs to golden values
  │◀──── Attestation result + short cert ────│
  │                                          │
```

### 3. Blake3 Chain Anchor

The first hash in the immutable audit chain is derived from TPM-sealed entropy:

```rust
let tpm_entropy = tpm2_get_random(32)?;
let chain_anchor = blake3::hash(&tpm_entropy);
// All subsequent hashes: blake3(prev_hash || event_bytes)
```

## Software Fallback (No TPM)

For VMs and containers without TPM:

| Platform | Fallback |
|----------|----------|
| Linux VM | `/dev/urandom` + kernel version hash |
| Docker container | Container ID + image digest |
| Windows VM | DPAPI + machine SID |
| macOS | Secure Enclave (T2/M-series) |

Software attestation provides weaker guarantees but still enables the blake3 chain and agent identity verification.

## Integration with Vault

Vault can verify TPM attestation before issuing agent certificates:

```hcl
# Vault policy: only TPM-attested agents get database credentials
path "database/creds/agent-*" {
  capabilities = ["read"]
  required_parameters = ["tpm_attestation"]
}
```

## PCR Golden Values

Golden PCR values are stored per agent version in Vault KV:

```
vault kv put secret/kubric/tpm/golden/coresec-0.2.0 \
  pcr0="sha256:abc..." \
  pcr4="sha256:def..." \
  pcr7="sha256:ghi..." \
  pcr14="sha256:jkl..."
```

Updated on every agent release by the CI/CD pipeline.
