# K-VENDOR-NUC-003 -- Cloud Misconfiguration Templates

## Overview

Cloud templates detect misconfigurations in AWS, Azure, GCP, and other cloud platforms. These templates check for publicly exposed resources, overly permissive IAM policies, unencrypted storage, and other cloud-specific security issues.

## Template Structure

```yaml
id: aws-s3-bucket-public-read
info:
  name: AWS S3 Bucket Public Read Access
  author: projectdiscovery
  severity: high
  description: S3 bucket allows anonymous read access
  tags: cloud,aws,s3,misconfiguration

http:
  - method: GET
    path:
      - "https://{{bucket}}.s3.amazonaws.com/"
    matchers:
      - type: word
        words:
          - "<ListBucketResult"
        condition: and
      - type: status
        status:
          - 200
```

## Coverage by Cloud Provider

| Provider | Template Count (approx) | Key Checks |
|---|---|---|
| **AWS** | ~200 | S3 exposure, IAM misconfig, SG rules, RDS public, Lambda env vars |
| **Azure** | ~100 | Blob storage, NSG rules, Key Vault access, App Service auth |
| **GCP** | ~80 | GCS buckets, Firewall rules, IAM bindings, Cloud SQL flags |
| **Kubernetes** | ~60 | Dashboard exposure, etcd access, kubelet API, RBAC misconfig |
| **General** | ~40 | DNS zone transfer, CORS misconfig, exposed admin panels |

## Kubric Integration

VDR runs cloud templates against tenant infrastructure on a configurable schedule (default: weekly). The scan targets are derived from:

- **Asset inventory** -- Cloud resource ARNs/IDs from K-SVC tenant configuration
- **DNS enumeration** -- Subdomains discovered during reconnaissance
- **Certificate transparency** -- Domains found via CT log monitoring

Findings map to OSCAL controls:
- AWS S3 public access --> NIST 800-53 AC-3 (Access Enforcement)
- Unencrypted storage --> PCI DSS Req 3 (Protect Stored Data)
- Overly permissive IAM --> ISO 27001 A.8.5 (Secure Authentication)

## MITRE ATT&CK Mapping

- T1530 -- Data from Cloud Storage Object
- T1078.004 -- Valid Accounts: Cloud Accounts
- T1580 -- Cloud Infrastructure Discovery
