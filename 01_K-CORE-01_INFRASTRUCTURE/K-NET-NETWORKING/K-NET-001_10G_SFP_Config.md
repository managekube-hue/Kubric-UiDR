# K-NET-001 — 10G SFP+ Network Configuration

> **NIC:** Intel X710-DA2 Dual-Port 10G SFP+  
> **Cabling:** DAC (Direct Attach Copper) Twinax between nodes  
> **MTU:** 9000 (Jumbo Frames) for storage and workload traffic  
> **Bond:** 802.3ad LACP for redundancy

---

## 1. Hardware Inventory

### 1.1 NIC Details Per Node

| Node | NIC PCI Address | Port 1 (p3p1) | Port 2 (p3p2) | Firmware |
|------|----------------|----------------|----------------|----------|
| pve-kubric-01 | 0000:03:00.0 | 10.0.50.21/24 | Bond slave | 8.30 |
| pve-kubric-02 | 0000:03:00.0 | 10.0.50.22/24 | Bond slave | 8.30 |
| pve-kubric-03 | 0000:03:00.0 | 10.0.50.23/24 | Bond slave | 8.30 |

### 1.2 DAC Cable Topology

```
pve-kubric-01 (p3p1) ──────── 10G Switch Port 1
pve-kubric-01 (p3p2) ──────── 10G Switch Port 2
pve-kubric-02 (p3p1) ──────── 10G Switch Port 3
pve-kubric-02 (p3p2) ──────── 10G Switch Port 4
pve-kubric-03 (p3p1) ──────── 10G Switch Port 5
pve-kubric-03 (p3p2) ──────── 10G Switch Port 6

Alternative (no switch — direct attach):
pve-kubric-01 (p3p1) ──────── pve-kubric-02 (p3p1)
pve-kubric-01 (p3p2) ──────── pve-kubric-03 (p3p1)
pve-kubric-02 (p3p2) ──────── pve-kubric-03 (p3p2)
```

### 1.3 Verify NIC Detection

```bash
# On each node
lspci | grep -i x710
# 03:00.0 Ethernet controller: Intel Corporation Ethernet Controller X710 for 10GbE SFP+ (rev 02)
# 03:00.1 Ethernet controller: Intel Corporation Ethernet Controller X710 for 10GbE SFP+ (rev 02)

# Verify driver
ethtool -i p3p1
# driver: i40e
# version: 2.23.17
# firmware-version: 8.30

# Check link status
ethtool p3p1 | grep -E "Speed|Link"
# Speed: 10000Mb/s
# Link detected: yes
```

---

## 2. Intel X710 Firmware & Driver

### 2.1 Update i40e Driver

```bash
# Check current driver version
modinfo i40e | grep version

# Install from Intel (if needed)
wget https://downloadmirror.intel.com/772530/i40e-2.23.17.tar.gz
tar xzf i40e-2.23.17.tar.gz
cd i40e-2.23.17/src
make install
modprobe -r i40e && modprobe i40e
```

### 2.2 Update NVM Firmware

```bash
# Download Intel Ethernet NVM Update Tool
wget https://downloadmirror.intel.com/772530/700Series_NVMUpdatePackage_v8_30_Linux.tar.gz
tar xzf 700Series_NVMUpdatePackage_v8_30_Linux.tar.gz
cd 700Series/Linux_x64
./nvmupdate64e -u -l -o results.xml
reboot
```

---

## 3. MTU 9000 (Jumbo Frames) Configuration

### 3.1 Enable Jumbo Frames on Physical Ports

```bash
# Set MTU on each physical port
ip link set p3p1 mtu 9000
ip link set p3p2 mtu 9000

# Verify
ip link show p3p1 | grep mtu
# mtu 9000

# Persistent via /etc/network/interfaces (see Section 5)
```

### 3.2 Verify End-to-End Jumbo Frame Support

```bash
# Test from node 1 to node 2 (subtract 28 bytes for IP+ICMP headers)
ping -c 5 -M do -s 8972 10.0.50.22
# If successful: jumbo frames work end-to-end

# If "Message too long": check intermediate switch MTU
ping -c 5 -M do -s 8972 10.0.50.23
```

### 3.3 Switch MTU Configuration

