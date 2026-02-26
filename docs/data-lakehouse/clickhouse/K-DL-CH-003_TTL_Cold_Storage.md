# K-DL-CH-003 — ClickHouse TTL Cold Storage

## Overview

ClickHouse uses TTL (Time-To-Live) expressions to automatically tier data from hot SSD storage to cold object storage (MinIO/S3). This reduces storage costs while maintaining query access to historical telemetry.

## Tiering Strategy

| Tier | Age | Storage | Compression | Use Case |
|------|-----|---------|-------------|----------|
| **Hot** | 0–7 days | NVMe SSD (local) | LZ4 | Real-time queries, alerting, dashboards |
| **Warm** | 7–30 days | SSD (replicated) | ZSTD(3) | Investigation, trending, weekly reports |
| **Cold** | 30–365 days | MinIO S3 (object) | ZSTD(9) | Compliance, forensics, annual review |
| **Archive** | >365 days | Deleted or external | — | Regulatory mandates only |

## TTL Configuration

```sql
-- Apply TTL policy to OCSF events table
ALTER TABLE ocsf_events
  MODIFY TTL
    event_time + INTERVAL 7 DAY TO VOLUME 'warm',
    event_time + INTERVAL 30 DAY TO VOLUME 'cold',
    event_time + INTERVAL 365 DAY DELETE;
```

## Storage Policy (config.xml)

```xml
<storage_configuration>
  <disks>
    <hot>
      <path>/var/lib/clickhouse/hot/</path>
    </hot>
    <warm>
      <path>/var/lib/clickhouse/warm/</path>
    </warm>
    <cold>
      <type>s3</type>
      <endpoint>http://minio.kubric.svc.cluster.local:9000/kubric-clickhouse-cold/</endpoint>
      <access_key_id>${MINIO_ACCESS_KEY}</access_key_id>
      <secret_access_key>${MINIO_SECRET_KEY}</secret_access_key>
    </cold>
  </disks>
  <policies>
    <kubric_tiered>
      <volumes>
        <hot>
          <disk>hot</disk>
          <max_data_part_size_bytes>1073741824</max_data_part_size_bytes>
        </hot>
        <warm>
          <disk>warm</disk>
          <max_data_part_size_bytes>10737418240</max_data_part_size_bytes>
        </warm>
        <cold>
          <disk>cold</disk>
        </cold>
      </volumes>
    </kubric_tiered>
  </policies>
</storage_configuration>
```

## Monitoring

```sql
-- Check current data distribution across tiers
SELECT
  disk_name,
  formatReadableSize(sum(bytes_on_disk)) AS size,
  count() AS parts
FROM system.parts
WHERE database = 'kubric' AND table = 'ocsf_events' AND active
GROUP BY disk_name;
```

## Per-Tenant Retention

Tenant-specific retention is enforced via `WHERE tenant_id = ...` materialized views that route to different TTL policies based on SLA tier.

| SLA Tier | Hot | Warm | Cold | Total Retention |
|----------|-----|------|------|-----------------|
| Starter | 3d | 14d | 90d | 90 days |
| Professional | 7d | 30d | 365d | 1 year |
| Enterprise | 14d | 90d | 730d | 2 years |
