# K-XRO-SD-002 -- Cilium Tetragon eBPF Security Enforcement

**License:** Apache 2.0  
**Role:** Kernel-level security observability and enforcement on Kubernetes nodes via eBPF.  
**Complement:** Tetragon handles K8s enforcement; Aya-rs eBPF (in CoreSec) handles bare-metal/VM endpoints.

---

## 1. Architecture

```
┌─────────────────────────────────────────────────┐
│  Kubernetes Node                                 │
│                                                  │
│  ┌──────────────┐   gRPC    ┌────────────────┐  │
│  │  Tetragon    │◄─────────►│  CoreSec Go    │  │
│  │  (eBPF DaemonSet)        │  gRPC Client   │  │
│  │  TracingPolicy│          └───────┬────────┘  │
│  └──────────────┘                   │ NATS      │
│         │                           ▼            │
│         │ eBPF hooks         kubric.edr.process.>│
│         ▼                                        │
│  ┌──────────────┐   Hubble  ┌────────────────┐  │
│  │  Kernel      │──────────►│  Hubble UI     │  │
│  │  Syscalls    │           │  Flow Visibility│  │
│  └──────────────┘           └────────────────┘  │
└─────────────────────────────────────────────────┘
```

---

## 2. Helm Installation

```bash
# Add Cilium Helm repo
helm repo add cilium https://helm.cilium.io
helm repo update

# Install Tetragon (standalone, does not require Cilium CNI)
helm install tetragon cilium/tetragon \
  --namespace kube-system \
  --version 1.2.0 \
  --set tetragon.grpc.address="localhost:54321" \
  --set tetragon.exportAllowList='{"event_set":["PROCESS_EXEC","PROCESS_EXIT","PROCESS_KPROBE","PROCESS_TRACEPOINT"]}' \
  --set tetragon.fieldFilters='[{"event_set":["PROCESS_EXEC"],"fields":"process.binary,process.arguments,process.pod","action":"INCLUDE"}]' \
  --set tetragon.enableProcessCred=true \
  --set tetragon.enableProcessNs=true \
  --set export.stdout.enabledCommand=true \
  --set export.stdout.enabledArgs=true

# Verify installation
kubectl get pods -n kube-system -l app.kubernetes.io/name=tetragon
kubectl logs -n kube-system -l app.kubernetes.io/name=tetragon -c export-stdout --tail=5
```

---

## 3. TracingPolicy CRDs

### 3.1 Process Execution Monitoring

```yaml
# deployments/k8s/tetragon/process-monitor.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-process-monitor
  namespace: kube-system
spec:
  kprobes:
    - call: "security_bprm_check"
      syscall: false
      args:
        - index: 0
          type: "linux_binprm"
      selectors:
        - matchNamespaces:
            - namespace: kubric-agents
              operator: In
            - namespace: kubric-services
              operator: In
            - namespace: kubric-kai
              operator: In
        - matchBinaries:
            - operator: NotIn
              values:
                - "/usr/bin/kubric-agent"
                - "/usr/local/bin/tetragon"
                - "/usr/bin/pause"
          matchActions:
            - action: Post
              rateLimit: "1m"
              rateLimitScope: process
```

### 3.2 Block Unauthorized Process Execution

```yaml
# deployments/k8s/tetragon/block-unauthorized.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-block-unauthorized-exec
spec:
  kprobes:
    - call: "security_bprm_check"
      syscall: false
      args:
        - index: 0
          type: "linux_binprm"
      selectors:
        - matchNamespaces:
            - namespace: kubric-services
              operator: In
          matchBinaries:
            - operator: NotIn
              values:
                - "/usr/bin/ksvc"
                - "/usr/bin/vdr"
                - "/usr/bin/kic"
                - "/usr/bin/noc"
                - "/bin/sh"
                - "/usr/bin/curl"    # health checks only
          matchActions:
            - action: Sigkill   # Kill unauthorized process immediately
```

### 3.3 Network Egress Control

