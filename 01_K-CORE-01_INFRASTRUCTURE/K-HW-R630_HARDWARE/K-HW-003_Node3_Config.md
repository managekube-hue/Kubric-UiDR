# K-HW-003 — Dell R630 Node 3 Hardware Configuration

> **Hostname:** pve-kubric-03  
> **Role:** K8s Worker + GPU Inference (vLLM/Ollama) + MinIO Object Storage  
> **iDRAC IP:** 10.0.100.13  
> **Management IP:** 10.0.100.23 (eno1)  
> **Storage IP:** 10.0.200.23 (eno2)  
> **Workload IP:** 10.0.50.23 (p3p1 — 10G SFP+)  
> **GPU:** Optional NVIDIA Tesla T4 or A2 (low-profile PCIe)

---

## 1. iDRAC Initial Access & Credentials

```bash
# Connect to iDRAC dedicated NIC port
ip addr add 192.168.0.100/24 dev eth0

# Change default password
racadm -r 192.168.0.120 -u root -p calvin \
  set iDRAC.Users.2.Password "K8r1c!N0d3_03#iDR@C"

# Create kubric-admin
racadm -r 192.168.0.120 -u root -p "K8r1c!N0d3_03#iDR@C" <<'EOF'
set iDRAC.Users.3.Enable Enabled
set iDRAC.Users.3.UserName kubric-admin
set iDRAC.Users.3.Password "${IDRAC_KUBRIC_PASS}"
set iDRAC.Users.3.Privilege 0x1ff
set iDRAC.Users.3.IpmiLanPrivilege Administrator
EOF

# Set production network
racadm -r 192.168.0.120 -u root -p "K8r1c!N0d3_03#iDR@C" <<'EOF'
set iDRAC.IPv4.Address 10.0.100.13
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

```bash
IDRAC="10.0.100.13"
CRED="-u kubric-admin -p ${IDRAC_KUBRIC_PASS}"

# Performance profile
racadm -r $IDRAC $CRED set BIOS.SysProfileSettings.SysProfile PerfOptimized
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcCStates Disabled
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcC1E Disabled
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcTurboMode Enabled
racadm -r $IDRAC $CRED set BIOS.ProcSettings.LogicalProc Enabled

# SR-IOV for 10G NIC
racadm -r $IDRAC $CRED set BIOS.IntegratedDevices.SriovGlobalEnable Enabled

# VT-d / IOMMU — CRITICAL for GPU passthrough
racadm -r $IDRAC $CRED set BIOS.ProcSettings.ProcVirtualization Enabled
racadm -r $IDRAC $CRED set BIOS.IntegratedDevices.IoatDmaEngine Enabled

# Memory
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
  -rl r10 -wp wb -rp ra -ss 256 \
  -pdkey:Disk.Bay.0:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.1:Enclosure.Internal.0-1:RAID.Integrated.1-1 \
  -name "OS_RAID10" -size 480GB
```

### 3.2 RAID 5 — Data Volume (4× 2TB NVMe)

```bash
racadm -r $IDRAC $CRED storage createvd RAID.Integrated.1-1 \
  -rl r5 -wp wb -rp ara -ss 512 \
  -pdkey:Disk.Bay.2:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.3:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.4:Enclosure.Internal.0-1:RAID.Integrated.1-1,Disk.Bay.5:Enclosure.Internal.0-1:RAID.Integrated.1-1 \
  -name "DATA_RAID5" -size max
```

### 3.3 Apply RAID

```bash
racadm -r $IDRAC $CRED jobqueue create RAID.Integrated.1-1
racadm -r $IDRAC $CRED serveraction powercycle
```

---

## 4. Network Configuration

### 4.1 Physical NIC Mapping

| Interface | NIC Model | IP Address | VLAN | Purpose |
|-----------|-----------|------------|------|---------|
| eno1 | Broadcom 5720 1GbE | 10.0.100.23/24 | native | Management / Proxmox API |
| eno2 | Broadcom 5720 1GbE | 10.0.200.23/24 | 200 | Storage / Ceph + MinIO |
| p3p1 | Intel X710 10G SFP+ | 10.0.50.23/24 | 50 | K8s pod traffic / workload |
| p3p2 | Intel X710 10G SFP+ | — | — | Bond slave / LACP pair |

### 4.2 Proxmox Network Interfaces

```ini
# /etc/network/interfaces on pve-kubric-03

