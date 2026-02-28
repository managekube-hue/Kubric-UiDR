# K-DL-CH-001 — ClickHouse Cluster Configuration

> **Cluster Name:** `kubric_cluster`  
> **Shards:** 2 (pve-kubric-01 + pve-kubric-02)  
> **Keeper Nodes:** 3 (all Proxmox hosts)  
> **Version:** ClickHouse 24.8 LTS  
> **Ports:** 8123 (HTTP), 9000 (native), 9440 (native TLS), 9009 (inter-server), 9181 (Keeper), 9363 (Prometheus)  
> **Storage:** XFS on Ceph RBD, 500 GB per shard

---

## 1. Cluster Architecture

```
┌──────────────────────────────────────────────────────────┐
│                   kubric_cluster                         │
│                                                          │
│  ┌─────────────────┐         ┌─────────────────┐        │
│  │  Shard 01        │         │  Shard 02        │        │
│  │  10.0.50.21      │         │  10.0.50.22      │        │
│  │  pve-kubric-01   │         │  pve-kubric-02   │        │
│  │  VM 211          │         │  VM 212          │        │
│  │                  │         │                  │        │
│  │  Databases:      │         │  Databases:      │        │
│  │   kubric_telemetry│         │   kubric_telemetry│        │
│  │   kubric_ocsf    │         │   kubric_ocsf    │        │
│  └─────────────────┘         └─────────────────┘        │
│                                                          │
│  ┌─────────────────────────────────────────────┐        │
│  │  ClickHouse Keeper (Raft consensus)          │        │
│  │  Node 1: 10.0.50.21:9181  (ID=1)            │        │
│  │  Node 2: 10.0.50.22:9181  (ID=2)            │        │
│  │  Node 3: 10.0.50.23:9181  (ID=3)            │        │
│  └─────────────────────────────────────────────┘        │
└──────────────────────────────────────────────────────────┘
```

---

## 2. Cluster Configuration

### 2.1 Remote Servers (Sharding)

File: `/etc/clickhouse-server/config.d/cluster.xml`

```xml
<?xml version="1.0"?>
<clickhouse>
  <remote_servers>
    <!-- 2-shard cluster, no replication (single replica per shard) -->
    <kubric_cluster>
      <shard>
        <weight>1</weight>
        <internal_replication>true</internal_replication>
        <replica>
          <host>10.0.50.21</host>
          <port>9440</port>
          <secure>1</secure>
          <user>kubric_repl</user>
          <password>CHANGEME_REPL_PASS</password>
        </replica>
      </shard>
      <shard>
        <weight>1</weight>
        <internal_replication>true</internal_replication>
        <replica>
          <host>10.0.50.22</host>
          <port>9440</port>
          <secure>1</secure>
          <user>kubric_repl</user>
          <password>CHANGEME_REPL_PASS</password>
        </replica>
      </shard>
    </kubric_cluster>
  </remote_servers>

  <distributed_ddl>
    <path>/clickhouse/task_queue/ddl</path>
    <profile>default</profile>
  </distributed_ddl>
</clickhouse>
```

### 2.2 Macros (Per-Node)

File: `/etc/clickhouse-server/config.d/macros.xml`

**Shard 1 (10.0.50.21):**

```xml
<?xml version="1.0"?>
<clickhouse>
  <macros>
    <cluster>kubric_cluster</cluster>
    <shard>01</shard>
    <replica>clickhouse-s1</replica>
  </macros>
</clickhouse>
```

**Shard 2 (10.0.50.22):**

```xml
<?xml version="1.0"?>
<clickhouse>
  <macros>
    <cluster>kubric_cluster</cluster>
    <shard>02</shard>
    <replica>clickhouse-s2</replica>
  </macros>
</clickhouse>
```

### 2.3 ClickHouse Keeper Configuration

File: `/etc/clickhouse-keeper/keeper_config.xml`

