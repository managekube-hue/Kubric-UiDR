$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$repoRoot = Split-Path -Parent $root
$frontend = Join-Path $repoRoot "frontend"

if (-not (Test-Path $frontend)) {
  Write-Host "Frontend directory not found: $frontend"
  exit 1
}

Push-Location $frontend
try {
  if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "npm is required but not found in PATH."
    exit 1
  }

  Write-Host "Installing frontend dependencies..."
  npm ci

  Write-Host "Building Next.js frontend..."
  npm run build

  Write-Host "Frontend build succeeded."
} finally {
  Pop-Location
}
