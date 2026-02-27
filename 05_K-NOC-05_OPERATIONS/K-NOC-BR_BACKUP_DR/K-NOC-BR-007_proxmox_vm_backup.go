// Package noc provides NOC operations tooling.
// K-NOC-BR-007 — Proxmox VM Backup: trigger and monitor VM backups via Proxmox REST API.
package noc

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ProxmoxVM represents a QEMU virtual machine on a Proxmox node.
type ProxmoxVM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Memory int64  `json:"maxmem"`
	CPU    int    `json:"cpus"`
	Uptime int64  `json:"uptime"`
}

// ProxmoxBackupInfo represents a backup object stored in a Proxmox storage pool.
type ProxmoxBackupInfo struct {
	VolID  string `json:"volid"`
	VMID   int    `json:"vmid"`
	Size   int64  `json:"size"`
	CT     string `json:"content"`
	Format string `json:"format"`
	Notes  string `json:"notes"`
}

// ProxmoxBackup manages Proxmox VM backups via the Proxmox REST API.
type ProxmoxBackup struct {
	BaseURL    string
	TokenID    string
	Token      string
	HTTPClient *http.Client
}

// NewProxmoxBackup reads PROXMOX_URL, PROXMOX_TOKEN_ID, and PROXMOX_TOKEN from the environment.
func NewProxmoxBackup() *ProxmoxBackup {
	baseURL := os.Getenv("PROXMOX_URL")
	if baseURL == "" {
		baseURL = "https://proxmox.local:8006/api2/json"
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed certs common in Proxmox
	}
	return &ProxmoxBackup{
		BaseURL: baseURL,
		TokenID: os.Getenv("PROXMOX_TOKEN_ID"),
		Token:   os.Getenv("PROXMOX_TOKEN"),
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (pb *ProxmoxBackup) authHeader() string {
	return fmt.Sprintf("PVEAPIToken=%s=%s", pb.TokenID, pb.Token)
}

func (pb *ProxmoxBackup) doRequest(ctx context.Context, method, path string, body url.Values) (json.RawMessage, error) {
	fullURL := pb.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewBufferString(body.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", pb.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := pb.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Proxmox %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, fmt.Errorf("read response body: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Proxmox %s %s returned %d: %s", method, path, resp.StatusCode, respBody)
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if jsonErr := json.Unmarshal(respBody, &envelope); jsonErr != nil {
		return nil, fmt.Errorf("decode Proxmox response: %w", jsonErr)
	}
	return envelope.Data, nil
}

// ListVMs returns all QEMU VMs on the given node.
func (pb *ProxmoxBackup) ListVMs(ctx context.Context, node string) ([]ProxmoxVM, error) {
	data, err := pb.doRequest(ctx, http.MethodGet, fmt.Sprintf("/nodes/%s/qemu", node), nil)
	if err != nil {
		return nil, err
	}
	var vms []ProxmoxVM
	if parseErr := json.Unmarshal(data, &vms); parseErr != nil {
		return nil, fmt.Errorf("parse VM list: %w", parseErr)
	}
	return vms, nil
}

// CreateBackup starts a backup task for vmid on the given node and storage pool.
// It returns the UPID (Universal Process ID) of the background task.
func (pb *ProxmoxBackup) CreateBackup(ctx context.Context, node string, vmid int, storage string) (string, error) {
	params := url.Values{}
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	params.Set("storage", storage)
	params.Set("mode", "snapshot")
	params.Set("compress", "zstd")

	data, err := pb.doRequest(ctx, http.MethodPost, fmt.Sprintf("/nodes/%s/vzdump", node), params)
	if err != nil {
		return "", err
	}

	var upid string
	if parseErr := json.Unmarshal(data, &upid); parseErr != nil {
		return "", fmt.Errorf("parse backup UPID: %w", parseErr)
	}
	return upid, nil
}

// GetTaskStatus returns the status string of a Proxmox task identified by its UPID.
func (pb *ProxmoxBackup) GetTaskStatus(ctx context.Context, node, upid string) (string, error) {
	data, err := pb.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/nodes/%s/tasks/%s/status", node, url.PathEscape(upid)), nil)
	if err != nil {
		return "", err
	}

	var status struct {
		Status string `json:"status"`
	}
	if parseErr := json.Unmarshal(data, &status); parseErr != nil {
		return "", fmt.Errorf("parse task status: %w", parseErr)
	}
	return status.Status, nil
}

// WaitForTask polls until the Proxmox task completes or the timeout elapses.
func (pb *ProxmoxBackup) WaitForTask(ctx context.Context, node, upid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		status, err := pb.GetTaskStatus(ctx, node, upid)
		if err != nil {
			return err
		}
		if status == "stopped" {
			return nil
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout waiting for task %q after %s", upid, timeout)
}

// ListBackups lists backup content in the given storage pool.
func (pb *ProxmoxBackup) ListBackups(ctx context.Context, storage string) ([]ProxmoxBackupInfo, error) {
	data, err := pb.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/storage/%s/content?content=backup", url.PathEscape(storage)), nil)
	if err != nil {
		return nil, err
	}
	var infos []ProxmoxBackupInfo
	if parseErr := json.Unmarshal(data, &infos); parseErr != nil {
		return nil, fmt.Errorf("parse backup list: %w", parseErr)
	}
	return infos, nil
}
