# K-SOC-DET-008 -- Tetragon K8s eBPF Enforcement Policies

**License:** Apache 2.0  
**Role:** Kernel-level process, network, and file monitoring for Kubric K8s workloads via Cilium Tetragon TracingPolicy CRDs.

---

## 1. TracingPolicy: Process Execution Monitoring

```yaml
# deployments/k8s/tetragon/policies/process-exec-monitor.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-process-exec
  annotations:
    kubric.io/description: "Monitor all process executions in Kubric namespaces"
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
            - namespace: kubric-agents
              operator: In
            - namespace: kubric-kai
              operator: In
            - namespace: kubric-data
              operator: In
          matchActions:
            - action: Post
  tracepoints:
    - subsystem: "sched"
      event: "sched_process_exec"
      args:
        - index: 0
          type: "nop"
      selectors:
        - matchNamespaces:
            - namespace: kubric-services
              operator: In
          matchActions:
            - action: Post
```

---

## 2. TracingPolicy: Network Connection Tracking

```yaml
# deployments/k8s/tetragon/policies/network-connect-track.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-network-connect
  annotations:
    kubric.io/description: "Track all TCP connections from Kubric pods"
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
          matchActions:
            - action: Post
    - call: "tcp_close"
      syscall: false
      args:
        - index: 0
          type: "sock"
      selectors:
        - matchNamespaces:
            - namespace: kubric-services
              operator: In
          matchActions:
            - action: Post

    # Block egress to known-bad ports from service pods
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
              operator: "DPort"
              values:
                - "4444"   # Metasploit default
                - "5555"   # Common RAT
                - "6666"   # IRC C2
                - "1337"   # Common backdoor
          matchActions:
            - action: Sigkill
            - action: Post
```

---

## 3. TracingPolicy: File Access Auditing

```yaml
# deployments/k8s/tetragon/policies/file-access-audit.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-file-access
  annotations:
    kubric.io/description: "Audit access to sensitive files in Kubric pods"
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
                # Kubric configuration files
                - "/etc/kubric/"
                - "/var/lib/kubric/secrets/"
                # System credential files
                - "/etc/shadow"
                - "/etc/gshadow"
                - "/root/.ssh/"
                - "/home/"
                # Kubernetes secrets
                - "/var/run/secrets/kubernetes.io/"
                # TLS certificates
                - "/etc/ssl/private/"
                - "/certs/"
          matchNamespaces:
            - namespace: kubric-services
              operator: In
            - namespace: kubric-agents
              operator: In
          matchActions:
            - action: Post

    # Detect write attempts to read-only config
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
            - action: Sigkill
            - action: Post
```

---

## 4. Kubric Namespace-Specific Policies

### 4.1 KAI AI Namespace — Prevent Shell Escapes

```yaml
# deployments/k8s/tetragon/policies/kai-shell-prevent.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-kai-shell-prevent
spec:
  kprobes:
    - call: "security_bprm_check"
      syscall: false
      args:
        - index: 0
          type: "linux_binprm"
      selectors:
        - matchNamespaces:
            - namespace: kubric-kai
              operator: In
          matchBinaries:
            - operator: In
              values:
                - "/bin/sh"
                - "/bin/bash"
                - "/bin/zsh"
                - "/usr/bin/python3"
                - "/usr/bin/perl"
                - "/usr/bin/ruby"
                - "/usr/bin/nc"
                - "/usr/bin/ncat"
                - "/usr/bin/wget"
                - "/usr/bin/curl"
          matchActions:
            - action: Sigkill
            - action: Post
```

### 4.2 Data Namespace — Prevent Data Exfiltration

```yaml
# deployments/k8s/tetragon/policies/data-exfil-prevent.yaml
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: kubric-data-exfil-prevent
spec:
  kprobes:
    - call: "tcp_connect"
      syscall: false
      args:
        - index: 0
          type: "sock"
      selectors:
        - matchNamespaces:
            - namespace: kubric-data
              operator: In
          matchArgs:
            - index: 0
              operator: "NotDAddr"
              values:
                # Only allow connections to internal services
                - "10.0.0.0/8"
                - "172.16.0.0/12"
          matchActions:
            - action: Sigkill
            - action: Post
```

---

## 5. Go gRPC Integration with CoreSec