auto lo
iface lo inet loopback

auto eno1
iface eno1 inet manual

auto vmbr0
iface vmbr0 inet static
    address 10.0.100.23/24
    gateway 10.0.100.1
    bridge-ports eno1
    bridge-stp off
    bridge-fd 0
    dns-nameservers 10.0.100.1 1.1.1.1
    dns-search kubric.local

auto eno2
iface eno2 inet manual

auto vmbr1
iface vmbr1 inet static
    address 10.0.200.23/24
    bridge-ports eno2
    bridge-stp off
    bridge-fd 0
    mtu 9000

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
    address 10.0.50.23/24
    bridge-ports bond0
    bridge-stp off
    bridge-fd 0
    mtu 9000
```

---

## 5. Proxmox VE Installation

```bash
# Install from Proxmox VE 8.x USB
# Target: OS_RAID10, ext4
# Hostname: pve-kubric-03.kubric.local
# IP: 10.0.100.23/24, GW: 10.0.100.1

# Post-install
sed -i 's/^deb/#deb/' /etc/apt/sources.list.d/pve-enterprise.list
echo "deb http://download.proxmox.com/debian/pve bookworm pve-no-subscription" \
  > /etc/apt/sources.list.d/pve-no-subscription.list
apt-get update && apt-get dist-upgrade -y

apt-get install -y htop iotop iperf3 ethtool lm-sensors net-tools \
  chrony smartmontools nvme-cli fio pve-headers-$(uname -r)

hostnamectl set-hostname pve-kubric-03

cat >> /etc/hosts <<'EOF'
10.0.100.21  pve-kubric-01 pve-kubric-01.kubric.local
10.0.100.22  pve-kubric-02 pve-kubric-02.kubric.local
10.0.100.23  pve-kubric-03 pve-kubric-03.kubric.local
EOF
```

---

## 6. GPU Passthrough Configuration (Tesla T4 / A2)

### 6.1 IOMMU / VFIO Kernel Setup

```bash
# Edit GRUB for IOMMU
sed -i 's/GRUB_CMDLINE_LINUX_DEFAULT="quiet"/GRUB_CMDLINE_LINUX_DEFAULT="quiet intel_iommu=on iommu=pt"/' /etc/default/grub
update-grub

# Load VFIO modules
cat > /etc/modules-load.d/vfio.conf <<'EOF'
vfio
vfio_iommu_type1
vfio_pci
vfio_virqfd
EOF

# Blacklist nouveau (open-source NVIDIA driver)
cat > /etc/modprobe.d/blacklist-nouveau.conf <<'EOF'
blacklist nouveau
blacklist lbm-nouveau
options nouveau modeset=0
alias nouveau off
alias lbm-nouveau off
EOF

update-initramfs -u -k all
reboot
```

### 6.2 Identify GPU PCI Address

```bash
# After reboot, verify IOMMU is active
dmesg | grep -e DMAR -e IOMMU
# Should show: DMAR: IOMMU enabled

# Find GPU
lspci -nn | grep -i nvidia
# Example output: 3b:00.0 3D controller [0302]: NVIDIA Corporation TU104GL [Tesla T4] [10de:1eb8]

# Note the vendor:device ID: 10de:1eb8 for T4, 10de:25b6 for A2
GPU_PCI_ID="10de:1eb8"

# Bind to VFIO
echo "options vfio-pci ids=${GPU_PCI_ID}" > /etc/modprobe.d/vfio-pci.conf
update-initramfs -u -k all
reboot
```

### 6.3 Verify GPU Isolation

```bash
# After reboot — confirm VFIO owns the GPU
lspci -nnk -s 3b:00.0
# Kernel driver in use: vfio-pci   ← correct

# GPU is now available for VM passthrough
```

### 6.4 Assign GPU to VM (vLLM or Ollama VM)

```bash
# When creating VM, add hostpci device:
qm set <VMID> -hostpci0 3b:00.0,pcie=1,x-vga=off

# Inside the VM, install NVIDIA drivers:
# (See K-HV-VM-005 for full vLLM GPU node setup)
```

---

## 7. Storage Layout — Node 3

### 7.1 Data Volume Partitioning

```bash
# Node 3 splits data between MinIO and Ceph
lsblk -o NAME,SIZE,MODEL

