param(
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"

Write-Host "[kubric] Deploying sandbox-no-vault overlay..." -ForegroundColor Cyan

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    throw "kubectl not found in PATH"
}

kubectl get namespace $Namespace 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    kubectl create namespace $Namespace | Out-Null
}

kubectl apply -k infra/k8s/overlays/sandbox-no-vault

$deployments = @("k-svc", "vdr", "kic", "noc", "kai-core", "web", "n8n")
foreach ($d in $deployments) {
    Write-Host "[kubric] Waiting for deployment/$d" -ForegroundColor DarkCyan
    kubectl rollout status deployment/$d -n $Namespace --timeout=300s
}

Write-Host "[kubric] Sandbox no-vault deployment complete." -ForegroundColor Green
Write-Host "[kubric] Next check: kubectl get pods -n $Namespace" -ForegroundColor Yellow
