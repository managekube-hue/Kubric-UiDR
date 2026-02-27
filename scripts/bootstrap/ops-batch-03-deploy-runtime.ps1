param(
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"

Write-Host "[batch-03] Running no-vault deployment..." -ForegroundColor Cyan
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/deploy-sandbox-no-vault.ps1 -Namespace $Namespace

Write-Host "[batch-03] Waiting for statefulsets..." -ForegroundColor Cyan
$statefulsets = @("clickhouse", "nats", "postgresql")
foreach ($s in $statefulsets) {
    kubectl rollout status statefulset/$s -n $Namespace --timeout=600s
}

Write-Host "[batch-03] Deployment batch PASSED" -ForegroundColor Green
