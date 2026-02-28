# K-VENDOR-NUC-005 -- SaaS Application Templates

## Overview

SaaS templates detect misconfigurations and vulnerabilities in third-party cloud applications used by tenant organizations. These templates check for exposed admin interfaces, insecure integrations, token leaks, and SaaS-specific attack surfaces.

## Coverage by SaaS Category

| Category | Applications | Example Checks |
|---|---|---|
| **Collaboration** | Slack, Teams, Confluence, Jira | Webhook exposure, public channels, open signups |
| **Dev platforms** | GitHub, GitLab, Bitbucket, Jenkins | Public repos, CI token leaks, pipeline injection |
| **Identity** | Okta, Auth0, Azure AD | SSO misconfiguration, MFA bypass, open registration |
| **Storage** | SharePoint, Google Drive, Box | Public sharing links, anonymous access |
| **Monitoring** | Grafana, Kibana, Datadog | Unauthenticated dashboards, API key exposure |
| **Email** | Exchange, Google Workspace | Open relay, SPF/DKIM/DMARC misconfiguration |

## Template Structure

```yaml
id: grafana-default-credentials
info:
  name: Grafana Default Credentials
  author: projectdiscovery
  severity: high
  description: Grafana instance using default admin/admin credentials
  tags: saas,grafana,default-login

http:
  - raw:
      - |
        POST /login HTTP/1.1
        Host: {{Hostname}}
        Content-Type: application/json

        {"user":"admin","password":"admin"}
    matchers:
      - type: word
        words:
          - '"message":"Logged in"'
```

## Kubric Integration

VDR runs SaaS templates against tenant external assets and known SaaS endpoints. Integration points:

1. **Asset discovery** -- K-SVC tenant profiles list provisioned SaaS applications
2. **Template targeting** -- VDR selects templates matching the tenant's SaaS inventory
3. **Result enrichment** -- Findings are tagged with the affected SaaS vendor and OSCAL control mapping
4. **Remediation guidance** -- KAI-Analyst generates vendor-specific remediation steps

SaaS scan results map to compliance frameworks:
- Exposed admin panels --> SOC 2 CC6.1 (Logical Access)
- Default credentials --> PCI DSS Req 8 (Identify & Authenticate)
- Public data shares --> ISO 27001 A.8.12 (Data Leakage Prevention)

## MITRE ATT&CK Mapping

- T1199 -- Trusted Relationship (SaaS supply chain)
- T1078.004 -- Valid Accounts: Cloud Accounts
- T1213 -- Data from Information Repositories
