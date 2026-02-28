# K-NOC-MDM-003 -- Android Enterprise Integration

**Role:** Mobile Device Management for Android endpoints using Android Enterprise APIs, integrated with Kubric NOC for policy enforcement, compliance, and remote management.

---

## 1. Architecture

```
┌──────────────┐  Android       ┌──────────────┐  REST API    ┌──────────────┐
│  Android     │  Management   │  Google EMM  │◄────────────►│  Go Service  │
│  Devices     │  API          │  API         │              │  (NOC/MDM)   │
│              │               │              │              │              │
│  Work        │               │  Play Store  │              │  Policy mgmt │
│  Profile     │               │  Management  │              │  Compliance  │
│              │               │  API         │              │  NATS publish│
└──────────────┘               └──────────────┘              └──────────────┘
```

---

## 2. Android Management API Client

```go
// internal/noc/mdm/android.go
package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
	"golang.org/x/oauth2/google"
)

const androidMgmtAPI = "https://androidmanagement.googleapis.com/v1"

// AndroidClient manages Android Enterprise devices.
type AndroidClient struct {
	httpClient  *http.Client
	projectID   string
	enterpriseID string
	nc          *nats.Conn
	tenantID    string
}

func NewAndroidClient(nc *nats.Conn, tenantID, projectID, enterpriseID string) (*AndroidClient, error) {
	ctx := context.Background()

	// Use service account credentials
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/androidmanagement")
	if err != nil {
		return nil, fmt.Errorf("google credentials: %w", err)
	}

	return &AndroidClient{
		httpClient:   creds.Client(ctx),
		projectID:    projectID,
		enterpriseID: enterpriseID,
		nc:           nc,
		tenantID:     tenantID,
	}, nil
}

func (ac *AndroidClient) apiURL(path string) string {
	return fmt.Sprintf("%s/enterprises/%s/%s", androidMgmtAPI, ac.enterpriseID, path)
}

func (ac *AndroidClient) doRequest(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("android api %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ── Data Types ──────────────────────────────────────────────

type Device struct {
	Name              string            `json:"name"`
	UserName          string            `json:"userName"`
	ManagementMode    string            `json:"managementMode"`
	State             string            `json:"state"`
	AppliedState      string            `json:"appliedState"`
	PolicyName        string            `json:"policyName"`
	EnrollmentTime    string            `json:"enrollmentTime"`
	LastStatusReport  string            `json:"lastStatusReportTime"`
	HardwareInfo      HardwareInfo      `json:"hardwareInfo"`
	SoftwareInfo      SoftwareInfo      `json:"softwareInfo"`
	NetworkInfo       NetworkInfo       `json:"networkInfo"`
	NonComplianceDetails []NonCompliance `json:"nonComplianceDetails"`
}

type HardwareInfo struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
	SerialNumber string `json:"serialNumber"`
	DeviceBasebandVersion string `json:"deviceBasebandVersion"`
}

type SoftwareInfo struct {
	AndroidVersion     string `json:"androidVersion"`
	SecurityPatchLevel string `json:"securityPatchLevel"`
	AndroidBuildNumber string `json:"androidBuildNumber"`
	PrimaryLanguageCode string `json:"primaryLanguageCode"`
}

type NetworkInfo struct {
	IMEI          string `json:"imei"`
	MEID          string `json:"meid"`
	WifiMacAddress string `json:"wifiMacAddress"`
	NetworkOperatorName string `json:"networkOperatorName"`
}

type NonCompliance struct {
	SettingName        string `json:"settingName"`
	NonComplianceReason string `json:"nonComplianceReason"`
	FieldPath          string `json:"fieldPath"`
	CurrentValue       string `json:"currentValue"`
	InstallationFailureReason string `json:"installationFailureReason,omitempty"`
}
```

---

## 3. Policy Management

