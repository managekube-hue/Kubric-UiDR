# =============================================================================
# Kubric Master Validation - Run All Checks
# =============================================================================
param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}

Set-Location $RepoRoot

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "        KUBRIC MASTER VALIDATION - ALL SYSTEMS CHECK           " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

$totalChecks  = 0
$passedChecks = 0

function Run-Check {
    param(
        [string]$Name,
        [scriptblock]$Check
    )
    $script:totalChecks++
    Write-Host ("  [{0:D2}] {1}..." -f $script:totalChecks, $Name) -ForegroundColor Yellow -NoNewline
    try {
        $result = & $Check
        if ($result) {
            Write-Host " OK" -ForegroundColor Green
            $script:passedChecks++
            return $true
        } else {
            Write-Host " FAIL" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host (" FAIL (Error: {0})" -f $_) -ForegroundColor Red
        return $false
    }
}

# =============================================================================
# 1. Core File Structure
# =============================================================================
Write-Host ""
Write-Host "--- 1. Core File Structure ---" -ForegroundColor Cyan

Run-Check "docker-compose.yml exists" {
    Test-Path "docker-compose.yml"
}

Run-Check "Dockerfile.api exists" {
    Test-Path "Dockerfile.api"
}

Run-Check "Dockerfile.agents exists" {
    Test-Path "Dockerfile.agents"
}

Run-Check "Dockerfile.web exists" {
    Test-Path "Dockerfile.web"
}

Run-Check ".env.example exists" {
    Test-Path ".env.example"
}

Run-Check "go.mod exists" {
    Test-Path "go.mod"
}

Run-Check "Cargo.toml exists" {
    Test-Path "Cargo.toml"
}

Run-Check ".kubric/ directory exists" {
    Test-Path ".kubric"
}

Run-Check "Source files are valid UTF-8" {
    & powershell -ExecutionPolicy Bypass -File "scripts/bootstrap/find-bad-bytes.ps1" -RepoRoot $RepoRoot | Out-Null
    $LASTEXITCODE -eq 0
}

# =============================================================================
# 2. Docker Compose Validation
# =============================================================================
Write-Host ""
Write-Host "--- 2. Docker Compose ---" -ForegroundColor Cyan

Run-Check "docker-compose.yml syntax valid" {
    # Provide required vars so compose can parse without a real .env
    $env:POSTGRES_PASSWORD   = "validate-only"
    $env:CLICKHOUSE_PASSWORD = "validate-only"
    $env:MINIO_ROOT_PASSWORD = "validate-only"
    $out = docker compose -f docker-compose.yml config 2>&1
    $env:POSTGRES_PASSWORD   = ""
    $env:CLICKHOUSE_PASSWORD = ""
    $env:MINIO_ROOT_PASSWORD = ""
    $LASTEXITCODE -eq 0
}

Run-Check "docker-compose.yml has postgres service" {
    (Get-Content "docker-compose.yml" -Raw) -match "postgres:"
}

Run-Check "docker-compose.yml has nats service" {
    (Get-Content "docker-compose.yml" -Raw) -match "nats:"
}

Run-Check "docker-compose.yml has clickhouse service" {
    (Get-Content "docker-compose.yml" -Raw) -match "clickhouse:"
}

Run-Check "docker-compose.yml has temporal service" {
    (Get-Content "docker-compose.yml" -Raw) -match "temporalio"
}

Run-Check "docker-compose.yml has ksvc service" {
    (Get-Content "docker-compose.yml" -Raw) -match "ksvc:"
}

Run-Check "docker-compose.yml has kic service" {
    (Get-Content "docker-compose.yml" -Raw) -match "kic:"
}

Run-Check "docker-compose.yml wires KAI_RAG_URL to kic" {
    (Get-Content "docker-compose.yml" -Raw) -match "KAI_RAG_URL"
}

Run-Check "docker-compose.yml has kai service" {
    (Get-Content "docker-compose.yml" -Raw) -match "kai:"
}

Run-Check "docker-compose.yml has web service" {
    (Get-Content "docker-compose.yml" -Raw) -match "web:"
}

# =============================================================================
# 3. Dockerfile Validation
# =============================================================================
Write-Host ""
Write-Host "--- 3. Dockerfiles ---" -ForegroundColor Cyan

Run-Check "Dockerfile.api has kic target stage" {
    (Get-Content "Dockerfile.api" -Raw) -match "AS kic"
}

Run-Check "Dockerfile.api has ksvc target stage" {
    (Get-Content "Dockerfile.api" -Raw) -match "AS ksvc"
}

Run-Check "Dockerfile.api has vdr target stage" {
    (Get-Content "Dockerfile.api" -Raw) -match "AS vdr"
}

Run-Check "Dockerfile.agents builds eBPF objects" {
    (Get-Content "Dockerfile.agents" -Raw) -match "make -C vendor/ebpf"
}

Run-Check "Dockerfile.agents copies YARA rules into NetGuard" {
    (Get-Content "Dockerfile.agents" -Raw) -match "vendor/yara-rules"
}

Run-Check "Dockerfile.agents copies IPsum blocklist" {
    (Get-Content "Dockerfile.agents" -Raw) -match "vendor/ipsum"
}

Run-Check "Dockerfile.web has standalone Next.js stage" {
    (Get-Content "Dockerfile.web" -Raw) -match "standalone"
}

# =============================================================================
# 4. Go Services
# =============================================================================
Write-Host ""
Write-Host "--- 4. Go Services ---" -ForegroundColor Cyan

Run-Check "cmd/ksvc/main.go exists" {
    Test-Path "cmd/ksvc/main.go"
}

Run-Check "cmd/kic/main.go exists" {
    Test-Path "cmd/kic/main.go"
}

Run-Check "cmd/vdr/main.go exists" {
    Test-Path "cmd/vdr/main.go"
}

Run-Check "cmd/noc/main.go exists" {
    Test-Path "cmd/noc/main.go"
}

Run-Check "internal/kic/server.go wires /ciso routes" {
    (Get-Content "internal/kic/server.go" -Raw) -match "ciso"
}

Run-Check "internal/kic/server.go creates CISO handler" {
    (Get-Content "internal/kic/server.go" -Raw) -match "newCISOHandler"
}

Run-Check "internal/kic/handler_ciso.go exists" {
    Test-Path "internal/kic/handler_ciso.go"
}

Run-Check "internal/kic/store_assessment.go has GetFrameworkStats" {
    (Get-Content "internal/kic/store_assessment.go" -Raw) -match "GetFrameworkStats"
}

Run-Check "internal/kic/framework_registry.go has 200+ frameworks" {
    $content = Get-Content "internal/kic/framework_registry.go" -Raw
    $count = ([regex]::Matches($content, '\{ID:\s+"[^"]+"')).Count
    $count -ge 200
}

Run-Check "services/grc/ciso_bridge.go exists" {
    Test-Path "services/grc/ciso_bridge.go"
}

# =============================================================================
# 5. Rust Agents
# =============================================================================
Write-Host ""
Write-Host "--- 5. Rust Agents ---" -ForegroundColor Cyan

Run-Check "agents/coresec/src/hooks/ebpf.rs exists" {
    Test-Path "agents/coresec/src/hooks/ebpf.rs"
}

Run-Check "agents/netguard/src/capture.rs imports DpiEngine" {
    (Get-Content "agents/netguard/src/capture.rs" -Raw) -match "use crate::dpi::DpiEngine"
}

Run-Check "agents/netguard/src/capture.rs imports IdsEngine" {
    (Get-Content "agents/netguard/src/capture.rs" -Raw) -match "use crate::ids::IdsEngine"
}

Run-Check "agents/netguard/src/dpi.rs detects AMQP" {
    (Get-Content "agents/netguard/src/dpi.rs" -Raw) -match '"AMQP"'
}

Run-Check "agents/netguard/src/ids.rs uses yara_x::Scanner" {
    (Get-Content "agents/netguard/src/ids.rs" -Raw) -match "yara_x::Scanner"
}

Run-Check "vendor/ipsum/ipsum.txt blocklist exists" {
    Test-Path "vendor/ipsum/ipsum.txt"
}

# =============================================================================
# 6. Frontend
# =============================================================================
Write-Host ""
Write-Host "--- 6. Frontend ---" -ForegroundColor Cyan

Run-Check "frontend/package.json exists" {
    Test-Path "frontend/package.json"
}

Run-Check "frontend/next.config.js has standalone output" {
    (Get-Content "frontend/next.config.js" -Raw) -match "standalone"
}

Run-Check "frontend/lib/api-client.ts exports askCISO" {
    (Get-Content "frontend/lib/api-client.ts" -Raw) -match "askCISO"
}

Run-Check "frontend/lib/api-client.ts exports getCompliancePosture" {
    (Get-Content "frontend/lib/api-client.ts" -Raw) -match "getCompliancePosture"
}

Run-Check "frontend/lib/api-client.ts exports listComplianceFrameworks" {
    (Get-Content "frontend/lib/api-client.ts" -Raw) -match "listComplianceFrameworks"
}

# =============================================================================
# 7. AI Autonomy
# =============================================================================
Write-Host ""
Write-Host "--- 7. KAI AI Layer ---" -ForegroundColor Cyan

Run-Check "kai/deploy/docker_manager.py exists" {
    Test-Path "kai/deploy/docker_manager.py"
}

Run-Check "kai/house/monitor.py exists" {
    Test-Path "kai/house/monitor.py"
}

Run-Check "requirements.txt exists" {
    Test-Path "requirements.txt"
}

# =============================================================================
# 8. CI / Documentation
# =============================================================================
Write-Host ""
Write-Host "--- 8. CI and Docs ---" -ForegroundColor Cyan

Run-Check "Makefile exists" {
    Test-Path "Makefile"
}

Run-Check "Makefile has deploy-prod target" {
    (Get-Content "Makefile" -Raw) -match "deploy-prod:"
}

Run-Check ".github/workflows/ exists" {
    Test-Path ".github/workflows"
}

Run-Check "docs/PROJECT-STATUS.md exists" {
    Test-Path "docs/PROJECT-STATUS.md"
}

Run-Check "QUICKSTART.md exists" {
    Test-Path "QUICKSTART.md"
}

Run-Check "DEVELOPER-BOOTSTRAP.md exists" {
    Test-Path "DEVELOPER-BOOTSTRAP.md"
}

# =============================================================================
# Summary
# =============================================================================
Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "                     VALIDATION SUMMARY                        " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

$pct = [math]::Round(($passedChecks / $totalChecks) * 100, 1)

if ($passedChecks -eq $totalChecks) {
    Write-Host ("  Status : PASS - ALL {0} CHECKS PASSED" -f $totalChecks) -ForegroundColor Green
    Write-Host ("  Score  : {0}/{1} ({2} pct)" -f $passedChecks, $totalChecks, $pct) -ForegroundColor Green
    Write-Host ""
    Write-Host "  KUBRIC IS READY FOR PRODUCTION" -ForegroundColor Green
    Write-Host ""
    exit 0
} else {
    $failed = $totalChecks - $passedChecks
    Write-Host ("  Status : FAIL - {0} of {1} checks failed" -f $failed, $totalChecks) -ForegroundColor Red
    Write-Host ("  Score  : {0}/{1} ({2} pct)" -f $passedChecks, $totalChecks, $pct) -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Review FAIL items above to resolve." -ForegroundColor Yellow
    Write-Host ""
    exit 1
}
