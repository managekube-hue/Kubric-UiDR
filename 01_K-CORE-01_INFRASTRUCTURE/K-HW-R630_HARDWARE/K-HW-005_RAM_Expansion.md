# K-HW-005 — RAM Expansion Planning (R630 Nodes)

> **Current:** 12× 16GB DDR4-2400 RDIMMs = 192 GB per node  
> **Target:** 12× 32GB DDR4-2400 RDIMMs = 384 GB per node  
> **Platform:** Dell PowerEdge R630 (2-socket, 24 DIMM slots)

---

## 1. Current Memory Configuration

### 1.1 Verify Installed DIMMs

```bash
# On each Proxmox node
dmidecode -t memory | grep -E "Size|Speed|Locator|Part|Manufacturer" | head -48

# Or via iDRAC
racadm -r $IDRAC $CRED get BIOS.MemSettings
racadm -r $IDRAC $CRED hwinventory | grep -A5 "DIMM"

# Summary command
lshw -short -class memory
free -g
```

### 1.2 Current Layout (Per Node)

| CPU Socket | Channel A | Channel B | Channel C | Channel D |
|-----------|-----------|-----------|-----------|-----------|
| CPU 0 | A1: 16GB | B1: 16GB | C1: 16GB | D1: 16GB |
| CPU 0 | A2: 16GB | B2: 16GB | — | — |
| CPU 1 | E1: 16GB | F1: 16GB | G1: 16GB | H1: 16GB |
| CPU 1 | E2: 16GB | F2: 16GB | — | — |

**Total per node:** 12× 16GB = 192 GB  
**Running at:** DDR4-2400, Dual Rank, 1.2V

---

## 2. Target Configuration (384 GB per Node)

### 2.1 Dell Compatible DIMMs

| Part Number | Description | Qty/Node | Total/Cluster | Est. Price |
|-------------|-------------|----------|---------------|------------|
| SNPCPC7GC/32G | Dell 32GB DDR4-2400 RDIMM 2Rx4 | 12 | 36 | ~$35/ea |
| A8711888 | Dell OEM 32GB DDR4-2400 RDIMM | 12 | 36 | ~$40/ea |
| HMA84GR7AFR4N-UH | SK Hynix 32GB 2Rx4 PC4-2400T-R | 12 | 36 | ~$30/ea |
| M393A4K40BB1-CRC | Samsung 32GB 2Rx4 PC4-2400T-R | 12 | 36 | ~$32/ea |

> **Recommendation:** Use SK Hynix HMA84GR7AFR4N-UH or Samsung M393A4K40BB1-CRC for cost/compatibility balance. All DIMMs must be identical across a node for optimal interleaving.

### 2.2 Target Layout — Full Population

| CPU Socket | Channel A | Channel B | Channel C | Channel D |
|-----------|-----------|-----------|-----------|-----------|
| CPU 0 | A1: 32GB | B1: 32GB | C1: 32GB | D1: 32GB |
| CPU 0 | A2: 32GB | B2: 32GB | — | — |
| CPU 1 | E1: 32GB | F1: 32GB | G1: 32GB | H1: 32GB |
| CPU 1 | E2: 32GB | F2: 32GB | — | — |

**Total per node:** 12× 32GB = 384 GB

---

## 3. DIMM Population Rules for Optimal Interleaving

### 3.1 R630 Memory Architecture

The R630 with E5-2600 v4 CPUs has **4 memory channels per CPU**, each supporting up to **3 DIMMs per channel (DPC)**.

**Dell R630 DIMM Slot Labels:**
```
CPU 0: A1 A2 A3 | B1 B2 B3 | C1 C2 C3 | D1 D2 D3
CPU 1: E1 E2 E3 | F1 F2 F3 | G1 G2 G3 | H1 H2 H3
```

### 3.2 Population Order (Dell Recommended)

**Rule 1:** Always populate white slots first (slot 1 in each channel).  
**Rule 2:** Balance across channels before adding DPC.  
**Rule 3:** Balance across CPUs.

| # DIMMs | Slots to Populate |
|---------|-------------------|
| 2 | A1, E1 |
| 4 | A1, B1, E1, F1 |
| 8 | A1, B1, C1, D1, E1, F1, G1, H1 |
| 12 | A1, A2, B1, B2, C1, D1, E1, E2, F1, F2, G1, H1 |
| 16 | A1, A2, B1, B2, C1, C2, D1, D2, E1, E2, F1, F2, G1, G2, H1, H2 |
| 24 (full) | All slots populated |

> **Kubric runs 12 DIMMs** — this uses 2 DPC on channels A/B/E/F and 1 DPC on C/D/G/H. Channels A and B get slightly more bandwidth.

### 3.3 Interleaving Verification

```bash
# After upgrade, verify interleaving mode
dmidecode -t 19
# Should show memory regions spanning both sockets

# Check NUMA topology
numactl --hardware
# Expected:
#   node 0: 192 GB (CPU 0)
#   node 1: 192 GB (CPU 1)

# Verify DDR4 speed
dmidecode -t 17 | grep -E "Speed|Configured"
# Expected: Speed: 2400 MT/s, Configured Memory Speed: 2400 MT/s
```

---

## 4. Upgrade Procedure

### 4.1 Pre-Upgrade Checklist

```bash
# 1. Backup running VMs / containers
qm list | grep running
vzdump --all --storage local --mode snapshot

# 2. Drain K8s node (if applicable)
kubectl drain pve-kubric-01 --ignore-daemonsets --delete-emptydir-data

# 3. Record current memory
free -g > /tmp/pre_upgrade_mem.txt
dmidecode -t memory > /tmp/pre_upgrade_dimms.txt

# 4. Graceful shutdown
shutdown -h now
```

