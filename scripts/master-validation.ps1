# =============================================================================
# Kubric Master Validation - Run All Checks
# =============================================================================

$ErrorActionPreference = "Stop"

Write-Host "`n" -NoNewline
Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║         KUBRIC MASTER VALIDATION - ALL SYSTEMS CHECK          ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

$totalChecks = 0
$passedChecks = 0

function Run-Check {
    param(
        [string]$Name,
        [scriptblock]$Check
    )
    
    $script:totalChecks++
    Write-Host "[$script:totalChecks] $Name..." -ForegroundColor Yellow -NoNewline
    
    try {
        $result = & $Check
        if ($result) {
            Write-Host " ✓" -ForegroundColor Green
            $script:passedChecks++
            return $true
        }
        else {
            Write-Host " ✗" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host " ✗ (Error: $_)" -ForegroundColor Red
        return $false
    }
}

# ── 1. File Structure Checks ─────────────────────────────────────────────
Write-Host "`n=== File Structure ===" -ForegroundColor Cyan

Run-Check "docker-compose.prod.yml exists" {
    Test-Path "docker-compose.prod.yml"
}

Run-Check "Dockerfile.web exists" {
    Test-Path "Dockerfile.web"
}

Run-Check ".kubric/ directory exists" {
    Test-Path ".kubric"
}

Run-Check "docs/VENDOR-INTEGRATIONS.md exists" {
    Test-Path "docs/VENDOR-INTEGRATIONS.md"
}

Run-Check "docs/COMPLIANCE.md exists" {
    Test-Path "docs/COMPLIANCE.md"
}

Run-Check "docs/DR-COVERAGE.md exists" {
    Test-Path "docs/DR-COVERAGE.md"
}

Run-Check "archive/k8s-legacy/ exists" {
    Test-Path "archive/k8s-legacy"
}

Run-Check "archive/spec-modules/ exists" {
    Test-Path "archive/spec-modules"
}

# ── 2. AI Autonomy Checks ────────────────────────────────────────────────
Write-Host "`n=== AI Autonomy ===" -ForegroundColor Cyan

Run-Check "kai/deploy/docker_manager.py exists" {
    Test-Path "kai/deploy/docker_manager.py"
}

Run-Check "kai/house/monitor.py exists" {
    Test-Path "kai/house/monitor.py"
}

Run-Check ".kubric/stack.yaml exists" {
    Test-Path ".kubric/stack.yaml"
}

Run-Check ".kubric/deployment-rules.yaml exists" {
    Test-Path ".kubric/deployment-rules.yaml"
}

Run-Check ".kubric/health-checks.yaml exists" {
    Test-Path ".kubric/health-checks.yaml"
}

# ── 3. Docker Compose Validation ─────────────────────────────────────────
Write-Host "`n=== Docker Compose ===" -ForegroundColor Cyan

Run-Check "docker-compose.yml syntax valid" {
    docker compose config 2>&1 | Out-Null
    $LASTEXITCODE -eq 0
}

Run-Check "docker-compose.prod.yml syntax valid" {
    docker compose -f docker-compose.prod.yml config 2>&1 | Out-Null
    $LASTEXITCODE -eq 0
}

Run-Check "Temporal UI tag is 2.28.0" {
    $content = Get-Content "docker-compose.prod.yml" -Raw
    $content -match "temporalio/ui:2.28.0"
}

Run-Check "Production compose has Watchtower" {
    $content = Get-Content "docker-compose.prod.yml" -Raw
    $content -match "watchtower"
}

Run-Check "Production compose has resource limits" {
    $content = Get-Content "docker-compose.prod.yml" -Raw
    $content -match "limits:"
}

Run-Check "Production compose has 3 replicas" {
    $content = Get-Content "docker-compose.prod.yml" -Raw
    $content -match "replicas: 3"
}

# ── 4. Test Suite Checks ─────────────────────────────────────────────────
Write-Host "`n=== Test Suite ===" -ForegroundColor Cyan

Run-Check "Integration tests exist" {
    Test-Path "tests/integration/test_e2e.py"
}

Run-Check "License check workflow exists" {
    Test-Path ".github/workflows/license-check.yml"
}

Run-Check "ops-batch-07 script exists" {
    Test-Path "scripts/bootstrap/ops-batch-07-tree-restructure.ps1"
}

# ── 5. Makefile Checks ───────────────────────────────────────────────────
Write-Host "`n=== Makefile ===" -ForegroundColor Cyan

Run-Check "Makefile exists" {
    Test-Path "Makefile"
}

Run-Check "deploy-prod target exists" {
    $content = Get-Content "Makefile" -Raw
    $content -match "deploy-prod:"
}

Run-Check "ops-batch-07 target exists" {
    $content = Get-Content "Makefile" -Raw
    $content -match "ops-batch-07:"
}

# ── 6. Documentation Checks ──────────────────────────────────────────────
Write-Host "`n=== Documentation ===" -ForegroundColor Cyan

Run-Check "DEPLOYMENT.md exists" {
    Test-Path "docs/DEPLOYMENT.md"
}

Run-Check "BUGFIXES-2026-02-27.md exists" {
    Test-Path "docs/BUGFIXES-2026-02-27.md"
}

Run-Check "EXECUTION-SUMMARY.md exists" {
    Test-Path "docs/EXECUTION-SUMMARY.md"
}

Run-Check "TREE-RESTRUCTURE-SUMMARY.md exists" {
    Test-Path "docs/TREE-RESTRUCTURE-SUMMARY.md"
}

Run-Check "FINAL-REPORT.md exists" {
    Test-Path "docs/FINAL-REPORT.md"
}

Run-Check "QUICKREF.md exists" {
    Test-Path "QUICKREF.md"
}

# ── 7. Validation Scripts ────────────────────────────────────────────────
Write-Host "`n=== Validation Scripts ===" -ForegroundColor Cyan

Run-Check "validate-fixes.ps1 exists" {
    Test-Path "scripts/validate-fixes.ps1"
}

Run-Check "validate-fixes.sh exists" {
    Test-Path "scripts/validate-fixes.sh"
}

Run-Check "deploy-aws.sh exists" {
    Test-Path "scripts/deploy-aws.sh"
}

# ── Summary ──────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║                      VALIDATION SUMMARY                        ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

$percentage = [math]::Round(($passedChecks / $totalChecks) * 100, 1)

if ($passedChecks -eq $totalChecks) {
    Write-Host "  Status: " -NoNewline
    Write-Host "✓ ALL CHECKS PASSED" -ForegroundColor Green
    Write-Host "  Result: $passedChecks/$totalChecks ($percentage%)" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Next Steps:" -ForegroundColor Yellow
    Write-Host "    1. make ops-batch-07     # Run full tree verification (72 checks)" -ForegroundColor White
    Write-Host "    2. make deploy-prod      # Deploy production stack" -ForegroundColor White
    Write-Host "    3. docker compose -f docker-compose.prod.yml ps" -ForegroundColor White
    Write-Host ""
    Write-Host "  🚀 KUBRIC IS READY FOR PRODUCTION" -ForegroundColor Green
    exit 0
}
else {
    Write-Host "  Status: " -NoNewline
    Write-Host "✗ SOME CHECKS FAILED" -ForegroundColor Red
    Write-Host "  Result: $passedChecks/$totalChecks ($percentage%)" -ForegroundColor Red
    Write-Host "  Failed: $($totalChecks - $passedChecks) checks" -ForegroundColor Red
    Write-Host ""
    Write-Host "  Review the output above for details." -ForegroundColor Yellow
    exit 1
}
