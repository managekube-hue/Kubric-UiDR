param(
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"

function Invoke-Strict {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(ValueFromRemainingArguments = $true)]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed ($LASTEXITCODE): $FilePath $($Arguments -join ' ')"
    }
}

function Test-ServiceHealthz {
    param(
        [Parameter(Mandatory = $true)][string]$Service,
        [Parameter(Mandatory = $true)][int]$Port
    )

    $path = "/api/v1/namespaces/$Namespace/services/http:${Service}:${Port}/proxy/healthz"
    & kubectl get --raw $path 1>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Health probe failed for $Service on port $Port"
    }
}

Write-Host "[batch-04] Checking pod health..." -ForegroundColor Cyan
$pods = kubectl get pods -n $Namespace --no-headers
if (-not $pods) {
    throw "No pods found in namespace $Namespace"
}

$bad = @()
$pods -split "`n" | ForEach-Object {
    if ($_ -match "CrashLoopBackOff|Error|ImagePullBackOff|Pending") {
        $bad += $_
    }
}

if ($bad.Count -gt 0) {
    Write-Host "[batch-04] Unhealthy pods:" -ForegroundColor Red
    $bad | ForEach-Object { Write-Host "  $_" }
    throw "Runtime health check failed"
}

Write-Host "[batch-04] Checking service endpoints..." -ForegroundColor Cyan
$services = @("k-svc", "vdr", "kic", "noc", "kai-core", "web", "n8n", "postgresql", "nats", "clickhouse")
foreach ($svc in $services) {
    Invoke-Strict kubectl get svc $svc -n $Namespace
}

Write-Host "[batch-04] Running service health probes via Kubernetes API proxy..." -ForegroundColor Cyan
Test-ServiceHealthz -Service "k-svc" -Port 8080
Test-ServiceHealthz -Service "vdr" -Port 8081
Test-ServiceHealthz -Service "kic" -Port 8082
Test-ServiceHealthz -Service "noc" -Port 8083
Test-ServiceHealthz -Service "kai-core" -Port 8100

Write-Host "[batch-04] Runtime smoke PASSED" -ForegroundColor Green
