# K-VENDOR-YAR-001 -- YARA Rules Index

## Overview

YARA is a pattern-matching engine for identifying and classifying malware samples. Kubric vendors community YARA rules from the Yara-Rules project and uses the **yara-x** Rust crate (BSD-2) for native in-process scanning.

## Source

- **Repository**: `https://github.com/Yara-Rules/rules.git`
- **Format**: `.yar` / `.yara` files
- **License**: BSD / Apache 2.0 mix (per-file headers)
- **Sync script**: `scripts/vendor-pull.sh yara`

## Kubric Integration

| Consumer | Integration Method | Notes |
|---|---|---|
| **NetGuard** (Rust agent) | `yara_x::Compiler` + `Scanner` in `ids.rs` | Scans TCP/UDP payloads for malicious byte patterns |
| **CoreSec** (Rust agent) | FIM-triggered file scans | When FIM detects a new/modified file, YARA scans it for malware signatures |
| **KAI-Analyst** (Python) | References matched rule names in reports | Maps YARA hits to malware families for investigation context |
| **VDR** (Go service) | Indexes rule metadata | Tracks rule counts and last-update timestamps |

## License Notes

The `yara-x` crate is BSD-2 licensed, allowing direct Rust dependency. Vendor rule files are BSD/Apache 2.0 -- no copyleft restrictions apply to their use as data.

## Document Map

| Doc ID | Title |
|---|---|
| YAR-002 | Malware Signature Rules |
| YAR-003 | PII / Sensitive Data Rules |
