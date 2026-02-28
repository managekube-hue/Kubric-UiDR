# K-HW-001 — Dell R630 Node 1 Hardware Configuration

> **Hostname:** pve-kubric-01  
> **Role:** Primary K8s control plane + ClickHouse Shard 1  
> **iDRAC IP:** 10.0.100.11  
> **Management IP:** 10.0.100.21 (eno1)  
> **Storage IP:** 10.0.200.21 (eno2)  
> **Workload IP:** 10.0.50.21 (p3p1 — 10G SFP+)

---

## 1. iDRAC Initial Access & Credentials

### 1.1 Default First-Boot Access

Connect laptop to iDRAC dedicated NIC port on rear panel. Default IP is `192.168.0.120`.

```bash
# Set static IP on laptop interface for initial access
ip addr add 192.168.0.100/24 dev eth0
ping -c 3 192.168.0.120

# Login via browser: https://192.168.0.120
# Default credentials: root / calvin
```

### 1.2 racadm Credential Rotation

```bash
# Install racadm CLI (from Dell OpenManage)
apt-get install -y srvadmin-idracadm7

# Change default password immediately
racadm -r 192.168.0.120 -u root -p calvin \
  set iDRAC.Users.2.Password "K8r1c!N0d3_01#iDR@C"

# Create Kubric admin account (UserID 3)
racadm -r 10.0.100.11 -u root -p "K8r1c!N0d3_01#iDR@C" <<'EOF'
set iDRAC.Users.3.Enable Enabled
set iDRAC.Users.3.UserName kubric-admin
set iDRAC.Users.3.Password "${IDRAC_KUBRIC_PASS}"
set iDRAC.Users.3.Privilege 0x1ff
set iDRAC.Users.3.IpmiLanPrivilege Administrator
EOF
```

**Environment Variable:**
```
IDRAC_KUBRIC_PASS=<generate with: openssl rand -base64 24>
```

### 1.3 Set Production iDRAC Network

```bash
racadm -r 192.168.0.120 -u root -p "K8r1c!N0d3_01#iDR@C" <<'EOF'
set iDRAC.IPv4.Address 10.0.100.11
set iDRAC.IPv4.Netmask 255.255.255.0
set iDRAC.IPv4.Gateway 10.0.100.1
set iDRAC.IPv4.DHCPEnable Disabled
set iDRAC.IPv4.DNSFromDHCP Disabled
set iDRAC.IPv4.DNS1 10.0.100.1
set iDRAC.NIC.VLanEnable Enabled
set iDRAC.NIC.VLanID 100
set iDRAC.NIC.DNSDomainName kubric.local
EOF
```

---

## 2. BIOS Configuration

### 2.1 Performance BIOS Profile

```bash
IDRAC="10.0.100.11"
CRED="-u kubric-admin -p ${IDRAC_KUBRIC_PASS}"

# System Profile → Performance
racadm -r $IDRAC $CRED set BIOS.SysProfileSettings.SysProfile PerfOptimized

# Disable C-States (prevent frequency scaling latency)
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcCStates Disabled
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcC1E Disabled

# Enable Turbo Boost
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcTurboMode Enabled

# Enable Hyperthreading (2x logical cores for K8s scheduler)
racadm -r $IDRAC $CRED set BIOS.ProcSettings.LogicalProc Enabled

# Enable SR-IOV for 10G NIC Virtual Functions
racadm -r $IDRAC $CRED set BIOS.IntegratedDevices.SriovGlobalEnable Enabled

# Enable IOMMU / VT-d for GPU passthrough readiness
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcVirtualization Enabled
racadm -r $IDRAC $CRED set BIOS.IntegratedDevices.IoatDmaEngine Enabled

# Memory Operating Mode → Optimizer (maximum bandwidth)
racadm -r $IDRAC $CRED set BIOS.MemSettings.MemOpMode OptimizerMode

# Node Interleaving → Disabled (NUMA-aware for ClickHouse)
racadm -r $IDRAC $CRED set BIOS.MemSettings.NodeInterleave Disabled

# Boot Mode → UEFI
racadm -r $IDRAC $CRED set BIOS.BiosBootSettings.BootMode Uefi

# Apply pending BIOS changes — requires reboot
racadm -r $IDRAC $CRED jobqueue create BIOS.Setup.1-1
```

