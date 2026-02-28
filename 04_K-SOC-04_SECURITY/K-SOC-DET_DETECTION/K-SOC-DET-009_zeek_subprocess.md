# K-SOC-DET-009 -- Zeek Network Monitor as Subprocess

**License:** BSD-3-Clause — scripts are data, safe to vendor.  
**Vendored at:** `vendor/zeek/scripts/`  
**Role:** Deep network protocol analysis, TLS fingerprinting (JA3/JA3S), and behavioral analytics via Zeek subprocess invocation from Go services.

---

## 1. Architecture

```
┌────────────────────────────────────────────────┐
│  Go Service (NetGuard / NOC)                    │
│                                                 │
│  exec.Command("zeek") ──► Zeek process          │
│           │                    │                 │
│           │ PCAP input         │ JSON logs       │
│           ▼                    ▼                 │
│       /tmp/capture.pcap   /tmp/zeek-logs/        │
│                            ├── conn.log          │
│                            ├── dns.log           │
│                            ├── ssl.log           │
│                            ├── http.log          │
│                            ├── files.log         │
│                            └── ja3.log           │
│                                │                 │
│  Parse JSON ◄──────────────────┘                 │
│       │                                          │
│       │ NATS publish                             │
│       ▼                                          │
│  kubric.ndr.zeek.{tenant_id}                     │
└────────────────────────────────────────────────┘
```

---

## 2. Zeek Installation (Docker)

```dockerfile
# docker/zeek/Dockerfile
FROM zeek/zeek:7.0.2

# Install JA3 plugin
RUN zkg install ja3 --force

# Copy vendored Kubric scripts
COPY vendor/zeek/scripts/ /opt/kubric/zeek-scripts/

# Enable JSON log output
RUN echo '@load policy/tuning/json-logs' >> /opt/zeek/share/zeek/site/local.zeek && \
    echo '@load /opt/kubric/zeek-scripts/kubric.zeek' >> /opt/zeek/share/zeek/site/local.zeek

ENV PATH="/opt/zeek/bin:$PATH"
```

---

## 3. Vendored Zeek Scripts

```zeek
# vendor/zeek/scripts/kubric.zeek
# Kubric-specific Zeek configuration

@load base/protocols/conn
@load base/protocols/dns
@load base/protocols/http
@load base/protocols/ssl
@load base/files/hash-all-files
@load policy/protocols/ssl/validate-certs
@load policy/protocols/ssl/log-hostcerts-only
@load policy/tuning/json-logs

# JA3 TLS fingerprinting
@load ja3

# Custom notice for suspicious connections
module Kubric;

export {
    redef enum Notice::Type += {
        Kubric::C2_Beacon_Detected,
        Kubric::DNS_Tunnel_Suspected,
        Kubric::Long_Connection,
        Kubric::Self_Signed_Cert,
    };
}

# Alert on connections lasting > 8 hours (potential C2 keep-alive)
event connection_state_remove(c: connection) {
    if ( c$duration > 8hr && c$conn$proto == tcp ) {
        NOTICE([
            $note=Kubric::Long_Connection,
            $conn=c,
            $msg=fmt("Long-lived TCP connection: %s:%s -> %s:%s duration=%s",
                c$id$orig_h, c$id$orig_p, c$id$resp_h, c$id$resp_p, c$duration),
            $identifier=cat(c$id$orig_h, c$id$resp_h),
            $suppress_for=1hr
        ]);
    }
}

# Alert on self-signed certificates
event ssl_established(c: connection) {
    if ( c$ssl?$validation_status && c$ssl$validation_status == "self signed certificate" ) {
        NOTICE([
            $note=Kubric::Self_Signed_Cert,
            $conn=c,
            $msg=fmt("Self-signed TLS cert: %s -> %s:%s subject=%s",
                c$id$orig_h, c$id$resp_h, c$id$resp_p,
                c$ssl?$subject ? c$ssl$subject : "unknown"),
            $identifier=cat(c$id$resp_h, c$id$resp_p)
        ]);
    }
}

# DNS tunnel detection: high-entropy subdomains
event dns_request(c: connection, msg: dns_msg, query: string, qtype: count, qclass: count) {
    local parts = split_string(query, /\./);
    if ( |parts| > 4 ) {
        local subdomain = parts[0];
        if ( |subdomain| > 40 ) {
            NOTICE([
                $note=Kubric::DNS_Tunnel_Suspected,
                $conn=c,
                $msg=fmt("Suspected DNS tunnel: query=%s len=%d", query, |subdomain|),
                $identifier=cat(c$id$orig_h, query)
            ]);
        }
    }
}
```

