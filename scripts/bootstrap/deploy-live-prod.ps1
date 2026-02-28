param(
    [string]$Namespace = "kubric",
    [string]$KSvcTag = "latest",
    [string]$VdrTag = "latest",
    [string]$KicTag = "latest",
    [string]$NocTag = "latest",
    [string]$KaiTag = "latest",
    [string]$WebTag = "latest",
    [string]$N8nTag = "latest"
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

Write-Host "[live] Deploying production overlay..." -ForegroundColor Cyan

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    throw "kubectl not found in PATH"
}

& kubectl get namespace $Namespace 1>$null 2>$null
if ($LASTEXITCODE -ne 0) {
    Invoke-Strict kubectl create namespace $Namespace
}

kubectl kustomize infra/k8s/overlays/prod --load-restrictor LoadRestrictionsNone | kubectl apply -f -
if ($LASTEXITCODE -ne 0) { throw "kustomize apply failed" }

Write-Host "[live] Pinning deployment images..." -ForegroundColor Cyan
Invoke-Strict kubectl set image deployment/k-svc -n $Namespace k-svc="kubric/k-svc:$KSvcTag"
Invoke-Strict kubectl set image deployment/vdr -n $Namespace vdr="kubric/vdr:$VdrTag"
Invoke-Strict kubectl set image deployment/kic -n $Namespace kic="kubric/kic:$KicTag"
Invoke-Strict kubectl set image deployment/noc -n $Namespace noc="kubric/noc:$NocTag"
Invoke-Strict kubectl set image deployment/kai-core -n $Namespace kai-core="kubric/kai:$KaiTag"
Invoke-Strict kubectl set image deployment/web -n $Namespace web="kubric/web:$WebTag"
Invoke-Strict kubectl set image deployment/n8n -n $Namespace n8n="n8nio/n8n:$N8nTag"

Write-Host "[live] Waiting for data plane statefulsets..." -ForegroundColor Cyan
$statefulsets = @("postgresql", "nats", "clickhouse")
foreach ($s in $statefulsets) {
    Invoke-Strict kubectl rollout status statefulset/$s -n $Namespace --timeout=900s
}

Write-Host "[live] Waiting for service deployments..." -ForegroundColor Cyan
$deployments = @("k-svc", "vdr", "kic", "noc", "kai-core", "web", "n8n")
foreach ($d in $deployments) {
    Invoke-Strict kubectl rollout status deployment/$d -n $Namespace --timeout=900s
}

Write-Host "[live] Running runtime smoke checks..." -ForegroundColor Cyan
Invoke-Strict powershell -ExecutionPolicy Bypass -File "$PSScriptRoot\ops-batch-04-runtime-smoke.ps1" -Namespace $Namespace

Write-Host "[live] Production deployment PASSED" -ForegroundColor Green
