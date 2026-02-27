// Package noc provides NOC operations tooling.
// K-NOC-MDM-001 — MicroMDM iOS Device Management: manage iOS devices via MicroMDM REST API.
package noc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// MDMDevice represents an iOS device enrolled in MicroMDM.
type MDMDevice struct {
	UDID         string    `json:"udid"`
	SerialNumber string    `json:"serial_number"`
	DeviceName   string    `json:"device_name"`
	OSVersion    string    `json:"os_version"`
	Model        string    `json:"model"`
	LastSeen     time.Time `json:"last_seen"`
	EnrolledAt   time.Time `json:"enrolled_at"`
	IsSupervised bool      `json:"is_supervised"`
}

// MDMCommand is a raw MDM command payload.
type MDMCommand struct {
	RequestType string         `json:"request_type"`
	Params      map[string]any `json:"params,omitempty"`
}

// MicroMDMClient interacts with the MicroMDM REST API.
type MicroMDMClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewMicroMDMClient reads MICROMDM_URL and MICROMDM_API_KEY from the environment.
func NewMicroMDMClient() *MicroMDMClient {
	baseURL := os.Getenv("MICROMDM_URL")
	if baseURL == "" {
		baseURL = "https://mdm.local"
	}
	return &MicroMDMClient{
		BaseURL:    baseURL,
		APIKey:     os.Getenv("MICROMDM_API_KEY"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MicroMDMClient) basicAuth() string {
	// MicroMDM uses Basic auth with "micromdm" as user and the API key as password.
	creds := base64.StdEncoding.EncodeToString([]byte("micromdm:" + m.APIKey))
	return "Basic " + creds
}

func (m *MicroMDMClient) doJSON(ctx context.Context, method, path string, body interface{}) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, m.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", m.basicAuth())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MicroMDM %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respData, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("MicroMDM %s %s returned %d: %s", method, path, resp.StatusCode, respData)
	}
	return json.RawMessage(respData), nil
}

// ListDevices returns all enrolled iOS devices.
func (m *MicroMDMClient) ListDevices(ctx context.Context) ([]MDMDevice, error) {
	raw, err := m.doJSON(ctx, http.MethodGet, "/v1/devices", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Devices []MDMDevice `json:"devices"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse devices response: %w", parseErr)
	}
	return envelope.Devices, nil
}

// GetDevice retrieves a single device by UDID.
func (m *MicroMDMClient) GetDevice(ctx context.Context, udid string) (*MDMDevice, error) {
	raw, err := m.doJSON(ctx, http.MethodGet, "/v1/devices/"+udid, nil)
	if err != nil {
		return nil, err
	}

	var device MDMDevice
	if parseErr := json.Unmarshal(raw, &device); parseErr != nil {
		return nil, fmt.Errorf("parse device response: %w", parseErr)
	}
	return &device, nil
}

// SendCommand sends an MDM command to a device identified by UDID.
func (m *MicroMDMClient) SendCommand(ctx context.Context, udid string, cmd MDMCommand) error {
	payload := map[string]any{
		"udid":         udid,
		"request_type": cmd.RequestType,
	}
	for k, v := range cmd.Params {
		payload[k] = v
	}
	_, err := m.doJSON(ctx, http.MethodPut, "/v1/commands", payload)
	return err
}

// LockDevice sends a DeviceLock MDM command to the specified device.
func (m *MicroMDMClient) LockDevice(ctx context.Context, udid string, message string) error {
	return m.SendCommand(ctx, udid, MDMCommand{
		RequestType: "DeviceLock",
		Params:      map[string]any{"Message": message},
	})
}

// EraseDevice sends an EraseDevice MDM command to wipe the device.
func (m *MicroMDMClient) EraseDevice(ctx context.Context, udid string) error {
	return m.SendCommand(ctx, udid, MDMCommand{
		RequestType: "EraseDevice",
	})
}

// InstallApp sends an InstallApplication MDM command pointing at a manifest URL.
func (m *MicroMDMClient) InstallApp(ctx context.Context, udid, manifestURL string) error {
	return m.SendCommand(ctx, udid, MDMCommand{
		RequestType: "InstallApplication",
		Params:      map[string]any{"ManifestURL": manifestURL},
	})
}

// SetPasscode sends a ForcePINChange command to require a new passcode.
func (m *MicroMDMClient) SetPasscode(ctx context.Context, udid string) error {
	return m.SendCommand(ctx, udid, MDMCommand{
		RequestType: "RequireDeviceEnrollment",
		Params:      map[string]any{"ForcePINChange": true},
	})
}
