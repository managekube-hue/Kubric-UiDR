# K-VENDOR-ZEK-005 -- JA3/JA4 TLS Fingerprinting

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | TLS client/server fingerprint analysis         |
| Format      | Zeek JA3/JA4 script package + ssl.log fields   |
| Consumer    | NetGuard agent, KAI-HUNTER                     |

## Purpose

JA3 and JA4 TLS fingerprinting scripts generate deterministic hashes
of TLS ClientHello and ServerHello parameters. These fingerprints
identify malware C2 channels, even when traffic is encrypted, by
matching known-bad TLS handshake signatures.

## Fingerprint Types

| Type    | Description                          | Log Field         |
|---------|--------------------------------------|--------------------|
| JA3     | Client TLS fingerprint (MD5)         | ssl.log `ja3`      |
| JA3S    | Server TLS fingerprint (MD5)         | ssl.log `ja3s`     |
| JA4     | Client fingerprint (next-gen, tuple) | ssl.log `ja4`      |
| JA4S    | Server fingerprint (next-gen)        | ssl.log `ja4s`     |

## Known-Bad Fingerprint Matching

- NetGuard maintains a hash set of JA3/JA3S values associated with
  known malware families (Cobalt Strike, Metasploit, Sliver, etc.).
- Matches generate alerts on `kubric.netguard.tls.alert`.
- KAI-HUNTER uses JA3 anomaly clustering to detect unknown C2.

## Integration Flow

1. Zeek JA3 package is loaded via `@load ja3` in `local.zeek`.
2. JA3/JA4 hashes are appended to every `ssl.log` entry.
3. NetGuard compares hashes against the threat intel hash set.
4. Matches and anomalous clusters are published to NATS.
5. KAI-TRIAGE correlates JA3 hits with IP reputation data.

## Notes

- JA3 package is BSD-licensed (Salesforce open source).
- Fingerprint database is updated alongside MISP threat intel feeds.