```go
// internal/tetragon/consumer.go
package tetragon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	tetragonAPI "github.com/cilium/tetragon/api/v1/tetragon"
	nats "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

type EventConsumer struct {
	grpcAddr string
	nc       *nats.Conn
	tenantID string
}

func NewEventConsumer(grpcAddr, tenantID string, nc *nats.Conn) *EventConsumer {
	return &EventConsumer{
		grpcAddr: grpcAddr,
		nc:       nc,
		tenantID: tenantID,
	}
}

func (ec *EventConsumer) Start(ctx context.Context) error {
	conn, err := grpc.NewClient(
		ec.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50*1024*1024)),
	)
	if err != nil {
		return fmt.Errorf("grpc connect: %w", err)
	}
	defer conn.Close()

	client := tetragonAPI.NewFineGuidanceSensorsClient(conn)
	stream, err := client.GetEvents(ctx, &tetragonAPI.GetEventsRequest{
		AllowList: []*tetragonAPI.Filter{
			{
				EventSet: []tetragonAPI.EventType{
					tetragonAPI.EventType_PROCESS_EXEC,
					tetragonAPI.EventType_PROCESS_EXIT,
					tetragonAPI.EventType_PROCESS_KPROBE,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("tetragon stream: %w", err)
	}

	marshaler := protojson.MarshalOptions{EmitUnpopulated: false}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}

		ocsf, natsSubject := ec.toOCSF(resp)
		if ocsf == nil {
			// Log raw event for debugging
			raw, _ := marshaler.Marshal(resp)
			_ = ec.nc.Publish(
				fmt.Sprintf("kubric.edr.raw.%s", ec.tenantID),
				raw,
			)
			continue
		}

		data, _ := json.Marshal(ocsf)
		if err := ec.nc.Publish(natsSubject, data); err != nil {
			return fmt.Errorf("nats publish: %w", err)
		}
	}
}

// OCSFProcessActivity - OCSF class 1007
type OCSFProcessActivity struct {
	ClassUID    int                    `json:"class_uid"`
	ActivityID  int                    `json:"activity_id"`
	CategoryUID int                    `json:"category_uid"`
	SeverityID  int                    `json:"severity_id"`
	Time        string                 `json:"time"`
	StatusID    int                    `json:"status_id"`
	Process     *OCSFProcess           `json:"process"`
	Actor       *OCSFActor             `json:"actor,omitempty"`
	Device      *OCSFDevice            `json:"device,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Unmapped    map[string]interface{} `json:"unmapped,omitempty"`
}

type OCSFProcess struct {
	PID     uint32 `json:"pid"`
	Name    string `json:"name"`
	CmdLine string `json:"cmd_line"`
	UID     int32  `json:"uid"`
	File    *struct {
		Path string `json:"path"`
	} `json:"file,omitempty"`
}

type OCSFActor struct {
	Process *OCSFProcess `json:"process,omitempty"`
}

type OCSFDevice struct {
	Hostname string `json:"hostname"`
}

func (ec *EventConsumer) toOCSF(resp *tetragonAPI.GetEventsResponse) (*OCSFProcessActivity, string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	meta := map[string]interface{}{
		"product": map[string]string{
			"name":        "Tetragon",
			"vendor_name": "Cilium",
		},
		"tenant_uid": ec.tenantID,
	}

	switch ev := resp.Event.(type) {
	case *tetragonAPI.GetEventsResponse_ProcessExec:
		p := ev.ProcessExec.Process
		ns := ""
		podName := ""
		if p.Pod != nil {
			ns = p.Pod.Namespace
			podName = p.Pod.Name
		}

		severity := 1 // Info
		if ev.ProcessExec.Process.Binary == "/bin/sh" ||
			ev.ProcessExec.Process.Binary == "/bin/bash" {
			severity = 3 // Medium — shell exec in pod
		}

		return &OCSFProcessActivity{
			ClassUID:    1007,
			ActivityID:  1,
			CategoryUID: 1,
			SeverityID:  severity,
			Time:        now,
			StatusID:    1,
			Process: &OCSFProcess{
				PID:     p.Pid.GetValue(),
				Name:    p.Binary,
				CmdLine: p.Arguments,
				UID:     int32(p.ProcessCredentials.GetUid().GetValue()),
				File:    &struct{ Path string }{Path: p.Binary},
			},
			Device:   &OCSFDevice{Hostname: podName},
			Metadata: meta,
			Unmapped: map[string]interface{}{
				"k8s_namespace": ns,
				"k8s_pod":       podName,
				"action":        "exec",
			},
		}, fmt.Sprintf("kubric.edr.process.%s", ec.tenantID)

	case *tetragonAPI.GetEventsResponse_ProcessKprobe:
		kp := ev.ProcessKprobe
		p := kp.Process
		ns := ""
		if p.Pod != nil {
			ns = p.Pod.Namespace
		}

		return &OCSFProcessActivity{
			ClassUID:    1007,
			ActivityID:  99, // Other
			CategoryUID: 1,
			SeverityID:  3,
			Time:        now,
			StatusID:    1,
			Process: &OCSFProcess{
				PID:  p.Pid.GetValue(),
				Name: p.Binary,
			},
			Metadata: meta,
			Unmapped: map[string]interface{}{
				"k8s_namespace": ns,
				"kprobe_func":   kp.FunctionName,
				"kprobe_action": kp.Action.String(),
			},
		}, fmt.Sprintf("kubric.edr.process.%s", ec.tenantID)

	default:
		return nil, ""
	}
}
```

---

## 6. Apply Policies

```bash
# Apply all Kubric Tetragon policies
kubectl apply -f deployments/k8s/tetragon/policies/

# Verify policies are active
kubectl get tracingpolicies -A

# Monitor events in real-time
kubectl logs -n kube-system -l app.kubernetes.io/name=tetragon \
  -c export-stdout -f | jq '.process_exec.process.binary // .process_kprobe.function_name'

# Check for enforcement actions (Sigkill events)
kubectl logs -n kube-system -l app.kubernetes.io/name=tetragon \
  -c export-stdout -f | jq 'select(.process_kprobe.action == "KPROBE_ACTION_SIGKILL")'
```

---

## 7. Event Flow

```
Tetragon eBPF (kernel) ──gRPC──► Go EventConsumer ──OCSF JSON──► NATS
                                                                   │
                                                   kubric.edr.process.{tenant}
                                                                   │
                                        ┌──────────────────────────┼───────────────┐
                                        ▼                          ▼               ▼
                                   CoreSec                   ClickHouse      Incident
                                   (Sigma eval)             (telemetry)      Stitcher
```