### 4.2 Physical Installation

```
1. Power off server and disconnect power cables
2. Wait 30 seconds for capacitors to discharge
3. Note existing DIMM placement (photograph if possible)
4. Remove 12× 16GB DIMMs:
   - Push retention clips outward on each side of DIMM slot
   - Lift DIMM straight up
5. Install 12× 32GB DIMMs in SAME slots:
   A1, A2, B1, B2, C1, D1 (CPU 0)
   E1, E2, F1, F2, G1, H1 (CPU 1)
   - Align DIMM notch with slot key
   - Press down firmly until both clips snap closed
6. Close chassis, reconnect power
7. Power on — BIOS will detect new memory and may retrain
```

### 4.3 Post-Upgrade Verification

```bash
# BIOS should auto-detect new DIMMs — check iDRAC
racadm -r $IDRAC $CRED hwinventory | grep -A3 "DIMM.Socket"

# Boot into Proxmox, verify
free -g
# Expected: ~377 GB (384 GB minus kernel/reserved)

dmidecode -t 17 | grep "Size" | sort | uniq -c
# Expected: 12 × "Size: 32768 MB"

numactl --hardware
# node 0 cpus: 0-13 28-41   memory: 192 GB
# node 1 cpus: 14-27 42-55  memory: 192 GB

# Run stress test
apt-get install -y stress-ng
stress-ng --vm 4 --vm-bytes 80G --timeout 300s --metrics
```

---

## 5. Memory Test with memtest86+

### 5.1 Create memtest86+ Boot USB

```bash
# On workstation
wget https://memtest.org/download/v7.00/mt86plus_7.00_64.grub.iso
dd if=mt86plus_7.00_64.grub.iso of=/dev/sdX bs=4M status=progress
```

### 5.2 Run Memory Test

```
1. Boot from USB on each node
2. Select "Test All Memory"
3. Let run for minimum 2 full passes (approximately 4-8 hours per node at 384GB)
4. Record results:
   - Pass count
   - Errors found
   - Test duration
5. Any errors → replace the failing DIMM immediately
```

### 5.3 Alternative: In-OS Memory Test

```bash
# Quick validation without reboot (less thorough)
apt-get install -y memtester
# Test 16GB blocks (don't test all RAM while OS is running)
memtester 16G 1

# Or use stress-ng with memory verification
stress-ng --vm 8 --vm-bytes 32G --vm-method all --verify --timeout 600s
```

---

## 6. Memory Allocation Plan (Per Node at 384 GB)

### 6.1 Node 1 (pve-kubric-01) — K8s Control Plane + ClickHouse

| Workload | Allocated RAM | Notes |
|----------|---------------|-------|
| ClickHouse Shard 1 | 128 GB | `max_memory_usage`, `mark_cache_size` |
| K8s Control Plane | 8 GB | etcd, apiserver, scheduler, controller-manager |
| K8s Worker Pods | 128 GB | Kubric services (ksvc, vdr, noc, kai-api) |
| Proxmox OS + Buffers | 16 GB | Kernel, page cache, ZFS ARC |
| Ceph OSD Cache | 16 GB | Bluestore cache for NVMe OSDs |
| Monitoring Stack | 8 GB | Prometheus, node-exporter, AlertManager |
| **Reserved Headroom** | **80 GB** | Burst capacity |

### 6.2 Node 2 (pve-kubric-02) — Worker + ClickHouse + PostgreSQL

| Workload | Allocated RAM | Notes |
|----------|---------------|-------|
| ClickHouse Shard 2 | 128 GB | Replication pair with Node 1 |
| PostgreSQL Primary | 32 GB | `shared_buffers=8GB`, `effective_cache_size=24GB` |
| K8s Worker Pods | 128 GB | Kubric services |
| Proxmox OS + Buffers | 16 GB | Kernel, page cache |
| Ceph OSD Cache | 16 GB | Bluestore cache |
| **Reserved Headroom** | **64 GB** | Burst capacity |

### 6.3 Node 3 (pve-kubric-03) — Worker + GPU + MinIO

| Workload | Allocated RAM | Notes |
|----------|---------------|-------|
| vLLM / Ollama LLM VM | 128 GB | Model loading, KV cache, batch inference |
| K8s Worker Pods | 96 GB | Kubric services |
| MinIO | 16 GB | Object store caching |
| Proxmox OS + Buffers | 16 GB | Kernel, page cache |
| Ceph OSD Cache | 16 GB | Bluestore cache |
| **Reserved Headroom** | **112 GB** | Future model scaling |

---

## 7. BIOS Memory Settings After Upgrade

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  racadm -r $IDRAC $CRED <<'CMDS'
    # Optimizer mode — max bandwidth
    set BIOS.MemSettings.MemOpMode OptimizerMode
    # Disable node interleaving — maintain NUMA topology
    set BIOS.MemSettings.NodeInterleave Disabled
    # Enable memory scrubbing for ECC error detection
    set BIOS.MemSettings.DemandScrub Enabled
    # Patrol scrub — background ECC scanning
    set BIOS.MemSettings.PatrolScrub Standard
CMDS
  racadm -r $IDRAC $CRED jobqueue create BIOS.Setup.1-1
done
```

---

## 8. Kubernetes Memory Resource Limits

After upgrade, adjust K8s node capacity:

```yaml
# kubelet config on each node
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
systemReserved:
  cpu: "2"
  memory: "16Gi"
kubeReserved:
  cpu: "2"
  memory: "8Gi"
evictionHard:
  memory.available: "2Gi"
  nodefs.available: "10%"
```

```bash
# Verify node allocatable after kubelet restart
kubectl describe node pve-kubric-01 | grep -A5 "Allocatable"
# Expected: memory ~350Gi allocatable
```
