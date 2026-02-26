# K-VENDOR-ZEK-002 -- Zeek Base Protocol Logs

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Network protocol metadata extraction           |
| Format      | Zeek TSV / JSON structured logs                |
| Consumer    | NetGuard agent, KAI-ANALYST                    |

## Purpose

Zeek base protocol analyzers generate structured logs for every
connection and supported protocol. NetGuard parses these logs for
baseline network visibility and anomaly detection.

## Key Log Types

| Log File    | Protocol  | Key Fields                              |
|-------------|-----------|-----------------------------------------|
| conn.log    | TCP/UDP   | duration, bytes, state, service         |
| dns.log     | DNS       | query, qtype, rcode, answers            |
| http.log    | HTTP      | method, uri, status, user_agent         |
| ssl.log     | TLS       | version, cipher, server_name, JA3       |
| smtp.log    | SMTP      | from, to, subject, tls                  |
| files.log   | File xfer | mime_type, md5, sha1, source            |
| notice.log  | Zeek      | Notice framework alerts                 |

## Integration Flow

1. NetGuard spawns Zeek on the monitored interface or PCAP capture.
2. Zeek produces JSON logs to a watched output directory.
3. NetGuard tails log files and publishes parsed events to NATS.
4. Connection metadata feeds NetGuard's DPI and RITA beacon analysis.
5. KAI-ANALYST queries stored Zeek logs for forensic investigations.

## Notes

- BSD-3-Clause license permits direct log parsing and script bundling.
- Log rotation is handled by Zeek's built-in LogRotationInterval.
