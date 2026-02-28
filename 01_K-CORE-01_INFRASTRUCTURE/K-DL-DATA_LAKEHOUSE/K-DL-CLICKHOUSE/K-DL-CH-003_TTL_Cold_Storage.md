# K-DL-CH-003 — TTL & Cold Storage Tiering

> **Applies To:** kubric_cluster (ClickHouse 24.8 LTS)  
> **Hot Tier:** NVMe-backed Ceph RBD (500 GB per shard)  
> **Cold Tier:** MinIO S3 at 10.0.50.23:9000  
> **Archive Bucket:** `kubric-ch-cold`  
> **Retention:** Hot 30d → Cold 90d → Delete 365d

---

## 1. Storage Policy Architecture

```
┌──────────────────────────────────────────────────────┐
│                  Data Lifecycle                       │
│                                                      │
│  Ingest → [Hot: NVMe/Ceph]  → [Cold: MinIO S3]  → Delete │
│           0-30 days            30-90 days          365d    │
│           Fast queries          Slow queries       Purged  │
│           ZSTD(1)              ZSTD(5)                     │
└──────────────────────────────────────────────────────┘
```

---

## 2. Configure S3 Cold Storage

### 2.1 MinIO Bucket Setup

```bash
# On MinIO node (10.0.50.23)
mc alias set kubric http://10.0.50.23:9000 kubric-admin "${MINIO_ROOT_PASS}"

# Create cold storage bucket
mc mb kubric/kubric-ch-cold
mc mb kubric/kubric-ch-cold-archive

# Create dedicated user for ClickHouse
mc admin user add kubric clickhouse-s3 "${CH_S3_SECRET}"
mc admin policy attach kubric readwrite --user clickhouse-s3

# Lifecycle policy: delete archived data after 365 days
mc ilm rule add kubric/kubric-ch-cold-archive --expire-days 365
```

### 2.2 ClickHouse S3 Disk Configuration

File: `/etc/clickhouse-server/config.d/storage.xml`

Apply on **both shards**:

```xml
<?xml version="1.0"?>
<clickhouse>
  <storage_configuration>
    <disks>
      <!-- Hot disk: local (Ceph RBD mounted at /var/lib/clickhouse) -->
      <hot>
        <type>local</type>
        <path>/var/lib/clickhouse/data/</path>
        <keep_free_space_bytes>10737418240</keep_free_space_bytes> <!-- 10 GB -->
      </hot>

      <!-- Cold disk: MinIO S3 -->
      <cold_s3>
        <type>s3</type>
        <endpoint>http://10.0.50.23:9000/kubric-ch-cold/data/</endpoint>
        <access_key_id>clickhouse-s3</access_key_id>
        <secret_access_key>CHANGEME_CH_S3_SECRET</secret_access_key>
        <metadata_path>/var/lib/clickhouse/disks/cold_s3/</metadata_path>
        <cache_enabled>true</cache_enabled>
        <cache_path>/var/lib/clickhouse/disks/cold_s3_cache/</cache_path>
        <data_cache_enabled>true</data_cache_enabled>
        <data_cache_max_size>10737418240</data_cache_max_size> <!-- 10 GB local cache -->
        <skip_access_check>false</skip_access_check>
        <send_metadata>true</send_metadata>
        <max_single_part_upload_size>33554432</max_single_part_upload_size> <!-- 32 MB -->
        <min_upload_part_size>16777216</min_upload_part_size> <!-- 16 MB -->
        <max_connections>50</max_connections>
      </cold_s3>

      <!-- Archive disk: MinIO S3 (higher compression, rarely queried) -->
      <archive_s3>
        <type>s3</type>
        <endpoint>http://10.0.50.23:9000/kubric-ch-cold-archive/data/</endpoint>
        <access_key_id>clickhouse-s3</access_key_id>
        <secret_access_key>CHANGEME_CH_S3_SECRET</secret_access_key>
        <metadata_path>/var/lib/clickhouse/disks/archive_s3/</metadata_path>
        <cache_enabled>false</cache_enabled>
        <skip_access_check>false</skip_access_check>
        <send_metadata>true</send_metadata>
      </archive_s3>
    </disks>

    <policies>
      <!-- Tiered policy: hot → cold → archive -->
      <tiered>
        <volumes>
          <hot_volume>
            <disk>hot</disk>
            <max_data_part_size_bytes>1073741824</max_data_part_size_bytes> <!-- 1 GB max part -->
          </hot_volume>
          <cold_volume>
            <disk>cold_s3</disk>
            <prefer_not_to_merge>true</prefer_not_to_merge>
          </cold_volume>
          <archive_volume>
            <disk>archive_s3</disk>
            <prefer_not_to_merge>true</prefer_not_to_merge>
          </archive_volume>
        </volumes>
        <move_factor>0.2</move_factor>
      </tiered>

      <!-- S3-only policy for less critical data -->
      <s3_only>
        <volumes>
          <s3_volume>
            <disk>cold_s3</disk>
          </s3_volume>
        </volumes>
      </s3_only>
    </policies>
  </storage_configuration>
</clickhouse>
```

