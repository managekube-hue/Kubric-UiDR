# K-VENDOR-SUR-004 -- Emerging Web Attack Rules

## Category

The `emerging-web_server.rules` and `emerging-web_specific_apps.rules` families detect attacks targeting web applications and servers. These rules cover SQLi, XSS, LFI/RFI, SSRF, deserialization exploits, and CVE-specific payloads.

## Coverage Areas

| Sub-category | Examples |
|---|---|
| **SQL Injection** | Union-based, blind, error-based SQLi patterns |
| **Cross-Site Scripting** | Reflected/stored XSS payloads in headers, params, body |
| **Path Traversal / LFI** | `../etc/passwd`, `....//` bypasses, null byte injection |
| **Remote Code Execution** | Log4Shell, Spring4Shell, MOVEit, Confluence OGNL |
| **SSRF** | Cloud metadata endpoint access (`169.254.169.254`) |
| **Deserialization** | Java/PHP/Python object injection patterns |

## Rule Structure

```
alert http $EXTERNAL_NET any -> $HOME_NET any (
    msg:"ET WEB_SERVER SQL Injection SELECT FROM in URI";
    flow:established,to_server;
    http.uri; content:"SELECT"; nocase;
    content:"FROM"; nocase; distance:0;
    sid:2031200; rev:4;
    classtype:web-application-attack;
)
```

## Kubric Integration

NetGuard inspects HTTP payloads (method, URI, headers, body) against these rules. The Go-based **K-SVC** API gateway also leverages these patterns for its WAF layer (OWASP CRS + ET web rules). Detections generate:

- OCSF `HttpActivity` events on `kubric.{tenant}.detection.web.v1`
- Automatic correlation with Nuclei CVE template IDs (see NUC-002)
- KAI-Analyst context: attacker IP geolocation, target endpoint, payload hash

## MITRE ATT&CK Mapping

- T1190 -- Exploit Public-Facing Application
- T1059.007 -- JavaScript execution via XSS
- T1505.003 -- Web Shell (post-exploitation indicators)

## Update Cadence

Critical CVE signatures are typically added within 24 hours by the ET community. Run `scripts/vendor-pull.sh suricata` to sync.