### 2.2 Commit BIOS & Reboot

```bash
racadm -r $IDRAC $CRED serveraction powercycle
# Monitor job status:
racadm -r $IDRAC $CRED jobqueue view
```

---

## 3. RAID Configuration

### 3.1 RAID Controller Identification

```bash
# List storage controllers
racadm -r $IDRAC $CRED storage get controllers
# Expected: RAID.Integrated.1-1 (PERC H730P Mini)

# List physical disks
racadm -r $IDRAC $CRED storage get pdisks
```

### 3.2 RAID 10 — OS Volume (2× 480GB SSD)

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

**Result:** ~480 GB usable RAID 10 volume for Proxmox VE OS + local-lvm.

### 3.3 RAID 5 — Data Volume (4× 2TB NVMe)

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

**Result:** ~6 TB usable RAID 5 volume for ClickHouse data, Ceph OSD, and container storage.

### 3.4 Apply RAID Configuration

```bash
racadm -r $IDRAC $CRED jobqueue create RAID.Integrated.1-1
racadm -r $IDRAC $CRED serveraction powercycle
# Wait for RAID initialization (~2-4 hours for full init)
watch -n 30 "racadm -r $IDRAC $CRED storage get vdisks"
```

---

## 4. Network Configuration

### 4.1 Physical NIC Mapping

| Interface | NIC Model | IP Address | VLAN | Purpose |
|-----------|-----------|------------|------|---------|
| eno1 | Broadcom 5720 1GbE | 10.0.100.21/24 | native | Management / Proxmox API |
| eno2 | Broadcom 5720 1GbE | 10.0.200.21/24 | 200 | Storage / Ceph cluster |
| p3p1 | Intel X710 10G SFP+ | 10.0.50.21/24 | 50 | K8s pod traffic / workload |
| p3p2 | Intel X710 10G SFP+ | — | — | Bond slave / LACP pair |

### 4.2 Proxmox Network Configuration

File: `/etc/network/interfaces` on pve-kubric-01

```ini
# Loopback
auto lo
iface lo inet loopback

# Management — eno1 (Broadcom 1GbE Port 1)
auto eno1
iface eno1 inet manual

auto vmbr0
iface vmbr0 inet static
    address 10.0.100.21/24
    gateway 10.0.100.1
    bridge-ports eno1
    bridge-stp off
    bridge-fd 0
    dns-nameservers 10.0.100.1 1.1.1.1
    dns-search kubric.local

# Storage — eno2 (Broadcom 1GbE Port 2, VLAN 200)
auto eno2
iface eno2 inet manual

auto vmbr1
iface vmbr1 inet static
    address 10.0.200.21/24
    bridge-ports eno2
    bridge-stp off
    bridge-fd 0
    mtu 9000
#   Ceph cluster network — no default gateway

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
    address 10.0.50.21/24
    bridge-ports bond0
    bridge-stp off
    bridge-fd 0
    mtu 9000
```

### 4.3 Apply & Verify

```bash
ifreload -a
ip addr show vmbr0 | grep "inet "
ip addr show vmbr1 | grep "inet "
ip addr show vmbr2 | grep "inet "

# Verify bond
cat /proc/net/bonding/bond0

# Test connectivity
ping -c 3 10.0.100.1       # gateway
ping -c 3 10.0.200.22      # storage to node2
ping -c 3 -s 8972 10.0.50.22  # jumbo frame test to node2
```

---

## 5. Proxmox VE Installation

### 5.1 Install via ISO

```bash
# Write Proxmox VE 8.x ISO to USB
dd if=proxmox-ve_8.2-1.iso of=/dev/sdX bs=4M status=progress

# During install:
#   Target disk: OS_RAID10 virtual disk
#   Filesystem: ext4
#   Country/Timezone: UTC
#   Hostname: pve-kubric-01.kubric.local
#   IP: 10.0.100.21/24, GW: 10.0.100.1, DNS: 10.0.100.1
```

