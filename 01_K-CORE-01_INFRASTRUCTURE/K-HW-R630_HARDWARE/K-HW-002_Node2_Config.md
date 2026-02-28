# K-HW-002 — Dell R630 Node 2 Hardware Configuration

> **Hostname:** pve-kubric-02  
> **Role:** K8s Worker + ClickHouse Shard 2 + PostgreSQL Primary  
> **iDRAC IP:** 10.0.100.12  
> **Management IP:** 10.0.100.22 (eno1)  
> **Storage IP:** 10.0.200.22 (eno2)  
> **Workload IP:** 10.0.50.22 (p3p1 — 10G SFP+)

---

## 1. iDRAC Initial Access & Credentials

### 1.1 First-Boot Access

```bash
# Connect to iDRAC dedicated NIC port (rear panel)
# Default: https://192.168.0.120  root / calvin
ip addr add 192.168.0.100/24 dev eth0
curl -k https://192.168.0.120/redfish/v1/Systems/System.Embedded.1 | jq .HostName
```

### 1.2 racadm Credential Setup

```bash
# Change default password
racadm -r 192.168.0.120 -u root -p calvin \
  set iDRAC.Users.2.Password "K8r1c!N0d3_02#iDR@C"

# Create kubric-admin account
racadm -r 192.168.0.120 -u root -p "K8r1c!N0d3_02#iDR@C" <<'EOF'
set iDRAC.Users.3.Enable Enabled
set iDRAC.Users.3.UserName kubric-admin
set iDRAC.Users.3.Password "${IDRAC_KUBRIC_PASS}"
set iDRAC.Users.3.Privilege 0x1ff
set iDRAC.Users.3.IpmiLanPrivilege Administrator
EOF
```

### 1.3 Set Production Network

```bash
racadm -r 192.168.0.120 -u root -p "K8r1c!N0d3_02#iDR@C" <<'EOF'
set iDRAC.IPv4.Address 10.0.100.12
set iDRAC.IPv4.Netmask 255.255.255.0
set iDRAC.IPv4.Gateway 10.0.100.1
set iDRAC.IPv4.DHCPEnable Disabled
set iDRAC.IPv4.DNS1 10.0.100.1
set iDRAC.NIC.VLanEnable Enabled
set iDRAC.NIC.VLanID 100
set iDRAC.NIC.DNSDomainName kubric.local
EOF
```

---

## 2. BIOS Configuration

### 2.1 Performance Profile (Identical to Node 1)

```bash
IDRAC="10.0.100.12"
CRED="-u kubric-admin -p ${IDRAC_KUBRIC_PASS}"

# Performance-optimized profile
racadm -r $IDRAC $CRED set BIOS.SysProfileSettings.SysProfile PerfOptimized

# Disable C-States
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcCStates Disabled
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcC1E Disabled

# Enable Turbo Boost
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcTurboMode Enabled

# Enable Hyperthreading
racadm -r $IDRAC $CRED set BIOS.ProcSettings.LogicalProc Enabled

# Enable SR-IOV
racadm -r $IDRAC $CRED set BIOS.IntegratedDevices.SriovGlobalEnable Enabled

# Enable VT-d / IOMMU
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcVirtualization Enabled

# Memory Optimizer + NUMA enabled
racadm -r $IDRAC $CRED set BIOS.MemSettings.MemOpMode OptimizerMode
racadm -r $IDRAC $CRED set BIOS.MemSettings.NodeInterleave Disabled

# UEFI boot
racadm -r $IDRAC $CRED set BIOS.BiosBootSettings.BootMode Uefi

# Commit and reboot
racadm -r $IDRAC $CRED jobqueue create BIOS.Setup.1-1
racadm -r $IDRAC $CRED serveraction powercycle
```

---

## 3. RAID Configuration

### 3.1 RAID 10 — OS Volume (2× 480GB SSD)

```bash
racadm -r $IDRAC $CRED storage createvd RAID.Integrated.1-1 \
  -rl r10 \
  -wp wb \
  -rp ra \
  -ss 256 \
  -pdkey:Disk.Bay.0:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.1:Enclosure.Internal.0-1:RAID.Integrated.1-1 \
  -name "OS_RAID10" \
  -size 480GB
```

### 3.2 RAID 5 — Data Volume (4× 2TB NVMe)

