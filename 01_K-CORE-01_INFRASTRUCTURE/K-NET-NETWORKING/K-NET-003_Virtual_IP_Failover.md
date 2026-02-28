# K-NET-003 — Virtual IP Failover (Keepalived VRRP)

> **Primary VIP:** 10.0.100.100 — K8s API Server  
> **Secondary VIP:** 10.0.100.101 — Kubric API Gateway  
> **VRRP:** Virtual Router Redundancy Protocol via Keepalived  
> **Nodes:** All 3 Proxmox hosts participate

---

## 1. Install Keepalived

```bash
# On ALL 3 nodes
apt-get update && apt-get install -y keepalived

# Enable and start
systemctl enable keepalived
```

---

## 2. VIP Assignments

| VIP | Purpose | VRRP Instance | Priority (Node1/2/3) |
|-----|---------|---------------|----------------------|
| 10.0.100.100 | K8s API (kube-apiserver:6443) | VI_K8S | 150 / 140 / 130 |
| 10.0.100.101 | Kubric API Gateway (ksvc:8080) | VI_KUBRIC | 130 / 150 / 140 |

> **Design:** VIPs are split across nodes for load distribution. Node 1 is primary for K8s API; Node 2 is primary for Kubric API.

---

## 3. Health Check Scripts

### 3.1 K8s API Health Check

File: `/etc/keepalived/check_k8s_api.sh` (all nodes)

```bash
#!/bin/bash
# Check if kube-apiserver is responding on this node
# Returns 0 (healthy) or 1 (unhealthy)

APISERVER_URL="https://127.0.0.1:6443/healthz"
TIMEOUT=3

response=$(curl -sk --max-time $TIMEOUT -o /dev/null -w "%{http_code}" "$APISERVER_URL" 2>/dev/null)

if [ "$response" = "200" ]; then
    exit 0
else
    # Log failure for debugging
    echo "$(date): K8s API health check failed (HTTP $response)" >> /var/log/keepalived-health.log
    exit 1
fi
```

```bash
chmod +x /etc/keepalived/check_k8s_api.sh
```

### 3.2 Kubric API Health Check

File: `/etc/keepalived/check_kubric_api.sh` (all nodes)

```bash
#!/bin/bash
# Check if Kubric API gateway (ksvc) is responding
KUBRIC_URL="http://127.0.0.1:8080/healthz"
TIMEOUT=3

response=$(curl -s --max-time $TIMEOUT -o /dev/null -w "%{http_code}" "$KUBRIC_URL" 2>/dev/null)

if [ "$response" = "200" ]; then
    exit 0
else
    echo "$(date): Kubric API health check failed (HTTP $response)" >> /var/log/keepalived-health.log
    exit 1
fi
```

```bash
chmod +x /etc/keepalived/check_kubric_api.sh
```

### 3.3 HAProxy Health Check (if fronting services)

File: `/etc/keepalived/check_haproxy.sh`

```bash
#!/bin/bash
# Check if HAProxy is running and responsive
if ! pidof haproxy > /dev/null 2>&1; then
    exit 1
fi

response=$(curl -s --max-time 2 -o /dev/null -w "%{http_code}" "http://127.0.0.1:9999/stats" 2>/dev/null)
if [ "$response" = "200" ] || [ "$response" = "401" ]; then
    exit 0
else
    exit 1
fi
```

```bash
chmod +x /etc/keepalived/check_haproxy.sh
```

---

## 4. Keepalived Configuration

### 4.1 Node 1 — pve-kubric-01

File: `/etc/keepalived/keepalived.conf`

```ini
global_defs {
    router_id pve-kubric-01
    enable_script_security
    script_user root
    vrrp_garp_master_refresh 60
    vrrp_garp_master_repeat 3
    vrrp_garp_interval 0.001
}

# ─── Health Check Definitions ───

vrrp_script chk_k8s_api {
    script "/etc/keepalived/check_k8s_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

vrrp_script chk_kubric_api {
    script "/etc/keepalived/check_kubric_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

# ─── VRRP Instance: K8s API VIP ───

vrrp_instance VI_K8S {
    state MASTER
    interface vmbr0
    virtual_router_id 51
    priority 150
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass K8sVRRP#51!
    }

    unicast_src_ip 10.0.100.21
    unicast_peer {
        10.0.100.22
        10.0.100.23
    }

    virtual_ipaddress {
        10.0.100.100/24 dev vmbr0 label vmbr0:vip0
    }

    track_script {
        chk_k8s_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_K8S"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_K8S"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_K8S"
}

# ─── VRRP Instance: Kubric API VIP ───

vrrp_instance VI_KUBRIC {
    state BACKUP
    interface vmbr0
    virtual_router_id 52
    priority 130
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass KubrVRRP#52!
    }

    unicast_src_ip 10.0.100.21
    unicast_peer {
        10.0.100.22
        10.0.100.23
    }

    virtual_ipaddress {
        10.0.100.101/24 dev vmbr0 label vmbr0:vip1
    }

    track_script {
        chk_kubric_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_KUBRIC"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_KUBRIC"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_KUBRIC"
}
```

