# =============================================================================
# Kubric Validation Script - Test All Fixes (Windows)
# =============================================================================

$ErrorActionPreference = "Stop"

Write-Host "=== Kubric Validation Suite ===" -ForegroundColor Green
Write-Host ""

# Test 1: Validate docker-compose.prod.yml syntax
Write-Host "[1/4] Validating docker-compose.prod.yml..." -ForegroundColor Yellow
try {
    docker compose -f docker-compose.prod.yml config | Out-Null
    Write-Host "✓ docker-compose.prod.yml syntax valid" -ForegroundColor Green
} catch {
    Write-Host "✗ docker-compose.prod.yml syntax invalid" -ForegroundColor Red
    exit 1
}

# Test 2: Build frontend image
Write-Host "[2/4] Building frontend image..." -ForegroundColor Yellow
try {
    Push-Location frontend
    docker build -t kubric-web:test -f ..\Dockerfile.web . 2>&1 | Out-Null
    Write-Host "✓ Frontend image built successfully" -ForegroundColor Green
    Pop-Location
} catch {
    Write-Host "✗ Frontend build failed" -ForegroundColor Red
    Pop-Location
    exit 1
}

# Test 3: Check KAI Python modules exist
Write-Host "[3/4] Checking KAI Python modules..." -ForegroundColor Yellow
if ((Test-Path "kai\deploy\docker_manager.py") -and (Test-Path "kai\house\monitor.py")) {
    Write-Host "✓ KAI autonomy modules present" -ForegroundColor Green
} else {
    Write-Host "✗ KAI autonomy modules missing" -ForegroundColor Red
    exit 1
}

# Test 4: Verify service names in compose file
Write-Host "[4/4] Verifying service names..." -ForegroundColor Yellow
$content = Get-Content docker-compose.prod.yml -Raw
if ($content -match "kai-python:") {
    Write-Host "✓ Service name 'kai-python' found in docker-compose.prod.yml" -ForegroundColor Green
} else {
    Write-Host "✗ Service name 'kai-python' not found" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== All Validations Passed ===" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. docker compose -f docker-compose.prod.yml up -d"
Write-Host "  2. docker compose -f docker-compose.prod.yml ps"
Write-Host "  3. docker exec -it kubric-uidr-kai-python-1 python --version"
Write-Host ""