```xml
<?xml version="1.0"?>
<clickhouse>
  <keeper_server>
    <tcp_port>9181</tcp_port>
    <server_id>NODE_ID</server_id>  <!-- 1, 2, or 3 per host -->

    <log_storage_path>/var/lib/clickhouse-keeper/log</log_storage_path>
    <snapshot_storage_path>/var/lib/clickhouse-keeper/snapshots</snapshot_storage_path>

    <coordination_settings>
      <operation_timeout_ms>10000</operation_timeout_ms>
      <session_timeout_ms>30000</session_timeout_ms>
      <raft_logs_level>warning</raft_logs_level>
      <auto_forwarding>true</auto_forwarding>
      <shutdown_timeout>5000</shutdown_timeout>
      <startup_timeout>60000</startup_timeout>
    </coordination_settings>

    <raft_configuration>
      <server>
        <id>1</id>
        <hostname>10.0.50.21</hostname>
        <port>9234</port>
      </server>
      <server>
        <id>2</id>
        <hostname>10.0.50.22</hostname>
        <port>9234</port>
      </server>
      <server>
        <id>3</id>
        <hostname>10.0.50.23</hostname>
        <port>9234</port>
      </server>
    </raft_configuration>
  </keeper_server>
</clickhouse>
```

### 2.4 ZooKeeper Client Config (points to Keeper)

File: `/etc/clickhouse-server/config.d/zookeeper.xml`

```xml
<?xml version="1.0"?>
<clickhouse>
  <zookeeper>
    <node>
      <host>10.0.50.21</host>
      <port>9181</port>
    </node>
    <node>
      <host>10.0.50.22</host>
      <port>9181</port>
    </node>
    <node>
      <host>10.0.50.23</host>
      <port>9181</port>
    </node>
    <session_timeout_ms>30000</session_timeout_ms>
    <operation_timeout_ms>10000</operation_timeout_ms>
  </zookeeper>
</clickhouse>
```

---

## 3. Database & Table Creation

### 3.1 Create Databases

```sql
-- Run on any node (distributed DDL)
CREATE DATABASE IF NOT EXISTS kubric_telemetry ON CLUSTER kubric_cluster;
CREATE DATABASE IF NOT EXISTS kubric_ocsf ON CLUSTER kubric_cluster;
```

### 3.2 Core Telemetry Tables

