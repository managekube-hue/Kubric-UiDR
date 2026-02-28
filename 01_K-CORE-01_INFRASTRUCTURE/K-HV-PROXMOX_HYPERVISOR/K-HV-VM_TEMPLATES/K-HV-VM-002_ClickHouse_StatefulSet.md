# K-HV-VM-002 — ClickHouse StatefulSet VM Template

> **Template VM ID:** 9002  
> **Base Image:** Ubuntu 24.04 LTS cloud image  
> **vCPU:** 8 | **RAM:** 32 GB | **OS Disk:** 50 GB | **Data Disk:** 500 GB (Ceph RBD)  
> **Target Nodes:** pve-kubric-01 (shard 1), pve-kubric-02 (shard 2)  
> **Network:** vmbr2 (VLAN 50 — Workload), vmbr1 (VLAN 200 — Storage replication)  
> **Ports:** 8123 (HTTP), 9000 (native TCP), 9440 (native TLS), 9363 (Prometheus)

---

## 1. Create VM Template

```bash
qm create 9002 --name clickhouse-template --ostype l26 \
  --cpu cputype=host --cores 8 --sockets 1 \
  --memory 32768 --balloon 16384 \
  --numa 1 \
  --net0 virtio,bridge=vmbr2,tag=50 \
  --net1 virtio,bridge=vmbr1,tag=200 \
  --scsihw virtio-scsi-single --machine q35 \
  --agent enabled=1

# Import OS disk
qm importdisk 9002 /var/lib/vz/template/iso/noble-server-cloudimg-amd64.raw kubric-ceph
qm set 9002 --scsi0 kubric-ceph:vm-9002-disk-0,iothread=1,discard=on,ssd=1
qm resize 9002 scsi0 50G

# Add data disk (500 GB for ClickHouse data)
qm set 9002 --scsi1 kubric-ceph:500,iothread=1,discard=on,ssd=1

# Cloud-init drive
qm set 9002 --ide2 kubric-ceph:cloudinit
qm set 9002 --boot order=scsi0
qm set 9002 --serial0 socket --vga serial0

# Cloud-init defaults
qm set 9002 --ciuser kubric --sshkeys /root/.ssh/kubric_deploy.pub
qm set 9002 --nameserver 10.0.100.1 --searchdomain kubric.local
```

---

## 2. Cloud-Init User Data

File: `/var/lib/vz/snippets/clickhouse-cloud.yml`

