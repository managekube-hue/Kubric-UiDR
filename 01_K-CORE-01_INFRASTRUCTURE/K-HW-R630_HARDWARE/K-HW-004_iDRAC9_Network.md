# K-HW-004 — iDRAC9 Network Configuration (All Nodes)

> **Scope:** iDRAC9 management plane for pve-kubric-01, pve-kubric-02, pve-kubric-03  
> **Management VLAN:** 100  
> **Monitoring:** SNMPv3 → Prometheus SNMP Exporter

---

## 1. iDRAC IP Address Assignments

| Node | iDRAC IP | Management IP | Hostname |
|------|----------|---------------|----------|
| Node 1 | 10.0.100.11 | 10.0.100.21 | pve-kubric-01 |
| Node 2 | 10.0.100.12 | 10.0.100.22 | pve-kubric-02 |
| Node 3 | 10.0.100.13 | 10.0.100.23 | pve-kubric-03 |

---

## 2. Dedicated Management VLAN Configuration

### 2.1 racadm — All Nodes

Run for each node, substituting `$IDRAC` and `$NODE_NUM`:

```bash
# Variables per node
# Node 1: IDRAC=10.0.100.11
# Node 2: IDRAC=10.0.100.12
# Node 3: IDRAC=10.0.100.13
CRED="-u kubric-admin -p ${IDRAC_KUBRIC_PASS}"

for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  racadm -r $IDRAC $CRED <<'CMDS'
    # Dedicated NIC mode (not shared with host)
    set iDRAC.NIC.Selection Dedicated
    set iDRAC.NIC.Enable Enabled
    set iDRAC.NIC.Speed 1000
    set iDRAC.NIC.Duplex Full
    set iDRAC.NIC.AutoNegotiation Enabled

    # VLAN 100 for management isolation
    set iDRAC.NIC.VLanEnable Enabled
    set iDRAC.NIC.VLanID 100
    set iDRAC.NIC.VLanPriority 0

    # DNS
    set iDRAC.IPv4.DNSFromDHCP Disabled
    set iDRAC.IPv4.DNS1 10.0.100.1
    set iDRAC.IPv4.DNS2 1.1.1.1
    set iDRAC.NIC.DNSDomainName kubric.local

    # Disable IPv6 (simplify management plane)
    set iDRAC.IPv6.Enable Disabled

    # Timeout & security
    set iDRAC.WebServer.Timeout 1800
    set iDRAC.WebServer.TLSProtocol TLS_1_2_Only
    set iDRAC.IPMILan.Enable Enabled
    set iDRAC.IPMILan.PrivLimit Administrator
CMDS
done
```

### 2.2 Switch Configuration (Management Switch)

```
! Cisco IOS / Dell OS10 example for management switch port
interface GigabitEthernet1/0/1
  description iDRAC-pve-kubric-01
  switchport mode access
  switchport access vlan 100
  spanning-tree portfast
  no shutdown

interface GigabitEthernet1/0/2
  description iDRAC-pve-kubric-02
  switchport mode access
  switchport access vlan 100
  spanning-tree portfast
  no shutdown

interface GigabitEthernet1/0/3
  description iDRAC-pve-kubric-03
  switchport mode access
  switchport access vlan 100
  spanning-tree portfast
  no shutdown

! Management VLAN SVI
interface Vlan100
  description Kubric-Management
  ip address 10.0.100.1 255.255.255.0
  no shutdown
```

---

## 3. SNMPv3 Configuration

### 3.1 Enable SNMPv3 on All iDRACs

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  racadm -r $IDRAC $CRED <<'CMDS'
    # Enable SNMP agent
    set iDRAC.SNMP.AgentEnable Enabled
    set iDRAC.SNMP.AgentCommunity ""

    # SNMPv3 user
    set iDRAC.SNMP.Alert.1.Enable Enabled
    set iDRAC.SNMP.Alert.1.DestAddr 10.0.100.21
    set iDRAC.SNMP.TrapFormat SNMPv3

    # SNMPv3 credentials
    set iDRAC.Users.4.Enable Enabled
    set iDRAC.Users.4.UserName snmp-monitor
    set iDRAC.Users.4.Password "${SNMPV3_PASS}"
    set iDRAC.Users.4.Privilege 0x01
    set iDRAC.Users.4.SNMPv3Enable Enabled
    set iDRAC.Users.4.SNMPv3AuthenticationType SHA
    set iDRAC.Users.4.SNMPv3PrivacyType AES
