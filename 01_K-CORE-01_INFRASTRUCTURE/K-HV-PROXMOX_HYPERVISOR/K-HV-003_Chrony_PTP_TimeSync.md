# K-HV-003 — Chrony NTP + PTP Precision Time Sync

> **Why:** OCSF event correlation, Blake3 hash chain ordering, and distributed ClickHouse queries require sub-millisecond clock agreement across all nodes.  
> **NTP:** Chrony for millisecond-accuracy sync to external sources  
> **PTP:** linuxptp for sub-microsecond sync between cluster nodes  
> **Scope:** All 3 Proxmox hosts + VMs/containers

---

## 1. Why Precise Time Matters for Kubric

| Component | Time Sensitivity | Consequence of Drift |
|-----------|-----------------|---------------------|
| OCSF Event Correlation | <1ms | Events appear out of order; SOC triage fails |
| Blake3 Chain Ordering | <100µs | Immutable audit chain gaps; compliance violation |
| ClickHouse Distributed | <10ms | INSERT deduplication failures; query result skew |
| Ceph Cluster | <50ms | OSD heartbeat false positives; unnecessary recovery |
| TLS Certificate Validation | <1s | Certificate validation failures |
| Kerberos Authentication | <5min | Authentication failures (if AD-integrated) |

---

## 2. Chrony NTP Configuration

### 2.1 Install Chrony

```bash
# On ALL nodes (Proxmox includes chrony by default)
apt-get install -y chrony
systemctl enable chrony
```

### 2.2 /etc/chrony/chrony.conf — Node 1 (Stratum 2 Primary)

Node 1 syncs to external NTP pools and serves as local stratum for the cluster:

```ini
# ──────────────────────────────────────────────
# External NTP sources (Stratum 1/2 servers)
# ──────────────────────────────────────────────
pool 0.us.pool.ntp.org iburst maxsources 4
pool 1.us.pool.ntp.org iburst maxsources 4
pool time.cloudflare.com iburst maxsources 2
server time.google.com iburst prefer

# NIST time servers (high accuracy)
server time-a-g.nist.gov iburst
server time-b-g.nist.gov iburst

# ──────────────────────────────────────────────
# Local stratum — serve time to cluster nodes
# ──────────────────────────────────────────────
# Allow cluster nodes to sync from this server
allow 10.0.100.0/24
allow 10.0.200.0/24
allow 10.0.50.0/24

# Serve time even if not externally synced (fallback)
local stratum 3 orphan

# ──────────────────────────────────────────────
# Drift and logging
# ──────────────────────────────────────────────
driftfile /var/lib/chrony/drift
logdir /var/log/chrony
log tracking measurements statistics

# ──────────────────────────────────────────────
# Security
# ──────────────────────────────────────────────
# Restrict command access
cmdallow 127.0.0.1
cmdallow 10.0.100.0/24

# Key file for authenticated NTP (optional)
keyfile /etc/chrony/chrony.keys

# ──────────────────────────────────────────────
# Step threshold — allow large correction on boot only
# ──────────────────────────────────────────────
makestep 1.0 3
# Step the clock if offset > 1s, but only in first 3 updates

# RTC sync
rtcsync
hwclockfile /etc/adjtime

# Leap second handling
leapsectz right/UTC

# ──────────────────────────────────────────────
# NTS (Network Time Security) — if supported
# ──────────────────────────────────────────────
# server time.cloudflare.com iburst nts
# Uncomment above and remove non-NTS cloudflare entry for NTS
```

### 2.3 /etc/chrony/chrony.conf — Nodes 2 and 3

Nodes 2 and 3 sync primarily from Node 1, with external pools as fallback:

```ini
# Primary — sync from Node 1 (local stratum)
server 10.0.200.21 iburst prefer minpoll 4 maxpoll 6
# minpoll 4 = every 16s, maxpoll 6 = every 64s (tighter than default)

# Fallback — external NTP pools
pool 0.us.pool.ntp.org iburst maxsources 2
server time.google.com iburst

# Allow VMs and containers to sync from this node
allow 10.0.50.0/24

# Serve time locally if Node 1 is unreachable
local stratum 4 orphan

# Drift and logging
driftfile /var/lib/chrony/drift
logdir /var/log/chrony
log tracking measurements statistics

# Step on boot only
makestep 1.0 3
rtcsync
hwclockfile /etc/adjtime
leapsectz right/UTC
```