---

## 4. Go Subprocess Execution

```go
// internal/zeek/runner.go
package zeek

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	nats "github.com/nats-io/nats.go"
)

// ZeekRunner executes Zeek as a subprocess and parses JSON output.
type ZeekRunner struct {
	binaryPath string
	scriptsDir string
	logDir     string
	nc         *nats.Conn
	tenantID   string
}

func NewZeekRunner(nc *nats.Conn, tenantID string) *ZeekRunner {
	return &ZeekRunner{
		binaryPath: getEnv("ZEEK_BIN", "/opt/zeek/bin/zeek"),
		scriptsDir: getEnv("ZEEK_SCRIPTS", "vendor/zeek/scripts"),
		logDir:     getEnv("ZEEK_LOG_DIR", "/tmp/zeek-logs"),
		nc:         nc,
		tenantID:   tenantID,
	}
}

// AnalyzePCAP runs Zeek against a PCAP file and returns parsed logs.
func (zr *ZeekRunner) AnalyzePCAP(ctx context.Context, pcapFile string) error {
	// Create temporary output directory
	outDir, err := os.MkdirTemp(zr.logDir, "zeek-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(outDir)

	// Build Zeek command
	cmd := exec.CommandContext(ctx, zr.binaryPath,
		"-r", pcapFile,
		"-C",                        // ignore checksum errors
		"LogAscii::use_json=T",     // force JSON output
		fmt.Sprintf("Log::default_logdir=%s", outDir),
		"local",                     // load local.zeek
		filepath.Join(zr.scriptsDir, "kubric.zeek"),
	)

	cmd.Dir = outDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zeek exec: %w (output: %s)", err, string(output))
	}

	// Parse generated log files
	logFiles := []string{
		"conn.log", "dns.log", "ssl.log", "http.log",
		"files.log", "notice.log",
	}

	for _, logFile := range logFiles {
		logPath := filepath.Join(outDir, logFile)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue
		}

		events, err := zr.parseZeekLog(logPath)
		if err != nil {
			continue
		}

		for _, event := range events {
			ocsf := zr.toOCSF(logFile, event)
			data, _ := json.Marshal(ocsf)
			subject := fmt.Sprintf("kubric.ndr.zeek.%s", zr.tenantID)
			_ = zr.nc.Publish(subject, data)
		}
	}

	return nil
}

// AnalyzeLive starts Zeek in live capture mode on an interface.
func (zr *ZeekRunner) AnalyzeLive(ctx context.Context, iface string) error {
	outDir, err := os.MkdirTemp(zr.logDir, "zeek-live-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, zr.binaryPath,
		"-i", iface,
		"-C",
		"LogAscii::use_json=T",
		fmt.Sprintf("Log::default_logdir=%s", outDir),
		"local",
		filepath.Join(zr.scriptsDir, "kubric.zeek"),
	)

	cmd.Dir = outDir
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("zeek live start: %w", err)
	}

	// Tail logs in background
	go zr.tailLogs(ctx, outDir)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			_ = scanner.Text() // Zeek stdout messages
		}
	}()

	return cmd.Wait()
}

func (zr *ZeekRunner) parseZeekLog(logPath string) ([]map[string]interface{}, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

func (zr *ZeekRunner) tailLogs(ctx context.Context, logDir string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	offsets := make(map[string]int64) // track read positions

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			entries, _ := os.ReadDir(logDir)
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) != ".log" {
					continue
				}
				logPath := filepath.Join(logDir, entry.Name())
				offset := offsets[logPath]

				f, err := os.Open(logPath)
				if err != nil {
					continue
				}
				f.Seek(offset, io.SeekStart)

				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if len(line) == 0 || line[0] == '#' {
						continue
					}
					var event map[string]interface{}
					if json.Unmarshal([]byte(line), &event) == nil {
						ocsf := zr.toOCSF(entry.Name(), event)
						data, _ := json.Marshal(ocsf)
						_ = zr.nc.Publish(
							fmt.Sprintf("kubric.ndr.zeek.%s", zr.tenantID), data,
						)
					}
				}

				pos, _ := f.Seek(0, io.SeekCurrent)
				offsets[logPath] = pos
				f.Close()
			}
		}
	}
}

// toOCSF converts a Zeek log entry to OCSF NetworkActivity (4001).
func (zr *ZeekRunner) toOCSF(logFile string, event map[string]interface{}) map[string]interface{} {
	ocsf := map[string]interface{}{
		"class_uid":    4001, // NetworkActivity
		"category_uid": 4,   // Network Activity
		"severity_id":  1,   // Info
		"time":         time.Now().UTC().Format(time.RFC3339Nano),
		"metadata": map[string]interface{}{
			"product": map[string]string{
				"name":        "Zeek",
				"vendor_name": "Zeek Project",
			},
			"tenant_uid": zr.tenantID,
		},
	}

	switch logFile {
	case "conn.log":
		ocsf["activity_id"] = 6 // Traffic
		ocsf["src_endpoint"] = map[string]interface{}{
			"ip":   event["id.orig_h"],
			"port": event["id.orig_p"],
		}
		ocsf["dst_endpoint"] = map[string]interface{}{
			"ip":   event["id.resp_h"],
			"port": event["id.resp_p"],
		}
		ocsf["connection_info"] = map[string]interface{}{
			"protocol_name": event["proto"],
			"uid":           event["uid"],
		}
		ocsf["unmapped"] = map[string]interface{}{
			"zeek_log":  "conn",
			"duration":  event["duration"],
			"orig_bytes": event["orig_bytes"],
			"resp_bytes": event["resp_bytes"],
			"conn_state": event["conn_state"],
			"service":    event["service"],
		}

	case "dns.log":
		ocsf["class_uid"] = 4003 // DNS Activity
		ocsf["activity_id"] = 1  // Query
		ocsf["query"] = map[string]interface{}{
			"hostname": event["query"],
			"type":     event["qtype_name"],
		}
		ocsf["src_endpoint"] = map[string]interface{}{
			"ip": event["id.orig_h"],
		}
		ocsf["unmapped"] = map[string]interface{}{
			"zeek_log": "dns",
			"rcode":    event["rcode_name"],
			"answers":  event["answers"],
		}

	case "ssl.log":
		ocsf["activity_id"] = 6
		ocsf["tls"] = map[string]interface{}{
			"version":     event["version"],
			"cipher":      event["cipher"],
			"server_name": event["server_name"],
			"ja3":         event["ja3"],
			"ja3s":        event["ja3s"],
		}
		ocsf["unmapped"] = map[string]interface{}{
			"zeek_log":          "ssl",
			"subject":           event["subject"],
			"issuer":            event["issuer"],
			"validation_status": event["validation_status"],
		}

	case "http.log":
		ocsf["class_uid"] = 4002 // HTTP Activity
		ocsf["activity_id"] = 1
		ocsf["http_request"] = map[string]interface{}{
			"http_method": event["method"],
			"url": map[string]interface{}{
				"hostname": event["host"],
				"path":     event["uri"],
			},
		}
		ocsf["http_response"] = map[string]interface{}{
			"code": event["status_code"],
		}
		ocsf["unmapped"] = map[string]interface{}{
			"zeek_log":   "http",
			"user_agent": event["user_agent"],
			"referrer":   event["referrer"],
		}

	case "notice.log":
		ocsf["severity_id"] = 3 // Medium
		ocsf["unmapped"] = map[string]interface{}{
			"zeek_log":    "notice",
			"notice_type": event["note"],
			"message":     event["msg"],
		}
	}

	return ocsf
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

---

## 5. JA3/JA3S TLS Fingerprinting

JA3 fingerprints are generated by the Zeek `ja3` package and appear in `ssl.log`. Known malicious JA3 hashes are maintained in a lookup table.

```go
// internal/zeek/ja3_lookup.go
package zeek

