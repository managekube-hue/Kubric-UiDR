# K-VENDOR-MISP-005 -- MISP Object Templates

## Overview

MISP objects are structured JSON templates that define how related attributes are grouped together. They enforce a schema for complex indicators -- for example, grouping a malware sample's hash, filename, compilation timestamp, and C2 URL into a single logical object.

## Data Structure

```json
{
  "name": "file",
  "description": "File object",
  "version": 24,
  "attributes": {
    "filename":  { "misp-attribute": "filename",  "ui-priority": 0 },
    "md5":       { "misp-attribute": "md5",        "ui-priority": 1 },
    "sha256":    { "misp-attribute": "sha256",     "ui-priority": 2 },
    "size-in-bytes": { "misp-attribute": "size-in-bytes", "ui-priority": 3 }
  }
}
```

## Key Object Templates Used by Kubric

| Object | Attributes | Kubric Usage |
|---|---|---|
| **file** | filename, md5, sha256, size, mime-type | CoreSec FIM events, YARA match context |
| **ip-port** | ip, port, protocol, first-seen, last-seen | NetGuard flow records |
| **domain-ip** | domain, ip, registration-date | DNS resolution correlation |
| **email** | from, to, subject, attachment | Email-based IOC sharing |
| **vulnerability** | CVE ID, CVSS, summary, affected products | VDR vulnerability records |
| **x509** | serial, issuer, subject, validity | TLS certificate tracking |

## Kubric Integration

When VDR ingests IOCs from a MISP instance via REST API, the response contains MISP objects that group related attributes. VDR deserializes these using the object template schemas to:

1. Extract all attribute values into Kubric's internal IOC format
2. Preserve the relationships between attributes (e.g., a file hash linked to its C2 domain)
3. Map MISP object types to OCSF observable categories

KAI-Analyst uses object templates to structure its investigation reports. When multiple IOCs are related (same intrusion set), they are grouped using MISP object semantics before being published to `kubric.kai.analyst.report`.

## File Layout

```
vendor/misp/objects/
  file/definition.json
  ip-port/definition.json
  domain-ip/definition.json
  ...
```

Each object type has its own directory containing a `definition.json` schema file.
