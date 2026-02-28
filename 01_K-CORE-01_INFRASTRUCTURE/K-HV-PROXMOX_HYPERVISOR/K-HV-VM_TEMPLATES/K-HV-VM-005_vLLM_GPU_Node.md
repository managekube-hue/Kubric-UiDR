# K-HV-VM-005 — vLLM GPU Inference Node

> **Template VM ID:** 9005  
> **Base Image:** Ubuntu 24.04 LTS cloud image  
> **vCPU:** 8 | **RAM:** 32 GB | **Disk:** 200 GB (Ceph RBD, model cache)  
> **Target Node:** pve-kubric-03 (GPU node)  
> **VM ID:** 232  
> **IP:** 10.0.50.36/24  
> **GPU:** NVIDIA Tesla T4 16GB or A2 16GB (VFIO passthrough)  
> **Ports:** 8000 (vLLM OpenAI-compatible API), 9102 (Prometheus metrics)  
> **Use Case:** High-throughput batch inference, OpenAI-compatible API for KAI orchestration

---

## 1. Create VM Template

```bash
qm create 9005 --name vllm-template --ostype l26 \
  --cpu cputype=host --cores 8 --sockets 1 \
  --memory 32768 --balloon 0 \
  --net0 virtio,bridge=vmbr2,tag=50 \
  --scsihw virtio-scsi-single --machine q35 \
  --agent enabled=1 \
  --bios ovmf

qm importdisk 9005 /var/lib/vz/template/iso/noble-server-cloudimg-amd64.raw kubric-ceph
qm set 9005 --scsi0 kubric-ceph:vm-9005-disk-0,iothread=1,discard=on,ssd=1
qm resize 9005 scsi0 200G

qm set 9005 --efidisk0 kubric-ceph:1,efitype=4m,pre-enrolled-keys=1

qm set 9005 --ide2 kubric-ceph:cloudinit
qm set 9005 --boot order=scsi0
qm set 9005 --serial0 socket --vga serial0
qm set 9005 --ciuser kubric --sshkeys /root/.ssh/kubric_deploy.pub
qm set 9005 --nameserver 10.0.100.1 --searchdomain kubric.local

# GPU passthrough
qm set 9005 --hostpci0 3b:00.0,pcie=1,x-vga=0
```

---

## 2. Cloud-Init User Data

File: `/var/lib/vz/snippets/vllm-cloud.yml`

```yaml
#cloud-config
hostname: vllm-kubric
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
  - python3
  - python3-pip
  - python3-venv
  - git

write_files:
  # vLLM systemd service
  - path: /etc/systemd/system/vllm.service
    permissions: "0644"
    content: |
      [Unit]
      Description=vLLM OpenAI-Compatible Inference Server
      After=network-online.target
      Wants=network-online.target
      StartLimitIntervalSec=300
      StartLimitBurst=3

      [Service]
      Type=simple
      User=kubric
      Group=kubric
      WorkingDirectory=/opt/vllm
      EnvironmentFile=/opt/vllm/vllm.env
      ExecStart=/opt/vllm/venv/bin/python -m vllm.entrypoints.openai.api_server \
        --host 0.0.0.0 \
        --port 8000 \
        --model ${VLLM_MODEL} \
        --max-model-len ${VLLM_MAX_MODEL_LEN} \
        --gpu-memory-utilization ${VLLM_GPU_MEM_UTIL} \
        --tensor-parallel-size ${VLLM_TP_SIZE} \
        --dtype ${VLLM_DTYPE} \
        --quantization ${VLLM_QUANTIZATION} \
        --max-num-seqs ${VLLM_MAX_SEQS} \
        --download-dir /opt/vllm/models \
        --trust-remote-code \
        --enforce-eager
      Restart=on-failure
      RestartSec=30
      StandardOutput=journal
      StandardError=journal
      SyslogIdentifier=vllm

      # Resource limits
      LimitNOFILE=65536
      MemoryMax=30G

      [Install]
      WantedBy=multi-user.target

  # vLLM environment
  - path: /opt/vllm/vllm.env
    permissions: "0640"
    content: |
      # Model selection
      VLLM_MODEL=TheBloke/Mistral-7B-Instruct-v0.2-AWQ
      VLLM_MAX_MODEL_LEN=8192
      VLLM_GPU_MEM_UTIL=0.90
      VLLM_TP_SIZE=1
      VLLM_DTYPE=auto
      VLLM_QUANTIZATION=awq
      VLLM_MAX_SEQS=64

      # HuggingFace cache
      HF_HOME=/opt/vllm/hf-cache
      TRANSFORMERS_CACHE=/opt/vllm/hf-cache

      # CUDA
      CUDA_VISIBLE_DEVICES=0

  # Sysctl
  - path: /etc/sysctl.d/90-vllm.conf
    permissions: "0644"
    content: |
      vm.swappiness = 1
      vm.overcommit_memory = 1
      kernel.shmmax = 17179869184
      net.core.somaxconn = 65535

runcmd:
  - systemctl enable --now qemu-guest-agent

  # Install NVIDIA driver
  - ubuntu-drivers install

  # Install NVIDIA Container Toolkit
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

  # Create vLLM directories
  - mkdir -p /opt/vllm/{models,hf-cache,venv}
  - chown -R kubric:kubric /opt/vllm

  # Install vLLM in virtualenv
  - |
    sudo -u kubric bash -c '
      python3 -m venv /opt/vllm/venv
      source /opt/vllm/venv/bin/activate
      pip install --upgrade pip wheel
      pip install vllm==0.6.4
      pip install flash-attn --no-build-isolation
    '

  - sysctl --system
  - systemctl daemon-reload
  - systemctl enable vllm

final_message: "vLLM GPU VM ready. Start: systemctl start vllm"
```

