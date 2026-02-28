# K-HV-VM-004 — Ollama LLM Inference Server

> **Template VM ID:** 9004  
> **Base Image:** Ubuntu 24.04 LTS cloud image  
> **vCPU:** 8 | **RAM:** 32 GB | **Disk:** 100 GB (Ceph RBD)  
> **Target Node:** pve-kubric-03 (GPU node)  
> **VM ID:** 231  
> **IP:** 10.0.50.35/24  
> **GPU:** NVIDIA Tesla T4 16GB or A2 16GB (VFIO passthrough)  
> **Ports:** 11434 (Ollama API), 9102 (Prometheus metrics)

---

## 1. VFIO GPU Passthrough Prerequisites

On the Proxmox host (pve-kubric-03), GPU passthrough must be configured:

```bash
# Verify IOMMU is enabled (done in K-HW-003)
dmesg | grep -i iommu
# Expected: DMAR: IOMMU enabled

# Identify GPU PCI IDs
lspci -nn | grep -i nvidia
# Example: 3b:00.0 3D controller [0302]: NVIDIA Corporation TU104GL [Tesla T4] [10de:1eb8]
# Example: 3b:00.0 3D controller [0302]: NVIDIA Corporation GA107 [A2] [10de:25b6]

# Verify VFIO binding
lspci -ks 3b:00.0 | grep "Kernel driver"
# Expected: Kernel driver in use: vfio-pci
```

---

## 2. Create VM Template

```bash
qm create 9004 --name ollama-template --ostype l26 \
  --cpu cputype=host --cores 8 --sockets 1 \
  --memory 32768 --balloon 0 \
  --net0 virtio,bridge=vmbr2,tag=50 \
  --scsihw virtio-scsi-single --machine q35 \
  --agent enabled=1 \
  --bios ovmf

# Import OS disk
qm importdisk 9004 /var/lib/vz/template/iso/noble-server-cloudimg-amd64.raw kubric-ceph
qm set 9004 --scsi0 kubric-ceph:vm-9004-disk-0,iothread=1,discard=on,ssd=1
qm resize 9004 scsi0 100G

# EFI disk for OVMF/GPU passthrough
qm set 9004 --efidisk0 kubric-ceph:1,efitype=4m,pre-enrolled-keys=1

# Cloud-init
qm set 9004 --ide2 kubric-ceph:cloudinit
qm set 9004 --boot order=scsi0
qm set 9004 --serial0 socket --vga serial0
qm set 9004 --ciuser kubric --sshkeys /root/.ssh/kubric_deploy.pub
qm set 9004 --nameserver 10.0.100.1 --searchdomain kubric.local

# GPU passthrough (PCI device)
qm set 9004 --hostpci0 3b:00.0,pcie=1,x-vga=0
```

---

## 3. Cloud-Init User Data

File: `/var/lib/vz/snippets/ollama-cloud.yml`

```yaml
#cloud-config
hostname: ollama-kubric
manage_etc_hosts: true
timezone: UTC

users:
  - name: kubric
    groups: [sudo, docker]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... kubric-deploy

package_update: true
packages:
  - curl
  - wget
  - htop
  - jq
  - pciutils
  - qemu-guest-agent
  - linux-headers-generic
  - build-essential

write_files:
  # Ollama systemd override
  - path: /etc/systemd/system/ollama.service.d/override.conf
    permissions: "0644"
    content: |
      [Service]
      Environment="OLLAMA_HOST=0.0.0.0:11434"
      Environment="OLLAMA_MODELS=/opt/ollama/models"
      Environment="OLLAMA_NUM_PARALLEL=4"
      Environment="OLLAMA_MAX_LOADED_MODELS=2"
      Environment="OLLAMA_KEEP_ALIVE=10m"
      Environment="OLLAMA_FLASH_ATTENTION=1"

  # Sysctl for GPU workloads
  - path: /etc/sysctl.d/90-gpu.conf
    permissions: "0644"
    content: |
      vm.swappiness = 1
      vm.overcommit_memory = 1
      kernel.shmmax = 8589934592

runcmd:
  # Enable guest agent
  - systemctl enable --now qemu-guest-agent

  # Install NVIDIA driver
  - |
    ubuntu-drivers install
    # Or specific version:
    # apt-get install -y nvidia-driver-550-server nvidia-utils-550-server

  # Verify GPU
  - nvidia-smi

  # Install NVIDIA Container Toolkit (for future Docker usage)
  - |
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
      gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
      sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
      tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
    apt-get update
    apt-get install -y nvidia-container-toolkit

  # Install Docker
  - curl -fsSL https://get.docker.com | sh
  - nvidia-ctk runtime configure --runtime=docker
  - systemctl restart docker

  # Install Ollama
  - curl -fsSL https://ollama.com/install.sh | sh
  - mkdir -p /opt/ollama/models
  - chown ollama:ollama /opt/ollama/models

  # Apply systemd override
  - systemctl daemon-reload
  - systemctl enable ollama

  # Apply sysctl
  - sysctl --system

final_message: "Ollama GPU VM ready. Pull models then start."
```

```bash
qm set 9004 --cicustom "user=local:snippets/ollama-cloud.yml"
qm template 9004
```

---

## 4. Clone and Deploy

```bash
qm clone 9004 231 --name ollama-kubric --full --target pve-kubric-03 --storage kubric-ceph
qm set 231 --ipconfig0 ip=10.0.50.35/24,gw=10.0.50.1
qm start 231

ssh kubric@10.0.50.35 "cloud-init status --wait"
```

---

## 5. Pull Models