```yaml
#cloud-config
hostname: clickhouse-${instance_id}
manage_etc_hosts: true
timezone: UTC

users:
  - name: kubric
    groups: [sudo]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... kubric-deploy

package_update: true
package_upgrade: true
packages:
  - curl
  - wget
  - htop
  - net-tools
  - jq
  - qemu-guest-agent
  - nvme-cli

disk_setup:
  /dev/sdb:
    table_type: gpt
    layout: true
    overwrite: false

fs_setup:
  - label: clickhouse-data
    filesystem: xfs
    device: /dev/sdb1
    opts: "-f -L clickhouse-data"

mounts:
  - ["/dev/sdb1", "/var/lib/clickhouse", "xfs", "noatime,nodiratime", "0", "2"]

write_files:
  # ClickHouse repo
  - path: /etc/apt/sources.list.d/clickhouse.list
    permissions: "0644"
    content: |
      deb [signed-by=/usr/share/keyrings/clickhouse-keyring.gpg] https://packages.clickhouse.com/deb stable main

  # ClickHouse server config
  - path: /etc/clickhouse-server/config.d/kubric.xml
    permissions: "0644"
    content: |
      <?xml version="1.0"?>
      <clickhouse>
        <logger>
          <level>information</level>
          <log>/var/log/clickhouse-server/clickhouse-server.log</log>
          <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
          <size>500M</size>
          <count>5</count>
        </logger>

        <listen_host>0.0.0.0</listen_host>
        <http_port>8123</http_port>
        <tcp_port>9000</tcp_port>
        <tcp_port_secure>9440</tcp_port_secure>
        <interserver_http_port>9009</interserver_http_port>

        <!-- TLS -->
        <openSSL>
          <server>
            <certificateFile>/etc/clickhouse-server/certs/server.crt</certificateFile>
            <privateKeyFile>/etc/clickhouse-server/certs/server.key</privateKeyFile>
            <caConfig>/etc/clickhouse-server/certs/ca.crt</caConfig>
            <verificationMode>relaxed</verificationMode>
            <loadDefaultCAFile>true</loadDefaultCAFile>
          </server>
        </openSSL>

        <!-- Data paths -->
        <path>/var/lib/clickhouse/</path>
        <tmp_path>/var/lib/clickhouse/tmp/</tmp_path>
        <user_files_path>/var/lib/clickhouse/user_files/</user_files_path>
        <format_schema_path>/var/lib/clickhouse/format_schemas/</format_schema_path>

        <!-- Memory limits -->
        <max_server_memory_usage_to_ram_ratio>0.8</max_server_memory_usage_to_ram_ratio>
        <max_memory_usage>25769803776</max_memory_usage> <!-- 24 GB -->

        <!-- Merge tree -->
        <merge_tree>
          <max_suspicious_broken_parts>100</max_suspicious_broken_parts>
          <parts_to_delay_insert>150</parts_to_delay_insert>
          <parts_to_throw_insert>300</parts_to_throw_insert>
          <max_part_loading_threads>4</max_part_loading_threads>
          <min_bytes_for_wide_part>10485760</min_bytes_for_wide_part>
          <min_rows_for_wide_part>0</min_rows_for_wide_part>
        </merge_tree>

        <!-- Prometheus metrics -->
        <prometheus>
          <endpoint>/metrics</endpoint>
          <port>9363</port>
          <metrics>true</metrics>
          <events>true</events>
          <asynchronous_metrics>true</asynchronous_metrics>
        </prometheus>

        <!-- Background tasks -->
        <background_pool_size>8</background_pool_size>
        <background_schedule_pool_size>4</background_schedule_pool_size>
        <background_merges_mutations_concurrency_ratio>4</background_merges_mutations_concurrency_ratio>
      </clickhouse>

  # Cluster / replication config (per-shard, overridden post-clone)
  - path: /etc/clickhouse-server/config.d/cluster.xml
    permissions: "0644"
    content: |
      <?xml version="1.0"?>
      <clickhouse>
        <remote_servers>
          <kubric_cluster>
            <shard>
              <replica>
                <host>10.0.50.21</host>
                <port>9440</port>
                <secure>1</secure>
                <user>kubric_repl</user>
                <password>CHANGEME_CH_REPL</password>
              </replica>
            </shard>
            <shard>
              <replica>
                <host>10.0.50.22</host>
                <port>9440</port>
                <secure>1</secure>
                <user>kubric_repl</user>
                <password>CHANGEME_CH_REPL</password>
              </replica>
            </shard>
          </kubric_cluster>
        </remote_servers>

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
        </zookeeper>

        <macros>
          <cluster>kubric_cluster</cluster>
          <shard>SHARD_ID</shard>
          <replica>REPLICA_ID</replica>
        </macros>

        <distributed_ddl>
          <path>/clickhouse/task_queue/ddl</path>
        </distributed_ddl>
      </clickhouse>

  # ClickHouse users
  - path: /etc/clickhouse-server/users.d/kubric_users.xml
    permissions: "0640"
    content: |
      <?xml version="1.0"?>
      <clickhouse>
        <users>
          <default>
            <password_sha256_hex>CHANGEME_DEFAULT_HASH</password_sha256_hex>
            <networks><ip>::/0</ip></networks>
            <profile>default</profile>
            <quota>default</quota>
            <access_management>1</access_management>
          </default>

          <kubric_writer>
            <password_sha256_hex>CHANGEME_WRITER_HASH</password_sha256_hex>
            <networks>
              <ip>10.0.50.0/24</ip>
            </networks>
            <profile>writer</profile>
            <quota>default</quota>
            <allow_databases>
              <database>kubric_telemetry</database>
              <database>kubric_ocsf</database>
            </allow_databases>
          </kubric_writer>

          <kubric_reader>
            <password_sha256_hex>CHANGEME_READER_HASH</password_sha256_hex>
            <networks>
              <ip>10.0.50.0/24</ip>
            </networks>
            <profile>reader</profile>
            <quota>default</quota>
            <readonly>1</readonly>
          </kubric_reader>

          <kubric_repl>
            <password_sha256_hex>CHANGEME_REPL_HASH</password_sha256_hex>
            <networks>
              <ip>10.0.50.21</ip>
              <ip>10.0.50.22</ip>
              <ip>10.0.200.21</ip>
              <ip>10.0.200.22</ip>
            </networks>
            <profile>default</profile>
          </kubric_repl>
        </users>

        <profiles>
          <writer>
            <max_memory_usage>8589934592</max_memory_usage>
            <max_execution_time>300</max_execution_time>
            <max_insert_block_size>1048576</max_insert_block_size>
          </writer>
          <reader>
            <max_memory_usage>4294967296</max_memory_usage>
            <max_execution_time>60</max_execution_time>
            <readonly>1</readonly>
          </reader>
        </profiles>
      </clickhouse>

  # Sysctl tuning
  - path: /etc/sysctl.d/90-clickhouse.conf
    permissions: "0644"
    content: |
      vm.max_map_count = 262144
      vm.swappiness = 1
      net.core.somaxconn = 65535
      net.ipv4.tcp_max_syn_backlog = 65535
      fs.file-max = 2097152
      net.core.rmem_max = 16777216
      net.core.wmem_max = 16777216

  # Limits
  - path: /etc/security/limits.d/clickhouse.conf
    permissions: "0644"
    content: |
      clickhouse soft nofile 262144
      clickhouse hard nofile 262144
      clickhouse soft nproc 131072
      clickhouse hard nproc 131072

runcmd:
  # Install ClickHouse
  - |
    apt-get install -y apt-transport-https ca-certificates curl gnupg
    curl -fsSL 'https://packages.clickhouse.com/rpm/lts/repodata/repomd.xml.key' | \
      gpg --dearmor -o /usr/share/keyrings/clickhouse-keyring.gpg
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y clickhouse-server clickhouse-client clickhouse-common-static

  # Create data directories
  - mkdir -p /var/lib/clickhouse/{data,metadata,tmp,user_files,format_schemas}
  - chown -R clickhouse:clickhouse /var/lib/clickhouse

  # Create TLS certs directory
  - mkdir -p /etc/clickhouse-server/certs
  - chown clickhouse:clickhouse /etc/clickhouse-server/certs

  # Apply sysctl
  - sysctl --system

  # Enable services
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
  - systemctl enable clickhouse-server

final_message: "ClickHouse VM ready. Set macros/passwords, then: systemctl start clickhouse-server"
```

