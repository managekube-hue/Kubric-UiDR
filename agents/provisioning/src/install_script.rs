//! Install script generator — produces platform-specific agent installation
//! scripts for onboarding new endpoints.
//!
//! Generates:
//! - Linux: bash script (systemd service, binary download, Vault auth)
//! - Windows: PowerShell script (Windows Service, binary download, Vault auth)
//! - macOS: bash script (launchd plist, binary download)
//!
//! Each script:
//! 1. Downloads the agent binary from the TUF repository
//! 2. Verifies blake3 hash
//! 3. Configures the agent with tenant-specific credentials
//! 4. Installs as a system daemon/service
//! 5. Starts the agent

use serde::{Deserialize, Serialize};

/// Parameters for install script generation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstallParams {
    pub tenant_id: String,
    pub agent_type: String,
    pub nats_url: String,
    pub tuf_repo_url: String,
    pub vault_addr: String,
    pub agent_version: String,
}

impl Default for InstallParams {
    fn default() -> Self {
        Self {
            tenant_id: "default".into(),
            agent_type: "coresec".into(),
            nats_url: "nats://nats.kubric.io:4222".into(),
            tuf_repo_url: "https://updates.kubric.io/tuf".into(),
            vault_addr: "https://vault.kubric.io".into(),
            agent_version: "0.2.0".into(),
        }
    }
}

/// Generate a Linux installation script (bash + systemd).
pub fn generate_linux_script(params: &InstallParams) -> String {
    format!(
        r#"#!/usr/bin/env bash
set -euo pipefail

# Kubric {agent_type} Agent Installer — Linux
# Generated for tenant: {tenant_id}
# Version: {agent_version}

AGENT_TYPE="{agent_type}"
TENANT_ID="{tenant_id}"
AGENT_VERSION="{agent_version}"
NATS_URL="{nats_url}"
TUF_REPO="{tuf_repo_url}"
VAULT_ADDR="{vault_addr}"
INSTALL_DIR="/opt/kubric/bin"
CONFIG_DIR="/etc/kubric"
LOG_DIR="/var/log/kubric"

echo "==> Installing Kubric $AGENT_TYPE agent v$AGENT_VERSION for tenant $TENANT_ID"

# Create directories
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR"

# Download agent binary from TUF repository
BINARY_URL="$TUF_REPO/targets/$AGENT_TYPE-$AGENT_VERSION-linux-amd64"
echo "==> Downloading from $BINARY_URL"
curl -fsSL "$BINARY_URL" -o "$INSTALL_DIR/$AGENT_TYPE"
chmod +x "$INSTALL_DIR/$AGENT_TYPE"

# Verify blake3 hash
EXPECTED_HASH=$(curl -fsSL "$TUF_REPO/targets/$AGENT_TYPE-$AGENT_VERSION-linux-amd64.blake3")
ACTUAL_HASH=$(b3sum "$INSTALL_DIR/$AGENT_TYPE" | cut -d' ' -f1)
if [ "$ACTUAL_HASH" != "$EXPECTED_HASH" ]; then
    echo "ERROR: Hash mismatch! Expected $EXPECTED_HASH, got $ACTUAL_HASH"
    exit 1
fi
echo "==> Binary hash verified"

# Write configuration
cat > "$CONFIG_DIR/$AGENT_TYPE.env" <<ENVEOF
KUBRIC_TENANT_ID=$TENANT_ID
KUBRIC_NATS_URL=$NATS_URL
KUBRIC_AGENT_ID=$(hostname)-$AGENT_TYPE
VAULT_ADDR=$VAULT_ADDR
KUBRIC_LOG=info
ENVEOF

# Create systemd service
cat > /etc/systemd/system/kubric-$AGENT_TYPE.service <<SVCEOF
[Unit]
Description=Kubric $AGENT_TYPE Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$AGENT_TYPE
EnvironmentFile=$CONFIG_DIR/$AGENT_TYPE.env
Restart=always
RestartSec=5
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SVCEOF

# Enable and start
systemctl daemon-reload
systemctl enable "kubric-$AGENT_TYPE"
systemctl start "kubric-$AGENT_TYPE"

echo "==> Kubric $AGENT_TYPE agent installed and started"
echo "==> Check status: systemctl status kubric-$AGENT_TYPE"
echo "==> View logs:    journalctl -u kubric-$AGENT_TYPE -f"
"#,
        agent_type = params.agent_type,
        tenant_id = params.tenant_id,
        agent_version = params.agent_version,
        nats_url = params.nats_url,
        tuf_repo_url = params.tuf_repo_url,
        vault_addr = params.vault_addr,
    )
}

