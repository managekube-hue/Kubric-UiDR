# K-HV-VM-001 — Go API Server Cloud-Init Template

> **Template VM ID:** 9001  
> **Base Image:** Ubuntu 24.04 LTS (Noble Numbat) cloud image  
> **vCPU:** 4 | **RAM:** 8 GB | **Disk:** 50 GB (Ceph RBD)  
> **Target Nodes:** pve-kubric-01, pve-kubric-02  
> **Network:** vmbr2 (VLAN 50 — Workload)  
> **Role:** Runs Kubric Go API binary, connects to PostgreSQL + ClickHouse + NATS

---

## 1. Download Ubuntu Cloud Image

```bash
# On pve-kubric-01
cd /var/lib/vz/template/iso/

wget https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img

# Convert to QCOW2 if needed (Proxmox prefers raw for Ceph)
qemu-img convert -f qcow2 -O raw noble-server-cloudimg-amd64.img noble-server-cloudimg-amd64.raw
```

---

## 2. Create VM Template

```bash
# Create VM shell
qm create 9001 --name kubric-api-template --ostype l26 \
  --cpu cputype=host --cores 4 --sockets 1 \
  --memory 8192 --balloon 4096 \
  --net0 virtio,bridge=vmbr2,tag=50 \
  --scsihw virtio-scsi-single --machine q35 \
  --agent enabled=1 \
  --bios ovmf

# Import disk to Ceph
qm importdisk 9001 noble-server-cloudimg-amd64.raw kubric-ceph

# Attach imported disk
qm set 9001 --scsi0 kubric-ceph:vm-9001-disk-0,iothread=1,discard=on,ssd=1

# Add Cloud-Init drive
qm set 9001 --ide2 kubric-ceph:cloudinit

# Set boot order
qm set 9001 --boot order=scsi0

# Enable serial console for cloud-init
qm set 9001 --serial0 socket --vga serial0

# Resize disk
qm resize 9001 scsi0 50G
```

---

## 3. Cloud-Init Configuration

```bash
# Set cloud-init parameters
qm set 9001 --ciuser kubric --cipassword "$(openssl rand -base64 16)"
qm set 9001 --sshkeys /root/.ssh/kubric_deploy.pub
qm set 9001 --ipconfig0 ip=dhcp
qm set 9001 --nameserver 10.0.100.1
qm set 9001 --searchdomain kubric.local

# Custom cloud-init snippets
pvesh create /storage/local/content --filename kubric-api-cloud.yml --content snippets
```

### 3.1 Cloud-Init User Data

File: `/var/lib/vz/snippets/kubric-api-cloud.yml`

```yaml
#cloud-config
hostname: kubric-api-${instance_id}
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
package_upgrade: true
packages:
  - curl
  - wget
  - htop
  - net-tools
  - jq
  - ca-certificates
  - gnupg
  - unzip
  - qemu-guest-agent

write_files:
  # Kubric API systemd unit
  - path: /etc/systemd/system/kubric-api.service
    permissions: "0644"
    content: |
      [Unit]
      Description=Kubric API Server
      After=network-online.target
      Wants=network-online.target
      StartLimitIntervalSec=60
      StartLimitBurst=5

      [Service]
      Type=simple
      User=kubric
      Group=kubric
      WorkingDirectory=/opt/kubric
      ExecStart=/opt/kubric/bin/kubric-api serve
      Restart=on-failure
      RestartSec=5
      StandardOutput=journal
      StandardError=journal
      SyslogIdentifier=kubric-api

      # Environment
      EnvironmentFile=/opt/kubric/etc/kubric-api.env

      # Security hardening
      NoNewPrivileges=true
      ProtectSystem=strict
      ProtectHome=true
      ReadWritePaths=/opt/kubric/data /var/log/kubric
      PrivateTmp=true
      ProtectKernelModules=true
      ProtectKernelTunables=true

      # Resource limits
      LimitNOFILE=65536
      LimitNPROC=4096
      MemoryMax=6G
      CPUQuota=350%

      [Install]
      WantedBy=multi-user.target

  # Kubric API environment
  - path: /opt/kubric/etc/kubric-api.env
    permissions: "0640"
    content: |
      # ─── Server ───
      KUBRIC_LISTEN_ADDR=0.0.0.0:8080
      KUBRIC_GRPC_ADDR=0.0.0.0:9090
      KUBRIC_METRICS_ADDR=0.0.0.0:9102
      KUBRIC_LOG_LEVEL=info
      KUBRIC_LOG_FORMAT=json

      # ─── PostgreSQL ───
      KUBRIC_PG_HOST=10.0.50.22
      KUBRIC_PG_PORT=5432
      KUBRIC_PG_DATABASE=kubric_core
      KUBRIC_PG_USER=kubric_api
      KUBRIC_PG_PASSWORD=CHANGEME_PG_PASS
      KUBRIC_PG_SSLMODE=require
      KUBRIC_PG_POOL_MAX=50
      KUBRIC_PG_POOL_MIN=10

      # ─── ClickHouse ───
      KUBRIC_CH_HOSTS=10.0.50.21:9440,10.0.50.22:9440
      KUBRIC_CH_DATABASE=kubric_telemetry
      KUBRIC_CH_USER=kubric_writer
      KUBRIC_CH_PASSWORD=CHANGEME_CH_PASS
      KUBRIC_CH_TLS=true

      # ─── NATS ───
      KUBRIC_NATS_URL=nats://10.0.50.21:4222,nats://10.0.50.22:4222,nats://10.0.50.23:4222
      KUBRIC_NATS_CREDS=/opt/kubric/etc/nats-kubric-api.creds

      # ─── MinIO (S3) ───
      KUBRIC_S3_ENDPOINT=http://10.0.50.23:9000
      KUBRIC_S3_ACCESS_KEY=kubric-api
      KUBRIC_S3_SECRET_KEY=CHANGEME_MINIO_PASS
      KUBRIC_S3_BUCKET=kubric-data
      KUBRIC_S3_REGION=us-east-1

      # ─── KAI (AI Backend) ───
      KUBRIC_KAI_GRPC_ADDR=10.0.50.23:50051

      # ─── JWT ───
      KUBRIC_JWT_SECRET=CHANGEME_JWT_SECRET
      KUBRIC_JWT_EXPIRY=24h

  # Sysctl tuning for Go HTTP server
  - path: /etc/sysctl.d/90-kubric-api.conf
    permissions: "0644"
    content: |
      net.core.somaxconn = 65535
      net.ipv4.tcp_max_syn_backlog = 65535
      net.ipv4.tcp_tw_reuse = 1
      net.ipv4.tcp_fin_timeout = 15
      net.core.netdev_max_backlog = 65535
      net.ipv4.ip_local_port_range = 1024 65535
      vm.swappiness = 10
      fs.file-max = 2097152

  # Directory structure
  - path: /opt/kubric/.keep
    permissions: "0644"
    content: ""

runcmd:
  # Create directories
  - mkdir -p /opt/kubric/{bin,etc,data} /var/log/kubric
  - chown -R kubric:kubric /opt/kubric /var/log/kubric

  # Apply sysctl
  - sysctl --system

  # Enable guest agent
  - systemctl enable --now qemu-guest-agent

  # Install Go runtime (for plugin support)
  - |
    GO_VERSION=1.23.4
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/golang.sh
    rm /tmp/go.tar.gz

  # Pull latest API binary from MinIO (CI artifact)
  - |
    curl -sL https://dl.min.io/client/mc/release/linux-amd64/mc -o /usr/local/bin/mc
    chmod +x /usr/local/bin/mc
    mc alias set kubric http://10.0.50.23:9000 kubric-admin CHANGEME_MINIO_ADMIN
    mc cp kubric/kubric-artifacts/latest/kubric-api /opt/kubric/bin/kubric-api
    chmod +x /opt/kubric/bin/kubric-api
    chown kubric:kubric /opt/kubric/bin/kubric-api

  # Enable and start service
  - systemctl daemon-reload
  - systemctl enable kubric-api.service

  # Log rotation
  - |
    cat > /etc/logrotate.d/kubric-api <<'EOF'
    /var/log/kubric/*.log {
        daily
        rotate 14
        compress
        delaycompress
        missingok
        notifempty
        create 0644 kubric kubric
        postrotate
            systemctl reload kubric-api || true
        endscript
    }
    EOF

final_message: "Kubric API VM ready. Run: systemctl start kubric-api"
```

