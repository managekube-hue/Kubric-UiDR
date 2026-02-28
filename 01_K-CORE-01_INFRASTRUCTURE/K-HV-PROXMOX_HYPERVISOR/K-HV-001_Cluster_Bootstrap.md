# K-HV-001 — Proxmox VE Cluster Bootstrap

> **Cluster Name:** kubric-cluster  
> **Nodes:** pve-kubric-01 (10.0.100.21), pve-kubric-02 (10.0.100.22), pve-kubric-03 (10.0.100.23)  
> **Storage:** local-lvm, Ceph RBD, NFS  
> **Version:** Proxmox VE 8.2+

---

## 1. Prerequisites

```bash
# On ALL nodes — ensure /etc/hosts is correct
cat >> /etc/hosts <<'EOF'
10.0.100.21  pve-kubric-01 pve-kubric-01.kubric.local
10.0.100.22  pve-kubric-02 pve-kubric-02.kubric.local
10.0.100.23  pve-kubric-03 pve-kubric-03.kubric.local
EOF

# Verify time sync (critical for cluster)
chronyc tracking | grep "System time"
# Must be <1ms offset

# Verify SSH connectivity between all nodes
ssh root@10.0.100.22 hostname
ssh root@10.0.100.23 hostname

# Ensure no firewall blocks (ports: 5404-5405/udp, 8006/tcp, 22/tcp)
iptables -L -n | grep -i drop
```

---

## 2. Cluster Creation

### 2.1 Initialize Cluster on Node 1

```bash
# On pve-kubric-01
pvecm create kubric-cluster --link0 10.0.200.21

# Verify
pvecm status
pvecm nodes
```

### 2.2 Join Remaining Nodes

```bash
# On pve-kubric-02
pvecm add 10.0.100.21 --link0 10.0.200.22

# On pve-kubric-03
pvecm add 10.0.100.21 --link0 10.0.200.23
```

### 2.3 Verify Three-Node Cluster

```bash
pvecm status
# Cluster information
# ───────────────────
# Name:    kubric-cluster
# Config:  3
# Quorum:  3 (3 expected)
# Nodes:   3

pvecm nodes
# Membership: 3 nodes
#    1  pve-kubric-01 (local)
#    2  pve-kubric-02
#    3  pve-kubric-03
```

---

## 3. Network Bridge Configuration

### 3.1 Bridge Definitions (All Nodes)

Each node needs 3 bridges (already defined in K-HW-001/2/3 interfaces):

| Bridge | Physical Port | Purpose | VLAN | MTU |
|--------|--------------|---------|------|-----|
| vmbr0 | eno1 | Management + VM access | native | 1500 |
| vmbr1 | eno2 | Storage (Ceph, NFS) | 200 | 9000 |
| vmbr2 | bond0 (p3p1+p3p2) | Workload (K8s pods) | 50 | 9000 |

### 3.2 Verify Bridges

```bash
# On each node
brctl show
# bridge name   bridge id         STP enabled   interfaces
# vmbr0         8000.xxxx         no            eno1
# vmbr1         8000.xxxx         no            eno2
# vmbr2         8000.xxxx         no            bond0

ip addr show vmbr0 | grep "inet "
ip addr show vmbr1 | grep "inet "
ip addr show vmbr2 | grep "inet "
```

---

## 4. Storage Configuration

### 4.1 Local LVM (Per-Node)

Already created by Proxmox installer:

```bash
# Verify local storage
pvesm status
# Name        Type     Status   Total     Used     Available   %
# local       dir      active   XX GB     XX GB    XX GB       XX%
# local-lvm   lvmthin  active   XX GB     XX GB    XX GB       XX%

# Expand local-lvm if needed
lvextend -l +100%FREE /dev/pve/data
```

### 4.2 Ceph RBD Pool (See K-HV-002 for Full Setup)