```yaml
# deployments/k8s/tetragon/network-egress.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-network-egress-control
spec:
  kprobes:
    - call: "tcp_connect"
      syscall: false
      args:
        - index: 0
          type: "sock"
      selectors:
        - matchNamespaces:
            - namespace: kubric-services
              operator: In
          matchArgs:
            - index: 0
              operator: "NotDAddr"
              values:
                # Allowed egress destinations
                - "10.0.0.0/8"       # Internal cluster
                - "172.16.0.0/12"    # Internal services
                - "169.254.169.254"  # IMDS (for cloud metadata)
          matchActions:
            - action: Post
            # Log but don't block — alert for review
```

### 3.4 File Access Auditing

```yaml
# deployments/k8s/tetragon/file-audit.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-file-access-audit
spec:
  kprobes:
    - call: "security_file_open"
      syscall: false
      return: true
      args:
        - index: 0
          type: "file"
      returnArg:
        index: 0
        type: "int"
      selectors:
        - matchArgs:
            - index: 0
              operator: "Prefix"
              values:
                - "/etc/kubric/"
                - "/var/lib/kubric/"
                - "/etc/shadow"
                - "/etc/passwd"
                - "/root/.ssh/"
          matchActions:
            - action: Post
    - call: "security_file_permission"
      syscall: false
      args:
        - index: 0
          type: "file"
        - index: 1
          type: "int"
      selectors:
        - matchArgs:
            - index: 0
              operator: "Prefix"
              values:
                - "/etc/kubric/"
            - index: 1
              operator: "Mask"
              values:
                - "2"   # MAY_WRITE
          matchActions:
            - action: Post
```

---

## 4. gRPC Client Integration (Go)

```go
// internal/tetragon/client.go
package tetragon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cilium/tetragon/api/v1/tetragon"
	nats "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TetragonClient consumes Tetragon gRPC events and publishes to NATS.
type TetragonClient struct {
	grpcAddr string
	nc       *nats.Conn
}

func NewTetragonClient(grpcAddr string, nc *nats.Conn) *TetragonClient {
	return &TetragonClient{grpcAddr: grpcAddr, nc: nc}
}

// StreamEvents connects to Tetragon gRPC and streams events to NATS.
func (tc *TetragonClient) StreamEvents(ctx context.Context) error {
	conn, err := grpc.NewClient(
		tc.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("tetragon grpc dial: %w", err)
	}
	defer conn.Close()

	client := tetragon.NewFineGuidanceSensorsClient(conn)
	stream, err := client.GetEvents(ctx, &tetragon.GetEventsRequest{
		AllowList: []*tetragon.Filter{
			{
				EventSet: []tetragon.EventType{
					tetragon.EventType_PROCESS_EXEC,
					tetragon.EventType_PROCESS_EXIT,
					tetragon.EventType_PROCESS_KPROBE,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("tetragon get events: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tetragon recv: %w", err)
		}

		ocsf, subject, err := tc.mapToOCSF(resp)
		if err != nil {
			continue // skip unmappable events
		}

		data, _ := json.Marshal(ocsf)
		if err := tc.nc.Publish(subject, data); err != nil {
			return fmt.Errorf("nats publish: %w", err)
		}
	}
}

// OCSFProcessActivity maps Tetragon events to OCSF class 1007 (Process Activity).
type OCSFProcessActivity struct {
	ClassUID    int                    `json:"class_uid"`    // 1007
	ActivityID  int                    `json:"activity_id"`  // 1=Launch, 2=Terminate
	CategoryUID int                    `json:"category_uid"` // 1 = System Activity
	SeverityID  int                    `json:"severity_id"`
	Time        string                 `json:"time"`
	Process     map[string]interface{} `json:"process"`
	Actor       map[string]interface{} `json:"actor,omitempty"`
	Device      map[string]interface{} `json:"device,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Unmapped    map[string]interface{} `json:"unmapped,omitempty"`
}

