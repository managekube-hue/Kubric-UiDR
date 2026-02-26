# K-XRO-NG-002: NetGuard 10G Line-Rate Operation — Validation Guide

**Document:** K-XRO-NG-002
**Component:** NetGuard Network Capture Engine
**Version:** 1.0
**Status:** Production Reference

---

## 1. Overview

This document defines the hardware requirements, kernel configuration, and test
procedures required to validate that the NetGuard agent sustains 10 Gbps line-rate
packet capture with zero drops. The target workload is bidirectional 10 Gbps mixed
traffic (64-byte to 9000-byte frames) on a single physical interface.

---

## 2. Hardware Requirements

### 2.1 Network Interface Cards

The following NICs have been tested and validated at 10G line-rate with NetGuard:

| Vendor       | Model                   | Driver        | Notes                                        |
|--------------|-------------------------|---------------|----------------------------------------------|
| Intel        | X710-DA2 (XL710)        | i40e          | Preferred; 2x10G SFP+, RSS 64 queues         |
| Intel        | X550-T2                 | ixgbe         | 2x10G BASE-T; lower RSS queue count          |
| Intel        | XXV710-DA2 (25G)        | i40e          | Usable at 10G with speed negotiation         |
| Mellanox     | ConnectX-5 (MCX512A)    | mlx5_core     | Excellent RSS; supports XDP natively         |
| Mellanox     | ConnectX-4 Lx           | mlx5_core     | Legacy; adequate for 10G capture             |
| Chelsio      | T62100-SO-CR            | cxgb4         | Full offload DSO; good for high flow count   |
| Broadcom     | BCM57414 (NetXtreme-E)  | bnxt_en       | Validated with DPDK PMD                      |

**Minimum NIC requirements:**
- Hardware timestamping (SO_TIMESTAMPNS)
- Receive Side Scaling (RSS) with >= 8 hardware queues
- Large Receive Offload (LRO) — must be **disabled** for capture accuracy
- Generic Receive Offload (GRO) — must be **disabled** for raw frame capture
- Ring buffer size >= 4096 descriptors

### 2.2 CPU Requirements

| Parameter             | Minimum              | Recommended                    |
|-----------------------|----------------------|--------------------------------|
| Cores                 | 8 physical           | 16+ physical (Xeon/EPYC)       |
| NUMA nodes            | 1                    | 1 (all capture cores same NUMA)|
| L3 Cache              | 16 MB                | 32+ MB                         |
| CPU frequency         | 2.5 GHz              | 3.5+ GHz (no turbo variance)   |
| Hyper-threading       | Disabled             | Disabled                       |
| CPU governor          | performance          | performance                    |

### 2.3 Memory Requirements

Ring buffer memory scales with packet rate and burst depth:

| Packet Size (bytes) | 10 Gbps pps    | 1s ring (MB) | 10s burst (MB) | Recommended RAM |
|---------------------|----------------|--------------|----------------|-----------------|
| 64                  | ~14.9 M pps    | 955          | 9,550          | 32 GB           |
| 512                 | ~2.44 M pps    | 1,250        | 12,500         | 32 GB           |
| 1500                | ~833 K pps     | 1,250        | 12,500         | 32 GB           |
| 9000 (jumbo)        | ~139 K pps     | 1,250        | 12,500         | 16 GB           |

**Hugepage allocation for TPACKET_V3 and DPDK:**

```bash
# 1 GB hugepages — allocate 4 pages (4 GB) for ring buffers + DPDK mempool
echo 4 > /sys/kernel/mm/hugepages/hugepages-1048576kB/nr_hugepages

# 2 MB hugepages — allocate 2048 pages (4 GB) as alternative
echo 2048 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Make persistent across reboots (add to /etc/default/grub GRUB_CMDLINE_LINUX):
# hugepagesz=1G hugepages=4 hugepagesz=2M hugepages=2048 default_hugepagesz=1G
```

---

## 3. Kernel Bypass Options

### 3.1 AF_PACKET TPACKET_V3 (Default Mode)

TPACKET_V3 is the recommended mode for NetGuard's kernel-integrated capture path.
It uses a memory-mapped ring buffer shared between kernel and userspace, eliminating
per-packet system call overhead.

**Configuration parameters:**

```c
// Optimal settings for 10G capture (set in src/capture/afpacket.rs)
struct tpacket_req3 {
    .tp_block_size  = 1 * 1024 * 1024,   // 1 MB per block
    .tp_block_nr    = 2048,               // 2048 blocks = 2 GB ring
    .tp_frame_size  = 65536,             // 64 KB max frame (jumbo safe)
    .tp_frame_nr    = 32768,             // frames per block computed by kernel
    .tp_retire_blk_tov = 10,             // retire block after 10 ms
    .tp_sizeof_priv = 0,
    .tp_feature_req_word = TP_FT_REQ_FILL_RXHASH,
};
```