```bash
# After Ceph is configured:
pvesm add rbd kubric-ceph \
  --pool kubric-vms \
  --monhost 10.0.200.21,10.0.200.22,10.0.200.23 \
  --content images,rootdir \
  --krbd 0

pvesm status | grep kubric-ceph
```

### 4.3 NFS Share (Optional — for ISOs and Backups)

```bash
# If running NFS server on Node 1 or external NAS:
pvesm add nfs kubric-nfs \
  --server 10.0.200.21 \
  --export /mnt/nfs-share \
  --content iso,backup,snippets \
  --options vers=4.2

# Create NFS export on node 1 (if hosted locally)
apt-get install -y nfs-kernel-server
mkdir -p /mnt/nfs-share/{iso,backup,snippets}
cat >> /etc/exports <<'EOF'
/mnt/nfs-share 10.0.200.0/24(rw,sync,no_subtree_check,no_root_squash)
EOF
exportfs -ra
systemctl restart nfs-kernel-server
```

### 4.4 MinIO S3 for Backups (from Node 3)

```bash
# Proxmox Backup Server can use S3 via PBS proxy,
# but for vzdump, use NFS or local storage
# MinIO is accessed directly by services, not PVE storage
```

---

## 5. HA Group Creation

### 5.1 Create HA Groups

```bash
# Create HA group for critical services — all nodes eligible
ha-manager groupadd kubric-ha-all \
  --nodes pve-kubric-01,pve-kubric-02,pve-kubric-03 \
  --nofailback 1

# Create HA group preferring Node 1+2 (database services)
ha-manager groupadd kubric-ha-db \
  --nodes pve-kubric-01:2,pve-kubric-02:2,pve-kubric-03:1 \
  --nofailback 0 \
  --restricted 0

# Create HA group for GPU workloads (Node 3 only)
ha-manager groupadd kubric-ha-gpu \
  --nodes pve-kubric-03 \
  --nofailback 1 \
  --restricted 1
```

### 5.2 Add VMs to HA

```bash
# Example: Add ClickHouse VM (VMID 200) to HA
ha-manager add vm:200 --group kubric-ha-db --state started --max_restart 3 --max_relocate 2

# Add K8s control plane (VMID 100) to HA
ha-manager add vm:100 --group kubric-ha-all --state started --max_restart 3

# Add GPU inference VM (VMID 300) to HA
ha-manager add vm:300 --group kubric-ha-gpu --state started --max_restart 2

# View HA status
ha-manager status
```

---

## 6. Cloud-init Template Preparation

### 6.1 Download Base Image

```bash
# On pve-kubric-01 (template will sync via Ceph or manually copy)
cd /tmp

# Ubuntu 22.04 cloud image
wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img

# Verify checksum
sha256sum jammy-server-cloudimg-amd64.img
```

### 6.2 Create Template VM

```bash
# Create VM 9000 as template
qm create 9000 --name ubuntu-2204-cloud --memory 2048 --cores 2 --net0 virtio,bridge=vmbr2

# Import cloud image as disk
qm importdisk 9000 /tmp/jammy-server-cloudimg-amd64.img local-lvm

# Attach disk
qm set 9000 --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-9000-disk-0

# Add cloud-init drive
qm set 9000 --ide2 local-lvm:cloudinit

# Set boot order
qm set 9000 --boot c --bootdisk scsi0

# Set serial console (required for cloud-init)
qm set 9000 --serial0 socket --vga serial0

# Set cloud-init defaults
qm set 9000 --ciuser kubric
qm set 9000 --cipassword "${CI_DEFAULT_PASS}"
qm set 9000 --sshkeys /root/.ssh/authorized_keys
qm set 9000 --ipconfig0 ip=dhcp
qm set 9000 --nameserver 10.0.100.1
qm set 9000 --searchdomain kubric.local

# Enable QEMU guest agent
qm set 9000 --agent enabled=1

# Convert to template
qm template 9000
```

