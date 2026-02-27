// Package noc provides NOC operations tooling.
// K-NOC-MDM-002 — Headwind MDM Android Management: manage Android devices via Headwind MDM REST API.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AndroidDevice represents an Android device enrolled in Headwind MDM.
type AndroidDevice struct {
	DeviceNumber string    `json:"deviceNumber"`
	Imei         string    `json:"imei"`
	SerialNumber string    `json:"serialNumber"`
	Description  string    `json:"description"`
	OsVersion    string    `json:"osVersion"`
	LastUpdate   time.Time `json:"lastUpdate"`
	Enrolled     bool      `json:"enrolled"`
}

// HeadwindClient manages Android devices via the Headwind MDM REST API.
type HeadwindClient struct {
	BaseURL    string
	Login      string
	Password   string
	token      string
	HTTPClient *http.Client
}

// NewHeadwindClient reads HEADWIND_URL, HEADWIND_LOGIN, and HEADWIND_PASSWORD from the environment.
func NewHeadwindClient() *HeadwindClient {
	baseURL := os.Getenv("HEADWIND_URL")
	if baseURL == "" {
		baseURL = "http://mdm.local"
	}
	return &HeadwindClient{
		BaseURL:    baseURL,
		Login:      os.Getenv("HEADWIND_LOGIN"),
		Password:   os.Getenv("HEADWIND_PASSWORD"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate logs in to the Headwind MDM API and stores the session token.
func (h *HeadwindClient) Authenticate(ctx context.Context) error {
	payload := map[string]string{
		"login":    h.Login,
		"password": h.Password,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal auth payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.BaseURL+"/rest/public/auth/signin", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("headwind auth: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("headwind auth returned %d: %s", resp.StatusCode, respData)
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if parseErr := json.Unmarshal(respData, &result); parseErr != nil {
		return fmt.Errorf("parse auth response: %w", parseErr)
	}
	if result.Data.Token == "" {
		return fmt.Errorf("headwind auth: empty token in response")
	}
	h.token = result.Data.Token
	return nil
}

func (h *HeadwindClient) doJSON(ctx context.Context, method, path string, body interface{}) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, h.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Headwind %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respData, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}

	// Re-authenticate on 401 and retry once.
	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := h.Authenticate(ctx); authErr != nil {
			return nil, fmt.Errorf("re-auth after 401: %w", authErr)
		}
		return h.doJSON(ctx, method, path, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Headwind %s %s returned %d: %s", method, path, resp.StatusCode, respData)
	}
	return json.RawMessage(respData), nil
}

// ListDevices returns all enrolled Android devices.
func (h *HeadwindClient) ListDevices(ctx context.Context) ([]AndroidDevice, error) {
	raw, err := h.doJSON(ctx, http.MethodGet, "/rest/public/device/all", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data []AndroidDevice `json:"data"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse device list: %w", parseErr)
	}
	return envelope.Data, nil
}

// GetDevice retrieves a single device by its device number.
func (h *HeadwindClient) GetDevice(ctx context.Context, deviceNumber string) (*AndroidDevice, error) {
	raw, err := h.doJSON(ctx, http.MethodGet, "/rest/public/device/"+deviceNumber, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data AndroidDevice `json:"data"`
	}
	if parseErr := json.Unmarshal(raw, &envelope); parseErr != nil {
		return nil, fmt.Errorf("parse device response: %w", parseErr)
	}
	return &envelope.Data, nil
}

// LockDevice sends a device lock command.
func (h *HeadwindClient) LockDevice(ctx context.Context, deviceNumber string) error {
	payload := map[string]string{"deviceNumber": deviceNumber}
	_, err := h.doJSON(ctx, http.MethodPost, "/rest/public/device/lock", payload)
	return err
}

// WipeDevice sends a remote wipe command.
func (h *HeadwindClient) WipeDevice(ctx context.Context, deviceNumber string) error {
	payload := map[string]string{"deviceNumber": deviceNumber}
	_, err := h.doJSON(ctx, http.MethodPost, "/rest/public/device/wipe", payload)
	return err
}

// PushConfig pushes an arbitrary configuration key-value map to the device.
func (h *HeadwindClient) PushConfig(ctx context.Context, deviceNumber string, config map[string]any) error {
	payload := map[string]any{
		"deviceNumber": deviceNumber,
		"config":       config,
	}
	_, err := h.doJSON(ctx, http.MethodPost, "/rest/public/device/config", payload)
	return err
}

// InstallApp triggers the installation of an application package on the device.
func (h *HeadwindClient) InstallApp(ctx context.Context, deviceNumber, pkg string) error {
	payload := map[string]any{
		"deviceNumber": deviceNumber,
		"pkg":          pkg,
	}
	_, err := h.doJSON(ctx, http.MethodPost, "/rest/public/device/install-applications", payload)
	return err
}