### 5.2 Post-Install Configuration

```bash
# Disable enterprise repo (unless licensed)
sed -i 's/^deb/#deb/' /etc/apt/sources.list.d/pve-enterprise.list

# Add no-subscription repo
echo "deb http://download.proxmox.com/debian/pve bookworm pve-no-subscription" \
  > /etc/apt/sources.list.d/pve-no-subscription.list

apt-get update && apt-get dist-upgrade -y

# Install utilities
apt-get install -y htop iotop iperf3 ethtool lm-sensors net-tools \
  chrony smartmontools nvme-cli fio

# Configure hostname
hostnamectl set-hostname pve-kubric-01

# Set /etc/hosts
cat >> /etc/hosts <<'EOF'
10.0.100.21  pve-kubric-01 pve-kubric-01.kubric.local
10.0.100.22  pve-kubric-02 pve-kubric-02.kubric.local
10.0.100.23  pve-kubric-03 pve-kubric-03.kubric.local
EOF
```

---

## 6. Storage Layout

### 6.1 LVM for OS Disk

```bash
# Proxmox installer creates:
#   /dev/sda1 — EFI System (512MB)
#   /dev/sda2 — root (remainder)
#
# local-lvm thin pool for VM images:
lvcreate -L 200G -n data pve
lvconvert --type thin-pool pve/data
```

### 6.2 Data Volume Preparation

```bash
# Format RAID 5 data volume for Ceph OSD or direct mount
# Identify the DATA_RAID5 disk:
lsblk -o NAME,SIZE,MODEL,SERIAL

# If raw for Ceph OSD — leave unformatted (Ceph uses raw block)
# If direct mount for ClickHouse:
mkfs.xfs -f -L ch-data /dev/sdb
mkdir -p /mnt/clickhouse-data
echo "LABEL=ch-data /mnt/clickhouse-data xfs defaults,noatime,nodiratime 0 2" >> /etc/fstab
mount -a
```

---

## 7. Verification Checklist

```bash
# BIOS Settings
racadm -r 10.0.100.11 $CRED get BIOS.ProcSettings.ProcCStates
# Expected: Disabled

racadm -r 10.0.100.11 $CRED get BIOS.ProcSettings.ProcTurboMode
# Expected: Enabled

racadm -r 10.0.100.11 $CRED get BIOS.IntegratedDevices.SriovGlobalEnable
# Expected: Enabled

# RAID Status
racadm -r 10.0.100.11 $CRED storage get vdisks
# Expected: OS_RAID10 (Optimal), DATA_RAID5 (Optimal)

# Network
ethtool eno1 | grep "Link detected"    # yes
ethtool p3p1 | grep "Speed"            # 10000Mb/s
cat /proc/net/bonding/bond0 | grep "MII Status"  # up

# Memory
free -g
# Expected: ~189 GB (192GB minus kernel reserved)

# CPU
lscpu | grep -E "Model name|Socket|Core|Thread"
# Expected: 2x Intel Xeon E5-2680 v4 (28 cores / 56 threads)

# Storage
pvesm status
# local, local-lvm should be active

# Proxmox cluster readiness
pvecm status
# Should show this node is standalone (before cluster join)
```

---

## 8. Node 1 Role Assignment Summary

| Component | Resource Allocation | Notes |
|-----------|-------------------|-------|
| K8s Control Plane | 4 vCPU, 8 GB RAM | etcd, kube-apiserver, scheduler |
| ClickHouse Shard 1 | 8 vCPU, 64 GB RAM | ReplicatedMergeTree, OCSF tables |
| Ceph OSD | 2 NVMe drives | Part of kubric-vms pool |
| Kubric API (ksvc) | 4 vCPU, 8 GB RAM | Go API server |
| Monitoring Stack | 2 vCPU, 4 GB RAM | Prometheus, node-exporter |
| **Total Reserved** | **20 vCPU, 84 GB** | Leaves headroom for burst |
