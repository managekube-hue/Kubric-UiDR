param(
    [string]$RepoRoot = "",
    [switch]$SkipDownloads = $false
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

Set-Location $RepoRoot

function Ensure-Dir([string]$path) {
    if (-not (Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

function Download-File([string]$url, [string]$dest) {
    Write-Host "Downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
}

function Expand-TarGz([string]$archive, [string]$dest) {
    Ensure-Dir $dest
    tar -xf $archive -C $dest
}

$srcDir = Join-Path $RepoRoot "vendor\sources"
$stageDir = Join-Path $RepoRoot "vendor\_staging"
Ensure-Dir $srcDir
Ensure-Dir $stageDir
Ensure-Dir (Join-Path $RepoRoot "vendor\ndpi")
Ensure-Dir (Join-Path $RepoRoot "vendor\zeek\policy\protocols\ssl")
Ensure-Dir (Join-Path $RepoRoot "vendor\zeek\scripts")
Ensure-Dir (Join-Path $RepoRoot "vendor\coreruleset\rules")
Ensure-Dir (Join-Path $RepoRoot "vendor\suricata")

$ndpiTar = Join-Path $srcDir "ntop-nDPI-5.0.tar.gz"
$zeekTar = Join-Path $srcDir "zeek-v7.0.2.tar.gz"
$crsTar  = Join-Path $srcDir "coreruleset-v4.6.0.tar.gz"
$etTar   = Join-Path $srcDir "suricata-emerging.rules.tar.gz"
$ja3File = Join-Path $RepoRoot "vendor\zeek\policy\protocols\ssl\ja3.zeek"
$intelJa3File = Join-Path $RepoRoot "vendor\zeek\policy\protocols\ssl\intel_ja3.zeek"

if (-not $SkipDownloads) {
    Download-File "https://github.com/ntop/nDPI/archive/refs/tags/5.0.tar.gz" $ndpiTar
    Download-File "https://github.com/zeek/zeek/archive/refs/tags/v7.0.2.tar.gz" $zeekTar
    Download-File "https://github.com/coreruleset/coreruleset/archive/refs/tags/v4.6.0.tar.gz" $crsTar
    Download-File "https://rules.emergingthreats.net/open/suricata-7.0/emerging.rules.tar.gz" $etTar
    Download-File "https://raw.githubusercontent.com/salesforce/ja3/master/zeek/ja3.zeek" $ja3File
    try {
        Download-File "https://raw.githubusercontent.com/salesforce/ja3/master/zeek/intel_ja3.zeek" $intelJa3File
    } catch {
        Write-Host "intel_ja3.zeek not available upstream; continuing"
    }
}

Expand-TarGz $ndpiTar (Join-Path $stageDir "ndpi")
Expand-TarGz $zeekTar (Join-Path $stageDir "zeek")
Expand-TarGz $crsTar (Join-Path $stageDir "coreruleset")
Expand-TarGz $etTar (Join-Path $stageDir "suricata")

$ndpiRoot = Get-ChildItem (Join-Path $stageDir "ndpi") -Directory | Select-Object -First 1
if ($ndpiRoot) {
    Ensure-Dir (Join-Path $RepoRoot "vendor\ndpi\include")
    Copy-Item (Join-Path $ndpiRoot.FullName "src\include\*.h") (Join-Path $RepoRoot "vendor\ndpi\include") -Force -ErrorAction SilentlyContinue
}

$zeekRoot = Get-ChildItem (Join-Path $stageDir "zeek") -Directory | Select-Object -First 1
if ($zeekRoot -and (Test-Path (Join-Path $zeekRoot.FullName "scripts"))) {
    Copy-Item (Join-Path $zeekRoot.FullName "scripts\*") (Join-Path $RepoRoot "vendor\zeek\scripts") -Recurse -Force
}

$crsRoot = Get-ChildItem (Join-Path $stageDir "coreruleset") -Directory | Select-Object -First 1
if ($crsRoot -and (Test-Path (Join-Path $crsRoot.FullName "rules"))) {
    Copy-Item (Join-Path $crsRoot.FullName "rules\*.conf") (Join-Path $RepoRoot "vendor\coreruleset\rules") -Force
}

$etRuleFiles = Get-ChildItem (Join-Path $stageDir "suricata") -Recurse -Filter "*.rules" -File
foreach ($f in $etRuleFiles) {
    Copy-Item $f.FullName (Join-Path $RepoRoot "vendor\suricata\$($f.Name)") -Force
}

Write-Host "NDR vendor sync complete."
