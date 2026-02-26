# K-VENDOR-SUR-003 -- Emerging C2 Rules

## Category

The `emerging-command_and_control.rules` family detects command-and-control (C2) communications. These rules identify beaconing, tunneling protocols, and known C2 framework traffic patterns on the wire.

## Detection Techniques

| Technique | Description | Example Frameworks |
|---|---|---|
| **Beacon interval** | Periodic HTTP/S callbacks at fixed or jittered intervals | Cobalt Strike, Sliver, Brute Ratel |
| **DNS tunneling** | Encoded data in DNS TXT/CNAME queries | Iodine, dnscat2, Cobalt Strike DNS |
| **Certificate fingerprint** | Known self-signed or default TLS certs | Metasploit, Cobalt Strike default JA3/JA3S |
| **URI patterns** | Hardcoded C2 URI paths and parameters | Havoc, PoshC2, Mythic |
| **Protocol abuse** | C2 over ICMP, DoH, WebSocket, or gRPC | Merlin (gRPC), Sliver (mTLS) |

## Rule Structure

```
alert tls $HOME_NET any -> $EXTERNAL_NET any (
    msg:"ET C2 Cobalt Strike Default TLS Certificate";
    tls.cert_subject; content:"Major Cobalt Strike";
    sid:2030500; rev:2;
    classtype:trojan-activity;
    metadata:former_category MALWARE;
)
```

## Kubric Integration

NetGuard's TLS SNI parser (`tls.rs`) extracts `ServerName` from ClientHello messages. The extracted SNI is cross-referenced against C2 domain IOCs from MISP warninglists and ET rules. When a C2 signature fires, the alert is enriched with:

- JA3/JA3S fingerprint (computed by NetGuard DPI module)
- MITRE mapping: T1071 (Application Layer Protocol), T1573 (Encrypted Channel)
- EPSS score if a related CVE exists

Alerts publish to `kubric.{tenant}.detection.network_ids.v1` and escalate to KAI-Analyst when severity >= HIGH.

## Update Cadence

C2 rules update daily from ET Open. New C2 framework signatures are typically added within 48 hours of public disclosure.