### 4.2 Node 2 — pve-kubric-02

File: `/etc/keepalived/keepalived.conf`

```ini
global_defs {
    router_id pve-kubric-02
    enable_script_security
    script_user root
    vrrp_garp_master_refresh 60
    vrrp_garp_master_repeat 3
}

vrrp_script chk_k8s_api {
    script "/etc/keepalived/check_k8s_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

vrrp_script chk_kubric_api {
    script "/etc/keepalived/check_kubric_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

# K8s API — BACKUP on Node 2
vrrp_instance VI_K8S {
    state BACKUP
    interface vmbr0
    virtual_router_id 51
    priority 140
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass K8sVRRP#51!
    }

    unicast_src_ip 10.0.100.22
    unicast_peer {
        10.0.100.21
        10.0.100.23
    }

    virtual_ipaddress {
        10.0.100.100/24 dev vmbr0 label vmbr0:vip0
    }

    track_script {
        chk_k8s_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_K8S"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_K8S"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_K8S"
}

# Kubric API — MASTER on Node 2
vrrp_instance VI_KUBRIC {
    state MASTER
    interface vmbr0
    virtual_router_id 52
    priority 150
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass KubrVRRP#52!
    }

    unicast_src_ip 10.0.100.22
    unicast_peer {
        10.0.100.21
        10.0.100.23
    }

    virtual_ipaddress {
        10.0.100.101/24 dev vmbr0 label vmbr0:vip1
    }

    track_script {
        chk_kubric_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_KUBRIC"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_KUBRIC"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_KUBRIC"
}
```

### 4.3 Node 3 — pve-kubric-03

File: `/etc/keepalived/keepalived.conf`

```ini
global_defs {
    router_id pve-kubric-03
    enable_script_security
    script_user root
    vrrp_garp_master_refresh 60
    vrrp_garp_master_repeat 3
}

vrrp_script chk_k8s_api {
    script "/etc/keepalived/check_k8s_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

vrrp_script chk_kubric_api {
    script "/etc/keepalived/check_kubric_api.sh"
    interval 3
    weight -20
    fall 3
    rise 2
}

# K8s API — lowest priority BACKUP
vrrp_instance VI_K8S {
    state BACKUP
    interface vmbr0
    virtual_router_id 51
    priority 130
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass K8sVRRP#51!
    }

    unicast_src_ip 10.0.100.23
    unicast_peer {
        10.0.100.21
        10.0.100.22
    }

    virtual_ipaddress {
        10.0.100.100/24 dev vmbr0 label vmbr0:vip0
    }

    track_script {
        chk_k8s_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_K8S"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_K8S"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_K8S"
}

# Kubric API — middle priority BACKUP
vrrp_instance VI_KUBRIC {
    state BACKUP
    interface vmbr0
    virtual_router_id 52
    priority 140
    advert_int 1
    nopreempt

    authentication {
        auth_type PASS
        auth_pass KubrVRRP#52!
    }

    unicast_src_ip 10.0.100.23
    unicast_peer {
        10.0.100.21
        10.0.100.22
    }

    virtual_ipaddress {
        10.0.100.101/24 dev vmbr0 label vmbr0:vip1
    }

    track_script {
        chk_kubric_api
    }

    notify_master "/etc/keepalived/notify.sh MASTER VI_KUBRIC"
    notify_backup "/etc/keepalived/notify.sh BACKUP VI_KUBRIC"
    notify_fault  "/etc/keepalived/notify.sh FAULT VI_KUBRIC"
}
```

---

## 5. Notification Script

File: `/etc/keepalived/notify.sh` (all nodes)

