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

Write-Host "[batch-03] Running no-vault deployment..." -ForegroundColor Cyan
Invoke-Strict powershell -ExecutionPolicy Bypass -File "$PSScriptRoot\deploy-sandbox-no-vault.ps1" -Namespace $Namespace

Write-Host "[batch-03] Waiting for statefulsets..." -ForegroundColor Cyan
$statefulsets = @("clickhouse", "nats", "postgresql")
foreach ($s in $statefulsets) {
    Invoke-Strict kubectl rollout status statefulset/$s -n $Namespace --timeout=600s
}

Write-Host "[batch-03] Deployment batch PASSED" -ForegroundColor Green
