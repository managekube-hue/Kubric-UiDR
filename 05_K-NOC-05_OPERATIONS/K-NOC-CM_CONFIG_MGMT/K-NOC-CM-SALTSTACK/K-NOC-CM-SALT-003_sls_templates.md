# K-NOC-CM-SALT-003 -- Salt SLS Templates

**Role:** Reusable Salt state templates for common configuration management patterns in the Kubric platform.

---

## 1. Template Index

| Template | Purpose | Target |
|----------|---------|--------|
| `baseline/init.sls` | Security baseline | All nodes |
| `monitoring/init.sls` | Monitoring agent setup | All nodes |
| `patching/emergency.sls` | Emergency CVE patching | Targeted nodes |
| `docker/init.sls` | Docker host configuration | Container hosts |
| `kubric_agent/init.sls` | Kubric agent lifecycle | All nodes |
| `hardening/cis.sls` | CIS benchmark hardening | All nodes |
| `backup/client.sls` | Bareos FileDaemon | All nodes |

---

## 2. Monitoring Agent Template

```yaml
# /srv/salt/kubric/monitoring/init.sls
# Deploy and configure monitoring agents

include:
  - kubric.monitoring.node_exporter
  - kubric.monitoring.promtail

# Node exporter for system metrics
kubric_node_exporter_pkg:
  pkg.installed:
    - name: prometheus-node-exporter

kubric_node_exporter_svc:
  service.running:
    - name: prometheus-node-exporter
    - enable: True
    - require:
      - pkg: kubric_node_exporter_pkg
```

```yaml
# /srv/salt/kubric/monitoring/node_exporter.sls
kubric_node_exporter_config:
  file.managed:
    - name: /etc/default/prometheus-node-exporter
    - contents: |
        ARGS="--collector.systemd --collector.processes --collector.tcpstat \
              --web.listen-address=:9100 \
              --web.telemetry-path=/metrics"
    - mode: '0644'

kubric_node_exporter:
  service.running:
    - name: prometheus-node-exporter
    - enable: True
    - watch:
      - file: kubric_node_exporter_config
```

```yaml
# /srv/salt/kubric/monitoring/promtail.sls
{% set promtail_version = salt['pillar.get']('promtail:version', '2.9.4') %}
{% set loki_url = salt['pillar.get']('loki:url', 'http://loki.internal:3100') %}

kubric_promtail_binary:
  file.managed:
    - name: /usr/local/bin/promtail
    - source: salt://kubric/monitoring/files/promtail-{{ promtail_version }}
    - mode: '0755'
    - user: root
    - group: root

kubric_promtail_config:
  file.managed:
    - name: /etc/promtail/config.yml
    - makedirs: True
    - template: jinja
    - contents: |
        server:
          http_listen_port: 9080
          grpc_listen_port: 0
        positions:
          filename: /var/lib/promtail/positions.yaml
        clients:
          - url: {{ loki_url }}/loki/api/v1/push
            tenant_id: {{ salt['pillar.get']('tenant_id', 'default') }}
        scrape_configs:
          - job_name: syslog
            static_configs:
              - targets: [localhost]
                labels:
                  job: syslog
                  host: {{ grains['id'] }}
                  __path__: /var/log/syslog
          - job_name: auth
            static_configs:
              - targets: [localhost]
                labels:
                  job: auth
                  host: {{ grains['id'] }}
                  __path__: /var/log/auth.log
          - job_name: kubric
            static_configs:
              - targets: [localhost]
                labels:
                  job: kubric-agent
                  host: {{ grains['id'] }}
                  __path__: /var/log/kubric/*.log

kubric_promtail_service:
  file.managed:
    - name: /etc/systemd/system/promtail.service
    - contents: |
        [Unit]
        Description=Promtail Log Agent
        After=network.target

        [Service]
        Type=simple
        ExecStart=/usr/local/bin/promtail -config.file=/etc/promtail/config.yml
        Restart=always
        RestartSec=5

        [Install]
        WantedBy=multi-user.target

kubric_promtail_running:
  service.running:
    - name: promtail
    - enable: True
    - watch:
      - file: kubric_promtail_config
      - file: kubric_promtail_binary
    - require:
      - file: kubric_promtail_service
```

---

## 3. Emergency Patching Template