```go
// internal/noc/mdm/android_policy.go
package mdm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Policy defines an Android Enterprise device policy.
type Policy struct {
	Name                    string               `json:"name,omitempty"`
	PasswordPolicies        []PasswordPolicy     `json:"passwordPolicies,omitempty"`
	Applications            []ApplicationPolicy  `json:"applications,omitempty"`
	ComplianceRules         []ComplianceRule     `json:"complianceRules,omitempty"`
	ScreenCaptureDisabled   bool                 `json:"screenCaptureDisabled"`
	CameraDisabled          bool                 `json:"cameraDisabled"`
	EncryptionPolicy        string               `json:"encryptionPolicy"`
	PlayStoreMode           string               `json:"playStoreMode"`
	StatusReportingSettings StatusReporting      `json:"statusReportingSettings"`
	KeyguardDisabledFeatures []string            `json:"keyguardDisabledFeatures,omitempty"`
	MaximumTimeToLock       string               `json:"maximumTimeToLock"`
	AutoDateAndTimeZone     string               `json:"autoDateAndTimeZone"`
}

type PasswordPolicy struct {
	PasswordMinimumLength     int    `json:"passwordMinimumLength"`
	PasswordQuality           string `json:"passwordQuality"`
	PasswordHistoryLength     int    `json:"passwordHistoryLength"`
	MaximumFailedPasswordsForWipe int `json:"maximumFailedPasswordsForWipe"`
	PasswordExpirationTimeout string `json:"passwordExpirationTimeout"`
	RequirePasswordUnlock     string `json:"requirePasswordUnlock"`
}

type ApplicationPolicy struct {
	PackageName    string              `json:"packageName"`
	InstallType    string              `json:"installType"` // FORCE_INSTALLED, AVAILABLE, BLOCKED
	DefaultPermissionPolicy string     `json:"defaultPermissionPolicy"`
	ManagedConfiguration map[string]interface{} `json:"managedConfiguration,omitempty"`
}

type ComplianceRule struct {
	NonComplianceDetailCondition NonComplianceCondition `json:"nonComplianceDetailCondition"`
	APILevelCondition            *APILevelCondition     `json:"apiLevelCondition,omitempty"`
	DisableApps                  bool                   `json:"disableApps"`
}

type NonComplianceCondition struct {
	SettingName         string `json:"settingName,omitempty"`
	NonComplianceReason string `json:"nonComplianceReason,omitempty"`
}

type APILevelCondition struct {
	MinAPILevel int `json:"minApiLevel"`
}

type StatusReporting struct {
	ApplicationReportsEnabled bool `json:"applicationReportsEnabled"`
	DeviceSettingsEnabled     bool `json:"deviceSettingsEnabled"`
	SoftwareInfoEnabled       bool `json:"softwareInfoEnabled"`
	MemoryInfoEnabled         bool `json:"memoryInfoEnabled"`
	NetworkInfoEnabled        bool `json:"networkInfoEnabled"`
	DisplayInfoEnabled        bool `json:"displayInfoEnabled"`
	HardwareStatusEnabled     bool `json:"hardwareStatusEnabled"`
	CommonCriteriaModeEnabled bool `json:"commonCriteriaModeEnabled"`
}

// KubricSecurityPolicy returns the standard Kubric security policy.
func KubricSecurityPolicy() *Policy {
	return &Policy{
		PasswordPolicies: []PasswordPolicy{
			{
				PasswordMinimumLength:         8,
				PasswordQuality:               "COMPLEXITY_LOW",
				PasswordHistoryLength:         5,
				MaximumFailedPasswordsForWipe: 10,
				PasswordExpirationTimeout:     "7776000s", // 90 days
				RequirePasswordUnlock:         "REQUIRE_EVERY_DAY",
			},
		},
		Applications: []ApplicationPolicy{
			{
				PackageName:             "com.kubric.agent",
				InstallType:            "FORCE_INSTALLED",
				DefaultPermissionPolicy: "GRANT",
			},
			{
				PackageName:             "com.microsoft.teams",
				InstallType:            "AVAILABLE",
				DefaultPermissionPolicy: "PROMPT",
			},
		},
		ScreenCaptureDisabled: true,
		EncryptionPolicy:      "ENABLED_WITH_PASSWORD",
		PlayStoreMode:         "WHITELIST",
		MaximumTimeToLock:     "300000000000", // 5 minutes in nanoseconds
		AutoDateAndTimeZone:   "AUTO_DATE_AND_TIME_ZONE_ENFORCED",
		KeyguardDisabledFeatures: []string{
			"TRUST_AGENTS",
			"UNREDACTED_NOTIFICATIONS",
		},
		StatusReportingSettings: StatusReporting{
			ApplicationReportsEnabled: true,
			DeviceSettingsEnabled:     true,
			SoftwareInfoEnabled:       true,
			MemoryInfoEnabled:         true,
			NetworkInfoEnabled:        true,
			HardwareStatusEnabled:     true,
		},
	}
}

// CreatePolicy creates a new policy in the enterprise.
func (ac *AndroidClient) CreatePolicy(ctx context.Context, policyID string, policy *Policy) error {
	body, _ := json.Marshal(policy)
	url := fmt.Sprintf("%s/policies/%s", ac.apiURL(""), policyID)
	_, err := ac.doRequest(ctx, "PATCH", url, bytes.NewReader(body))
	return err
}

// ListDevices returns all managed devices.
func (ac *AndroidClient) ListDevices(ctx context.Context) ([]Device, error) {
	body, err := ac.doRequest(ctx, "GET", ac.apiURL("devices"), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Devices []Device `json:"devices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Devices, nil
}
```

---

## 4. Compliance Checking

```go
// internal/noc/mdm/android_compliance.go
package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CheckCompliance runs compliance checks against all managed devices.
func (ac *AndroidClient) CheckCompliance(ctx context.Context) error {
	devices, err := ac.ListDevices(ctx)
	if err != nil {
		return err
	}

	for _, dev := range devices {
		issues := ac.evaluateCompliance(dev)
		if len(issues) > 0 {
			ac.publishComplianceEvent(dev, issues)
		}
	}
	return nil
}