```sql
-- ═══════════════════════════════════════════
-- Security Events (main table)
-- ═══════════════════════════════════════════
CREATE TABLE kubric_telemetry.security_events ON CLUSTER kubric_cluster
(
    -- Identity
    event_id        UUID DEFAULT generateUUIDv4(),
    tenant_id       UUID,
    
    -- Timestamps
    event_time      DateTime64(3, 'UTC'),
    ingest_time     DateTime64(3, 'UTC') DEFAULT now64(3),
    
    -- Source
    source_type     LowCardinality(String),   -- 'syslog', 'winlog', 'zeek', 'suricata', 'agent'
    source_host     String,
    source_ip       IPv4,
    
    -- Classification  
    severity        LowCardinality(String),   -- 'critical','high','medium','low','info'
    category        LowCardinality(String),   -- OCSF category
    class_uid       UInt32,                    -- OCSF class UID
    
    -- Network
    src_ip          Nullable(IPv4),
    dst_ip          Nullable(IPv4),
    src_port        Nullable(UInt16),
    dst_port        Nullable(UInt16),
    protocol        LowCardinality(Nullable(String)),
    
    -- Process
    process_name    Nullable(String),
    process_pid     Nullable(UInt32),
    process_cmd     Nullable(String),
    user_name       Nullable(String),
    
    -- Detection
    rule_name       Nullable(String),
    rule_id         Nullable(String),
    mitre_tactic    Nullable(String),
    mitre_technique Nullable(String),
    
    -- Payload
    raw_log         String CODEC(ZSTD(3)),
    metadata        String DEFAULT '{}' CODEC(ZSTD(3)),  -- JSON
    
    -- Partitioning / sorting
    event_date      Date DEFAULT toDate(event_time)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/security_events', '{replica}')
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, severity, source_type, event_time)
TTL event_date + INTERVAL 90 DAY DELETE
SETTINGS 
    index_granularity = 8192,
    merge_with_ttl_timeout = 86400,
    storage_policy = 'default';

-- Distributed view
CREATE TABLE kubric_telemetry.security_events_distributed ON CLUSTER kubric_cluster AS kubric_telemetry.security_events
ENGINE = Distributed('kubric_cluster', 'kubric_telemetry', 'security_events', sipHash64(tenant_id));


-- ═══════════════════════════════════════════
-- Agent Metrics (XRO agent telemetry)
-- ═══════════════════════════════════════════
CREATE TABLE kubric_telemetry.agent_metrics ON CLUSTER kubric_cluster
(
    agent_id        UUID,
    tenant_id       UUID,
    metric_time     DateTime64(3, 'UTC'),
    metric_name     LowCardinality(String),
    metric_value    Float64,
    labels          Map(String, String),
    metric_date     Date DEFAULT toDate(metric_time)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/agent_metrics', '{replica}')
PARTITION BY toYYYYMM(metric_date)
ORDER BY (tenant_id, agent_id, metric_name, metric_time)
TTL metric_date + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE kubric_telemetry.agent_metrics_distributed ON CLUSTER kubric_cluster AS kubric_telemetry.agent_metrics
ENGINE = Distributed('kubric_cluster', 'kubric_telemetry', 'agent_metrics', sipHash64(tenant_id));


-- ═══════════════════════════════════════════
-- DNS Logs
-- ═══════════════════════════════════════════
CREATE TABLE kubric_telemetry.dns_logs ON CLUSTER kubric_cluster
(
    event_id        UUID DEFAULT generateUUIDv4(),
    tenant_id       UUID,
    event_time      DateTime64(3, 'UTC'),
    src_ip          IPv4,
    query           String,
    query_type      LowCardinality(String),    -- A, AAAA, MX, TXT, CNAME
    response_code   LowCardinality(String),    -- NOERROR, NXDOMAIN, SERVFAIL
    answers         Array(String),
    is_suspicious   UInt8 DEFAULT 0,
    event_date      Date DEFAULT toDate(event_time)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/dns_logs', '{replica}')
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, src_ip, event_time)
TTL event_date + INTERVAL 60 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE kubric_telemetry.dns_logs_distributed ON CLUSTER kubric_cluster AS kubric_telemetry.dns_logs
ENGINE = Distributed('kubric_cluster', 'kubric_telemetry', 'dns_logs', sipHash64(tenant_id));


-- ═══════════════════════════════════════════
-- Network Flow (Zeek conn.log equivalent)
-- ═══════════════════════════════════════════
CREATE TABLE kubric_telemetry.network_flows ON CLUSTER kubric_cluster
(
    flow_id         UUID DEFAULT generateUUIDv4(),
    tenant_id       UUID,
    start_time      DateTime64(3, 'UTC'),
    end_time        DateTime64(3, 'UTC'),
    duration_ms     UInt64,
    src_ip          IPv4,
    dst_ip          IPv4,
    src_port        UInt16,
    dst_port        UInt16,
    protocol        LowCardinality(String),
    bytes_in        UInt64,
    bytes_out       UInt64,
    packets_in      UInt32,
    packets_out     UInt32,
    conn_state      LowCardinality(String),   -- Zeek: SF, S0, REJ, etc.
    service         LowCardinality(Nullable(String)),
    geo_src         LowCardinality(Nullable(String)),
    geo_dst         LowCardinality(Nullable(String)),
    flow_date       Date DEFAULT toDate(start_time)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/network_flows', '{replica}')
PARTITION BY toYYYYMM(flow_date)
ORDER BY (tenant_id, flow_date, dst_port, src_ip, start_time)
TTL flow_date + INTERVAL 60 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE kubric_telemetry.network_flows_distributed ON CLUSTER kubric_cluster AS kubric_telemetry.network_flows
ENGINE = Distributed('kubric_cluster', 'kubric_telemetry', 'network_flows', sipHash64(tenant_id));
```

### 3.3 Materialized Views

