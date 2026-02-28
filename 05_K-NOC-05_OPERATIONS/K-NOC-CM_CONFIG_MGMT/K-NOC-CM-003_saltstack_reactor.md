# K-NOC-CM-003 -- SaltStack Reactor Configuration

**Role:** Event-driven automation using Salt Reactor to trigger remediation, provisioning, and enforcement actions in response to NATS events and system state changes.

---

## 1. Architecture

```
┌──────────────┐  NATS event    ┌──────────────┐  Reactor    ┌──────────────┐
│  Kubric      │───────────────►│  Go Service  │────────────►│  Salt Master │
│  Services    │                │  (NOC)       │  salt-api   │              │
│              │                │              │  REST POST  │  Execute SLS │
└──────────────┘                │  Translate   │             │  Orchestrate │
                                │  to reactor  │             │  Remediate   │
┌──────────────┐  Salt event    │  triggers    │             │              │
│  Salt Minion │───────────────►│              │             │              │
│  (endpoint)  │  event bus     └──────────────┘             └──────────────┘
└──────────────┘
```

---

## 2. Salt Master Reactor Configuration

```yaml
# /etc/salt/master.d/reactor.conf
reactor:
  # Minion authentication events
  - 'salt/auth':
    - /srv/reactor/auto_accept.sls

  # Minion start events — apply baseline state
  - 'salt/minion/*/start':
    - /srv/reactor/minion_start.sls

  # Custom Kubric events
  - 'kubric/drift/detected':
    - /srv/reactor/drift_remediate.sls

  - 'kubric/incident/isolate':
    - /srv/reactor/network_isolate.sls

  - 'kubric/patch/apply':
    - /srv/reactor/patch_apply.sls

  - 'kubric/user/lock':
    - /srv/reactor/user_lock.sls

  # Beacon events
  - 'salt/beacon/*/inotify/*':
    - /srv/reactor/file_change_alert.sls

  - 'salt/beacon/*/service_status/*':
    - /srv/reactor/service_restart.sls
```

---

## 3. Reactor SLS Files

### 3.1 Auto-Accept Minion (With Verification)

```yaml
# /srv/reactor/auto_accept.sls
# Only accept minions whose ID matches our naming convention
{% set minion_id = data['id'] %}
{% if minion_id.startswith('kubric-') or minion_id.startswith('k-') %}
accept_key:
  wheel.key.accept:
    - match: {{ minion_id }}

apply_baseline:
  local.state.apply:
    - tgt: {{ minion_id }}
    - arg:
      - kubric.baseline
    - kwarg:
        queue: True
{% endif %}
```

### 3.2 Minion Start — Apply Baseline

```yaml
# /srv/reactor/minion_start.sls
apply_desired_state:
  local.state.highstate:
    - tgt: {{ data['id'] }}
    - kwarg:
        queue: True
        pillar:
          kubric_event: minion_start
          timestamp: {{ data['_stamp'] }}

report_online:
  local.event.fire:
    - tgt: {{ data['id'] }}
    - arg:
      - kubric/node/online
    - kwarg:
        data:
          minion_id: {{ data['id'] }}
          boot_time: {{ data['_stamp'] }}
```

### 3.3 Drift Remediation

```yaml
# /srv/reactor/drift_remediate.sls
# Triggered when kubric/drift/detected fires
{% set node = data.get('node_id', '') %}
{% set drifts = data.get('drifts', []) %}

{% if node %}
remediate_drift:
  local.state.apply:
    - tgt: {{ node }}
    - arg:
      - kubric.desired_state
    - kwarg:
        queue: True
        pillar:
          drift_event: True
          drift_items: {{ drifts | tojson }}
          auto_remediate: True

notify_remediation:
  local.event.fire_master:
    - arg:
      - kubric/drift/remediation_started
    - kwarg:
        data:
          node_id: {{ node }}
          drift_count: {{ drifts | length }}
{% endif %}
```

### 3.4 Incident Network Isolation

```yaml
# /srv/reactor/network_isolate.sls
# Emergency: isolate compromised endpoint from network
{% set target = data.get('minion_id', '') %}
{% set severity = data.get('severity', 0) %}

{% if target and severity >= 4 %}
isolate_network:
  local.state.apply:
    - tgt: {{ target }}
    - arg:
      - kubric.isolation.network_lockdown
    - kwarg:
        pillar:
          allow_salt_master: True
          allow_dns: False
          incident_id: {{ data.get('incident_id', 'unknown') }}
          isolation_reason: {{ data.get('reason', 'automated incident response') }}
{% endif %}
```