```bash
# Apply snippet
qm set 9002 --cicustom "user=local:snippets/clickhouse-cloud.yml"

# Convert to template
qm template 9002
```

---

## 3. Clone and Deploy Shards

### Shard 1 — Node 1

```bash
qm clone 9002 211 --name clickhouse-s1 --full --target pve-kubric-01 --storage kubric-ceph
qm set 211 \
  --ipconfig0 ip=10.0.50.21/24,gw=10.0.50.1 \
  --ipconfig1 ip=10.0.200.21/24
qm start 211

# Wait for cloud-init
ssh kubric@10.0.50.21 "cloud-init status --wait"

# Set shard macros
ssh kubric@10.0.50.21 "sudo sed -i \
  -e 's/SHARD_ID/01/' \
  -e 's/REPLICA_ID/clickhouse-s1/' \
  /etc/clickhouse-server/config.d/cluster.xml"

# Generate password hashes
CH_DEFAULT_HASH=$(echo -n "CHANGEME_DEFAULT" | sha256sum | head -c 64)
CH_WRITER_HASH=$(echo -n "CHANGEME_WRITER" | sha256sum | head -c 64)
CH_READER_HASH=$(echo -n "CHANGEME_READER" | sha256sum | head -c 64)
CH_REPL_HASH=$(echo -n "CHANGEME_REPL" | sha256sum | head -c 64)

ssh kubric@10.0.50.21 "sudo sed -i \
  -e \"s/CHANGEME_DEFAULT_HASH/${CH_DEFAULT_HASH}/\" \
  -e \"s/CHANGEME_WRITER_HASH/${CH_WRITER_HASH}/\" \
  -e \"s/CHANGEME_READER_HASH/${CH_READER_HASH}/\" \
  -e \"s/CHANGEME_REPL_HASH/${CH_REPL_HASH}/\" \
  /etc/clickhouse-server/users.d/kubric_users.xml"

# Start ClickHouse
ssh kubric@10.0.50.21 "sudo systemctl start clickhouse-server"
```

