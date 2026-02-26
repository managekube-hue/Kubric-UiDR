# K-VENDOR-ZEK-001 -- Zeek Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | The Zeek Project (CNCF / Corelight)         |
| License      | BSD-3-Clause                                |
| Integration  | Direct integration permitted (BSD)          |
| Consumers    | NetGuard agent, KAI-HUNTER, KAI-ANALYST      |

## Overview

Zeek is a network analysis framework that produces structured logs from
live traffic or PCAP files. Licensed under BSD-3-Clause, Zeek scripts
and log parsers may be directly integrated into Kubric components.

## Kubric Integration Points

- **NetGuard** runs Zeek as a subprocess on captured traffic and parses
  the resulting structured logs (conn.log, dns.log, http.log, etc.).
- **KAI-HUNTER** correlates Zeek logs with threat intelligence feeds
  for network-based hunt hypotheses.
- **KAI-ANALYST** uses Zeek protocol logs for deep-dive network
  forensic analysis during incident investigations.
- **Watchdog** manages Zeek binary versions and script package updates.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| ZEK-002        | Base Protocols     |
| ZEK-003        | Intel Framework    |
| ZEK-004        | HTTP Scripts       |
| ZEK-005        | JA3/TLS            |