func (tc *TetragonClient) mapToOCSF(
	resp *tetragon.GetEventsResponse,
) (*OCSFProcessActivity, string, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	switch ev := resp.Event.(type) {
	case *tetragon.GetEventsResponse_ProcessExec:
		p := ev.ProcessExec.Process
		ns := "default"
		podName := ""
		if p.Pod != nil {
			ns = p.Pod.Namespace
			podName = p.Pod.Name
		}
		return &OCSFProcessActivity{
			ClassUID:    1007,
			ActivityID:  1, // Launch
			CategoryUID: 1,
			SeverityID:  1,
			Time:        now,
			Process: map[string]interface{}{
				"pid":  p.Pid.GetValue(),
				"name": p.Binary,
				"cmd_line": p.Arguments,
				"uid":  p.ProcessCredentials.GetUid().GetValue(),
			},
			Device: map[string]interface{}{
				"hostname": p.Pod.GetName(),
			},
			Metadata: map[string]interface{}{
				"product": map[string]string{
					"name":        "Tetragon",
					"vendor_name": "Cilium",
				},
			},
			Unmapped: map[string]interface{}{
				"k8s_namespace": ns,
				"k8s_pod":       podName,
			},
		}, fmt.Sprintf("kubric.edr.process.%s", ns), nil

	case *tetragon.GetEventsResponse_ProcessExit:
		p := ev.ProcessExit.Process
		ns := "default"
		if p.Pod != nil {
			ns = p.Pod.Namespace
		}
		return &OCSFProcessActivity{
			ClassUID:    1007,
			ActivityID:  2, // Terminate
			CategoryUID: 1,
			SeverityID:  1,
			Time:        now,
			Process: map[string]interface{}{
				"pid":  p.Pid.GetValue(),
				"name": p.Binary,
			},
			Metadata: map[string]interface{}{
				"product": map[string]string{
					"name":        "Tetragon",
					"vendor_name": "Cilium",
				},
			},
		}, fmt.Sprintf("kubric.edr.process.%s", ns), nil

	default:
		return nil, "", fmt.Errorf("unsupported event type")
	}
}
```

---

## 5. Hubble Integration for Flow Visibility

```bash
# Install Hubble CLI
HUBBLE_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/hubble/master/stable.txt)
curl -L --remote-name-all \
  https://github.com/cilium/hubble/releases/download/$HUBBLE_VERSION/hubble-linux-amd64.tar.gz
sudo tar xzvfC hubble-linux-amd64.tar.gz /usr/local/bin

# Enable Hubble in Tetragon namespace
kubectl port-forward -n kube-system svc/hubble-relay 4245:443 &

# Observe flows in kubric namespaces
hubble observe --namespace kubric-services --protocol tcp -o jsonpb
hubble observe --namespace kubric-agents --verdict DROPPED -o jsonpb

# Kubric-specific flow filters
hubble observe \
  --from-namespace kubric-services \
  --to-namespace kubric-services \
  --type l7 \
  -o json | jq '.flow.l7.dns // .flow.l7.http'
```

### Hubble Relay Service

```yaml
# deployments/k8s/hubble-relay.yaml
apiVersion: v1
kind: Service
metadata:
  name: hubble-relay
  namespace: kube-system
spec:
  type: ClusterIP
  selector:
    k8s-app: hubble-relay
  ports:
    - port: 443
      targetPort: 4245
      protocol: TCP
```

---

## 6. Tetragon vs Aya-rs eBPF Boundary

| Capability | Tetragon (K8s) | Aya-rs (Bare-metal) |
|------------|---------------|---------------------|
| Deployment | DaemonSet on K8s | Compiled into XRO agent |
| Policy format | TracingPolicy CRD | Rust code + BPF maps |
| Management | kubectl / Helm | NATS config push |
| Enforcement | Sigkill, Override | XDP drop, TC redirect |
| Container awareness | Full (Pod, namespace) | None (host-level) |
| Use case | K8s workload protection | Endpoint/VM EDR |
| License | Apache 2.0 | Apache 2.0 (Aya) |

---

## 7. Monitoring & Alerts

```yaml
# deployments/k8s/tetragon/prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tetragon-alerts
  namespace: kube-system
spec:
  groups:
    - name: tetragon.rules
      rules:
        - alert: TetragonPolicyViolation
          expr: rate(tetragon_events_total{event_type="PROCESS_EXEC",action="sigkill"}[5m]) > 0
          for: 1m
          labels:
            severity: critical
            team: soc
          annotations:
            summary: "Tetragon killed unauthorized process"
            description: "{{ $labels.namespace }}/{{ $labels.pod }}: unauthorized process blocked"

        - alert: TetragonAgentDown
          expr: up{job="tetragon"} == 0
          for: 5m
          labels:
            severity: critical
            team: noc
          annotations:
            summary: "Tetragon agent not responding"
```