```
! On 10G switch (Cisco Nexus / Dell S4048)
interface Ethernet1/1-6
  mtu 9216
  no shutdown
```

---

## 4. Bond Configuration (802.3ad LACP)

### 4.1 Load Bonding Module

```bash
modprobe bonding
echo "bonding" >> /etc/modules-load.d/bonding.conf
```

### 4.2 Bond Status Verification

```bash
# Check bond is active
cat /proc/net/bonding/bond0

# Expected output:
# Ethernet Channel Bonding Driver: v6.1.x
# Bonding Mode: IEEE 802.3ad Dynamic link aggregation
# Transmit Hash Policy: layer3+4 (1)
# MII Status: up
# MII Polling Interval (ms): 100
# LACP rate: fast
#
# Slave Interface: p3p1
# MII Status: up
# Speed: 10000 Mbps
# Duplex: full
# Aggregator ID: 1
#
# Slave Interface: p3p2
# MII Status: up
# Speed: 10000 Mbps
# Duplex: full
# Aggregator ID: 1
```

### 4.3 Switch LACP Configuration

```
! Switch port-channel for each node
! Node 1 — ports 1,2
interface port-channel 1
  description bond0-pve-kubric-01
  switchport mode trunk
  switchport trunk allowed vlan 50,200
  mtu 9216
  lacp mode active
  lacp rate fast

interface Ethernet1/1
  channel-group 1 mode active
  mtu 9216

interface Ethernet1/2
  channel-group 1 mode active
  mtu 9216

! Repeat for Node 2 (port-channel 2, Ethernet1/3-4)
! Repeat for Node 3 (port-channel 3, Ethernet1/5-6)
```

---

## 5. Linux Network Configuration

### 5.1 /etc/network/interfaces — Full 10G Section

This is the workload network section (complements vmbr0/vmbr1 from K-HW-001):

```ini
# ==============================================
# 10G SFP+ Workload Network — Intel X710 Bond
# ==============================================

# Physical port 1
auto p3p1
iface p3p1 inet manual
    mtu 9000
    # Disable offloads that cause issues with bonds
    up ethtool -K p3p1 lro off
    up ethtool -G p3p1 rx 4096 tx 4096

# Physical port 2
auto p3p2
iface p3p2 inet manual
    mtu 9000
    up ethtool -K p3p2 lro off
    up ethtool -G p3p2 rx 4096 tx 4096

# LACP Bond
auto bond0
iface bond0 inet manual
    bond-slaves p3p1 p3p2
    bond-mode 802.3ad
    bond-miimon 100
    bond-downdelay 200
    bond-updelay 200
    bond-lacp-rate fast
    bond-xmit-hash-policy layer3+4
    mtu 9000

# Proxmox bridge for workload VLAN 50
auto vmbr2
iface vmbr2 inet static
    address 10.0.50.2X/24       # 21 for node1, 22, 23
    bridge-ports bond0
    bridge-stp off
    bridge-fd 0
    mtu 9000
```

> Replace `10.0.50.2X` with node-specific IP: `.21`, `.22`, `.23`

---

## 6. VLAN Trunking for Workload Isolation

### 6.1 VLAN Definitions

| VLAN ID | Name | Subnet | Purpose |
|---------|------|--------|---------|
| 50 | workload | 10.0.50.0/24 | K8s pod network, inter-service |
| 51 | k8s-services | 10.0.51.0/24 | K8s ClusterIP service range |
| 55 | monitoring | 10.0.55.0/24 | Prometheus, Grafana, exporters |
| 60 | dmz | 10.0.60.0/24 | External-facing services (Caddy) |
| 200 | storage | 10.0.200.0/24 | Ceph, MinIO, NFS |

### 6.2 VLAN Sub-interfaces on Bond

```ini
# /etc/network/interfaces — VLAN sub-interfaces

# Monitoring VLAN
auto bond0.55
iface bond0.55 inet manual
    mtu 9000
    vlan-raw-device bond0

auto vmbr3
iface vmbr3 inet static
    address 10.0.55.2X/24
    bridge-ports bond0.55
    bridge-stp off
    bridge-fd 0
    mtu 9000

# DMZ VLAN
auto bond0.60
iface bond0.60 inet manual
    mtu 1500
    vlan-raw-device bond0

auto vmbr4
iface vmbr4 inet static
    address 10.0.60.2X/24
    bridge-ports bond0.60
    bridge-stp off
    bridge-fd 0
```