### 2.4 Apply Chrony

```bash
# On ALL nodes
systemctl restart chrony

# Verify synchronization
chronyc tracking
# Expected output:
# Reference ID    : xxxxx (time.google.com)
# Stratum         : 2
# System time     : 0.000000123 seconds fast of NTP time
# Last offset     : +0.000000045 seconds

chronyc sources -v
# Shows all configured sources and their status

chronyc sourcestats -v
# Shows frequency and offset statistics
```

---

## 3. PTP Configuration (Sub-Microsecond)

### 3.1 Install linuxptp

```bash
# On ALL nodes
apt-get install -y linuxptp ethtool

# Verify hardware timestamping support on Intel X710
ethtool -T p3p1 | grep -A2 "Capabilities"
# Expected:
#   hardware-transmit
#   hardware-receive
#   hardware-raw-clock
```

### 3.2 PTP Grand Master (Node 1)

File: `/etc/linuxptp/ptp4l.conf` on pve-kubric-01

```ini
[global]
# Clock servo
clockClass              248
clockAccuracy           0xFE
offsetScaledLogVariance 0xFFFF
domainNumber            0
priority1               128
priority2               128

# All ports
network_transport       L2
delay_mechanism         E2E
time_stamping           hardware

# Logging
logging_level           6
verbose                 0
summary_interval        0

# Servo PI controller gains
step_threshold          1.0
first_step_threshold    0.00002
servo                   pi
pi_proportional_const   0.0
pi_integral_const       0.0
pi_proportional_scale   0.7
pi_proportional_exponent -0.3
pi_proportional_norm_max 0.7
pi_integral_scale        0.3
pi_integral_exponent     0.4
pi_integral_norm_max     0.3

# Announce interval
logAnnounceInterval     1
logSyncInterval         -3
logMinDelayReqInterval  -3

# Transport
dscp_event              46
dscp_general            46

[p3p1]
# PTP runs on the 10G SFP+ interface
```

### 3.3 PTP Slave (Nodes 2 and 3)

File: `/etc/linuxptp/ptp4l.conf` on pve-kubric-02/03

```ini
[global]
clockClass              255
clockAccuracy           0xFE
offsetScaledLogVariance 0xFFFF
domainNumber            0
priority1               200
priority2               200

network_transport       L2
delay_mechanism         E2E
time_stamping           hardware

logging_level           6
verbose                 0
summary_interval        0

step_threshold          1.0
first_step_threshold    0.00002
servo                   pi

logAnnounceInterval     1
logSyncInterval         -3
logMinDelayReqInterval  -3

dscp_event              46
dscp_general            46

[p3p1]
```

### 3.4 PHC-to-System Clock Sync (phc2sys)

```bash
# On ALL nodes — sync PTP hardware clock to system clock
# Create systemd service for phc2sys

cat > /etc/systemd/system/phc2sys.service <<'EOF'
[Unit]
Description=PTP Hardware Clock to System Clock Sync
After=ptp4l.service
Requires=ptp4l.service

[Service]
Type=simple
ExecStart=/usr/sbin/phc2sys -a -rr -z /var/run/ptp4l
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

### 3.5 Start PTP Services

```bash
# Create systemd service for ptp4l
cat > /etc/systemd/system/ptp4l.service <<'EOF'
[Unit]
Description=PTP4L IEEE 1588 PTP Clock
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/sbin/ptp4l -f /etc/linuxptp/ptp4l.conf -i p3p1
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
systemctl daemon-reload
systemctl enable ptp4l phc2sys
systemctl start ptp4l phc2sys
```

---

## 4. Verification Commands

### 4.1 Chrony Verification

```bash
# Check tracking (main status)
chronyc tracking
# Key metrics:
#   System time: should be <0.001 seconds offset
#   RMS offset:  should be <1ms

# Check sources
chronyc sources -v
# * = current source, + = valid alternative, ? = unreachable

# Check source statistics
chronyc sourcestats -v
# Freq ppm and Offset columns should be small

# Activity
chronyc activity
# 200 OK
# X sources online
```

### 4.2 PTP Verification

```bash
# Check PTP clock state
pmc -u -b 0 'GET CURRENT_DATA_SET'
# offsetFromMaster should be <1µs for hardware timestamping

