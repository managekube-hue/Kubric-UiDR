# K-VENDOR-MISP-004 -- MISP Warninglists

## Overview

MISP warninglists are curated lists of known-good and known-bad indicators (IPs, domains, hostnames, CIDRs) used to suppress false positives or flag known-malicious infrastructure. They act as reputation feeds for IOC validation.

## Data Structure

```json
{
  "name": "List of known Microsoft Azure Datacenter IP Ranges",
  "version": 3,
  "description": "Azure IP ranges - known good, suppress FPs",
  "type": "cidr",
  "matching_attributes": ["ip-src", "ip-dst"],
  "list": [
    "13.64.0.0/11",
    "20.33.0.0/16"
  ]
}
```

## Key Warninglists Used by Kubric

| List Category | Purpose | Example Lists |
|---|---|---|
| **Cloud providers** | Suppress FPs for AWS, Azure, GCP IPs | `microsoft-azure`, `amazon-aws`, `google-gcp` |
| **CDNs** | Suppress FPs for Cloudflare, Akamai, Fastly | `cloudflare`, `akamai` |
| **Known resolvers** | Suppress FPs for public DNS resolvers | `google-dns`, `quad9` |
| **Disposable emails** | Flag throwaway email domains | `disposable-email-domains` |
| **Parking domains** | Flag parked/sinkholed domains | `dax-sinkholed-domains` |
| **TOR exit nodes** | Flag anonymizing infrastructure | `tor-exit-nodes` |

## Kubric Integration

NetGuard's `ipsum_lookup` module loads warninglists at startup and cross-references every flow's destination IP/domain. The integration works in two modes:

1. **Suppress mode** -- Cloud provider and CDN lists reduce false positives. An IP matching a known-good list lowers the alert confidence score.
2. **Flag mode** -- TOR exit nodes, disposable email domains, and sinkholed domains raise the alert confidence score and trigger escalation to KAI-Analyst.

VDR uses warninglists during MISP IOC ingestion to pre-filter indicators before inserting them into the threat intelligence database.

## Update Cadence

Warninglists are updated weekly by the MISP community. Cloud provider IP ranges change frequently; re-sync with `scripts/vendor-pull.sh misp` before production deployments.