### 6.3 Apply

```bash
ifreload -a
# Verify VLANs
ip -d link show bond0.55
ip -d link show bond0.60
bridge vlan show
```

---

## 7. SR-IOV Virtual Functions (Optional)

### 7.1 Enable VFs on X710

```bash
# Create 8 VFs per port for VM direct NIC passthrough
echo 8 > /sys/class/net/p3p1/device/sriov_numvfs

# Persistent via udev
cat > /etc/udev/rules.d/70-sriov.rules <<'EOF'
ACTION=="add", SUBSYSTEM=="net", ENV{ID_NET_DRIVER}=="i40e", ATTR{device/sriov_numvfs}="8"
EOF

# Verify VFs
ip link show p3p1
# Should show 8 VFs listed

lspci | grep "Virtual Function"
# 03:02.0 Ethernet controller: Intel Corporation X710 VF
# ... (8 entries)
```

### 7.2 Assign VF to VM

```bash
# In Proxmox, add SR-IOV VF directly:
qm set <VMID> -hostpci1 03:02.0
# Provides near-native 10G performance to VM
```

---

## 8. Performance Tuning

### 8.1 Ring Buffer & Interrupt Coalescing

```bash
# Increase ring buffer
ethtool -G p3p1 rx 4096 tx 4096
ethtool -G p3p2 rx 4096 tx 4096

# Set interrupt coalescing for throughput
ethtool -C p3p1 adaptive-rx on adaptive-tx on rx-usecs 50 tx-usecs 50
ethtool -C p3p2 adaptive-rx on adaptive-tx on rx-usecs 50 tx-usecs 50

# CPU affinity — distribute interrupts across cores
# Set IRQ affinity for each queue
IRQS=$(grep p3p1 /proc/interrupts | awk '{print $1}' | tr -d ':')
CORE=0
for IRQ in $IRQS; do
    echo $CORE > /proc/irq/$IRQ/smp_affinity_list
    CORE=$((CORE + 1))
done
```

### 8.2 Kernel Tuning for 10G

```bash
cat >> /etc/sysctl.d/10g-tuning.conf <<'EOF'
# Network buffer sizes for 10G
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.core.rmem_default = 16777216
net.core.wmem_default = 16777216
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728

# Increase backlog for bursty traffic
net.core.netdev_max_backlog = 250000
net.core.somaxconn = 65535

# Enable TCP optimizations
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_no_metrics_save = 1
net.ipv4.tcp_congestion_control = bbr
net.core.default_qdisc = fq

# ARP tuning for large cluster
net.ipv4.neigh.default.gc_thresh1 = 4096
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh3 = 16384
EOF

sysctl -p /etc/sysctl.d/10g-tuning.conf
```

---

## 9. Verification & Benchmarking

### 9.1 iperf3 Throughput Test

```bash
# On node 2 (server)
iperf3 -s -B 10.0.50.22

# On node 1 (client)
iperf3 -c 10.0.50.22 -B 10.0.50.21 -t 30 -P 4
# Expected: ~18-19 Gbps (bonded) or ~9.3-9.4 Gbps (single port)

# With jumbo frames
iperf3 -c 10.0.50.22 -B 10.0.50.21 -t 30 -P 4 -M 8948
```

### 9.2 Latency Test

```bash
# Install sockperf
apt-get install -y sockperf

# Server (node 2)
sockperf sr --tcp -p 11111 -i 10.0.50.22

# Client (node 1) — ping-pong latency
sockperf pp --tcp -p 11111 -i 10.0.50.22 -t 10
# Expected: <20µs average for 10G DAC
```

### 9.3 Bond Failover Test

```bash
# Simulate port failure
ip link set p3p2 down

# Verify bond still works (single port)
ping -c 5 10.0.50.22
iperf3 -c 10.0.50.22 -t 5
# Expected: ~9.3 Gbps (single port)

# Restore
ip link set p3p2 up
cat /proc/net/bonding/bond0 | grep "MII Status"
```
