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

Write-Host "[kubric] Deploying sandbox-no-vault overlay..." -ForegroundColor Cyan

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    throw "kubectl not found in PATH"
}

& kubectl get namespace $Namespace 1>$null 2>$null
if ($LASTEXITCODE -ne 0) {
    Invoke-Strict kubectl create namespace $Namespace
}

kubectl kustomize infra/k8s/overlays/sandbox-no-vault --load-restrictor LoadRestrictionsNone | kubectl apply -f -
if ($LASTEXITCODE -ne 0) { throw "kustomize apply failed" }

$deployments = @("k-svc", "vdr", "kic", "noc", "kai-core", "web", "n8n")
foreach ($d in $deployments) {
    Write-Host "[kubric] Waiting for deployment/$d" -ForegroundColor DarkCyan
    Invoke-Strict kubectl rollout status deployment/$d -n $Namespace --timeout=300s
}

Write-Host "[kubric] Sandbox no-vault deployment complete." -ForegroundColor Green
Write-Host "[kubric] Next check: kubectl get pods -n $Namespace" -ForegroundColor Yellow
