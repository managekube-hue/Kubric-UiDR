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
$workspaceCargo = Get-Content (Join-Path $RepoRoot "Cargo.toml") -Raw
$coresecCargo = Get-Content (Join-Path $RepoRoot "agents\coresec\Cargo.toml") -Raw
$netguardCargo = Get-Content (Join-Path $RepoRoot "agents\netguard\Cargo.toml") -Raw
$perftraceCargo = Get-Content (Join-Path $RepoRoot "agents\perftrace\Cargo.toml") -Raw

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

# ── Rust dependency baseline cleanup checks ─────────────────────────────────
Assert-True -Condition ($workspaceCargo -notmatch "rdkafka") -Message "Workspace does not declare unused rdkafka dependency"
Assert-True -Condition ($workspaceCargo -notmatch "apache-avro") -Message "Workspace does not declare unused apache-avro dependency"
Assert-True -Condition ($workspaceCargo -notmatch "candle-transformers") -Message "Workspace does not declare unused candle-transformers dependency"
Assert-True -Condition ($workspaceCargo -notmatch "once_cell") -Message "Workspace does not declare unused once_cell dependency"
Assert-True -Condition ($workspaceCargo -notmatch "notify-debouncer-mini") -Message "Workspace does not declare unused notify-debouncer-mini dependency"

Assert-True -Condition ($coresecCargo -notmatch "prost") -Message "CoreSec does not declare unused prost/prost-types dependencies"
Assert-True -Condition ($coresecCargo -notmatch "once_cell") -Message "CoreSec does not declare unused once_cell dependency"
Assert-True -Condition ($coresecCargo -notmatch "notify-debouncer-mini") -Message "CoreSec does not declare unused notify-debouncer-mini dependency"

Assert-True -Condition ($netguardCargo -notmatch "prost") -Message "NetGuard does not declare unused prost/prost-types dependencies"
Assert-True -Condition ($netguardCargo -notmatch "once_cell") -Message "NetGuard does not declare unused once_cell dependency"
Assert-True -Condition ($netguardCargo -notmatch "notify-debouncer-mini") -Message "NetGuard does not declare unused notify-debouncer-mini dependency"

Assert-True -Condition ($perftraceCargo -notmatch "prost") -Message "PerfTrace does not declare unused prost/prost-types dependencies"
Assert-True -Condition ($perftraceCargo -notmatch "once_cell") -Message "PerfTrace does not declare unused once_cell dependency"

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

# ── CISO-Assistant GRC integration checks ─────────────────────────────────

$cisoHandler  = Join-Path $RepoRoot "internal\kic\handler_ciso.go"
$cisoBridge   = Join-Path $RepoRoot "services\grc\ciso_bridge.go"
$cisoNatsSub  = Join-Path $RepoRoot "docs\message-bus\subject-mapping\K-MB-SUB-016_grc.ciso.v1.md"
$apiClient    = Get-Content (Join-Path $RepoRoot "frontend\lib\api-client.ts") -Raw
$kicServer    = Get-Content (Join-Path $RepoRoot "internal\kic\server.go") -Raw
$kicStore     = Get-Content (Join-Path $RepoRoot "internal\kic\store_assessment.go") -Raw

Assert-True -Condition (Test-Path $cisoHandler) -Message "CISO-Assistant handler exists (internal/kic/handler_ciso.go)"
Assert-True -Condition (Test-Path $cisoBridge) -Message "GRC CISO bridge service exists (services/grc/ciso_bridge.go)"
Assert-True -Condition (Test-Path $cisoNatsSub) -Message "CISO NATS subject doc exists (K-MB-SUB-016_grc.ciso.v1.md)"

Assert-True -Condition ($kicServer -match "ciso") -Message "KIC server.go wires /ciso routes"
Assert-True -Condition ($kicServer -match "newCISOHandler") -Message "KIC server.go creates CISO handler"
Assert-True -Condition ($kicStore -match "GetFrameworkStats") -Message "Assessment store has GetFrameworkStats method"
Assert-True -Condition ($apiClient -match "askCISO") -Message "Frontend API client exports askCISO function"
Assert-True -Condition ($apiClient -match "getCompliancePosture") -Message "Frontend API client exports getCompliancePosture function"
Assert-True -Condition ($apiClient -match "listComplianceFrameworks") -Message "Frontend API client exports listComplianceFrameworks function"

# ── GRC 200-framework registry checks ─────────────────────────────────────

$fwRegistry = Join-Path $RepoRoot "internal\kic\framework_registry.go"
$fwIndex    = Join-Path $RepoRoot "07_K-GRC-07_COMPLIANCE\K-GRC-FW-000_framework_index.md"
Assert-True -Condition (Test-Path $fwRegistry) -Message "Framework registry exists (internal/kic/framework_registry.go)"
Assert-True -Condition (Test-Path $fwIndex) -Message "Framework index doc exists (K-GRC-FW-000_framework_index.md)"

$fwContent = Get-Content $fwRegistry -Raw
# Count framework entries (lines matching {ID: "...)
$fwEntries = ([regex]::Matches($fwContent, '\{ID:\s+"[^"]+"')).Count
Assert-True -Condition ($fwEntries -ge 200) -Message "Framework registry has $fwEntries frameworks (>= 200)"

# Verify key framework categories are present
Assert-True -Condition ($fwContent -match '"federal"') -Message "Framework registry includes federal category"
Assert-True -Condition ($fwContent -match '"international"') -Message "Framework registry includes international category"
Assert-True -Condition ($fwContent -match '"healthcare"') -Message "Framework registry includes healthcare category"
Assert-True -Condition ($fwContent -match '"financial"') -Message "Framework registry includes financial category"
Assert-True -Condition ($fwContent -match '"cloud"') -Message "Framework registry includes cloud category"
Assert-True -Condition ($fwContent -match '"supply_chain"') -Message "Framework registry includes supply_chain category"
Assert-True -Condition ($fwContent -match '"ai_ml"') -Message "Framework registry includes ai_ml category"
Assert-True -Condition ($fwContent -match '"iot_ics"') -Message "Framework registry includes iot_ics category"
Assert-True -Condition ($fwContent -match '"privacy"') -Message "Framework registry includes privacy category"
Assert-True -Condition ($fwContent -match '"risk"') -Message "Framework registry includes risk category"

# Verify assessment handler uses dynamic registry (not hardcoded 5 frameworks)
$assessHandler = Get-Content (Join-Path $RepoRoot "internal\kic\handler_assessment.go") -Raw
Assert-True -Condition ($assessHandler -match "FrameworkRegistry") -Message "Assessment handler uses dynamic 200-framework registry"

# Verify KAI_RAG_URL is in docker-compose KIC service
$dockerCompose = Get-Content (Join-Path $RepoRoot "docker-compose.yml") -Raw
Assert-True -Condition ($dockerCompose -match "KAI_RAG_URL") -Message "docker-compose.yml wires KAI_RAG_URL to KIC service"

# Verify KIC Docker image exists in Dockerfile.api
$dockerApi = Get-Content (Join-Path $RepoRoot "Dockerfile.api") -Raw
Assert-True -Condition ($dockerApi -match "AS kic") -Message "Dockerfile.api has KIC target image stage"

Write-Host "[batch-06] Audit remediation verification PASSED" -ForegroundColor Green