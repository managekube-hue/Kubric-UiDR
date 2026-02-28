# K-VENDOR-ZEK-003 -- Zeek Intelligence Framework

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Network-level threat intelligence matching     |
| Format      | Zeek Intel framework (intel.dat + scripts)     |
| Consumer    | NetGuard agent, KAI-HUNTER                     |

## Purpose

Zeek's Intelligence Framework matches observed network indicators
(IPs, domains, URLs, hashes) against loaded threat intel feeds in
real time. NetGuard loads MISP and custom IOC exports into Zeek's
intel tables.

## Intel Indicator Types

| Zeek Intel Type         | Observable Example              |
|-------------------------|---------------------------------|
| Intel::ADDR             | Malicious IP addresses          |
| Intel::DOMAIN           | C2 domain names                 |
| Intel::URL              | Phishing/malware URLs           |
| Intel::FILE_HASH        | Malware SHA-256 hashes          |
| Intel::EMAIL            | Threat actor email addresses    |
| Intel::CERT_HASH        | Malicious TLS certificate hash  |

## Integration Flow

1. KAI-HUNTER exports IOCs from MISP to Zeek `intel.dat` format.
2. NetGuard loads the intel file into the running Zeek instance.
3. Zeek generates `intel.log` entries when indicators match traffic.
4. NetGuard publishes matches to `kubric.netguard.intel.hit` on NATS.
5. KAI-TRIAGE escalates high-confidence intel hits for triage.

## Feed Management

- Intel feeds are refreshed on a configurable interval (default 15min).
- Feed source attribution is preserved in the `sources` field.
- Stale indicators are aged out based on MISP event expiry dates.
- BSD-3-Clause license permits bundling custom Zeek intel scripts.
