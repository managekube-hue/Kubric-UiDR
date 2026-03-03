Write-Host "?? Kubric Deployment" -ForegroundColor Cyan
Write-Host "====================" -ForegroundColor Cyan

$ErrorActionPreference = "Stop"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker is not installed or not on PATH"
}

$owner = if ($env:GHCR_OWNER) { $env:GHCR_OWNER } else { "managekube-hue" }
$pat = $env:GHCR_KUBRIC_PAT

$repoUrl = "https://github.com/$owner/Kubric-UiDR"
$status = 200
try {
    $resp = Invoke-WebRequest -Uri $repoUrl -Method Head -UseBasicParsing -TimeoutSec 10
    $status = [int]$resp.StatusCode
} catch {
    $status = 404
}

if ($status -eq 404 -or $status -eq 401) {
    Write-Host "?? Private repository/package access path detected"
    if (-not $pat) {
        $securePat = Read-Host "Enter GHCR_KUBRIC_PAT" -AsSecureString
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePat)
        $pat = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
    }

    $pat | docker login ghcr.io -u $owner --password-stdin | Out-Host
} else {
    Write-Host "?? Public repository path detected"
}

if (-not (Test-Path .env) -and (Test-Path deploy/.env.example)) {
    Copy-Item deploy/.env.example .env
    Write-Host "Created .env from deploy/.env.example"
}

docker compose -f deploy/docker-compose.yml pull | Out-Host
docker compose -f deploy/docker-compose.yml up -d | Out-Host

Write-Host "? Deployment started" -ForegroundColor Green