# Check PTP port state
pmc -u -b 0 'GET PORT_DATA_SET'
# portState: SLAVE (on nodes 2,3)
# portState: MASTER or GRAND_MASTER (on node 1)

# Check phc2sys output
journalctl -u phc2sys --since "5 minutes ago" | tail -10
# phc2sys[xxx]: p3p1 offset   -23 s2 freq   +1234 delay  450
# offset should be in nanoseconds (target: <100ns)
```

### 4.3 Cross-Node Time Comparison

```bash
# Compare system clocks across all nodes
for node in 10.0.100.21 10.0.100.22 10.0.100.23; do
  echo -n "$node: "
  ssh root@$node "date -u +%Y-%m-%dT%H:%M:%S.%N"
done

# Nanosecond timestamps should be within ~100ns of each other with PTP
# Or within ~1ms with Chrony-only

# Automated check script:
cat > /usr/local/bin/check-time-sync.sh <<'SCRIPT'
#!/bin/bash
REF=$(date +%s%N)
for node in 10.0.100.22 10.0.100.23; do
  REMOTE=$(ssh -o ConnectTimeout=2 root@$node "date +%s%N" 2>/dev/null)
  if [ -n "$REMOTE" ]; then
    DIFF=$((REMOTE - REF))
    echo "Offset to $node: ${DIFF}ns"
    if [ ${DIFF#-} -gt 1000000 ]; then
      echo "WARNING: Offset >1ms!"
    fi
  fi
done
SCRIPT
chmod +x /usr/local/bin/check-time-sync.sh
```

---

## 5. Prometheus Metrics

### 5.1 Chrony Exporter

```bash
# Install chrony_exporter
wget https://github.com/SuperQ/chrony_exporter/releases/download/v0.9.2/chrony_exporter-0.9.2.linux-amd64.tar.gz
tar xzf chrony_exporter-0.9.2.linux-amd64.tar.gz
cp chrony_exporter-0.9.2.linux-amd64/chrony_exporter /usr/local/bin/

cat > /etc/systemd/system/chrony-exporter.service <<'EOF'
[Unit]
Description=Chrony Exporter for Prometheus
After=chrony.service

[Service]
Type=simple
ExecStart=/usr/local/bin/chrony_exporter --chrony.address=unix:///run/chrony/chronyd.sock
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now chrony-exporter
# Metrics at: http://localhost:9123/metrics
```

### 5.2 Prometheus Scrape Config

```yaml
# Add to prometheus.yml:
  - job_name: 'chrony'
    static_configs:
      - targets:
        - '10.0.100.21:9123'
        - '10.0.100.22:9123'
        - '10.0.100.23:9123'
```

### 5.3 Key Alerts

```yaml
# Prometheus alert rules
groups:
  - name: time_sync
    rules:
      - alert: ClockDriftHigh
        expr: chrony_tracking_system_time_offset_seconds > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Clock drift >10ms on {{ $labels.instance }}"

      - alert: ClockDriftCritical
        expr: chrony_tracking_system_time_offset_seconds > 0.1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Clock drift >100ms on {{ $labels.instance }} — OCSF correlation at risk"

      - alert: NTPSourcesDown
        expr: chrony_activity_sources_online == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "No NTP sources available on {{ $labels.instance }}"
```

---

## 6. VM / Container Time Sync

### 6.1 KVM Guests

```bash
# VMs automatically sync via kvm-clock (paravirtual)
# Verify in guest:
cat /sys/devices/system/clocksource/clocksource0/current_clocksource
# Expected: kvm-clock

# Guest should also run chrony pointing to host:
# In guest /etc/chrony/chrony.conf:
# server 10.0.50.21 iburst prefer
```

### 6.2 LXC Containers

```bash
# LXC containers share the host kernel clock
# No separate time sync needed
# Verify:
pct exec <CTID> -- date -u
```

---

## 7. Troubleshooting

```bash
# Chrony not syncing
systemctl status chrony
chronyc sources -v
# Check: firewall blocking UDP 123

# PTP not locking
journalctl -u ptp4l --since "5 minutes ago"
# Common issues:
#   - "no timestamp" → hardware timestamping not supported
#   - "master offset" large → cable/switch issue
ethtool -T p3p1   # verify timestamping capabilities

# Large time step on reboot
# Expected: chrony will step clock once if offset > 1s
# makestep 1.0 3 allows this for first 3 updates only
journalctl -u chrony | grep "System clock was stepped"
```
