# K-NOC-CM-SALT-001 -- Salt Reactor Setup

**Role:** Step-by-step setup of the Salt Master reactor system for event-driven infrastructure automation within Kubric.

---

## 1. Prerequisites

```bash
# Salt Master + API installation on Ubuntu 22.04
curl -fsSL https://bootstrap.saltproject.io | sudo sh -s -- -M -P stable

# Install salt-api for REST interface
sudo apt-get install -y salt-api python3-pyopenssl

# Enable and start services
sudo systemctl enable --now salt-master salt-api
```

---

## 2. Salt Master Configuration

```yaml
# /etc/salt/master.d/kubric.conf
# File roots
file_roots:
  base:
    - /srv/salt
    - /srv/salt/kubric

# Pillar roots
pillar_roots:
  base:
    - /srv/pillar

# Auto-accept minions matching Kubric naming
autosign_file: /etc/salt/autosign.conf

# Event return to local job cache
event_return: rawfile_json
event_return_file: /var/log/salt/events.json

# Worker threads
worker_threads: 10

# Timeout settings
timeout: 60
gather_job_timeout: 30

# Log level
log_level: info
log_file: /var/log/salt/master

# Enable file_recv for minion file uploads (forensics)
file_recv: True
file_recv_max_size: 100  # MB
```

```yaml
# /etc/salt/master.d/reactor.conf
reactor:
  # Auto-register new Kubric minions
  - 'salt/auth':
    - /srv/reactor/auto_accept.sls

  # Baseline enforcement on minion start
  - 'salt/minion/*/start':
    - /srv/reactor/minion_start.sls

  # Kubric-specific event handlers
  - 'kubric/drift/detected':
    - /srv/reactor/drift_remediate.sls

  - 'kubric/incident/isolate':
    - /srv/reactor/network_isolate.sls

  - 'kubric/patch/emergency':
    - /srv/reactor/emergency_patch.sls

  - 'kubric/user/disable':
    - /srv/reactor/user_disable.sls

  - 'kubric/compliance/enforce':
    - /srv/reactor/compliance_enforce.sls
```

```yaml
# /etc/salt/master.d/api.conf
rest_tornado:
  port: 8000
  ssl_crt: /etc/salt/tls/salt-api.pem
  ssl_key: /etc/salt/tls/salt-api.key
  disable_ssl: false
  cors_origin: '*'

# PAM external auth for API access
external_auth:
  pam:
    kubric-noc%:
      - .*
      - '@wheel'
      - '@runner'
      - '@events'
```

---

## 3. Auto-Sign Configuration

```conf
# /etc/salt/autosign.conf
# Accept minions matching these patterns
kubric-*
k-node-*
k-edge-*
```

---

## 4. Directory Structure Setup

```bash
#!/usr/bin/env bash
# scripts/salt-reactor-setup.sh
set -euo pipefail

echo "=== Creating Salt directory structure ==="

# Salt state trees
sudo mkdir -p /srv/salt/kubric/{baseline,isolation,patching,compliance}
sudo mkdir -p /srv/reactor
sudo mkdir -p /srv/pillar
sudo mkdir -p /etc/salt/tls

echo "=== Generating TLS certificates for salt-api ==="
sudo openssl req -x509 -nodes -days 365 \
  -newkey rsa:4096 \
  -keyout /etc/salt/tls/salt-api.key \
  -out /etc/salt/tls/salt-api.pem \
  -subj "/CN=salt-api.kubric.internal/O=Kubric"

echo "=== Creating kubric-noc system user for API auth ==="
sudo useradd -r -s /usr/sbin/nologin kubric-noc || true
echo "kubric-noc:$(openssl rand -base64 32)" | sudo chpasswd

echo "=== Deploying reactor SLS files ==="
# Copy reactor files
for reactor in auto_accept minion_start drift_remediate network_isolate emergency_patch user_disable compliance_enforce; do
  if [ -f "config/salt/reactor/${reactor}.sls" ]; then
    sudo cp "config/salt/reactor/${reactor}.sls" "/srv/reactor/${reactor}.sls"
  fi
done

echo "=== Deploying baseline states ==="
if [ -d "config/salt/states/kubric" ]; then
  sudo cp -r config/salt/states/kubric/* /srv/salt/kubric/
fi

echo "=== Deploying pillar data ==="
if [ -d "config/salt/pillar" ]; then
  sudo cp -r config/salt/pillar/* /srv/pillar/
fi

echo "=== Restarting Salt services ==="
sudo systemctl restart salt-master
sudo systemctl restart salt-api

echo "=== Verifying Salt Master ==="
sudo salt-run manage.status

echo "=== Reactor setup complete ==="
```

