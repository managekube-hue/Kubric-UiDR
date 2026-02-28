# K-NOC-BR-009 -- Bareos Backup Configuration

**License:** AGPL 3.0 — runs as separate service.  
**Role:** Enterprise backup for customer endpoints and server workloads, managed by Kubric NOC.

---

## 1. Architecture

```
┌──────────────┐  FileDaemon   ┌──────────────┐   Storage    ┌──────────────┐
│  Customer    │──────────────►│  Bareos       │─────────────►│  MinIO       │
│  Endpoints   │  Port 9102   │  Director     │  S3 Backend  │  Object      │
│  (FD)        │              │  Port 9101    │              │  Storage     │
└──────────────┘              └──────┬────────┘              └──────────────┘
                                     │
                                     │ REST API
                                     ▼
                              ┌──────────────┐
                              │  Go Service  │
                              │  (NOC)       │
                              │              │
                              │  Schedule    │
                              │  Monitor     │
                              │  Report      │
                              └──────────────┘
```

---

## 2. Director Configuration

```conf
# /etc/bareos/bareos-dir.d/director/kubric-dir.conf
Director {
  Name = kubric-dir
  QueryFile = "/usr/lib/bareos/scripts/query.sql"
  Maximum Concurrent Jobs = 50
  Password = "@BAREOS_DIR_PASSWORD@"
  Messages = Daemon
  Auditing = yes

  # TLS for all director communications
  TLS Enable = yes
  TLS Require = yes
  TLS Verify Peer = yes
  TLS Certificate = /etc/bareos/tls/bareos-dir.pem
  TLS Key = /etc/bareos/tls/bareos-dir.key
  TLS CA Certificate File = /etc/bareos/tls/ca.pem
}
```

---

## 3. Storage Daemon — MinIO S3 Backend

```conf
# /etc/bareos/bareos-sd.d/device/minio-s3.conf
Device {
  Name = S3-MinIO-Device
  Media Type = S3
  Archive Device = "S3 Object Storage"
  Device Type = droplet
  Device Options = "profile=kubric-minio,bucket=bareos-backups,chunksize=100M"
  Label Media = yes
  Random Access = yes
  Automatic Mount = yes
  Removable Media = no
  Always Open = no
  Maximum Concurrent Jobs = 10
}

# /etc/bareos/bareos-sd.d/storage/minio-storage.conf
Storage {
  Name = S3-MinIO-Storage
  Address = minio.internal
  SD Port = 9103
  Password = "@BAREOS_SD_PASSWORD@"
  Device = S3-MinIO-Device
  Media Type = S3
  Maximum Concurrent Jobs = 10

  TLS Enable = yes
  TLS Require = yes
  TLS Certificate = /etc/bareos/tls/bareos-sd.pem
  TLS Key = /etc/bareos/tls/bareos-sd.key
  TLS CA Certificate File = /etc/bareos/tls/ca.pem
}
```

```ini
# /etc/bareos/bareos-sd.d/device/droplet/kubric-minio.profile
host = minio.internal
port = 9000
use_ssl = true
access_key = @MINIO_ACCESS_KEY@
secret_key = @MINIO_SECRET_KEY@
```

---

## 4. Job Templates

```conf
# /etc/bareos/bareos-dir.d/jobdefs/kubric-defaults.conf
JobDefs {
  Name = "KubricDefaultJob"
  Type = Backup
  Level = Incremental
  Storage = S3-MinIO-Storage
  Messages = Standard
  Pool = KubricIncremental
  Priority = 10
  Write Bootstrap = "/var/lib/bareos/%c.bsr"
  Full Backup Pool = KubricFull
  Differential Backup Pool = KubricDifferential
  Incremental Backup Pool = KubricIncremental

  # Retry configuration
  Maximum Concurrent Jobs = 5
  Reschedule On Error = yes
  Reschedule Interval = 1 hour
  Reschedule Times = 3
}

# /etc/bareos/bareos-dir.d/job/kubric-linux-servers.conf
Job {
  Name = "KubricLinuxServers"
  JobDefs = "KubricDefaultJob"
  Client = kubric-linux-fd
  FileSet = "LinuxServerFiles"
  Schedule = "KubricWeeklyCycle"
}

# /etc/bareos/bareos-dir.d/job/kubric-windows-workstations.conf
Job {
  Name = "KubricWindowsWorkstations"
  JobDefs = "KubricDefaultJob"
  Client = kubric-windows-fd
  FileSet = "WindowsWorkstationFiles"
  Schedule = "KubricWeeklyCycle"
}
```

---

## 5. FileSets

```conf
# /etc/bareos/bareos-dir.d/fileset/linux-servers.conf
FileSet {
  Name = "LinuxServerFiles"
  Enable VSS = no
  Include {
    Options {
      Signature = SHA256
      Compression = LZ4
      One FS = no
    }
    File = /etc
    File = /home
    File = /var/lib
    File = /opt
    File = /srv
  }
  Exclude {
    File = /var/lib/bareos
    File = /proc
    File = /sys
    File = /tmp
    File = /dev
    File = /run
    File = "*.tmp"
    File = "*.swap"
  }
}

# /etc/bareos/bareos-dir.d/fileset/windows-workstations.conf
FileSet {
  Name = "WindowsWorkstationFiles"
  Enable VSS = yes
  Include {
    Options {
      Signature = SHA256
      Compression = LZ4
      Drive Type = fixed
    }
    File = "C:/Users"
    File = "C:/ProgramData"
    File = "C:/Program Files"
  }
  Exclude {
    File = "C:/Users/*/AppData/Local/Temp"
    File = "C:/Windows/Temp"
    File = "*.tmp"
    File = "pagefile.sys"
    File = "hiberfil.sys"
  }
}
```