### Shard 2 — Node 2

```bash
qm clone 9002 212 --name clickhouse-s2 --full --target pve-kubric-02 --storage kubric-ceph
qm set 212 \
  --ipconfig0 ip=10.0.50.22/24,gw=10.0.50.1 \
  --ipconfig1 ip=10.0.200.22/24
qm start 212

ssh kubric@10.0.50.22 "cloud-init status --wait"
ssh kubric@10.0.50.22 "sudo sed -i \
  -e 's/SHARD_ID/02/' \
  -e 's/REPLICA_ID/clickhouse-s2/' \
  /etc/clickhouse-server/config.d/cluster.xml"

# Same password hashes as shard 1
ssh kubric@10.0.50.22 "sudo sed -i \
  -e \"s/CHANGEME_DEFAULT_HASH/${CH_DEFAULT_HASH}/\" \
  -e \"s/CHANGEME_WRITER_HASH/${CH_WRITER_HASH}/\" \
  -e \"s/CHANGEME_READER_HASH/${CH_READER_HASH}/\" \
  -e \"s/CHANGEME_REPL_HASH/${CH_REPL_HASH}/\" \
  /etc/clickhouse-server/users.d/kubric_users.xml"

ssh kubric@10.0.50.22 "sudo systemctl start clickhouse-server"
```

---

## 4. ClickHouse Keeper (ZooKeeper replacement)

Deploy ClickHouse Keeper on all 3 nodes for distributed coordination:

```bash
# On each node, add keeper config:
cat > /etc/clickhouse-keeper/keeper_config.xml <<'EOF'
<clickhouse>
  <keeper_server>
    <tcp_port>9181</tcp_port>
    <server_id>NODE_ID</server_id>
    <log_storage_path>/var/lib/clickhouse-keeper/log</log_storage_path>
    <snapshot_storage_path>/var/lib/clickhouse-keeper/snapshots</snapshot_storage_path>

    <coordination_settings>
      <operation_timeout_ms>10000</operation_timeout_ms>
      <session_timeout_ms>30000</session_timeout_ms>
      <raft_logs_level>warning</raft_logs_level>
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
EOF
```

---

## 5. Verification

```bash
# Cluster status
clickhouse-client --host 10.0.50.21 --port 9440 --secure \
  --user default --password "${CH_DEFAULT_PASS}" \
  --query "SELECT * FROM system.clusters WHERE cluster='kubric_cluster'"

# Keeper health
echo ruok | nc 10.0.50.21 9181   # Expected: imok

# Disk usage
clickhouse-client --query "SELECT formatReadableSize(total_space), formatReadableSize(free_space) FROM system.disks"

# Create test distributed table
clickhouse-client --host 10.0.50.21 --secure --query "
  CREATE DATABASE IF NOT EXISTS kubric_telemetry ON CLUSTER kubric_cluster;
"

# Prometheus metrics
curl -s http://10.0.50.21:9363/metrics | grep clickhouse_query
```