```yaml
# /srv/salt/kubric/patching/emergency.sls
{% set packages = salt['pillar.get']('packages', []) %}
{% set cve_id = salt['pillar.get']('cve_id', 'unknown') %}
{% set reboot_required = salt['pillar.get']('reboot_after', False) %}

# Log the emergency patch event
kubric_patch_log:
  cmd.run:
    - name: |
        echo '{"event":"emergency_patch_start","cve":"{{ cve_id }}","packages":{{ packages | tojson }},"timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' >> /var/log/kubric/patching.log

# Update package cache
kubric_patch_refresh:
  pkg.uptodate:
    - refresh: True
    {% if packages %}
    - name: kubric_targeted_patch
    {% endif %}

# Apply targeted patches
{% for pkg in packages %}
kubric_patch_{{ pkg.get('name', pkg) if pkg is mapping else pkg }}:
  pkg.installed:
    - name: {{ pkg.get('name', pkg) if pkg is mapping else pkg }}
    {% if pkg is mapping and pkg.get('version') %}
    - version: {{ pkg['version'] }}
    {% endif %}
    - refresh: True
{% endfor %}

# Restart affected services
kubric_patch_services_restart:
  cmd.run:
    - name: |
        # Restart services that use patched libraries
        needrestart -r a -l 2>/dev/null || true
    - onchanges_any:
      {% for pkg in packages %}
      - pkg: kubric_patch_{{ pkg.get('name', pkg) if pkg is mapping else pkg }}
      {% endfor %}

{% if reboot_required %}
kubric_patch_reboot:
  system.reboot:
    - message: "Emergency patch reboot for {{ cve_id }}"
    - at_time: +1
    - onchanges_any:
      {% for pkg in packages %}
      - pkg: kubric_patch_{{ pkg.get('name', pkg) if pkg is mapping else pkg }}
      {% endfor %}
{% endif %}

# Log completion
kubric_patch_complete:
  cmd.run:
    - name: |
        echo '{"event":"emergency_patch_complete","cve":"{{ cve_id }}","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' >> /var/log/kubric/patching.log
```

---

## 4. Docker Host Template

```yaml
# /srv/salt/kubric/docker/init.sls
{% set docker_version = salt['pillar.get']('docker:version', 'latest') %}

# Docker prerequisites
kubric_docker_prereqs:
  pkg.installed:
    - pkgs:
      - apt-transport-https
      - ca-certificates
      - curl
      - gnupg
      - lsb-release

# Docker GPG key
kubric_docker_gpg:
  cmd.run:
    - name: |
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    - creates: /usr/share/keyrings/docker-archive-keyring.gpg

# Docker repo
kubric_docker_repo:
  pkgrepo.managed:
    - humanname: Docker CE
    - name: "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu {{ salt['grains.get']('oscodename') }} stable"
    - file: /etc/apt/sources.list.d/docker.list
    - require:
      - cmd: kubric_docker_gpg

# Docker CE
kubric_docker_install:
  pkg.installed:
    - pkgs:
      - docker-ce
      - docker-ce-cli
      - containerd.io
      - docker-compose-plugin
    - require:
      - pkgrepo: kubric_docker_repo

# Docker daemon config
kubric_docker_daemon_config:
  file.managed:
    - name: /etc/docker/daemon.json
    - contents: |
        {
          "log-driver": "json-file",
          "log-opts": {
            "max-size": "50m",
            "max-file": "5"
          },
          "storage-driver": "overlay2",
          "live-restore": true,
          "default-ulimits": {
            "nofile": {
              "Name": "nofile",
              "Hard": 65536,
              "Soft": 65536
            }
          },
          "metrics-addr": "0.0.0.0:9323",
          "experimental": false
        }
    - mode: '0644'

kubric_docker_service:
  service.running:
    - name: docker
    - enable: True
    - watch:
      - file: kubric_docker_daemon_config
    - require:
      - pkg: kubric_docker_install
```

---

## 5. CIS Benchmark Hardening Template

