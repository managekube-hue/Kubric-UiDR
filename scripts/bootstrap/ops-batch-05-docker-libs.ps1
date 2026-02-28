param(
    [switch]$BuildKaiImage = $false,
    [string]$KaiImageTag = "kubric/kai:deps-check"
)

$ErrorActionPreference = "Stop"

function Assert-Contains {
    param(
        [Parameter(Mandatory = $true)][string]$Content,
        [Parameter(Mandatory = $true)][string]$Needle,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if ($Content -notmatch [regex]::Escape($Needle)) {
           throw ("[batch-05] Missing: {0} ({1})" -f $Label, $Needle)
    }

    Write-Host "[batch-05] OK: $Label" -ForegroundColor Green
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\")).Path

Write-Host "[batch-05] Validating Docker/library completeness..." -ForegroundColor Cyan

$kaiDocker = Get-Content (Join-Path $repoRoot "kai\Dockerfile") -Raw
$webDocker = Get-Content (Join-Path $repoRoot "Dockerfile.web") -Raw
$agentsDocker = Get-Content (Join-Path $repoRoot "Dockerfile.agents") -Raw
$kaiPyproject = Get-Content (Join-Path $repoRoot "kai\pyproject.toml") -Raw

Assert-Contains -Content $kaiDocker -Needle "ARG KAI_PIP_EXTRAS=" -Label "KAI extras arg exists"
Assert-Contains -Content $kaiDocker -Needle "ai,workflows,security,psa,ti" -Label "KAI extras include SOC/NOC/GRC/PSA/TI runtime"
Assert-Contains -Content $kaiDocker -Needle "composio-crewai" -Label "Composio dependency present in KAI Docker fallback"
Assert-Contains -Content $kaiDocker -Needle "ansible-runner" -Label "Ansible runner dependency present in KAI Docker fallback"
Assert-Contains -Content $kaiDocker -Needle "EXPOSE 8100" -Label "KAI image exposes Kubernetes service port"
Assert-Contains -Content $kaiDocker -Needle "--port" -Label "KAI uvicorn CLI contains port flag"
Assert-Contains -Content $kaiDocker -Needle "8100" -Label "KAI uvicorn binds to 8100"

Assert-Contains -Content $webDocker -Needle "NEXT_PUBLIC_KAI_URL=http://kai-core:8100" -Label "Web image default KAI URL matches in-cluster service"
Assert-Contains -Content $agentsDocker -Needle "coresec" -Label "CoreSec runtime stage exists"
Assert-Contains -Content $agentsDocker -Needle "libpcap0.8" -Label "Rust runtime includes libpcap dependency"

Assert-Contains -Content $kaiPyproject -Needle "composio-crewai" -Label "KAI manifest includes composio-crewai"
Assert-Contains -Content $kaiPyproject -Needle "ansible-runner" -Label "KAI manifest includes ansible-runner"
Assert-Contains -Content $kaiPyproject -Needle "checkov" -Label "KAI manifest includes checkov"
Assert-Contains -Content $kaiPyproject -Needle "presidio-analyzer" -Label "KAI manifest includes presidio"
Assert-Contains -Content $kaiPyproject -Needle "reportlab" -Label "KAI manifest includes reportlab"
Assert-Contains -Content $kaiPyproject -Needle "minio" -Label "KAI manifest includes minio"

if ($BuildKaiImage) {
    Write-Host "[batch-05] Building KAI image for dependency verification..." -ForegroundColor Cyan
    Push-Location (Join-Path $repoRoot "kai")
    try {
        docker build -t $KaiImageTag -f Dockerfile .
        if ($LASTEXITCODE -ne 0) {
            throw "[batch-05] KAI docker build failed ($LASTEXITCODE)"
        }
    }
    finally {
        Pop-Location
    }

    Write-Host "[batch-05] Inspecting installed Python packages in image..." -ForegroundColor Cyan
    $pipOut = docker run --rm $KaiImageTag python -m pip freeze
    if ($LASTEXITCODE -ne 0) {
        throw "[batch-05] Failed to inspect pip packages in built image ($LASTEXITCODE)"
    }

    Assert-Contains -Content $pipOut -Needle "crewai" -Label "Image contains crewai"
    Assert-Contains -Content $pipOut -Needle "temporalio" -Label "Image contains temporalio"
    Assert-Contains -Content $pipOut -Needle "ansible-runner" -Label "Image contains ansible-runner"
    Assert-Contains -Content $pipOut -Needle "clickhouse-connect" -Label "Image contains clickhouse-connect"
}

Write-Host "[batch-05] Docker/library completeness PASSED" -ForegroundColor Green