parted /dev/sdb --script mklabel gpt
parted /dev/sdb --script mkpart minio xfs 0% 70%     # ~4.2 TB for MinIO
parted /dev/sdb --script mkpart ceph-osd 70% 100%    # ~1.8 TB for Ceph OSD

# Format MinIO partition
mkfs.xfs -f -L minio-data /dev/sdb1
mkdir -p /mnt/minio-data
echo "LABEL=minio-data /mnt/minio-data xfs defaults,noatime,nodiratime 0 2" >> /etc/fstab

# Leave sdb2 raw for Ceph OSD
mount -a
```

### 7.2 MinIO Deployment

```bash
# Create MinIO directories
mkdir -p /mnt/minio-data/{kubric-artifacts,kubric-cold,kubric-backups,kubric-models}

# Deploy MinIO via Docker
docker run -d \
  --name kubric-minio \
  --restart unless-stopped \
  -p 9000:9000 \
  -p 9001:9001 \
  -v /mnt/minio-data:/data \
  -e MINIO_ROOT_USER=kubric-admin \
  -e MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASS}" \
  -e MINIO_BROWSER_REDIRECT_URL=https://minio.kubric.local \
  minio/minio:latest server /data --console-address ":9001"
```

**Environment Variable:**
```
MINIO_ROOT_PASS=<generate with: openssl rand -base64 24>
```

### 7.3 MinIO Bucket Setup

```bash
# Install mc (MinIO Client)
curl -o /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x /usr/local/bin/mc

# Configure alias
mc alias set kubric http://10.0.100.23:9000 kubric-admin "${MINIO_ROOT_PASS}"

# Create buckets
mc mb kubric/kubric-cold-storage        # ClickHouse cold tier
mc mb kubric/kubric-backups             # PostgreSQL backups
mc mb kubric/kubric-artifacts           # CI build artifacts
mc mb kubric/kubric-models              # LLM model weights
mc mb kubric/kubric-parquet             # DuckDB/Arrow parquet exports

# Set lifecycle — auto-delete backups after 90 days
cat > /tmp/lifecycle.json <<'EOF'
{
  "Rules": [
    {
      "ID": "expire-old-backups",
      "Status": "Enabled",
      "Prefix": "",
      "Expiration": { "Days": 90 }
    }
  ]
}
EOF
mc ilm import kubric/kubric-backups < /tmp/lifecycle.json

# Verify
mc ls kubric/
mc admin info kubric
```

---

## 8. Verification Checklist

```bash
# BIOS
racadm -r 10.0.100.13 $CRED get BIOS.ProcSettings.ProcCStates     # Disabled
racadm -r 10.0.100.13 $CRED get BIOS.ProcSettings.ProcTurboMode    # Enabled
racadm -r 10.0.100.13 $CRED get BIOS.ProcSettings.ProcVirtualization  # Enabled

# RAID
racadm -r 10.0.100.13 $CRED storage get vdisks
# OS_RAID10 (Optimal), DATA_RAID5 (Optimal)

# Network
ip addr show vmbr0 | grep "10.0.100.23"
ip addr show vmbr1 | grep "10.0.200.23"
ip addr show vmbr2 | grep "10.0.50.23"

# IOMMU
dmesg | grep IOMMU  # "IOMMU enabled"

# GPU (if installed)
lspci -nnk | grep -A 2 NVIDIA
# Should show vfio-pci as kernel driver

# MinIO
curl -s http://10.0.100.23:9000/minio/health/live
# Returns HTTP 200

mc admin info kubric
# Shows disk usage, uptime

# Storage
df -h /mnt/minio-data
```

---

## 9. Node 3 Role Assignment Summary

| Component | Resource Allocation | Notes |
|-----------|-------------------|-------|
| K8s Worker Node | 8 vCPU, 16 GB RAM | kubelet, kube-proxy |
| vLLM / Ollama VM | 16 vCPU, 64 GB RAM + GPU | LLM inference for KAI |
| MinIO | 2 vCPU, 8 GB RAM | S3-compatible object storage |
| Ceph OSD | — | Raw partition ~1.8 TB |
| Monitoring | 2 vCPU, 4 GB RAM | Grafana, Prometheus |
| **Total Reserved** | **28 vCPU, 92 GB** | GPU passthrough from host PCIe |