// KnownMaliciousJA3 contains JA3 hashes associated with malware families.
// Source: https://sslbl.abuse.ch/ja3-fingerprints/
var KnownMaliciousJA3 = map[string]string{
	"51c64c77e60f3980eea90869b68c58a8": "Cobalt Strike",
	"72a589da586844d7f0818ce684948eea": "Metasploit Meterpreter",
	"a0e9f5d64349fb13191bc781f81f42e1": "Trickbot",
	"e7d705a3286e19ea42f587b344ee6865": "AsyncRAT",
	"6734f37431670b3ab4292b8f60f29984": "Dridex",
	"b386946a5a44d1ddcc843bc75336dfce": "Emotet Epoch4",
}

// CheckJA3 returns the malware family if JA3 hash is known-malicious.
func CheckJA3(ja3Hash string) (family string, malicious bool) {
	family, malicious = KnownMaliciousJA3[ja3Hash]
	return
}
```

---

## 6. RITA Integration for Behavioral Analysis

```go
// internal/zeek/rita.go
package zeek

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// RITAAnalyzer runs RITA (Real Intelligence Threat Analytics) against Zeek logs.
type RITAAnalyzer struct {
	binaryPath string
}

func NewRITAAnalyzer() *RITAAnalyzer {
	return &RITAAnalyzer{
		binaryPath: getEnv("RITA_BIN", "/usr/local/bin/rita"),
	}
}