```yaml
# /srv/salt/kubric/hardening/cis.sls
# CIS Benchmark Level 1 — selected controls

# 1.1.1 - Disable unused filesystems
{% for fs in ['cramfs', 'freevxfs', 'jffs2', 'hfs', 'hfsplus', 'squashfs', 'udf'] %}
kubric_cis_disable_{{ fs }}:
  file.managed:
    - name: /etc/modprobe.d/kubric-{{ fs }}.conf
    - contents: |
        install {{ fs }} /bin/true
        blacklist {{ fs }}
{% endfor %}

# 1.4.1 - Ensure permissions on bootloader config
kubric_cis_grub_perms:
  file.managed:
    - name: /boot/grub/grub.cfg
    - mode: '0400'
    - user: root
    - group: root
    - replace: False

# 3.1 - Network parameters
kubric_cis_sysctl:
  sysctl.present:
    - names:
      - net.ipv4.ip_forward: 0
      - net.ipv4.conf.all.send_redirects: 0
      - net.ipv4.conf.default.send_redirects: 0
      - net.ipv4.conf.all.accept_source_route: 0
      - net.ipv4.conf.default.accept_source_route: 0
      - net.ipv4.conf.all.accept_redirects: 0
      - net.ipv4.conf.default.accept_redirects: 0
      - net.ipv4.conf.all.log_martians: 1
      - net.ipv4.icmp_echo_ignore_broadcasts: 1
      - net.ipv4.tcp_syncookies: 1
      - net.ipv6.conf.all.accept_redirects: 0
      - net.ipv6.conf.default.accept_redirects: 0

# 5.2 - SSH hardening
kubric_cis_ssh:
  file.managed:
    - name: /etc/ssh/sshd_config.d/kubric-cis.conf
    - contents: |
        Protocol 2
        LogLevel VERBOSE
        MaxAuthTries 4
        PermitRootLogin no
        PermitEmptyPasswords no
        PasswordAuthentication no
        X11Forwarding no
        MaxStartups 10:30:60
        ClientAliveInterval 300
        ClientAliveCountMax 3
        LoginGraceTime 60
    - mode: '0600'

kubric_cis_ssh_restart:
  service.running:
    - name: sshd
    - watch:
      - file: kubric_cis_ssh

# 6.1 - File permissions
kubric_cis_passwd_perms:
  file.managed:
    - name: /etc/passwd
    - mode: '0644'
    - user: root
    - group: root
    - replace: False

kubric_cis_shadow_perms:
  file.managed:
    - name: /etc/shadow
    - mode: '0640'
    - user: root
    - group: shadow
    - replace: False
```

---

## 6. Bareos Backup Client Template

```yaml
# /srv/salt/kubric/backup/client.sls
{% set bareos_password = salt['pillar.get']('bareos:client_password', '') %}
{% set bareos_director = salt['pillar.get']('bareos:director', 'kubric-dir') %}

kubric_bareos_fd:
  pkg.installed:
    - name: bareos-filedaemon

kubric_bareos_fd_config:
  file.managed:
    - name: /etc/bareos/bareos-fd.d/client/myself.conf
    - contents: |
        Client {
          Name = {{ grains['id'] }}-fd
          Maximum Concurrent Jobs = 10
          FD Port = 9102
          
          TLS Enable = yes
          TLS Require = yes
          TLS CA Certificate File = /etc/bareos/tls/ca.pem
        }
    - mode: '0640'
    - user: bareos
    - group: bareos

kubric_bareos_fd_director:
  file.managed:
    - name: /etc/bareos/bareos-fd.d/director/kubric-dir.conf
    - contents: |
        Director {
          Name = {{ bareos_director }}
          Password = "{{ bareos_password }}"
          
          TLS Enable = yes
          TLS Require = yes
          TLS CA Certificate File = /etc/bareos/tls/ca.pem
        }
    - mode: '0640'
    - user: bareos
    - group: bareos

kubric_bareos_fd_service:
  service.running:
    - name: bareos-fd
    - enable: True
    - watch:
      - file: kubric_bareos_fd_config
      - file: kubric_bareos_fd_director
```

---

## 7. Pillar Data Structure

```yaml
# /srv/pillar/top.sls
base:
  '*':
    - common
    - monitoring
  'kubric-docker-*':
    - docker
  'kubric-k8s-*':
    - kubernetes
```

```yaml
# /srv/pillar/common.sls
tenant_id: "{{ grains.get('kubric_tenant_id', 'default') }}"

loki:
  url: http://loki.internal:3100

bareos:
  director: kubric-dir
  client_password: {{ salt['cmd.run']('cat /etc/kubric/bareos-password 2>/dev/null || echo changeme') }}

kubric:
  nats_url: nats://nats.internal:4222
  agent_version: "1.0.0"
```
