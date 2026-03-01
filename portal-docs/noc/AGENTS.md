# NOC - Agent Management

**Module:** K-NOC-05  
**Purpose:** Deploy, monitor, and manage Kubric agents

---

## Agents Overview

| Agent | Purpose | Language | Port |
|-------|---------|----------|------|
| CoreSec | EDR - Process/file monitoring | Rust | - |
| NetGuard | NDR - Network traffic analysis | Rust | - |
| PerfTrace | Performance metrics | Rust | - |
| Watchdog | OTA updates and lifecycle | Rust | - |
| Provisioning | Agent enrollment | Rust | - |

---

## Deployment

### Install Script Generation
```bash
# Generate install script for Linux
curl -X POST http://noc:8083/api/v1/agents/install \
  -H "Content-Type: application/json" \
  -d '{"platform": "linux", "tenant_id": "acme-corp"}'

# Returns: bash install script with systemd service
```

### Manual Deployment
```bash
# Download agent binary
curl -O https://releases.kubric.security/coresec-v1.0.0-linux-amd64

# Verify hash
sha256sum coresec-v1.0.0-linux-amd64

# Install
sudo mv coresec-v1.0.0-linux-amd64 /usr/local/bin/coresec
sudo chmod +x /usr/local/bin/coresec

# Create systemd service
sudo systemctl enable coresec
sudo systemctl start coresec
```

---

## Agent Configuration

### CoreSec (EDR)
```yaml
# /etc/kubric/coresec.yaml
tenant_id: acme-corp
nats_url: nats://nats.kubric.security:4222
nats_token: <from_vault>
detection:
  sigma_rules: /etc/kubric/sigma/
  yara_rules: /etc/kubric/yara/
hooks:
  ebpf: true  # Linux only
  etw: true   # Windows only
```

### NetGuard (NDR)
```yaml
# /etc/kubric/netguard.yaml
tenant_id: acme-corp
nats_url: nats://nats.kubric.security:4222
interface: eth0
dpi: true
ids: true
suricata_rules: /etc/kubric/suricata/
```

### PerfTrace
```yaml
# /etc/kubric/perftrace.yaml
tenant_id: acme-corp
nats_url: nats://nats.kubric.security:4222
interval: 10s
metrics:
  - cpu
  - memory
  - disk
  - network
```

---

## Monitoring

### Agent Health
```bash
# Check agent status
curl http://noc:8083/api/v1/agents

# Response
{
  "agents": [
    {
      "id": "coresec-host1",
      "status": "healthy",
      "last_heartbeat": "2026-02-28T10:30:00Z",
      "version": "v1.0.0"
    }
  ]
}
```

### Heartbeat Monitoring
- Agents send heartbeat every 30 seconds
- Alert if no heartbeat for 5 minutes
- Auto-restart if agent crashes

---

## OTA Updates

### Watchdog Agent
- Checks for updates every 24 hours
- Downloads delta patches (zstd compressed)
- Verifies blake3 hash + ed25519 signature
- Atomic binary replacement
- Rollback on failure

### Update Process
1. Watchdog checks TUF repository
2. Downloads new version metadata
3. Downloads delta patch or full binary
4. Verifies signature
5. Applies update
6. Restarts agent
7. Verifies health

---

## Troubleshooting

### Agent Not Reporting
```bash
# Check agent status
sudo systemctl status coresec

# Check logs
sudo journalctl -u coresec -f

# Test NATS connectivity
nats pub kubric.test "hello"
```

### High CPU Usage
```bash
# Check eBPF map pressure (Linux)
sudo bpftool map list

# Reduce detection rules
# Edit /etc/kubric/coresec.yaml
```

### Network Issues
```bash
# Check NetGuard interface
ip link show eth0

# Verify packet capture
sudo tcpdump -i eth0 -c 10
```

---

## Related Documentation

- [Infrastructure Management](INFRASTRUCTURE.md)
- [System Architecture](../architecture/ARCHITECTURE.md)
- [Agent Architecture](../architecture/ARCHITECTURE.md#agent-architecture-rust)
