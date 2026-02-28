# K-VENDOR-THV-003 -- TheHive Alert Schema

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Pre-incident alert ingestion data model        |
| Format      | JSON objects via TheHive REST API              |
| Consumer    | KAI-TRIAGE, KAI-KEEPER                         |

## Purpose

TheHive alerts represent unconfirmed security events awaiting triage.
KAI-TRIAGE creates alerts in TheHive after enrichment; analysts or
KAI-KEEPER promote confirmed alerts to cases.

## Alert Object Fields

| Field            | Type     | Description                          |
|------------------|----------|--------------------------------------|
| title            | string   | Alert summary line                   |
| description      | string   | Detailed event narrative             |
| severity         | int      | 1 (Low) to 4 (Critical)             |
| type             | string   | Source type (wazuh, sigma, falco)    |
| source           | string   | Originating detection system         |
| sourceRef        | string   | Unique ID from source system         |
| tlp              | int      | Traffic Light Protocol level         |
| tags             | string[] | MITRE technique IDs, kill chain phase|
| artifacts        | object[] | Observables (IPs, hashes, domains)   |
| status           | string   | New, Updated, Ignored, Imported      |

## Integration Flow

1. KAI-TRIAGE completes enrichment and scoring of an event.
2. POSTs alert JSON to `https://thehive-{tenant}/api/v1/alert`.
3. Deduplication uses `type + source + sourceRef` as the unique key.
4. SOC analyst reviews, or KAI-KEEPER auto-promotes if confidence
   exceeds the tenant auto-promote threshold.
5. Promotion calls `POST /api/v1/alert/{id}/merge/{caseId}` to
   attach the alert to an existing or new case.

## Alert Sources

| Source          | Alert Type    | Volume Estimate     |
|-----------------|---------------|---------------------|
| Wazuh           | wazuh         | High                |
| Sigma/Suricata  | detection     | Medium              |
| Falco           | runtime       | Medium              |
| Nuclei          | vuln_scan     | Low (scheduled)     |
| Zeek Intel      | intel_match   | Variable            |