```sql
-- ═══════════════════════════════════════════
-- Hourly severity aggregation for dashboards
-- ═══════════════════════════════════════════
CREATE MATERIALIZED VIEW kubric_telemetry.severity_hourly ON CLUSTER kubric_cluster
ENGINE = ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/severity_hourly', '{replica}')
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, hour, severity, source_type)
AS SELECT
    tenant_id,
    toStartOfHour(event_time) AS hour,
    severity,
    source_type,
    count() AS event_count,
    uniqExact(source_host) AS unique_hosts
FROM kubric_telemetry.security_events
GROUP BY tenant_id, hour, severity, source_type;

-- ═══════════════════════════════════════════
-- Top talkers (network)
-- ═══════════════════════════════════════════
CREATE MATERIALIZED VIEW kubric_telemetry.top_talkers_hourly ON CLUSTER kubric_cluster
ENGINE = ReplicatedAggregatingMergeTree('/clickhouse/tables/{shard}/kubric_telemetry/top_talkers_hourly', '{replica}')
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, hour, src_ip)
AS SELECT
    tenant_id,
    toStartOfHour(start_time) AS hour,
    src_ip,
    sumState(bytes_in + bytes_out) AS total_bytes,
    countState() AS flow_count,
    uniqState(dst_ip) AS unique_destinations
FROM kubric_telemetry.network_flows
GROUP BY tenant_id, hour, src_ip;
```

---

## 4. Performance Settings

File: `/etc/clickhouse-server/config.d/performance.xml`

```xml
<?xml version="1.0"?>
<clickhouse>
  <!-- Memory -->
  <max_server_memory_usage_to_ram_ratio>0.8</max_server_memory_usage_to_ram_ratio>

  <!-- Background operations -->
  <background_pool_size>8</background_pool_size>
  <background_schedule_pool_size>4</background_schedule_pool_size>
  <background_merges_mutations_concurrency_ratio>4</background_merges_mutations_concurrency_ratio>

  <!-- MergeTree -->
  <merge_tree>
    <max_suspicious_broken_parts>100</max_suspicious_broken_parts>
    <parts_to_delay_insert>150</parts_to_delay_insert>
    <parts_to_throw_insert>300</parts_to_throw_insert>
    <max_part_loading_threads>4</max_part_loading_threads>
    <min_bytes_for_wide_part>10485760</min_bytes_for_wide_part>
  </merge_tree>

  <!-- Query limits -->
  <max_concurrent_queries>100</max_concurrent_queries>
  <max_concurrent_insert_queries>20</max_concurrent_insert_queries>

  <!-- Compression -->
  <compression>
    <case>
      <min_part_size>10737418240</min_part_size>
      <min_part_size_ratio>0.01</min_part_size_ratio>
      <method>zstd</method>
      <level>3</level>
    </case>
  </compression>
</clickhouse>
```

---

## 5. Verification

```bash
# Cluster status
clickhouse-client --host 10.0.50.21 --secure --port 9440 \
  -u default --password "${CH_PASS}" \
  --query "SELECT cluster, shard_num, replica_num, host_name, is_local
           FROM system.clusters WHERE cluster='kubric_cluster'"

# Keeper status
echo ruok | nc 10.0.50.21 9181    # imok
echo ruok | nc 10.0.50.22 9181    # imok
echo ruok | nc 10.0.50.23 9181    # imok
echo stat | nc 10.0.50.21 9181    # leader/follower info

# Table sizes
clickhouse-client --query "
  SELECT database, table, formatReadableSize(sum(bytes)) AS size, sum(rows) AS rows
  FROM system.parts
  WHERE active
  GROUP BY database, table
  ORDER BY sum(bytes) DESC"

# Test distributed insert
clickhouse-client --query "
  INSERT INTO kubric_telemetry.security_events_distributed
    (tenant_id, event_time, source_type, source_host, source_ip, severity, category, raw_log)
  VALUES
    ('11111111-1111-1111-1111-111111111111', now(), 'test', 'test-host', '10.0.0.1', 'info', 'test', 'test log entry')"

# Verify data distributed across shards
clickhouse-client --host 10.0.50.21 --query "SELECT count() FROM kubric_telemetry.security_events"
clickhouse-client --host 10.0.50.22 --query "SELECT count() FROM kubric_telemetry.security_events"

# Prometheus metrics
curl -s http://10.0.50.21:9363/metrics | grep -c clickhouse_
```
