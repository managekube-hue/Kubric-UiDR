param(
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"

$required = @("kubectl", "git", "node", "npm", "go")
$missing = @()

Write-Host "[batch-01] Checking required tools..." -ForegroundColor Cyan
foreach ($tool in $required) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        $missing += $tool
        Write-Host "[MISSING] $tool" -ForegroundColor Red
    } else {
        Write-Host "[FOUND]   $tool" -ForegroundColor Green
    }
}

if ($missing.Count -gt 0) {
    throw "Missing required tools: $($missing -join ', ')"
}

Write-Host "[batch-01] Checking Kubernetes cluster access..." -ForegroundColor Cyan
kubectl cluster-info | Out-Null

kubectl get namespace $Namespace 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "[batch-01] Creating namespace $Namespace" -ForegroundColor Yellow
    kubectl create namespace $Namespace | Out-Null
}

Write-Host "[batch-01] Validating RBAC for namespace $Namespace" -ForegroundColor Cyan
$canApply = kubectl auth can-i create deployments -n $Namespace
if ($canApply -notmatch "yes") {
    throw "Current kube context cannot create deployments in namespace '$Namespace'"
}

Write-Host "[batch-01] Preflight PASSED" -ForegroundColor Green