```bash
qm set 9005 --cicustom "user=local:snippets/vllm-cloud.yml"
qm template 9005
```

---

## 3. Clone and Deploy

```bash
qm clone 9005 232 --name vllm-kubric --full --target pve-kubric-03 --storage kubric-ceph
qm set 232 --ipconfig0 ip=10.0.50.36/24,gw=10.0.50.1
qm start 232

ssh kubric@10.0.50.36 "cloud-init status --wait"
```

---

## 4. Model Selection Guide

| Model | VRAM | Context | Use Case |
|-------|------|---------|----------|
| `TheBloke/Mistral-7B-Instruct-v0.2-AWQ` | ~5 GB | 8K | General security analysis |
| `TheBloke/CodeLlama-7B-Instruct-AWQ` | ~5 GB | 16K | Detection rule generation |
| `TheBloke/Llama-2-13B-chat-AWQ` | ~9 GB | 4K | Deep threat analysis |
| `microsoft/Phi-3-mini-4k-instruct` | ~3 GB | 4K | Fast triage/classification |
| `BAAI/bge-large-en-v1.5` | ~1.3 GB | 512 | Embeddings for RAG |

### Switch Model

```bash
# Update model in env file
ssh kubric@10.0.50.36 "sudo sed -i \
  's|VLLM_MODEL=.*|VLLM_MODEL=TheBloke/CodeLlama-7B-Instruct-AWQ|' \
  /opt/vllm/vllm.env"

# Restart vLLM
ssh kubric@10.0.50.36 "sudo systemctl restart vllm"
```

---

## 5. Docker Alternative Deployment

For container-based deployment (useful for K8s migration later):

```bash
ssh kubric@10.0.50.36 "tee /opt/vllm/docker-compose.yml" <<'EOF'
version: "3.8"

services:
  vllm:
    image: vllm/vllm-openai:v0.6.4
    container_name: vllm-server
    restart: unless-stopped
    runtime: nvidia
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    ports:
      - "8000:8000"
    environment:
      - CUDA_VISIBLE_DEVICES=0
      - HF_HOME=/models/hf-cache
    volumes:
      - /opt/vllm/models:/models
      - /opt/vllm/hf-cache:/models/hf-cache
    command: >
      --model TheBloke/Mistral-7B-Instruct-v0.2-AWQ
      --max-model-len 8192
      --gpu-memory-utilization 0.90
      --quantization awq
      --max-num-seqs 64
      --dtype auto
      --trust-remote-code
      --enforce-eager
      --host 0.0.0.0
      --port 8000
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 120s
EOF
```

---

## 6. API Usage (OpenAI-Compatible)

### 6.1 Chat Completion

```bash
curl -s http://10.0.50.36:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "TheBloke/Mistral-7B-Instruct-v0.2-AWQ",
    "messages": [
      {"role": "system", "content": "You are a cybersecurity analyst. Analyze the following alert and provide MITRE ATT&CK context."},
      {"role": "user", "content": "Alert: PowerShell encoded command execution detected. Process: powershell.exe -enc SQBuAHYAbwBrAGUALQBXAGUAYgBSAGUAcQB1AGUAcwB0AA== Host: WORKSTATION-42 User: jsmith"}
    ],
    "max_tokens": 1024,
    "temperature": 0.2,
    "stream": false
  }' | jq '.choices[0].message.content'
```