### 2.3 Restart ClickHouse

```bash
# On both shard nodes
sudo systemctl restart clickhouse-server

# Verify disks
clickhouse-client --query "
  SELECT name, path, type, formatReadableSize(free_space) AS free,
         formatReadableSize(total_space) AS total
  FROM system.disks"
```

Expected:

```
┌─name────────┬─path──────────────────────────────────────────┬─type──┬─free───┬─total──┐
│ hot         │ /var/lib/clickhouse/data/                      │ local │ 380 GB │ 500 GB │
│ cold_s3     │ http://10.0.50.23:9000/kubric-ch-cold/data/   │ s3    │ 0 B    │ 0 B    │
│ archive_s3  │ http://10.0.50.23:9000/kubric-ch-cold-archive/│ s3    │ 0 B    │ 0 B    │
└─────────────┴───────────────────────────────────────────────┴───────┴────────┴────────┘
```

---

## 3. TTL Policies

### 3.1 Modify Existing Tables with TTL + Storage Policy

```sql
-- Security events: hot 30d → cold 90d → delete 365d
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY SETTING storage_policy = 'tiered';

ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY TTL
    event_date + INTERVAL 30 DAY TO VOLUME 'cold_volume',
    event_date + INTERVAL 90 DAY TO VOLUME 'archive_volume',
    event_date + INTERVAL 365 DAY DELETE;

-- Agent metrics: hot 7d → cold 30d → delete 90d
ALTER TABLE kubric_telemetry.agent_metrics ON CLUSTER kubric_cluster
  MODIFY SETTING storage_policy = 'tiered';

ALTER TABLE kubric_telemetry.agent_metrics ON CLUSTER kubric_cluster
  MODIFY TTL
    metric_date + INTERVAL 7 DAY TO VOLUME 'cold_volume',
    metric_date + INTERVAL 30 DAY DELETE;

-- DNS logs: hot 14d → cold 60d → delete 180d
ALTER TABLE kubric_telemetry.dns_logs ON CLUSTER kubric_cluster
  MODIFY SETTING storage_policy = 'tiered';

ALTER TABLE kubric_telemetry.dns_logs ON CLUSTER kubric_cluster
  MODIFY TTL
    event_date + INTERVAL 14 DAY TO VOLUME 'cold_volume',
    event_date + INTERVAL 60 DAY TO VOLUME 'archive_volume',
    event_date + INTERVAL 180 DAY DELETE;

-- Network flows: hot 7d → cold 60d → delete 180d
ALTER TABLE kubric_telemetry.network_flows ON CLUSTER kubric_cluster
  MODIFY SETTING storage_policy = 'tiered';

ALTER TABLE kubric_telemetry.network_flows ON CLUSTER kubric_cluster
  MODIFY TTL
    flow_date + INTERVAL 7 DAY TO VOLUME 'cold_volume',
    flow_date + INTERVAL 60 DAY TO VOLUME 'archive_volume',
    flow_date + INTERVAL 180 DAY DELETE;
```

### 3.2 Column-Level TTL (Selective Data Reduction)