CMDS
done
```

**Environment Variable:**
```
SNMPV3_PASS=<generate with: openssl rand -base64 16>
```

### 3.2 Prometheus SNMP Exporter Configuration

File: `/etc/prometheus/snmp_exporter/dell_idrac.yml`

```yaml
modules:
  dell_idrac:
    version: 3
    auth:
      security_level: authPriv
      username: snmp-monitor
      password: "${SNMPV3_PASS}"
      auth_protocol: SHA
      priv_protocol: AES
      priv_password: "${SNMPV3_PASS}"
    walk:
      - 1.3.6.1.4.1.674.10892.5       # Dell iDRAC MIB
    lookups:
      - source_indexes: [systemStateChassis]
        lookup: systemStateChassisStatus
    metrics:
      # System Status
      - name: idrac_system_status
        oid: 1.3.6.1.4.1.674.10892.5.4.200.10.1.2
        type: gauge
        help: Overall system health (3=OK, 4=Non-critical, 5=Critical)
      # CPU Temperature
      - name: idrac_temperature_celsius
        oid: 1.3.6.1.4.1.674.10892.5.4.700.20.1.6
        type: gauge
        help: Temperature probe reading in tenths of Celsius
      # Fan Speed RPM
      - name: idrac_fan_speed_rpm
        oid: 1.3.6.1.4.1.674.10892.5.4.700.12.1.6
        type: gauge
        help: Fan speed in RPM
      # Power Consumption Watts
      - name: idrac_power_consumption_watts
        oid: 1.3.6.1.4.1.674.10892.5.4.600.30.1.6
        type: gauge
        help: Current power consumption in watts
      # Storage Status
      - name: idrac_storage_status
        oid: 1.3.6.1.4.1.674.10892.5.4.200.10.1.12
        type: gauge
        help: Storage subsystem status
```

### 3.3 Test SNMP Connectivity

```bash
# From monitoring host
apt-get install -y snmp snmp-mibs-downloader

# Test SNMPv3
snmpwalk -v3 -u snmp-monitor \
  -l authPriv \
  -a SHA -A "${SNMPV3_PASS}" \
  -x AES -X "${SNMPV3_PASS}" \
  10.0.100.11 \
  1.3.6.1.4.1.674.10892.5.4.200.10.1.2

# Expected: Dell system status OID value
```

---

## 4. SSL Certificate Deployment

### 4.1 Generate CSR from iDRAC

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  NODE_NUM=$(echo $IDRAC | awk -F. '{print $4}')
  racadm -r $IDRAC $CRED sslcsrgen \
    -g -f /tmp/idrac_node${NODE_NUM}.csr \
    -subjectcn "idrac-${NODE_NUM}.kubric.local" \
    -subjectorg "Kubric Security" \
    -subjectou "Infrastructure" \
    -subjectcity "Dallas" \
    -subjectstate "TX" \
    -subjectcountry "US"
done
```

### 4.2 Sign with Internal CA (or Let's Encrypt)

```bash
# Using internal CA (step-ca or openssl)
for NODE_NUM in 11 12 13; do
  openssl x509 -req \
    -in /tmp/idrac_node${NODE_NUM}.csr \
    -CA /etc/ssl/kubric/ca.crt \
    -CAkey /etc/ssl/kubric/ca.key \
    -CAcreateserial \
    -out /tmp/idrac_node${NODE_NUM}.crt \
    -days 365 \
    -sha256
done
```

### 4.3 Upload Signed Cert

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  NODE_NUM=$(echo $IDRAC | awk -F. '{print $4}')
  racadm -r $IDRAC $CRED sslcertupload \
    -t 1 -f /tmp/idrac_node${NODE_NUM}.crt
done
```

---

## 5. Alert Policies for Hardware Failures

### 5.1 Configure Email Alerts

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  racadm -r $IDRAC $CRED <<'CMDS'
    # SMTP server (use internal relay or direct)
    set iDRAC.EmailAlert.SMTPServerIPAddress 10.0.100.21
    set iDRAC.EmailAlert.SMTPAuthentication Disabled

    # Alert recipient
    set iDRAC.EmailAlert.Address.1 infra-alerts@kubric.local
    set iDRAC.EmailAlert.Enable.1 Enabled

    # Alert categories — enable critical alerts
    set iDRAC.SNMP.Alert.1.Enable Enabled

    # System events
    set iDRAC.Alert.1.Category System
    set iDRAC.Alert.1.Severity Critical
    set iDRAC.Alert.1.Action Email,SNMPTrap

    # Storage events (disk failures)
    set iDRAC.Alert.2.Category Storage
    set iDRAC.Alert.2.Severity Warning
    set iDRAC.Alert.2.Action Email,SNMPTrap

    # Temperature events
    set iDRAC.Alert.3.Category Temperature
    set iDRAC.Alert.3.Severity Warning
    set iDRAC.Alert.3.Action Email,SNMPTrap

    # Power events
    set iDRAC.Alert.4.Category Power
    set iDRAC.Alert.4.Severity Critical
    set iDRAC.Alert.4.Action Email,SNMPTrap

    # Fan failure
    set iDRAC.Alert.5.Category Fan
    set iDRAC.Alert.5.Severity Warning
    set iDRAC.Alert.5.Action Email,SNMPTrap
CMDS
done
```