---

## 5. Baseline Salt States

```yaml
# /srv/salt/kubric/baseline/init.sls
# Kubric baseline security configuration

# System hardening
kubric_sshd_config:
  file.managed:
    - name: /etc/ssh/sshd_config
    - source: salt://kubric/baseline/files/sshd_config
    - mode: '0600'
    - user: root
    - group: root

kubric_sshd_restart:
  service.running:
    - name: sshd
    - enable: True
    - watch:
      - file: kubric_sshd_config

# Firewall
kubric_firewall:
  service.running:
    - name: firewalld
    - enable: True

# NTP
kubric_chrony:
  service.running:
    - name: chronyd
    - enable: True

# Audit logging
kubric_auditd:
  pkg.installed:
    - name: audit

kubric_auditd_running:
  service.running:
    - name: auditd
    - enable: True
    - require:
      - pkg: kubric_auditd

# Kubric agent
kubric_agent:
  service.running:
    - name: kubric-agent
    - enable: True
    - watch:
      - file: kubric_agent_config

kubric_agent_config:
  file.managed:
    - name: /etc/kubric/agent.toml
    - source: salt://kubric/baseline/files/agent.toml
    - template: jinja
    - mode: '0640'
    - user: root
    - group: kubric

# Required packages
kubric_required_packages:
  pkg.installed:
    - pkgs:
      - curl
      - jq
      - rsyslog
      - ca-certificates
      - lsof
      - strace
```

---

## 6. Network Isolation State

```yaml
# /srv/salt/kubric/isolation/network_lockdown.sls
# Emergency network isolation — blocks all traffic except Salt master

{% set salt_master_ip = salt['pillar.get']('salt_master_ip', '10.0.0.1') %}
{% set incident_id = salt['pillar.get']('incident_id', 'unknown') %}

# Flush existing rules
kubric_isolation_flush:
  iptables.flush:
    - table: filter

# Allow loopback
kubric_isolation_loopback:
  iptables.append:
    - table: filter
    - chain: INPUT
    - jump: ACCEPT
    - in-interface: lo

# Allow established connections (for Salt)
kubric_isolation_established:
  iptables.append:
    - table: filter
    - chain: INPUT
    - jump: ACCEPT
    - match: state
    - connstate: ESTABLISHED,RELATED

# Allow Salt master communication
kubric_isolation_salt_in:
  iptables.append:
    - table: filter
    - chain: INPUT
    - jump: ACCEPT
    - source: {{ salt_master_ip }}
    - proto: tcp

kubric_isolation_salt_out:
  iptables.append:
    - table: filter
    - chain: OUTPUT
    - jump: ACCEPT
    - destination: {{ salt_master_ip }}
    - proto: tcp

# Drop everything else
kubric_isolation_drop_input:
  iptables.append:
    - table: filter
    - chain: INPUT
    - jump: DROP

kubric_isolation_drop_output:
  iptables.append:
    - table: filter
    - chain: OUTPUT
    - jump: DROP

# Log isolation event
kubric_isolation_log:
  cmd.run:
    - name: |
        echo '{"event":"network_isolated","incident_id":"{{ incident_id }}","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' >> /var/log/kubric/isolation.log
    - creates: /var/log/kubric/isolation.log
```

---

## 7. Verification Commands

```bash
# Check reactor is loaded
sudo salt-run reactor.list

# Test-fire a Kubric event
sudo salt-call event.fire_master '{"node_id": "test-01", "drifts": []}' 'kubric/drift/detected'

# Verify minion connectivity
sudo salt '*' test.ping

# Show Salt event bus (real-time)
sudo salt-run state.event pretty=True

# List accepted keys
sudo salt-key -L
```