```sql
-- Drop raw_log after 60 days to save space while keeping metadata
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY COLUMN raw_log String
    TTL event_date + INTERVAL 60 DAY;

-- Drop process command lines after 30 days
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY COLUMN process_cmd Nullable(String)
    TTL event_date + INTERVAL 30 DAY;
```

---

## 4. TTL Merge Settings

```sql
-- Control TTL merge frequency
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY SETTING
    merge_with_ttl_timeout = 14400,             -- Check every 4 hours
    ttl_only_drop_parts = 1,                    -- Drop entire parts when all rows expired
    max_bytes_to_merge_at_max_space_in_pool = 161061273600;  -- 150 GB max merge
```

---

## 5. Compression Codec Tuning

```sql
-- Higher compression for cold data (applied on new inserts)
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY COLUMN raw_log String CODEC(ZSTD(5));

ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY COLUMN metadata String CODEC(ZSTD(5));

-- LowCardinality + Delta for timestamps
ALTER TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
  MODIFY COLUMN event_time DateTime64(3, 'UTC') CODEC(Delta, ZSTD(1));
```

---

## 6. Monitoring TTL & Storage

### 6.1 Check Part Locations

```sql
-- Which parts are on which disk
SELECT
    table,
    partition,
    name AS part_name,
    disk_name,
    formatReadableSize(bytes_on_disk) AS part_size,
    rows,
    modification_time
FROM system.parts
WHERE database = 'kubric_telemetry' AND active
ORDER BY modification_time;
```

### 6.2 TTL Status

```sql
-- Check TTL delete/move queues
SELECT
    database,
    table,
    result_part_name,
    elapsed,
    progress,
    result_part_path
FROM system.merges
WHERE is_mutation = 0
ORDER BY elapsed DESC;

-- Parts pending TTL move
SELECT
    database,
    table,
    count() AS parts_count,
    formatReadableSize(sum(bytes_on_disk)) AS total_size,
    min(modification_time) AS oldest_part,
    disk_name
FROM system.parts
WHERE active
GROUP BY database, table, disk_name
ORDER BY database, table, disk_name;
```

### 6.3 S3 Bucket Size

```bash
# Check cold storage usage
mc du kubric/kubric-ch-cold --depth 1
mc du kubric/kubric-ch-cold-archive --depth 1
```

---

## 7. Manual Data Movement

```sql
-- Force move specific partition to cold
ALTER TABLE kubric_telemetry.security_events
  MOVE PARTITION '202409' TO VOLUME 'cold_volume';

-- Force TTL merge (trigger TTL evaluation)
OPTIMIZE TABLE kubric_telemetry.security_events FINAL;

-- Move all expired parts
SYSTEM START TTL MERGES kubric_telemetry.security_events;
```

---

## 8. Backup Cold Data to External S3

```bash
# ClickHouse native backup to MinIO
clickhouse-client --query "
  BACKUP TABLE kubric_telemetry.security_events
  TO S3('http://10.0.50.23:9000/kubric-ch-cold/backups/security_events_$(date +%Y%m%d)',
        'clickhouse-s3', 'CHANGEME_CH_S3_SECRET')
  SETTINGS base_backup = S3('http://10.0.50.23:9000/kubric-ch-cold/backups/security_events_latest',
        'clickhouse-s3', 'CHANGEME_CH_S3_SECRET')
"
```

---

## 9. Verification

```bash
# Storage policies active
clickhouse-client --query "SELECT * FROM system.storage_policies"

# Disk free space
clickhouse-client --query "
  SELECT name, type, formatReadableSize(free_space), formatReadableSize(total_space)
  FROM system.disks"

# Parts by disk
clickhouse-client --query "
  SELECT disk_name, count() AS parts, formatReadableSize(sum(bytes_on_disk)) AS size
  FROM system.parts WHERE active
  GROUP BY disk_name"

# TTL rules
clickhouse-client --query "
  SELECT database, table, engine_full
  FROM system.tables
  WHERE database = 'kubric_telemetry'"

# S3 connectivity
clickhouse-client --query "
  SELECT * FROM system.disks WHERE type = 's3'"
```
