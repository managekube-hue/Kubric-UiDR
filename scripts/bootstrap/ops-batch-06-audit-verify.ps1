param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

function Assert-True {
    param(
        [Parameter(Mandatory = $true)][bool]$Condition,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if (-not $Condition) {
        throw "[batch-06] FAIL: $Message"
    }

    Write-Host "[batch-06] OK: $Message" -ForegroundColor Green
}

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

Write-Host "[batch-06] Verifying remediation of audit gaps..." -ForegroundColor Cyan

$ebpfExecObj = Join-Path $RepoRoot "vendor\ebpf\execve_hook.o"
$ebpfOpenObj = Join-Path $RepoRoot "vendor\ebpf\openat2_hook.o"
$agentsDocker = Get-Content (Join-Path $RepoRoot "Dockerfile.agents") -Raw
$ebpfRs = Get-Content (Join-Path $RepoRoot "agents\coresec\src\hooks\ebpf.rs") -Raw
$etwRs = Get-Content (Join-Path $RepoRoot "agents\coresec\src\hooks\etw.rs") -Raw
$perfAgent = Get-Content (Join-Path $RepoRoot "agents\perftrace\src\agent.rs") -Raw

$hasEbpfObjects = (Test-Path $ebpfExecObj) -and (Test-Path $ebpfOpenObj)
$dockerBuildsEbpf = $agentsDocker -match [regex]::Escape("make -C vendor/ebpf")
$ebpfAutoBuildCode = $ebpfRs -match [regex]::Escape("ensure_ebpf_object")
Assert-True -Condition ($hasEbpfObjects -or ($dockerBuildsEbpf -and $ebpfAutoBuildCode)) -Message "eBPF object availability is handled (objects present or auto-build path exists)"

Assert-True -Condition ($etwRs -notmatch [regex]::Escape("WMI fallback")) -Message "Windows hook file no longer uses legacy WMI fallback placeholder"
Assert-True -Condition ($etwRs -match [regex]::Escape("blocking_send")) -Message "Windows hook emits process events"

Assert-True -Condition ($perfAgent -notmatch "disk_read_bytes:\s*0") -Message "PerfTrace no longer hardcodes disk_read_bytes to zero"
Assert-True -Condition ($perfAgent -notmatch "disk_write_bytes:\s*0") -Message "PerfTrace no longer hardcodes disk_write_bytes to zero"
Assert-True -Condition ($perfAgent -match [regex]::Escape("collect_disk_io_bytes")) -Message "PerfTrace has disk I/O collector function"

$yaraCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\yara-rules") -Recurse -File -Filter *.yar -ErrorAction SilentlyContinue).Count
$sigmaYmlCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\sigma\rules") -Recurse -File -Filter *.yml -ErrorAction SilentlyContinue).Count
$sigmaYamlCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\sigma\rules") -Recurse -File -Filter *.yaml -ErrorAction SilentlyContinue).Count

Assert-True -Condition ($yaraCount -gt 0) -Message "YARA rules are populated ($yaraCount)"
Assert-True -Condition (($sigmaYmlCount + $sigmaYamlCount) -gt 0) -Message "Sigma rules are populated ($($sigmaYmlCount + $sigmaYamlCount))"

# ── NetGuard enrichment pipeline wiring ──────────────────────────────────────

$captureRs = Get-Content (Join-Path $RepoRoot "agents\netguard\src\capture.rs") -Raw
$dpiRs     = Get-Content (Join-Path $RepoRoot "agents\netguard\src\dpi.rs") -Raw
$idsRs     = Get-Content (Join-Path $RepoRoot "agents\netguard\src\ids.rs") -Raw
$tlsRs     = Get-Content (Join-Path $RepoRoot "agents\netguard\src\tls.rs") -Raw
$ipsumRs   = Get-Content (Join-Path $RepoRoot "agents\netguard\src\ipsum_lookup.rs") -Raw
$ritaRs    = Get-Content (Join-Path $RepoRoot "agents\netguard\src\rita_client.rs") -Raw

# TCP SYN-only filter removed — capture loop must accept all TCP payloads
Assert-True -Condition ($captureRs -notmatch "get_flags\(\) & 0x02 == 0") -Message "NetGuard capture accepts all TCP packets (SYN-only filter removed)"

# All enrichment modules are imported and called in capture.rs
Assert-True -Condition ($captureRs -match "use crate::dpi::DpiEngine") -Message "NetGuard capture.rs imports DpiEngine"
Assert-True -Condition ($captureRs -match "use crate::ids::IdsEngine") -Message "NetGuard capture.rs imports IdsEngine"
Assert-True -Condition ($captureRs -match "use crate::ipsum_lookup::IpsumLookup") -Message "NetGuard capture.rs imports IpsumLookup"
Assert-True -Condition ($captureRs -match "use crate::rita_client::RitaClient") -Message "NetGuard capture.rs imports RitaClient"
Assert-True -Condition ($captureRs -match "use crate::tls") -Message "NetGuard capture.rs imports TLS SNI parser"

# DPI has 15 protocols (AMQP, Redis, Kafka added)
Assert-True -Condition ($dpiRs -match '"AMQP"') -Message "DPI engine detects AMQP protocol"
Assert-True -Condition ($dpiRs -match '"Redis"') -Message "DPI engine detects Redis protocol"
Assert-True -Condition ($dpiRs -match '"Kafka"') -Message "DPI engine detects Kafka protocol"

# DPI classify is called from capture loop
Assert-True -Condition ($captureRs -match "dpi\.classify\(") -Message "DPI classify() is called in capture loop"

# YARA scan is called
Assert-True -Condition ($captureRs -match "ids\.scan\(") -Message "YARA IDS scan() is called in capture loop"

# TLS SNI parse is called
Assert-True -Condition ($captureRs -match "tls::parse_client_hello") -Message "TLS SNI parse_client_hello() is called in capture loop"

# IPsum lookup is called
Assert-True -Condition ($captureRs -match "ipsum\.lookup\(") -Message "IPsum lookup() is called in capture loop"

# RITA poll loop exists
Assert-True -Condition ($captureRs -match "rita_poll_loop") -Message "RITA polling loop is wired in capture.rs"

# IDS module has YARA scanning
Assert-True -Condition ($idsRs -match "yara_x::Scanner") -Message "IDS engine uses yara_x::Scanner"

# TLS module parses ClientHello
Assert-True -Condition ($tlsRs -match "parse_client_hello") -Message "TLS module has parse_client_hello function"

# IPsum module loads tab-separated blocklist
Assert-True -Condition ($ipsumRs -match "split\(") -Message "IPsum loads tab-separated blocklist format"

# RITA client calls HTTP endpoints
Assert-True -Condition ($ritaRs -match "beacons") -Message "RITA client queries beacon endpoint"
Assert-True -Condition ($ritaRs -match "dns/tunneling") -Message "RITA client queries DNS tunneling endpoint"
Assert-True -Condition ($ritaRs -match "long-connections") -Message "RITA client queries long-connections endpoint"

# IPsum seed data exists
$ipsumFile = Join-Path $RepoRoot "vendor\ipsum\ipsum.txt"
Assert-True -Condition (Test-Path $ipsumFile) -Message "IPsum blocklist file exists at vendor/ipsum/ipsum.txt"
$ipsumLines = (Get-Content $ipsumFile | Where-Object { $_ -and -not $_.StartsWith('#') }).Count
Assert-True -Condition ($ipsumLines -gt 0) -Message "IPsum blocklist has $ipsumLines entries"

# Dockerfile.agents copies vendor assets into NetGuard runtime image
Assert-True -Condition ($agentsDocker -match "vendor/yara-rules.*netguard|netguard.*vendor/yara-rules|COPY --from=builder /src/vendor/yara-rules") -Message "Dockerfile.agents copies YARA rules into NetGuard image"
Assert-True -Condition ($agentsDocker -match "vendor/ipsum.*netguard|netguard.*vendor/ipsum|COPY --from=builder /src/vendor/ipsum") -Message "Dockerfile.agents copies IPsum blocklist into NetGuard image"

Write-Host "[batch-06] Audit remediation verification PASSED" -ForegroundColor Green