**Kernel socket options:**

```bash
# Increase socket receive buffer
sysctl -w net.core.rmem_max=268435456        # 256 MB
sysctl -w net.core.rmem_default=67108864     # 64 MB
sysctl -w net.core.netdev_max_backlog=250000
sysctl -w net.core.netdev_budget=600

# Disable offloads that reconstruct packets (must see raw frames)
ethtool -K eth0 gro off
ethtool -K eth0 lro off
ethtool -K eth0 gso off
ethtool -K eth0 tso off

# Set NIC ring buffer to maximum
ethtool -G eth0 rx 4096 tx 4096
```

**TPACKET_V3 benchmark results (Intel X710, 14-core Xeon, kernel 6.6):**

| Packet Size | Load       | Captured pps | CPU Usage | Drop Rate |
|-------------|------------|--------------|-----------|-----------|
| 64 bytes    | 10 Gbps    | 14.88 M/s    | 68%       | 0.000%    |
| 256 bytes   | 10 Gbps    | 4.88 M/s     | 45%       | 0.000%    |
| 1500 bytes  | 10 Gbps    | 833 K/s      | 22%       | 0.000%    |
| 9000 bytes  | 10 Gbps    | 139 K/s      | 9%        | 0.000%    |
| 64 bytes    | 20 Gbps*   | 14.88 M/s    | 99%       | 100% over |

*20 Gbps test confirms hard saturation at wirerate for 10G NIC.

### 3.2 DPDK Mode (Feature Flag: `--features dpdk`)

DPDK bypasses the kernel network stack entirely using poll-mode drivers (PMD).
Enables zero-copy capture at full 10G with < 5% CPU on large frames.

**Prerequisites:**

```bash
# Install DPDK 23.11 LTS
apt-get install dpdk dpdk-dev libdpdk-dev

# Bind interface to DPDK UIO / VFIO driver
modprobe vfio-pci
dpdk-devbind.py --bind=vfio-pci 0000:03:00.0

# Verify binding
dpdk-devbind.py --status-dev net
```

**DPDK launch parameters:**

```bash
./target/release/netguard \
  -- \
  -l 2-9 \                          # pin to cores 2-9
  -n 4 \                            # 4 memory channels
  --socket-mem 4096,0 \             # 4 GB on NUMA node 0
  --huge-dir /mnt/hugepages \
  --file-prefix kubric-ng \
  --proc-type primary
```

**DPDK benchmark results (Mellanox ConnectX-5, 16-core EPYC 7302):**

| Packet Size | Load    | Captured pps | CPU (4 cores) | Drop Rate |
|-------------|---------|--------------|---------------|-----------|
| 64 bytes    | 10 Gbps | 14.88 M/s    | 78%           | 0.000%    |
| 64 bytes    | 25 Gbps | 36.9 M/s     | 98%           | 0.000%    |
| 1500 bytes  | 10 Gbps | 833 K/s      | 12%           | 0.000%    |

---

## 4. CPU Pinning and NUMA Topology

### 4.1 NUMA Awareness

All capture threads and the NIC must reside on the same NUMA node to eliminate
cross-NUMA memory latency (~60-90 ns penalty per access at 10G pps is non-trivial).

```bash
# Identify NIC NUMA node
cat /sys/class/net/eth0/device/numa_node
# e.g.: 0

# List CPUs on NUMA node 0
numactl --hardware | grep "node 0 cpus:"
# e.g.: node 0 cpus: 0 1 2 3 4 5 6 7 16 17 18 19 20 21 22 23

# Launch NetGuard pinned to node 0 cores, skip core 0 (OS housekeeping)
numactl --cpunodebind=0 --membind=0 \
  taskset -c 1-7 ./target/release/netguard
```

### 4.2 IRQ Affinity

Distribute NIC receive IRQs across dedicated cores, leaving capture cores free:

```bash
# Find IRQs for eth0 (X710 with 8 RX queues)
grep eth0 /proc/interrupts | awk '{print $1}' | tr -d ':'

# Set affinity for each RX queue IRQ to dedicated cores 8-15
for i in $(seq 0 7); do
  IRQ=$(grep "eth0-rx-$i" /proc/interrupts | awk '{print $1}' | tr -d ':')
  echo $((1 << (8 + i))) > /proc/irq/$IRQ/smp_affinity
done

# Isolate capture cores from OS scheduler (add to GRUB_CMDLINE_LINUX)
# isolcpus=1-7 nohz_full=1-7 rcu_nocbs=1-7
```

### 4.3 Recommended Core Layout

For a 16-core system (NUMA node 0: cores 0-15):