### 3.5 Emergency Patch Application

```yaml
# /srv/reactor/patch_apply.sls
{% set target = data.get('target', '*') %}
{% set packages = data.get('packages', []) %}
{% set cve = data.get('cve_id', '') %}

apply_patches:
  local.state.apply:
    - tgt: {{ target }}
    - arg:
      - kubric.patching.emergency
    - kwarg:
        pillar:
          packages: {{ packages | tojson }}
          cve_id: {{ cve }}
          emergency: True
        queue: True
```

### 3.6 User Account Lock

```yaml
# /srv/reactor/user_lock.sls
{% set target = data.get('minion_id', '*') %}
{% set username = data.get('username', '') %}

{% if username %}
lock_user:
  local.user.chloginclass:
    - tgt: {{ target }}
    - arg:
      - {{ username }}
    - kwarg:
        loginclass: disabled

disable_ssh:
  local.cmd.run:
    - tgt: {{ target }}
    - arg:
      - "passwd -l {{ username }} && pkill -u {{ username }}"
{% endif %}
```

---

## 4. Go Event Bridge — NATS to Salt

```go
// internal/noc/salt_reactor_bridge.go
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	nats "github.com/nats-io/nats.go"
)

// SaltReactorBridge forwards NATS events to Salt event bus via salt-api.
type SaltReactorBridge struct {
	nc          *nats.Conn
	saltAPIURL  string
	saltToken   string
	httpClient  *http.Client
}

func NewSaltReactorBridge(nc *nats.Conn, saltAPIURL, saltToken string) *SaltReactorBridge {
	return &SaltReactorBridge{
		nc:         nc,
		saltAPIURL: saltAPIURL,
		saltToken:  saltToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Start subscribes to relevant NATS subjects and forwards to Salt.
func (srb *SaltReactorBridge) Start(ctx context.Context) error {
	// Drift events -> Salt reactor
	_, err := srb.nc.Subscribe("kubric.noc.drift.>", func(msg *nats.Msg) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}
		srb.fireSaltEvent("kubric/drift/detected", event)
	})
	if err != nil {
		return fmt.Errorf("subscribe drift: %w", err)
	}

	// Incident isolation events -> Salt reactor
	_, err = srb.nc.Subscribe("kubric.soc.incident.isolate.>", func(msg *nats.Msg) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}
		srb.fireSaltEvent("kubric/incident/isolate", event)
	})
	if err != nil {
		return fmt.Errorf("subscribe isolate: %w", err)
	}

	// Patch events -> Salt reactor
	_, err = srb.nc.Subscribe("kubric.noc.patch.>", func(msg *nats.Msg) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}
		srb.fireSaltEvent("kubric/patch/apply", event)
	})
	if err != nil {
		return fmt.Errorf("subscribe patch: %w", err)
	}

	<-ctx.Done()
	return nil
}

// fireSaltEvent sends an event to the Salt master event bus via salt-api.
func (srb *SaltReactorBridge) fireSaltEvent(tag string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"client": "local",
		"fun":    "event.fire_master",
		"arg":    []interface{}{data, tag},
		"tgt":    "*",
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", srb.saltAPIURL+"/run", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", srb.saltToken)

	resp, err := srb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("salt-api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("salt-api error: %d", resp.StatusCode)
	}
	return nil
}
```

---

## 5. Salt Beacons for Continuous Monitoring

```yaml
# /srv/pillar/beacons.sls
beacons:
  inotify:
    - files:
        /etc/passwd:
          mask:
            - modify
        /etc/shadow:
          mask:
            - modify
        /etc/sudoers:
          mask:
            - modify
            - create
        /etc/ssh/sshd_config:
          mask:
            - modify
    - interval: 5
    - disable_during_state_run: True

  service_status:
    - services:
        sshd:
          onchangeonly: True
        kubric-agent:
          onchangeonly: True
        firewalld:
          onchangeonly: True
    - interval: 30
```

---

## 6. Salt-API Configuration

```yaml
# /etc/salt/master.d/api.conf
rest_tornado:
  port: 8000
  ssl_crt: /etc/salt/tls/salt-api.pem
  ssl_key: /etc/salt/tls/salt-api.key
  disable_ssl: false

external_auth:
  pam:
    kubric-noc:
      - .*
      - '@wheel'
      - '@runner'
      - '@events'
```