### 6.2 Batch Inference

```bash
# Multiple alerts in parallel
curl -s http://10.0.50.36:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "TheBloke/Mistral-7B-Instruct-v0.2-AWQ",
    "prompt": ["Classify this alert severity: Unusual outbound DNS to Tor exit node",
               "Classify this alert severity: Failed login attempt from internal IP",
               "Classify this alert severity: Mimikatz binary detected in temp folder"],
    "max_tokens": 256,
    "temperature": 0.1
  }' | jq '.choices[].text'
```

### 6.3 Python Client (KAI Integration)

```python
# kai/inference/vllm_client.py
from openai import OpenAI

client = OpenAI(
    base_url="http://10.0.50.36:8000/v1",
    api_key="not-needed",  # vLLM doesn't require API key by default
)

def analyze_alert(alert_text: str) -> str:
    response = client.chat.completions.create(
        model="TheBloke/Mistral-7B-Instruct-v0.2-AWQ",
        messages=[
            {"role": "system", "content": "You are a SOC analyst. Provide MITRE ATT&CK mapping and severity assessment."},
            {"role": "user", "content": alert_text},
        ],
        max_tokens=1024,
        temperature=0.2,
    )
    return response.choices[0].message.content

def get_embeddings(texts: list[str]) -> list[list[float]]:
    """Batch embedding generation for RAG."""
    response = client.embeddings.create(
        model="BAAI/bge-large-en-v1.5",
        input=texts,
    )
    return [item.embedding for item in response.data]
```

---

## 7. Performance Tuning

```bash
# Benchmark throughput
ssh kubric@10.0.50.36 <<'EOF'
source /opt/vllm/venv/bin/activate
python -m vllm.entrypoints.openai.api_server --help | grep -E "batch|parallel|buffer"

# Benchmark with vegeta (HTTP load tester)
apt-get install -y vegeta || go install github.com/tsenart/vegeta@latest

echo 'POST http://10.0.50.36:8000/v1/completions
Content-Type: application/json
@/tmp/vllm-bench.json' > /tmp/vegeta-target.txt

cat > /tmp/vllm-bench.json <<'JSON'
{
  "model": "TheBloke/Mistral-7B-Instruct-v0.2-AWQ",
  "prompt": "Analyze the following security log entry for threats:",
  "max_tokens": 128,
  "temperature": 0
}
JSON

vegeta attack -targets=/tmp/vegeta-target.txt -rate=10/s -duration=30s | vegeta report
EOF
```

### GPU Monitoring During Load

```bash
ssh kubric@10.0.50.36 "watch -n 1 nvidia-smi"

# Detailed GPU metrics
ssh kubric@10.0.50.36 "nvidia-smi dmon -s pucvmet -d 1"
# Columns: pwr, gtemp, mtemp, sm%, mem%, enc%, dec%, mclk, pclk
```

---

## 8. Prometheus Metrics

vLLM exposes Prometheus metrics natively:

```bash
# Scrape config for Prometheus
# Add to prometheus.yml on monitoring node:
#   - job_name: 'vllm'
#     static_configs:
#       - targets: ['10.0.50.36:8000']

# Available metrics
curl -s http://10.0.50.36:8000/metrics | grep -E "^vllm_"
# vllm:num_requests_running
# vllm:num_requests_waiting
# vllm:gpu_cache_usage_perc
# vllm:cpu_cache_usage_perc
# vllm:avg_prompt_throughput_toks_per_s
# vllm:avg_generation_throughput_toks_per_s
# vllm:request_success_total
# vllm:time_to_first_token_seconds
# vllm:e2e_request_latency_seconds
```

---

## 9. Verification

```bash
# GPU presence
ssh kubric@10.0.50.36 "nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv"
# Expected: Tesla T4, 16384 MiB, 550.xx.xx

# vLLM health
curl -s http://10.0.50.36:8000/health
# Expected: {"status":"ok"}

# Model loaded
curl -s http://10.0.50.36:8000/v1/models | jq '.data[].id'
# Expected: "TheBloke/Mistral-7B-Instruct-v0.2-AWQ"

# Quick inference
time curl -s http://10.0.50.36:8000/v1/completions -d '{
  "model": "TheBloke/Mistral-7B-Instruct-v0.2-AWQ",
  "prompt": "What is a zero-day exploit?",
  "max_tokens": 64
}' | jq '.choices[0].text'

# Systemd status
ssh kubric@10.0.50.36 "systemctl status vllm"
```
