$ErrorActionPreference = "Stop"

$required = @(
  "node",
  "npm",
  "go",
  "cargo",
  "rustc",
  "docker",
  "kubectl",
  "kustomize"
)

$missing = @()

Write-Host "Checking required CLI tools..."
foreach ($tool in $required) {
  $cmd = Get-Command $tool -ErrorAction SilentlyContinue
  if ($null -eq $cmd) {
    Write-Host "[MISSING] $tool"
    $missing += $tool
  } else {
    try {
      $version = & $tool --version 2>$null | Select-Object -First 1
      if ([string]::IsNullOrWhiteSpace($version)) {
        Write-Host "[FOUND]   $tool"
      } else {
        Write-Host "[FOUND]   $tool -> $version"
      }
    } catch {
      Write-Host "[FOUND]   $tool"
    }
  }
}

Write-Host ""
if ($missing.Count -gt 0) {
  Write-Host "Missing tools: $($missing -join ', ')"
  exit 1
}

Write-Host "All required tools are available."
exit 0