```bash
#!/bin/bash
# Keepalived state change notification
STATE=$1
INSTANCE=$2
HOSTNAME=$(hostname)
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Log locally
echo "${TIMESTAMP} ${HOSTNAME} ${INSTANCE} → ${STATE}" >> /var/log/keepalived-transitions.log

# Send to NATS for Kubric event processing
nats pub "kubric.infra.vrrp.transition" \
  "{\"timestamp\":\"${TIMESTAMP}\",\"node\":\"${HOSTNAME}\",\"instance\":\"${INSTANCE}\",\"state\":\"${STATE}\"}" \
  2>/dev/null || true

# Optional: Curl to Kubric webhook
curl -s -X POST http://10.0.100.101:8080/api/v1/webhooks/infra \
  -H "Content-Type: application/json" \
  -d "{\"event\":\"vrrp_transition\",\"node\":\"${HOSTNAME}\",\"instance\":\"${INSTANCE}\",\"state\":\"${STATE}\",\"timestamp\":\"${TIMESTAMP}\"}" \
  2>/dev/null || true
```

```bash
chmod +x /etc/keepalived/notify.sh
```

---

## 6. Start & Enable Keepalived

```bash
# On ALL nodes
systemctl daemon-reload
systemctl enable keepalived
systemctl start keepalived

# Check status
systemctl status keepalived
journalctl -u keepalived --since "5 minutes ago" --no-pager
```

---

## 7. Failover Testing Procedure

### 7.1 Verify Initial State

```bash
# Node 1 should hold VIP 10.0.100.100
ip addr show vmbr0 | grep "10.0.100.100"
# Should show: inet 10.0.100.100/24 scope global secondary vmbr0:vip0

# Node 2 should hold VIP 10.0.100.101
ssh root@10.0.100.22 "ip addr show vmbr0 | grep 10.0.100.101"

# From external host, both VIPs should be reachable
ping -c 3 10.0.100.100
ping -c 3 10.0.100.101

# K8s API via VIP
curl -sk https://10.0.100.100:6443/healthz
# Expected: ok

# Kubric API via VIP
curl -s http://10.0.100.101:8080/healthz
# Expected: {"status":"ok"}
```

### 7.2 Simulate Node 1 Failure

```bash
# On Node 1 — stop keepalived
systemctl stop keepalived

# Immediately check — VIP should migrate to Node 2
ssh root@10.0.100.22 "ip addr show vmbr0 | grep 10.0.100.100"
# Should now show 10.0.100.100 on Node 2

# Verify services still reachable
curl -sk https://10.0.100.100:6443/healthz   # via Node 2 now
curl -s http://10.0.100.101:8080/healthz      # still on Node 2

# Check notification log on Node 2
tail -5 /var/log/keepalived-transitions.log
# Expected: pve-kubric-02 VI_K8S → MASTER
```

### 7.3 Restore Node 1

```bash
# On Node 1 — restart keepalived
systemctl start keepalived

# With nopreempt, VIP stays on Node 2 (no flap)
# To force VIP back to Node 1, briefly restart keepalived on Node 2
```

### 7.4 Simulate Service Failure (not node failure)

```bash
# On Node 1 — stop kube-apiserver (or block health check port)
iptables -A INPUT -p tcp --dport 6443 -j DROP

# Health check script will fail → weight drops by 20
# Priority goes from 150 → 130 (same as Node 3)
# Node 2 (priority 140) takes over VIP

# Wait ~10 seconds for 3 failed checks
ssh root@10.0.100.22 "ip addr show vmbr0 | grep 10.0.100.100"

# Restore
iptables -D INPUT -p tcp --dport 6443 -j DROP
```

---

## 8. DNS Configuration

Point DNS records at both VIPs:

```
; kubric.local zone file additions
k8s-api.kubric.local.     IN  A  10.0.100.100
api.kubric.local.          IN  A  10.0.100.101
kubric.kubric.local.       IN  A  10.0.100.101

; for kubectl config
kubernetes.kubric.local.   IN  A  10.0.100.100
```

```bash
# kubeconfig should reference VIP
kubectl config set-cluster kubric \
  --server=https://10.0.100.100:6443 \
  --certificate-authority=/etc/kubernetes/pki/ca.crt
```

---

## 9. Verification Summary

```bash
# On each node
ip addr show vmbr0 | grep -E "vip|100\.(100|101)"
systemctl is-active keepalived
journalctl -u keepalived --since "1 hour ago" | grep -i "state"
cat /var/log/keepalived-transitions.log | tail -10
```