---

## 6. Schedules and Pools

```conf
# /etc/bareos/bareos-dir.d/schedule/weekly-cycle.conf
Schedule {
  Name = "KubricWeeklyCycle"
  Run = Full 1st sun at 01:00
  Run = Differential 2nd-5th sun at 01:00
  Run = Incremental mon-sat at 21:00
}

# /etc/bareos/bareos-dir.d/pool/full.conf
Pool {
  Name = KubricFull
  Pool Type = Backup
  Recycle = yes
  AutoPrune = yes
  Volume Retention = 365 days
  Maximum Volume Bytes = 50G
  Maximum Volumes = 100
  Label Format = "Full-"
  Storage = S3-MinIO-Storage
}

# /etc/bareos/bareos-dir.d/pool/differential.conf
Pool {
  Name = KubricDifferential
  Pool Type = Backup
  Recycle = yes
  AutoPrune = yes
  Volume Retention = 90 days
  Maximum Volume Bytes = 10G
  Maximum Volumes = 200
  Label Format = "Diff-"
  Storage = S3-MinIO-Storage
}

# /etc/bareos/bareos-dir.d/pool/incremental.conf
Pool {
  Name = KubricIncremental
  Pool Type = Backup
  Recycle = yes
  AutoPrune = yes
  Volume Retention = 30 days
  Maximum Volume Bytes = 5G
  Maximum Volumes = 500
  Label Format = "Inc-"
  Storage = S3-MinIO-Storage
}
```

---

## 7. Go REST API Integration

```go
// internal/noc/bareos.go
package noc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const bareosAPIURL = "http://bareos-dir:9101/api/v2"

// BareosClient communicates with the Bareos REST API.
type BareosClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

func NewBareosClient(token string) *BareosClient {
	return &BareosClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    bareosAPIURL,
		apiToken:   token,
	}
}

// JobStatus represents the status of a backup job.
type JobStatus struct {
	JobID     int       `json:"jobid"`
	Name      string    `json:"name"`
	Client    string    `json:"client"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Status    string    `json:"jobstatus"`
	StartTime time.Time `json:"starttime"`
	EndTime   time.Time `json:"endtime"`
	JobBytes  int64     `json:"jobbytes"`
	JobFiles  int64     `json:"jobfiles"`
	Errors    int       `json:"joberrors"`
}

func (bc *BareosClient) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method,
		fmt.Sprintf("%s%s", bc.baseURL, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bc.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ListJobs returns recent backup jobs.
func (bc *BareosClient) ListJobs(ctx context.Context) ([]JobStatus, error) {
	body, err := bc.doRequest(ctx, "GET", "/jobs?limit=100")
	if err != nil {
		return nil, err
	}

	var result struct {
		Jobs []JobStatus `json:"jobs"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Jobs, nil
}

// GetFailedJobs returns jobs that failed in the last 24 hours.
func (bc *BareosClient) GetFailedJobs(ctx context.Context) ([]JobStatus, error) {
	jobs, err := bc.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	var failed []JobStatus
	for _, j := range jobs {
		if j.Status == "E" || j.Status == "f" { // Error or Fatal
			if j.EndTime.After(cutoff) {
				failed = append(failed, j)
			}
		}
	}
	return failed, nil
}
```

---

## 8. Client Auto-Registration Script

```bash
#!/usr/bin/env bash
# scripts/bareos-client-register.sh
# Registers a new endpoint with the Bareos director

set -euo pipefail

CLIENT_NAME="${1:?Usage: $0 <client-name> <client-ip>}"
CLIENT_IP="${2:?Usage: $0 <client-name> <client-ip>}"
CLIENT_PASSWORD=$(openssl rand -base64 32)

cat > "/etc/bareos/bareos-dir.d/client/${CLIENT_NAME}.conf" <<EOF
Client {
  Name = ${CLIENT_NAME}-fd
  Address = ${CLIENT_IP}
  FD Port = 9102
  Password = "${CLIENT_PASSWORD}"
  Catalog = MyCatalog
  File Retention = 90 days
  Job Retention = 180 days
  AutoPrune = yes

  TLS Enable = yes
  TLS Require = yes
  TLS CA Certificate File = /etc/bareos/tls/ca.pem
}
EOF

cat > "/etc/bareos/bareos-dir.d/job/backup-${CLIENT_NAME}.conf" <<EOF
Job {
  Name = "Backup-${CLIENT_NAME}"
  JobDefs = "KubricDefaultJob"
  Client = ${CLIENT_NAME}-fd
  FileSet = "LinuxServerFiles"
  Schedule = "KubricWeeklyCycle"
}
EOF

# Reload director config
echo "reload" | bconsole

echo "Client ${CLIENT_NAME} registered. FD password: ${CLIENT_PASSWORD}"
```
