# Find and report non-ASCII bytes in docker-compose.yml
param(
    [string]$RepoRoot = "",
    [switch]$FailOnAnyNonAscii = $false,
    [switch]$IncludeVendor = $false
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

Set-Location $RepoRoot

$utf8Strict = New-Object System.Text.UTF8Encoding($false, $true)
$extensions = @(
    ".go", ".rs", ".toml", ".yaml", ".yml", ".json", ".md", ".xml",
    ".ts", ".tsx", ".js", ".jsx", ".py", ".sh", ".sql", ".proto",
    ".env", ".conf", ".txt"
)

$candidates = Get-ChildItem -Path $RepoRoot -Recurse -File |
    Where-Object {
        $rel = $_.FullName.Substring($RepoRoot.Length).TrimStart('\\')
		$excludeVendor = (-not $IncludeVendor) -and $rel.StartsWith("vendor\")
        -not $rel.StartsWith(".git") -and
        -not $rel.StartsWith("archive\") -and
        -not $excludeVendor -and
        ($extensions -contains $_.Extension.ToLower())
    }

$invalidUtf8 = @()
$nonAscii = @()

foreach ($file in $candidates) {
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)

    try {
        [void]$utf8Strict.GetString($bytes)
    } catch {
        $invalidUtf8 += $file.FullName
        continue
    }

    if ($bytes | Where-Object { $_ -gt 127 } | Select-Object -First 1) {
        $nonAscii += $file.FullName
    }
}

if ($invalidUtf8.Count -gt 0) {
    Write-Host "Invalid UTF-8 files:" -ForegroundColor Red
    $invalidUtf8 | ForEach-Object { Write-Host "  $_" -ForegroundColor Red }
}

Write-Host ("Scanned files: {0}" -f $candidates.Count)
Write-Host ("Valid UTF-8 files: {0}" -f ($candidates.Count - $invalidUtf8.Count))
Write-Host ("Files containing non-ASCII chars: {0}" -f $nonAscii.Count)

if ($FailOnAnyNonAscii -and $nonAscii.Count -gt 0) {
    Write-Host "Non-ASCII files detected (FailOnAnyNonAscii enabled):" -ForegroundColor Yellow
    $nonAscii | ForEach-Object { Write-Host "  $_" -ForegroundColor Yellow }
    exit 2
}

if ($invalidUtf8.Count -gt 0) {
    exit 1
}

Write-Host "UTF-8 validation passed." -ForegroundColor Green
