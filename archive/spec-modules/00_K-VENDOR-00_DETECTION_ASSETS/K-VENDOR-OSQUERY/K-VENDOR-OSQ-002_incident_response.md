# K-VENDOR-OSQ-002 -- osquery Incident Response Queries

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Endpoint forensic data collection              |
| Format      | osquery SQL scheduled queries (JSON packs)     |
| Consumer    | KAI-INVEST, KAI-HUNTER                         |

## Purpose

Pre-built osquery SQL packs for incident response. KAI-INVEST triggers
these queries to collect volatile endpoint state during active incidents.

## Key IR Query Categories

| Query Pack             | Tables Used                              |
|------------------------|------------------------------------------|
| Running processes      | `processes`, `process_open_sockets`      |
| Network connections    | `listening_ports`, `socket_events`       |
| Loaded modules         | `kernel_modules`, `drivers`              |
| Startup items          | `startup_items`, `scheduled_tasks`       |
| User accounts          | `users`, `logged_in_users`               |
| Installed software     | `programs`, `deb_packages`, `rpm_packages`|
| Browser extensions     | `chrome_extensions`, `firefox_addons`    |
| Open file handles      | `process_open_files`                     |

## Integration Flow

1. KAI-INVEST receives incident trigger on NATS.
2. Submits SQL queries to osqueryd via the Thrift/HTTP distributed API.
3. Collects JSON result sets for each query.
4. Normalizes results into forensic evidence records.
5. Publishes evidence chain to `kubric.kai.invest.evidence`.

## Example Query

```sql
SELECT pid, name, path, cmdline, uid, start_time
FROM processes
WHERE on_disk = 0;  -- processes running from deleted binaries
```

## Notes

- IR packs run on-demand only, never on a continuous schedule.
- Results are tagged with collection timestamp and agent ID.