type ComplianceIssue struct {
	Check       string `json:"check"`
	Status      string `json:"status"` // fail, warn
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

func (ac *AndroidClient) evaluateCompliance(dev Device) []ComplianceIssue {
	var issues []ComplianceIssue

	// Check encryption
	if dev.AppliedState != "ACTIVE" {
		issues = append(issues, ComplianceIssue{
			Check:       "device_state",
			Status:      "fail",
			Description: fmt.Sprintf("Device state: %s", dev.AppliedState),
			Remediation: "Re-enroll device or check management profile",
		})
	}

	// Check security patch level (must be within 90 days)
	if dev.SoftwareInfo.SecurityPatchLevel != "" {
		patchDate, err := time.Parse("2006-01-02", dev.SoftwareInfo.SecurityPatchLevel)
		if err == nil && time.Since(patchDate) > 90*24*time.Hour {
			issues = append(issues, ComplianceIssue{
				Check:       "security_patch",
				Status:      "fail",
				Description: fmt.Sprintf("Security patch level: %s (>90 days old)", dev.SoftwareInfo.SecurityPatchLevel),
				Remediation: "Force OS update via policy",
			})
		}
	}

	// Check non-compliance details from Google's built-in checks
	for _, nc := range dev.NonComplianceDetails {
		issues = append(issues, ComplianceIssue{
			Check:       nc.SettingName,
			Status:      "fail",
			Description: fmt.Sprintf("%s: %s", nc.SettingName, nc.NonComplianceReason),
			Remediation: "Apply corrective policy",
		})
	}

	return issues
}

func (ac *AndroidClient) publishComplianceEvent(dev Device, issues []ComplianceIssue) {
	severity := 3 // Medium
	if len(issues) > 3 {
		severity = 4 // High
	}

	event := map[string]interface{}{
		"class_uid":    6003, // OCSF ComplianceFinding
		"activity_id":  1,
		"category_uid": 6,
		"severity_id":  severity,
		"time":         time.Now().UTC().Format(time.RFC3339),
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("Android compliance: %s %s — %d issues",
				dev.HardwareInfo.Brand, dev.HardwareInfo.Model, len(issues)),
			"uid": fmt.Sprintf("android-compliance-%s", dev.Name),
		},
		"compliance": map[string]interface{}{
			"status":        "non-compliant",
			"status_detail": fmt.Sprintf("%d issues detected", len(issues)),
		},
		"resource": map[string]interface{}{
			"type": "Mobile Device",
			"name": fmt.Sprintf("%s %s", dev.HardwareInfo.Brand, dev.HardwareInfo.Model),
			"uid":  dev.HardwareInfo.SerialNumber,
		},
		"metadata": map[string]interface{}{
			"product":    map[string]string{"name": "Android Enterprise", "vendor_name": "Google"},
			"tenant_uid": ac.tenantID,
		},
		"unmapped": map[string]interface{}{
			"issues":         issues,
			"android_version": dev.SoftwareInfo.AndroidVersion,
			"patch_level":    dev.SoftwareInfo.SecurityPatchLevel,
			"management_mode": dev.ManagementMode,
		},
	}

	data, _ := json.Marshal(event)
	_ = ac.nc.Publish(
		fmt.Sprintf("kubric.noc.mdm.compliance.%s", ac.tenantID), data,
	)
}
```

---

## 5. Remote Actions

```go
// internal/noc/mdm/android_actions.go
package mdm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// LockDevice remotely locks a device.
func (ac *AndroidClient) LockDevice(ctx context.Context, deviceName string) error {
	url := fmt.Sprintf("%s/%s:issueCommand", androidMgmtAPI, deviceName)
	cmd := map[string]interface{}{
		"type": "LOCK",
	}
	body, _ := json.Marshal(cmd)
	_, err := ac.doRequest(ctx, "POST", url, bytes.NewReader(body))
	return err
}

// WipeDevice remotely wipes a device (factory reset).
func (ac *AndroidClient) WipeDevice(ctx context.Context, deviceName string, wipeDataFlags []string) error {
	url := fmt.Sprintf("%s/%s:issueCommand", androidMgmtAPI, deviceName)
	cmd := map[string]interface{}{
		"type":           "RESET_PASSWORD",
		"resetPasswordFlags": wipeDataFlags,
	}
	body, _ := json.Marshal(cmd)
	_, err := ac.doRequest(ctx, "POST", url, bytes.NewReader(body))
	return err
}

// ResetPassword forces a password reset on next unlock.
func (ac *AndroidClient) ResetPassword(ctx context.Context, deviceName, newPassword string) error {
	url := fmt.Sprintf("%s/%s:issueCommand", androidMgmtAPI, deviceName)
	cmd := map[string]interface{}{
		"type":        "RESET_PASSWORD",
		"newPassword": newPassword,
		"resetPasswordFlags": []string{"REQUIRE_ENTRY"},
	}
	body, _ := json.Marshal(cmd)
	_, err := ac.doRequest(ctx, "POST", url, bytes.NewReader(body))
	return err
}
```