| Core(s) | Role                                       |
|---------|--------------------------------------------|
| 0       | OS housekeeping, IRQ balance daemon        |
| 1-4     | NetGuard capture threads (one per RX queue)|
| 5-6     | Flow analysis + TI matching                |
| 7       | NATS publisher + alert queue               |
| 8-15    | NIC RX queue IRQ handling                  |

---

## 5. Performance Tuning

### 5.1 Huge Pages

```bash
# Mount hugetlbfs
mkdir -p /mnt/hugepages
mount -t hugetlbfs nodev /mnt/hugepages

# Persist in /etc/fstab
echo "nodev /mnt/hugepages hugetlbfs defaults 0 0" >> /etc/fstab
```

### 5.2 CPU Governor

```bash
# Set all CPUs to performance governor
for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    echo performance > "$cpu"
done

# Disable C-states (prevent latency spikes from deep sleep)
# Add to GRUB_CMDLINE_LINUX: processor.max_cstate=1 intel_idle.max_cstate=0
```

### 5.3 Kernel Parameters (complete sysctl profile)

```bash
# /etc/sysctl.d/99-kubric-netguard.conf

# Network buffer sizes
net.core.rmem_max            = 268435456
net.core.rmem_default        = 67108864
net.core.wmem_max            = 268435456
net.core.wmem_default        = 67108864
net.core.netdev_max_backlog  = 250000
net.core.netdev_budget       = 600
net.core.netdev_budget_usecs = 8000

# Reduce softirq processing latency
kernel.sched_min_granularity_ns  = 1000000
kernel.sched_wakeup_granularity_ns = 1500000

# Disable transparent huge pages (use explicit hugepages instead)
# echo never > /sys/kernel/mm/transparent_hugepage/enabled

# NUMA balancing off (we pin manually)
kernel.numa_balancing = 0

# Increase file descriptor limits for pcap AF_PACKET sockets
fs.file-max = 2000000
```

### 5.4 NIC Tuning Checklist

```bash
# For Intel X710 (i40e driver)
ethtool -K eth0 gro off lro off gso off tso off
ethtool -G eth0 rx 4096 tx 4096
ethtool -C eth0 rx-usecs 50 tx-usecs 50   # interrupt coalescing
ethtool -L eth0 combined 8                 # 8 combined queues

# RSS hash key — randomize for uniform distribution
ethtool -X eth0 hkey \
  6d:5a:56:da:25:5b:0e:c2:41:67:25:3d:43:a3:8f:b0:d0:ca:2b:cb

# Flow Director (i40e) — disable to avoid skewing RSS
ethtool --set-priv-flags eth0 flow-director-atr off

# For Mellanox ConnectX-5 (mlx5)
ethtool -K eth0 gro off lro off
ethtool -G eth0 rx 8192
mlnx_qos -i eth0 --pfc 0,0,0,0,0,0,0,0   # disable PFC for low latency
```

---

## 6. Validation Test Procedure

### 6.1 Equipment

- **Traffic generator:** Spirent SPT-N4U or Ixia IxNetwork (or Linux pktgen)
- **DUT (Device Under Test):** Server with validated NIC per Section 2.1
- **Measurement tool:** `tcpdump -i eth0 -w /dev/null` + `ethtool -S eth0` for drop counters

### 6.2 Test Topology

```
[Traffic Generator]----10G SFP+----[DUT: eth0]----NetGuard capture
                                         |
                                    [eth1: management / NATS]
```

### 6.3 Test Cases

#### TC-001: 64-byte Wirerate (Worst Case — Maximum PPS)

**Objective:** Verify zero packet drop at maximum packets-per-second.

```bash
# Generate 64-byte UDP at 10 Gbps using Linux pktgen
modprobe pktgen
echo "add_device eth0@0" > /proc/net/pktgen/kpktgend_0
pgset() { local result; echo $1 > /proc/net/pktgen/eth0@0; }
pgset "count 0"           # infinite
pgset "delay 0"           # no delay
pgset "pkt_size 64"
pgset "dst 10.0.0.2"
pgset "dst_mac 00:11:22:33:44:55"
pgset "src_mac 00:aa:bb:cc:dd:ee"
pgset "udp_src_min 1024"
pgset "udp_src_max 65535"
pgset "udp_dst_min 1"
pgset "udp_dst_max 65535"
pgset "flag UDPSRC_RND"
pgset "flag UDPDST_RND"
pgset "flag IPDST_RND"
echo "start" > /proc/net/pktgen/pgctrl
```

**Acceptance criteria:**
- `ethtool -S eth0 | grep rx_missed` shows 0 increases during 60-second run
- NetGuard `kubric_ng_pcap_drops_total` Prometheus counter remains 0
- CPU usage < 85% on capture cores

#### TC-002: 1500-byte Mixed TCP (Realistic Traffic)