### 3.2 Apply Snippet to Template

```bash
qm set 9001 --cicustom "user=local:snippets/kubric-api-cloud.yml"
```

---

## 4. Convert to Template

```bash
qm template 9001
```

---

## 5. Clone and Deploy

### 5.1 Clone API Instance

```bash
# Clone for Node 1 (primary API)
qm clone 9001 201 --name kubric-api-01 --full --target pve-kubric-01 --storage kubric-ceph
qm set 201 --ipconfig0 ip=10.0.50.101/24,gw=10.0.50.1
qm start 201

# Clone for Node 2 (API replica)
qm clone 9001 202 --name kubric-api-02 --full --target pve-kubric-02 --storage kubric-ceph
qm set 202 --ipconfig0 ip=10.0.50.102/24,gw=10.0.50.1
qm start 202
```

### 5.2 Post-Start Configuration

```bash
# Wait for cloud-init completion (2-3 minutes)
ssh kubric@10.0.50.101 "cloud-init status --wait"

# Update actual passwords
ssh kubric@10.0.50.101 "sudo sed -i \
  -e 's/CHANGEME_PG_PASS/${PG_API_PASS}/' \
  -e 's/CHANGEME_CH_PASS/${CH_WRITER_PASS}/' \
  -e 's/CHANGEME_MINIO_PASS/${MINIO_API_PASS}/' \
  -e 's/CHANGEME_JWT_SECRET/${JWT_SECRET}/' \
  /opt/kubric/etc/kubric-api.env"

# Start the service
ssh kubric@10.0.50.101 "sudo systemctl start kubric-api"
```

---

## 6. Health Check Verification

```bash
# HTTP health
curl -s http://10.0.50.101:8080/healthz | jq .
# Expected: {"status":"ok","version":"v0.1.0","uptime":"12s"}

# gRPC health
grpcurl -plaintext 10.0.50.101:9090 grpc.health.v1.Health/Check
# Expected: {"status":"SERVING"}

# Prometheus metrics
curl -s http://10.0.50.101:9102/metrics | head -20
# Expected: kubric_api_requests_total, kubric_api_request_duration_seconds, ...

# Systemd status
ssh kubric@10.0.50.101 "systemctl status kubric-api"
```

---

## 7. Update / Rolling Deploy

```bash
# Pull new binary
ssh kubric@10.0.50.101 "
  mc cp kubric/kubric-artifacts/${COMMIT_SHA}/kubric-api /opt/kubric/bin/kubric-api.new
  chmod +x /opt/kubric/bin/kubric-api.new
  sudo systemctl stop kubric-api
  mv /opt/kubric/bin/kubric-api /opt/kubric/bin/kubric-api.bak
  mv /opt/kubric/bin/kubric-api.new /opt/kubric/bin/kubric-api
  sudo systemctl start kubric-api
"

# Verify health
curl -s http://10.0.50.101:8080/healthz | jq .version
```
