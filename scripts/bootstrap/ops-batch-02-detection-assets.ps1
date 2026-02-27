$ErrorActionPreference = "Stop"

function Sync-Repo {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    if (Test-Path (Join-Path $Destination ".git")) {
        Write-Host "[batch-02] Updating $Destination" -ForegroundColor DarkCyan
        git -C $Destination pull --ff-only | Out-Null
    } else {
        Write-Host "[batch-02] Cloning $Url -> $Destination" -ForegroundColor DarkCyan
        if (Test-Path $Destination) { Remove-Item -Recurse -Force $Destination }
        git clone --depth 1 $Url $Destination | Out-Null
    }
}

Write-Host "[batch-02] Pulling core detection assets..." -ForegroundColor Cyan
Sync-Repo -Url "https://github.com/SigmaHQ/sigma.git" -Destination "vendor/sigma/rules"
Sync-Repo -Url "https://github.com/Yara-Rules/rules.git" -Destination "vendor/yara-rules"
Sync-Repo -Url "https://github.com/projectdiscovery/nuclei-templates.git" -Destination "vendor/nuclei-templates"

$sigmaCount = (Get-ChildItem -Path "vendor/sigma/rules" -Recurse -Include *.yml,*.yaml -ErrorAction SilentlyContinue | Measure-Object).Count
$yaraCount = (Get-ChildItem -Path "vendor/yara-rules" -Recurse -Include *.yar,*.yara -ErrorAction SilentlyContinue | Measure-Object).Count
$nucleiCount = (Get-ChildItem -Path "vendor/nuclei-templates" -Recurse -Include *.yaml -ErrorAction SilentlyContinue | Measure-Object).Count

Write-Host "[batch-02] Sigma rules:  $sigmaCount"
Write-Host "[batch-02] YARA rules:   $yaraCount"
Write-Host "[batch-02] Nuclei rules: $nucleiCount"

if ($sigmaCount -lt 100 -or $yaraCount -lt 50 -or $nucleiCount -lt 500) {
    throw "Detection asset counts too low; aborting rollout"
}

Write-Host "[batch-02] Detection asset sync PASSED" -ForegroundColor Green