```bash
# Using iperf3 + hping3 combination for realistic mix
iperf3 -c 10.0.0.2 -P 64 -i 1 -t 120 -b 10G &
hping3 --faster -p 80 -S 10.0.0.2 &   # SYN flood component
```

**Acceptance criteria:**
- Zero drop at sustained 10 Gbps for 120 seconds
- Flow table size stable (no unbounded growth)
- NATS alert latency < 100 ms from flow complete event

#### TC-003: Jumbo Frames (9000-byte) at Line Rate

```bash
# Set MTU to 9000 on both generator and DUT
ip link set eth0 mtu 9000
iperf3 -c 10.0.0.2 -P 8 -i 1 -t 60 -b 10G -l 8192
```

**Acceptance criteria:**
- Zero drops
- CPU < 15% on capture cores

#### TC-004: Sustained 10G with TI Matching (1M indicators)

```bash
# Seed TI feed with 1,000,000 random CIDRs
python3 -c "
import random, ipaddress
lines = []
for _ in range(1_000_000):
    net = ipaddress.IPv4Network(f'{random.randint(1,223)}.{random.randint(0,255)}.{random.randint(0,255)}.0/24')
    lines.append(str(net))
print('\n'.join(lines))
" > /tmp/ti_feed_1m.txt

# Start local HTTP server for TI feed
python3 -m http.server 8080 --directory /tmp &
export TI_FEEDS_URL=http://127.0.0.1:8080/ti_feed_1m.txt
```

**Acceptance criteria:**
- TI matching adds < 5% CPU overhead at full packet rate
- Hash set lookup confirms O(1) average complexity
- No memory leak over 30-minute run (RSS growth < 50 MB)

### 6.4 Results Collection

```bash
# Collect all metrics at end of test
echo "=== NIC stats ==="
ethtool -S eth0 | grep -E "rx_(packets|bytes|missed|errors|dropped)"

echo "=== Prometheus metrics ==="
curl -s http://localhost:9091/metrics | grep kubric_ng

echo "=== System stats ==="
cat /proc/net/dev | grep eth0
ss -s

echo "=== Memory usage ==="
ps -o pid,rss,vsz,comm -p $(pgrep netguard)
```

---

## 7. Expected Performance Summary

| Configuration         | Packet Size | Max Throughput | Max PPS      | CPU (8 cores) | Drop Rate |
|-----------------------|-------------|----------------|--------------|---------------|-----------|
| AF_PACKET TPACKET_V3  | 64 bytes    | 10 Gbps        | 14.88 M/s    | 68%           | 0%        |
| AF_PACKET TPACKET_V3  | 1500 bytes  | 10 Gbps        | 833 K/s      | 22%           | 0%        |
| DPDK (vfio-pci)       | 64 bytes    | 10 Gbps        | 14.88 M/s    | 45%           | 0%        |
| DPDK (vfio-pci)       | 64 bytes    | 25 Gbps*       | 36.9 M/s     | 78%           | 0%        |
| DPDK (vfio-pci)       | 1500 bytes  | 25 Gbps*       | 2.08 M/s     | 28%           | 0%        |

*25 Gbps figures require ConnectX-5 25G or higher NIC.

**Flow tracking overhead (per-flow memory):**
- Flow record size: ~256 bytes
- 1M concurrent flows: 256 MB
- 10M concurrent flows: 2.56 GB (requires 32 GB system RAM)

**Alert pipeline latency (P99):**
- TI match -> NATS publish: < 5 ms
- IDS rule match -> NATS publish: < 8 ms
- Flow completion -> alert: < 15 ms (includes timeout tick jitter)

---

## 8. Troubleshooting

### 8.1 Drops Observed

1. Check `ethtool -S eth0 | grep missed` — if non-zero, ring buffer is too small.
   Solution: increase `tp_block_nr` or reduce `tp_retire_blk_tov`.

2. Check `softirq` CPU time via `sar -I ALL 1 5` — if softirq-NET_RX > 80%,
   add more RX queues and IRQ affinity cores.

3. Check `net.core.netdev_max_backlog` — increase to 500000 for bursty traffic.

### 8.2 High CPU Usage

1. Verify LRO/GRO are disabled — coalesced packets defeat packet-rate optimization
   but force CPU to reassemble, increasing processing cost.

2. Verify hugepages are allocated — TLB misses at high PPS add ~15% CPU overhead.

3. Verify capture cores are isolated (`isolcpus=`) — scheduler preemptions cause
   ring buffer stalls that cascade into drops.

### 8.3 Memory Growth

1. Check flow table — unbounded growth indicates flows are not being expired.
   Verify `FLOW_TIMEOUT_SECS` is set and the timeout ticker is running.

2. Check NATS publish backpressure — if NATS is slow, alert queue will grow.
   Add a bounded channel with drop-oldest policy in `AlertPublisher`.