**Environment Variable:**
```
CI_DEFAULT_PASS=<generate with: openssl rand -base64 16>
```

### 6.3 Clone Template for New VMs

```bash
# Clone to new VM (full clone)
qm clone 9000 101 --name kubric-ksvc-01 --full --storage kubric-ceph

# Customize clone
qm set 101 --cores 4 --memory 8192
qm set 101 --ipconfig0 ip=10.0.50.31/24,gw=10.0.50.1
qm set 101 --net0 virtio,bridge=vmbr2
qm resize 101 scsi0 +50G

# Start
qm start 101
```

---

## 7. pvesh API Commands

### 7.1 Node Information

```bash
# List cluster nodes
pvesh get /nodes --output-format json-pretty

# Node status
pvesh get /nodes/pve-kubric-01/status --output-format json-pretty

# Resource usage
pvesh get /cluster/resources --type vm --output-format json-pretty
```

### 7.2 VM Operations via API

```bash
# List VMs
pvesh get /nodes/pve-kubric-01/qemu --output-format json-pretty

# Start VM
pvesh create /nodes/pve-kubric-01/qemu/101/status/start

# Stop VM
pvesh create /nodes/pve-kubric-01/qemu/101/status/stop

# Migrate VM to another node
pvesh create /nodes/pve-kubric-01/qemu/101/migrate \
  --target pve-kubric-02 --online 1
```

### 7.3 Storage Operations

```bash
# List storage
pvesh get /storage --output-format json-pretty

# List storage content
pvesh get /nodes/pve-kubric-01/storage/local/content --output-format json-pretty
```

### 7.4 API Token for Automation

```bash
# Create API token for Kubric automation
pveum user add kubric-automation@pve
pveum aclmod / -user kubric-automation@pve -role PVEAdmin
pveum user token add kubric-automation@pve kubric-token --privsep=0

# Token output:
# ┌──────────────┬──────────────────────────────────────┐
# │ key          │ value                                 │
# ├──────────────┼──────────────────────────────────────┤
# │ full-tokenid │ kubric-automation@pve!kubric-token   │
# │ info         │ {"privsep":"0"}                       │
# │ value        │ xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx │
# └──────────────┴──────────────────────────────────────┘

# Use in API calls:
curl -sk -H "Authorization: PVEAPIToken=kubric-automation@pve!kubric-token=<TOKEN>" \
  https://10.0.100.21:8006/api2/json/nodes
```

---

## 8. Backup Configuration

### 8.1 vzdump Schedule

```bash
# Create backup job for all VMs — weekly full, daily incremental
cat > /etc/pve/jobs.cfg <<'EOF'
vzdump: kubric-weekly-full
    all 1
    compress zstd
    dow sun
    enabled 1
    mailnotification always
    mailto infra@kubric.local
    mode snapshot
    node pve-kubric-01
    schedule sun 02:00
    storage kubric-nfs

vzdump: kubric-daily-snap
    all 1
    compress zstd
    enabled 1
    mailnotification failure
    mode snapshot
    node pve-kubric-01
    schedule mon-sat 03:00
    storage kubric-nfs
    maxfiles 7
EOF
```

---

## 9. Post-Bootstrap Verification

```bash
# Cluster health
pvecm status | grep -E "Quorate|Nodes"
# Quorate: Yes
# Nodes:   3

# All storage available
pvesm status
# All should show "active"

# HA status
ha-manager status

# Template available
qm list | grep "ubuntu-2204-cloud"

# Network bridges functional
for node in pve-kubric-01 pve-kubric-02 pve-kubric-03; do
  echo "=== $node ==="
  pvesh get /nodes/$node/network --output-format json | \
    python3 -c "import sys,json; [print(f\"{i['iface']:10s} {i.get('address','N/A'):16s} {i.get('type','')}\") for i in json.load(sys.stdin)]"
done

# Web UI accessible
curl -sk https://10.0.100.21:8006 | head -1
# Should return HTML
```