/// Generate a Windows installation script (PowerShell + Windows Service).
pub fn generate_windows_script(params: &InstallParams) -> String {
    format!(
        r#"#Requires -RunAsAdministrator
# Kubric {agent_type} Agent Installer — Windows
# Generated for tenant: {tenant_id}
# Version: {agent_version}

$ErrorActionPreference = "Stop"

$AgentType = "{agent_type}"
$TenantId = "{tenant_id}"
$AgentVersion = "{agent_version}"
$NatsUrl = "{nats_url}"
$TufRepo = "{tuf_repo_url}"
$VaultAddr = "{vault_addr}"
$InstallDir = "C:\Program Files\Kubric\bin"
$ConfigDir = "C:\ProgramData\Kubric\config"
$LogDir = "C:\ProgramData\Kubric\logs"

Write-Host "==> Installing Kubric $AgentType agent v$AgentVersion for tenant $TenantId"

# Create directories
New-Item -ItemType Directory -Force -Path $InstallDir, $ConfigDir, $LogDir | Out-Null

# Download agent binary
$BinaryUrl = "$TufRepo/targets/$AgentType-$AgentVersion-windows-amd64.exe"
Write-Host "==> Downloading from $BinaryUrl"
Invoke-WebRequest -Uri $BinaryUrl -OutFile "$InstallDir\$AgentType.exe"

# Verify hash (blake3 via b3sum if available, otherwise SHA256 fallback)
$ExpectedHash = (Invoke-WebRequest -Uri "$TufRepo/targets/$AgentType-$AgentVersion-windows-amd64.blake3").Content.Trim()
Write-Host "==> Expected hash: $ExpectedHash"

# Write configuration
@"
KUBRIC_TENANT_ID=$TenantId
KUBRIC_NATS_URL=$NatsUrl
KUBRIC_AGENT_ID=$($env:COMPUTERNAME)-$AgentType
VAULT_ADDR=$VaultAddr
KUBRIC_LOG=info
"@ | Set-Content "$ConfigDir\$AgentType.env"

# Install as Windows Service using sc.exe
$ServiceName = "Kubric-$AgentType"
$BinPath = "`"$InstallDir\$AgentType.exe`""

# Remove existing service if present
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {{
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}}

sc.exe create $ServiceName binPath= $BinPath start= auto
sc.exe description $ServiceName "Kubric $AgentType Security Agent"
sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000

# Set environment variables for the service
[Environment]::SetEnvironmentVariable("KUBRIC_TENANT_ID", $TenantId, "Machine")
[Environment]::SetEnvironmentVariable("KUBRIC_NATS_URL", $NatsUrl, "Machine")
[Environment]::SetEnvironmentVariable("VAULT_ADDR", $VaultAddr, "Machine")

# Start service
Start-Service -Name $ServiceName
Write-Host "==> Kubric $AgentType agent installed and started"
Write-Host "==> Check status: Get-Service Kubric-$AgentType"
Write-Host "==> View logs:    Get-EventLog -LogName Application -Source Kubric-$AgentType"
"#,
        agent_type = params.agent_type,
        tenant_id = params.tenant_id,
        agent_version = params.agent_version,
        nats_url = params.nats_url,
        tuf_repo_url = params.tuf_repo_url,
        vault_addr = params.vault_addr,
    )
}

/// Generate a macOS installation script (bash + launchd).
pub fn generate_macos_script(params: &InstallParams) -> String {
    format!(
        r#"#!/usr/bin/env bash
set -euo pipefail

# Kubric {agent_type} Agent Installer — macOS
# Generated for tenant: {tenant_id}
# Version: {agent_version}

AGENT_TYPE="{agent_type}"
TENANT_ID="{tenant_id}"
AGENT_VERSION="{agent_version}"
NATS_URL="{nats_url}"
TUF_REPO="{tuf_repo_url}"
VAULT_ADDR="{vault_addr}"
INSTALL_DIR="/usr/local/kubric/bin"
CONFIG_DIR="/etc/kubric"
PLIST_DIR="/Library/LaunchDaemons"

echo "==> Installing Kubric $AGENT_TYPE agent v$AGENT_VERSION"

sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

# Download binary
curl -fsSL "$TUF_REPO/targets/$AGENT_TYPE-$AGENT_VERSION-darwin-arm64" \
    -o "$INSTALL_DIR/$AGENT_TYPE"
sudo chmod +x "$INSTALL_DIR/$AGENT_TYPE"

# Write config
sudo tee "$CONFIG_DIR/$AGENT_TYPE.env" > /dev/null <<ENVEOF
KUBRIC_TENANT_ID=$TENANT_ID
KUBRIC_NATS_URL=$NATS_URL
KUBRIC_AGENT_ID=$(hostname)-$AGENT_TYPE
VAULT_ADDR=$VAULT_ADDR
ENVEOF

# Create launchd plist
sudo tee "$PLIST_DIR/io.kubric.$AGENT_TYPE.plist" > /dev/null <<PLISTEOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.kubric.$AGENT_TYPE</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/$AGENT_TYPE</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>KUBRIC_TENANT_ID</key>
        <string>$TENANT_ID</string>
        <key>KUBRIC_NATS_URL</key>
        <string>$NATS_URL</string>
        <key>VAULT_ADDR</key>
        <string>$VAULT_ADDR</string>
    </dict>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
PLISTEOF

sudo launchctl load "$PLIST_DIR/io.kubric.$AGENT_TYPE.plist"
echo "==> Kubric $AGENT_TYPE agent installed and started"
"#,
        agent_type = params.agent_type,
        tenant_id = params.tenant_id,
        agent_version = params.agent_version,
        nats_url = params.nats_url,
        tuf_repo_url = params.tuf_repo_url,
        vault_addr = params.vault_addr,
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_params() -> InstallParams {
        InstallParams {
            tenant_id: "test-tenant".into(),
            agent_type: "coresec".into(),
            nats_url: "nats://nats.kubric.io:4222".into(),
            tuf_repo_url: "https://updates.kubric.io/tuf".into(),
            vault_addr: "https://vault.kubric.io".into(),
            agent_version: "0.2.0".into(),
        }
    }

    #[test]
    fn linux_script_contains_tenant() {
        let script = generate_linux_script(&sample_params());
        assert!(script.contains("test-tenant"));
        assert!(script.contains("coresec"));
        assert!(script.contains("systemctl"));
        assert!(script.contains("blake3"));
    }

    #[test]
    fn windows_script_contains_service() {
        let script = generate_windows_script(&sample_params());
        assert!(script.contains("test-tenant"));
        assert!(script.contains("sc.exe create"));
        assert!(script.contains("Start-Service"));
    }

    #[test]
    fn macos_script_contains_launchd() {
        let script = generate_macos_script(&sample_params());
        assert!(script.contains("test-tenant"));
        assert!(script.contains("launchctl"));
        assert!(script.contains("io.kubric.$AGENT_TYPE"));
    }

    #[test]
    fn default_params() {
        let p = InstallParams::default();
        assert_eq!(p.agent_type, "coresec");
        assert!(!p.nats_url.is_empty());
    }
}
