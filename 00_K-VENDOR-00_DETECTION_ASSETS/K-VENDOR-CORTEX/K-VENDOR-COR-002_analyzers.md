# K-VENDOR-COR-002 -- Cortex Analyzers

| Field       | Value                                        |
|-------------|----------------------------------------------|
| Category    | Observable enrichment and threat intelligence |
| Format      | Cortex Analyzer JSON jobs via REST API        |
| Consumer    | KAI-TRIAGE, KAI-ANALYST                      |

## Purpose

Cortex analyzers perform automated enrichment of observables extracted
from alerts. Kubric submits analysis jobs via the Cortex REST API and
consumes the structured JSON reports returned.

## Key Analyzers Used by Kubric

| Analyzer               | Observable Type  | Enrichment Provided          |
|------------------------|------------------|------------------------------|
| VirusTotal_GetReport   | hash, url        | AV detection ratio, sandbox  |
| AbuseIPDB              | ip               | Abuse confidence score       |
| Shodan_Host            | ip               | Open ports, banners, CVEs    |
| DomainMailSPFDMARC     | domain           | SPF/DMARC policy validation  |
| MISP_2_1               | any              | MISP event correlation       |
| OTXQuery               | ip, domain, hash | AlienVault OTX pulse match   |
| URLhaus                | url, domain      | Malware URL database lookup  |

## Integration Flow

1. KAI-TRIAGE extracts observables from an incoming alert.
2. POSTs each observable to `https://cortex-{tenant}/api/job`.
3. Polls `api/job/{id}/waitreport` until the analyzer completes.
4. Merges enrichment results into the alert severity score.
5. Publishes enriched alert to `kubric.kai.triage.enriched`.

## Notes

- Analyzer API keys are injected from Vault per-tenant.
- Rate limits are enforced by Cortex; Kubric does not bypass them.
