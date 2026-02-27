// Package noc provides NOC operations tooling.
// K-NOC-INV-001 — Osquery Go SDK Client: connect to osquery via exec and run queries.
// Uses exec.CommandContext to invoke osqueryi since osquery-go is not in go.mod.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// PlatformInfo captures operating system version details.
type PlatformInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Major   int    `json:"major"`
	Minor   int    `json:"minor"`
	Arch    string `json:"arch"`
}

// Package represents an installed software package.
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	InstallTime string `json:"install_time"`
}

// ListeningPort represents a network port being listened on.
type ListeningPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}

// Process represents a running system process.
type Process struct {
	PID        int    `json:"pid"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	State      string `json:"state"`
	OnDisk     int    `json:"on_disk"`
	Resident   int64  `json:"resident_size"`
	TotalSize  int64  `json:"total_size"`
}

// OsqueryClient runs osquery queries against the local osquery daemon.
type OsqueryClient struct {
	socketPath string
}

// NewOsqueryClient reads OSQUERY_SOCKET from the environment.
func NewOsqueryClient() *OsqueryClient {
	sock := os.Getenv("OSQUERY_SOCKET")
	if sock == "" {
		sock = "/var/osquery/osquery.em"
	}
	return &OsqueryClient{socketPath: sock}
}

// Query executes a SQL query via osqueryi and returns the parsed rows.
func (c *OsqueryClient) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "osqueryi", "--json", sql)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osqueryi: %w — stderr: %s", err, stderr.String())
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("parse osqueryi JSON: %w", err)
	}
	return raw, nil
}

// queryStrings is a convenience wrapper that returns rows as map[string]string.
func (c *OsqueryClient) queryStrings(ctx context.Context, sql string) ([]map[string]string, error) {
	rawRows, err := c.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(rawRows))
	for _, r := range rawRows {
		row := make(map[string]string, len(r))
		for k, v := range r {
			row[k] = fmt.Sprintf("%v", v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// GetPlatformInfo returns OS version information from the local osquery daemon.
func (c *OsqueryClient) GetPlatformInfo(ctx context.Context) (*PlatformInfo, error) {
	rows, err := c.queryStrings(ctx, "SELECT name, version, build, major, minor, arch FROM os_version LIMIT 1;")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("os_version returned no rows")
	}
	r := rows[0]
	major, _ := strconv.Atoi(r["major"])
	minor, _ := strconv.Atoi(r["minor"])
	return &PlatformInfo{
		Name:    r["name"],
		Version: r["version"],
		Build:   r["build"],
		Major:   major,
		Minor:   minor,
		Arch:    r["arch"],
	}, nil
}

// GetInstalledPackages returns all installed packages detected by osquery.
func (c *OsqueryClient) GetInstalledPackages(ctx context.Context) ([]Package, error) {
	rows, err := c.queryStrings(ctx,
		"SELECT name, version, source, install_time FROM packages;")
	if err != nil {
		// Try Windows-style table as fallback.
		var err2 error
		rows, err2 = c.queryStrings(ctx, "SELECT name, version, '' AS source, '' AS install_time FROM programs;")
		if err2 != nil {
			return nil, fmt.Errorf("packages query: %w; programs query: %w", err, err2)
		}
	}
	pkgs := make([]Package, 0, len(rows))
	for _, r := range rows {
		pkgs = append(pkgs, Package{
			Name:        r["name"],
			Version:     r["version"],
			Source:      r["source"],
			InstallTime: r["install_time"],
		})
	}
	return pkgs, nil
}

// GetListeningPorts returns network ports currently being listened on with process details.
func (c *OsqueryClient) GetListeningPorts(ctx context.Context) ([]ListeningPort, error) {
	rows, err := c.queryStrings(ctx, `
		SELECT lp.port, lp.protocol, lp.pid, p.name, p.path
		FROM listening_ports lp
		LEFT JOIN processes p ON lp.pid = p.pid;`)
	if err != nil {
		return nil, err
	}
	ports := make([]ListeningPort, 0, len(rows))
	for _, r := range rows {
		port, _ := strconv.Atoi(r["port"])
		pid, _ := strconv.Atoi(r["pid"])
		proto, _ := strconv.Atoi(r["protocol"])
		protoStr := "tcp"
		if proto == 17 {
			protoStr = "udp"
		}
		ports = append(ports, ListeningPort{
			Port:     port,
			Protocol: protoStr,
			PID:      pid,
			Name:     r["name"],
			Path:     r["path"],
		})
	}
	return ports, nil
}

// GetRunningProcesses returns all currently running processes.
func (c *OsqueryClient) GetRunningProcesses(ctx context.Context) ([]Process, error) {
	rows, err := c.queryStrings(ctx, `
		SELECT pid, name, path, state, on_disk, resident_size, total_size
		FROM processes;`)
	if err != nil {
		return nil, err
	}
	procs := make([]Process, 0, len(rows))
	for _, r := range rows {
		pid, _ := strconv.Atoi(r["pid"])
		onDisk, _ := strconv.Atoi(r["on_disk"])
		resident, _ := strconv.ParseInt(r["resident_size"], 10, 64)
		total, _ := strconv.ParseInt(r["total_size"], 10, 64)
		procs = append(procs, Process{
			PID:       pid,
			Name:      r["name"],
			Path:      r["path"],
			State:     r["state"],
			OnDisk:    onDisk,
			Resident:  resident,
			TotalSize: total,
		})
	}
	return procs, nil
}
