param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

function Assert-DPath([string]$label, [string]$path) {
    if (-not $path) {
        Write-Host "[WARN] $label not set" -ForegroundColor Yellow
        return
    }
    if ($path -notlike "D:\*") {
        Write-Host "[FAIL] $label is not on D: -> $path" -ForegroundColor Red
    } else {
        Write-Host "[OK]   $label on D: -> $path" -ForegroundColor Green
    }
}

Write-Host "Kubric local storage verification" -ForegroundColor Cyan
Write-Host "RepoRoot: $RepoRoot"

Assert-DPath "KUBRIC_POSTGRES_DATA_DIR" $env:KUBRIC_POSTGRES_DATA_DIR
Assert-DPath "KUBRIC_CLICKHOUSE_DATA_DIR" $env:KUBRIC_CLICKHOUSE_DATA_DIR
Assert-DPath "KUBRIC_NEO4J_DATA_DIR" $env:KUBRIC_NEO4J_DATA_DIR
Assert-DPath "MINIKUBE_HOME" $env:MINIKUBE_HOME
Assert-DPath "KUBECONFIG" $env:KUBECONFIG

if ($RepoRoot -like "D:\*") {
    Write-Host "[OK]   Repo is on D: -> $RepoRoot" -ForegroundColor Green
} else {
    Write-Host "[WARN] Repo is not on D: -> $RepoRoot" -ForegroundColor Yellow
}

$dockerVhd = Get-ChildItem "$env:LOCALAPPDATA\Docker\wsl\disk" -Filter *.vhdx -ErrorAction SilentlyContinue | Select-Object -First 1
if ($dockerVhd) {
    if ($dockerVhd.FullName -like "D:\*") {
        Write-Host "[OK]   Docker Desktop disk image on D: -> $($dockerVhd.FullName)" -ForegroundColor Green
    } else {
        Write-Host "[WARN] Docker Desktop disk image path from LOCALAPPDATA is not on D: -> $($dockerVhd.FullName)" -ForegroundColor Yellow
        Write-Host "       Verify Docker Desktop 'Disk image location' setting." -ForegroundColor Yellow
    }
}

Write-Host "Verification complete." -ForegroundColor Cyan