```bash
ssh kubric@10.0.50.35 <<'EOF'
# Start Ollama
sudo systemctl start ollama
sleep 5

# ─── Security-Focused Models ───

# Primary analysis model (7B, fits in 16GB VRAM)
ollama pull llama3.1:8b

# Code analysis for detection rule generation
ollama pull codellama:7b

# Small fast model for classification/triage
ollama pull phi3:mini

# Embedding model for RAG
ollama pull nomic-embed-text

# Verify models loaded
ollama list
EOF
```

Expected output:

```
NAME               ID          SIZE    MODIFIED
llama3.1:8b        <hash>      4.7 GB  Just now
codellama:7b       <hash>      3.8 GB  Just now
phi3:mini          <hash>      2.3 GB  Just now
nomic-embed-text   <hash>      274 MB  Just now
```

---

## 6. Ollama Modelfile (Custom Security Model)

```bash
ssh kubric@10.0.50.35 "tee /opt/ollama/Modelfile.kubric-sec" <<'MODELFILE'
FROM llama3.1:8b

SYSTEM """
You are Kubric Security Analyst, an expert AI assistant specialized in:
- Cybersecurity threat analysis (MITRE ATT&CK framework)
- SIEM alert triage and prioritization
- Sigma/YARA/Suricata rule generation
- Incident response procedures
- Log analysis and anomaly detection
- Vulnerability assessment (CVE analysis)

When analyzing alerts:
1. Assess severity based on MITRE ATT&CK tactics/techniques
2. Cross-reference with known TTPs
3. Provide concrete remediation steps
4. Suggest relevant detection rules

Always include confidence levels (high/medium/low) and cite specific MITRE technique IDs.
Format responses with clear sections: Summary, Analysis, Recommendations, Detection Rules.
"""

PARAMETER temperature 0.3
PARAMETER top_k 40
PARAMETER top_p 0.9
PARAMETER num_ctx 8192
PARAMETER repeat_penalty 1.1
MODELFILE

# Create custom model
ssh kubric@10.0.50.35 "ollama create kubric-sec -f /opt/ollama/Modelfile.kubric-sec"
```

---

## 7. API Usage from KAI Service

```bash
# Test inference
curl -s http://10.0.50.35:11434/api/generate -d '{
  "model": "kubric-sec",
  "prompt": "Analyze this alert: Multiple failed SSH login attempts from 185.220.101.x followed by successful login. Source: auth.log, Rule: sigma/win_ssh_bruteforce",
  "stream": false
}' | jq '.response'

# Embeddings for RAG
curl -s http://10.0.50.35:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "Lateral movement via PsExec with ADMIN$ share access"
}' | jq '.embedding | length'
# Expected: 768

# List running models
curl -s http://10.0.50.35:11434/api/tags | jq '.models[].name'

# Model info
curl -s http://10.0.50.35:11434/api/show -d '{"name":"kubric-sec"}' | jq '.details'
```

---

## 8. Prometheus Metrics & Monitoring

```bash
# Ollama exposes basic metrics — add custom exporter
ssh kubric@10.0.50.35 "sudo tee /opt/ollama/metrics.sh" <<'SCRIPT'
#!/bin/bash
# Simple metrics exporter for Prometheus
while true; do
  cat <<METRICS | nc -l -p 9102 -q 1
HTTP/1.1 200 OK
Content-Type: text/plain

# HELP ollama_models_loaded Number of loaded models
# TYPE ollama_models_loaded gauge
ollama_models_loaded $(curl -s http://localhost:11434/api/tags 2>/dev/null | jq '.models | length' 2>/dev/null || echo 0)

# HELP nvidia_gpu_temp_celsius GPU temperature
# TYPE nvidia_gpu_temp_celsius gauge
nvidia_gpu_temp_celsius $(nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader,nounits 2>/dev/null || echo 0)

# HELP nvidia_gpu_utilization GPU utilization percent
# TYPE nvidia_gpu_utilization gauge
nvidia_gpu_utilization $(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null || echo 0)

# HELP nvidia_gpu_memory_used_mb GPU memory used
# TYPE nvidia_gpu_memory_used_mb gauge
nvidia_gpu_memory_used_mb $(nvidia-smi --query-gpu=memory.used --format=csv,noheader,nounits 2>/dev/null || echo 0)

# HELP nvidia_gpu_memory_total_mb GPU memory total
# TYPE nvidia_gpu_memory_total_mb gauge
nvidia_gpu_memory_total_mb $(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null || echo 0)
METRICS
done
SCRIPT

ssh kubric@10.0.50.35 "chmod +x /opt/ollama/metrics.sh"

# Run as systemd service
ssh kubric@10.0.50.35 "sudo tee /etc/systemd/system/ollama-metrics.service" <<'UNIT'
[Unit]
Description=Ollama Prometheus Metrics
After=ollama.service

[Service]
Type=simple
ExecStart=/opt/ollama/metrics.sh
Restart=always

[Install]
WantedBy=multi-user.target
UNIT

ssh kubric@10.0.50.35 "sudo systemctl daemon-reload && sudo systemctl enable --now ollama-metrics"
```

---

## 9. Verification

```bash
# GPU visible
ssh kubric@10.0.50.35 "nvidia-smi"
# Expected: Tesla T4 or A2 with 16GB

# Ollama running
ssh kubric@10.0.50.35 "systemctl status ollama"
curl -s http://10.0.50.35:11434/api/tags | jq '.models[].name'

# Inference test
time curl -s http://10.0.50.35:11434/api/generate -d '{
  "model": "phi3:mini",
  "prompt": "What is MITRE ATT&CK T1059?",
  "stream": false
}' | jq '.response' | head -5

# GPU utilization during inference
ssh kubric@10.0.50.35 "nvidia-smi dmon -s u -d 1 -c 5"

# Metrics endpoint
curl -s http://10.0.50.35:9102/metrics
```