```bash
racadm -r $IDRAC $CRED storage createvd RAID.Integrated.1-1 \
  -rl r5 \
  -wp wb \
  -rp ara \
  -ss 512 \
  -pdkey:Disk.Bay.2:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.3:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.4:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.5:Enclosure.Internal.0-1:RAID.Integrated.1-1 \
  -name "DATA_RAID5" \
  -size max
```

### 3.3 Apply & Initialize

```bash
racadm -r $IDRAC $CRED jobqueue create RAID.Integrated.1-1
racadm -r $IDRAC $CRED serveraction powercycle
```

---

## 4. Network Configuration

### 4.1 Physical NIC Mapping

| Interface | NIC Model | IP Address | VLAN | Purpose |
|-----------|-----------|------------|------|---------|
| eno1 | Broadcom 5720 1GbE | 10.0.100.22/24 | native | Management / Proxmox API |
| eno2 | Broadcom 5720 1GbE | 10.0.200.22/24 | 200 | Storage / Ceph cluster |
| p3p1 | Intel X710 10G SFP+ | 10.0.50.22/24 | 50 | K8s pod traffic / workload |
| p3p2 | Intel X710 10G SFP+ | — | — | Bond slave / LACP pair |

### 4.2 Proxmox Network Configuration

File: `/etc/network/interfaces` on pve-kubric-02

```ini
auto lo
iface lo inet loopback

# Management — eno1
auto eno1
iface eno1 inet manual

auto vmbr0
iface vmbr0 inet static
    address 10.0.100.22/24
    gateway 10.0.100.1
    bridge-ports eno1
    bridge-stp off
    bridge-fd 0
    dns-nameservers 10.0.100.1 1.1.1.1
    dns-search kubric.local

# Storage — eno2 (Ceph cluster)
auto eno2
iface eno2 inet manual

auto vmbr1
iface vmbr1 inet static
    address 10.0.200.22/24
    bridge-ports eno2
    bridge-stp off
    bridge-fd 0
    mtu 9000

# Workload — Intel X710 10G SFP+ Bond
auto p3p1
iface p3p1 inet manual
    mtu 9000

auto p3p2
iface p3p2 inet manual
    mtu 9000

auto bond0
iface bond0 inet manual
    bond-slaves p3p1 p3p2
    bond-mode 802.3ad
    bond-miimon 100
    bond-lacp-rate fast
    bond-xmit-hash-policy layer3+4
    mtu 9000

auto vmbr2
iface vmbr2 inet static
    address 10.0.50.22/24
    bridge-ports bond0
    bridge-stp off
    bridge-fd 0
    mtu 9000
```

---

## 5. Proxmox VE Installation

```bash
# Install from Proxmox VE 8.x USB
# Target disk: OS_RAID10 virtual disk
# Filesystem: ext4
# Hostname: pve-kubric-02.kubric.local
# IP: 10.0.100.22/24, GW: 10.0.100.1, DNS: 10.0.100.1

# Post-install
sed -i 's/^deb/#deb/' /etc/apt/sources.list.d/pve-enterprise.list
echo "deb http://download.proxmox.com/debian/pve bookworm pve-no-subscription" \
  > /etc/apt/sources.list.d/pve-no-subscription.list
apt-get update && apt-get dist-upgrade -y

apt-get install -y htop iotop iperf3 ethtool lm-sensors net-tools \
  chrony smartmontools nvme-cli fio

hostnamectl set-hostname pve-kubric-02

cat >> /etc/hosts <<'EOF'
10.0.100.21  pve-kubric-01 pve-kubric-01.kubric.local
10.0.100.22  pve-kubric-02 pve-kubric-02.kubric.local
10.0.100.23  pve-kubric-03 pve-kubric-03.kubric.local
EOF
```

---

## 6. Storage Layout for Node 2

### 6.1 Partitioning the Data Volume

Node 2 hosts both ClickHouse Shard 2 and the PostgreSQL primary. Split the DATA_RAID5 volume:

```bash
# Identify DATA_RAID5 device
lsblk -o NAME,SIZE,MODEL
# Typically /dev/sdb (~6TB)

# Create partitions
parted /dev/sdb --script mklabel gpt
parted /dev/sdb --script mkpart clickhouse xfs 0% 65%
parted /dev/sdb --script mkpart postgres xfs 65% 85%
parted /dev/sdb --script mkpart ceph-osd 85% 100%

# Format ClickHouse partition
mkfs.xfs -f -L ch-data-s2 /dev/sdb1
mkdir -p /mnt/clickhouse-data
echo "LABEL=ch-data-s2 /mnt/clickhouse-data xfs defaults,noatime,nodiratime 0 2" >> /etc/fstab

# Format PostgreSQL partition
mkfs.xfs -f -L pg-data /dev/sdb2
mkdir -p /mnt/postgres-data
echo "LABEL=pg-data /mnt/postgres-data xfs defaults,noatime,nodiratime 0 2" >> /etc/fstab

# Leave sdb3 raw for Ceph OSD

mount -a
```

### 6.2 PostgreSQL Data Directory Permissions

```bash
chown -R 999:999 /mnt/postgres-data   # postgres container UID
chmod 700 /mnt/postgres-data
```

---

## 7. PostgreSQL Primary Configuration

### 7.1 Deploy via Docker (on Proxmox host or K8s pod)

```bash
docker run -d \
  --name kubric-postgres \
  --restart unless-stopped \
  -e POSTGRES_USER=kubric \
  -e POSTGRES_PASSWORD="${PG_KUBRIC_PASS}" \
  -e POSTGRES_DB=kubric_core \
  -v /mnt/postgres-data:/var/lib/postgresql/data \
  -p 5432:5432 \
  postgres:16-bookworm \
  -c shared_buffers=4GB \
  -c effective_cache_size=12GB \
  -c work_mem=256MB \
  -c maintenance_work_mem=1GB \
  -c max_connections=200 \
  -c wal_level=replica \
  -c max_wal_senders=5 \
  -c wal_keep_size=2GB \
  -c ssl=on
```

**Environment Variable:**
```
PG_KUBRIC_PASS=<generate with: openssl rand -base64 24>
```

### 7.2 Create Kubric Databases

```sql
-- Connect: psql -h 10.0.100.22 -U kubric -d kubric_core
CREATE DATABASE kubric_soc;
CREATE DATABASE kubric_psa;
CREATE DATABASE kubric_grc;

-- Create application roles
CREATE ROLE ksvc_app LOGIN PASSWORD '${KSVC_DB_PASS}';
CREATE ROLE kai_app LOGIN PASSWORD '${KAI_DB_PASS}';
CREATE ROLE vdr_app LOGIN PASSWORD '${VDR_DB_PASS}';

GRANT CONNECT ON DATABASE kubric_core TO ksvc_app;
GRANT USAGE ON SCHEMA public TO ksvc_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ksvc_app;
```

---

## 8. Verification Checklist

```bash
# BIOS
racadm -r 10.0.100.12 $CRED get BIOS.ProcSettings.ProcCStates   # Disabled
racadm -r 10.0.100.12 $CRED get BIOS.ProcSettings.ProcTurboMode  # Enabled

# RAID
racadm -r 10.0.100.12 $CRED storage get vdisks
# OS_RAID10 (Optimal), DATA_RAID5 (Optimal)

# Network
ip addr show vmbr0 | grep "10.0.100.22"
ip addr show vmbr1 | grep "10.0.200.22"
ip addr show vmbr2 | grep "10.0.50.22"
cat /proc/net/bonding/bond0 | grep "MII Status"

# Storage mounts
df -h /mnt/clickhouse-data /mnt/postgres-data
# Verify XFS
xfs_info /mnt/clickhouse-data

# PostgreSQL
docker exec kubric-postgres pg_isready
psql -h 10.0.100.22 -U kubric -d kubric_core -c "SELECT version();"

# Memory
free -g   # ~189 GB

# CPU
nproc     # 56 (28 cores × 2 threads)
```

---

## 9. Node 2 Role Assignment Summary

| Component | Resource Allocation | Notes |
|-----------|-------------------|-------|
| K8s Worker Node | 8 vCPU, 16 GB RAM | kubelet, kube-proxy |
| ClickHouse Shard 2 | 8 vCPU, 64 GB RAM | Replica pair with Node 1 |
| PostgreSQL Primary | 4 vCPU, 16 GB RAM | kubric_core, kubric_soc, kubric_psa |
| Ceph OSD | — | Raw partition ~900 GB |
| Kubric Services | 4 vCPU, 8 GB RAM | VDR, NOC containers |
| **Total Reserved** | **24 vCPU, 104 GB** | Leaves headroom for burst |