### 5.2 Test Alert

```bash
racadm -r 10.0.100.11 $CRED testemail -i 1
# Check infra-alerts@kubric.local inbox
```

---

## 6. Power Budgeting Commands

### 6.1 Query Current Power

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  echo "=== $IDRAC ==="
  racadm -r $IDRAC $CRED get System.Power.Redundancy
  racadm -r $IDRAC $CRED get System.Power.Supply.1.InputWattage
  racadm -r $IDRAC $CRED get System.Power.Supply.2.InputWattage
done
```

### 6.2 Set Power Cap

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  # Enable power cap at 750W per node (R630 TDP ~750W fully loaded)
  racadm -r $IDRAC $CRED set System.Power.Cap.Enable Enabled
  racadm -r $IDRAC $CRED set System.Power.Cap.Watts 750
done
```

### 6.3 Power Budget Summary

| Node | PSU 1 | PSU 2 | Redundancy | Cap (W) | Notes |
|------|-------|-------|------------|---------|-------|
| pve-kubric-01 | 750W | 750W | 1+1 | 750 | CPU-bound (ClickHouse) |
| pve-kubric-02 | 750W | 750W | 1+1 | 750 | CPU+IO (PG + ClickHouse) |
| pve-kubric-03 | 750W | 750W | 1+1 | 750 | +75W for Tesla T4 GPU |

**Total Cluster Power Budget:** ~2250W peak (typical idle: ~900W)

---

## 7. Firmware Update Procedure

### 7.1 Check Current Firmware

```bash
for IDRAC in 10.0.100.11 10.0.100.12 10.0.100.13; do
  echo "=== $IDRAC ==="
  racadm -r $IDRAC $CRED getversion
done
```

### 7.2 Apply Firmware via racadm

```bash
# Download Dell Server Update Utility or use racadm
# Example: Update iDRAC firmware
racadm -r 10.0.100.11 $CRED update -f /tmp/iDRAC-with-Lifecycle-Controller_Firmware_XXXXX_LN_2.83.83.83_A00.exe

# Or via Redfish API
curl -k -u kubric-admin:${IDRAC_KUBRIC_PASS} \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"ImageURI": "http://10.0.100.21/firmware/idrac_2.83.exe"}' \
  https://10.0.100.11/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate
```

---

## 8. Redfish API Quick Reference

```bash
BASE="https://10.0.100.11/redfish/v1"
AUTH="-u kubric-admin:${IDRAC_KUBRIC_PASS}"

# System overview
curl -sk $AUTH $BASE/Systems/System.Embedded.1 | jq '{Status, PowerState, MemorySummary, ProcessorSummary}'

# Storage
curl -sk $AUTH $BASE/Systems/System.Embedded.1/Storage | jq '.Members[]."@odata.id"'

# Thermal (fans + temps)
curl -sk $AUTH $BASE/Chassis/System.Embedded.1/Thermal | jq '.Temperatures[] | {Name, ReadingCelsius, Status}'

# Power
curl -sk $AUTH $BASE/Chassis/System.Embedded.1/Power | jq '.PowerControl[] | {Name, PowerConsumedWatts, PowerCapacityWatts}'

# Lifecycle logs
curl -sk $AUTH $BASE/Managers/iDRAC.Embedded.1/LogServices/Lclog/Entries | jq '.Members[:5] | .[] | {Created, Message, Severity}'
```

---

## 9. Verification

```bash
# VLAN 100 connectivity from management workstation
ping -c 3 10.0.100.11
ping -c 3 10.0.100.12
ping -c 3 10.0.100.13

# HTTPS access
for ip in 10.0.100.11 10.0.100.12 10.0.100.13; do
  curl -sk https://$ip/redfish/v1 | jq .RedfishVersion
done
# Expected: "1.17.0" or similar

# SNMPv3
for ip in 10.0.100.11 10.0.100.12 10.0.100.13; do
  snmpget -v3 -u snmp-monitor -l authPriv \
    -a SHA -A "${SNMPV3_PASS}" \
    -x AES -X "${SNMPV3_PASS}" \
    $ip sysDescr.0
done
```