// AnalyzeBeacons detects C2 beacon patterns in Zeek conn.log data.
func (r *RITAAnalyzer) AnalyzeBeacons(ctx context.Context, dbName string) ([]BeaconResult, error) {
	cmd := exec.CommandContext(ctx, r.binaryPath,
		"show-beacons", dbName, "--json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rita show-beacons: %w", err)
	}

	var results []BeaconResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("parse rita output: %w", err)
	}

	return results, nil
}

// ImportZeekLogs imports Zeek logs into RITA database.
func (r *RITAAnalyzer) ImportZeekLogs(ctx context.Context, logDir, dbName string) error {
	cmd := exec.CommandContext(ctx, r.binaryPath,
		"import", logDir, dbName,
	)
	return cmd.Run()
}

type BeaconResult struct {
	Score         float64 `json:"score"`
	Source        string  `json:"src"`
	Destination   string  `json:"dst"`
	Connections   int     `json:"connection_count"`
	AvgBytes      float64 `json:"avg_bytes"`
	TSScore       float64 `json:"ts_score"`
	DSScore       float64 `json:"ds_score"`
	DurScore      float64 `json:"dur_score"`
	HistScore     float64 `json:"hist_score"`
}
```

---

## 7. Docker Compose

```yaml
# docker-compose.yml (snippet)
services:
  zeek:
    build:
      context: .
      dockerfile: docker/zeek/Dockerfile
    volumes:
      - zeek-logs:/var/log/zeek
      - pcap-data:/pcap:ro
      - ./vendor/zeek/scripts:/opt/kubric/zeek-scripts:ro
    network_mode: host
    cap_add:
      - NET_RAW
      - NET_ADMIN
    command: ["zeek", "-i", "eth0", "-C", "LogAscii::use_json=T", "local"]

volumes:
  zeek-logs:
  pcap-data:
```
