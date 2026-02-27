param(
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"

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
    kubectl get svc $svc -n $Namespace | Out-Null
}

Write-Host "[batch-04] Running in-cluster health probes..." -ForegroundColor Cyan
kubectl run curl-smoke --rm -i --restart=Never -n $Namespace --image=curlimages/curl:8.8.0 -- \
  sh -c "curl -fsS http://k-svc:8080/healthz ; curl -fsS http://vdr:8081/healthz ; curl -fsS http://kic:8082/healthz ; curl -fsS http://noc:8083/healthz ; curl -fsS http://kai-core:8100/healthz" | Out-Null

Write-Host "[batch-04] Runtime smoke PASSED" -ForegroundColor Green
