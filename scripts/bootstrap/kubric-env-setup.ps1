# Kubric-UiDR Environment Setup Script
# Installs and verifies all tools to D:\ drive
# Run from: Visual Studio Developer PowerShell

param(
    [switch]$VerifyOnly
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  KUBRIC-UiDR ENVIRONMENT SETUP" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan

# ?? Set all tool paths to D:\ ?????????????????????????????????????????????????
$toolsDir    = "D:\kubric-tools"
$goDir       = "D:\go"
$cargoDir    = "D:\.cargo"
$rustupDir   = "D:\.rustup"
$minikubeDir = "D:\.minikube"
$kubeDir     = "D:\.kube"
$pythonDir   = "D:\Python312"
$nodeDir     = "D:\"

foreach ($d in @($toolsDir, $goDir, $minikubeDir, $kubeDir)) {
    New-Item -ItemType Directory -Force -Path $d | Out-Null
}

# ?? Set environment variables permanently ????????????????????????????????????
[Environment]::SetEnvironmentVariable("CARGO_HOME",    $cargoDir,    "User")
[Environment]::SetEnvironmentVariable("RUSTUP_HOME",   $rustupDir,   "User")
[Environment]::SetEnvironmentVariable("GOPATH",        $goDir,       "User")
[Environment]::SetEnvironmentVariable("MINIKUBE_HOME", $minikubeDir, "User")
[Environment]::SetEnvironmentVariable("KUBECONFIG",    "$kubeDir\config", "User")

$addToPath = "$goDir\bin;$cargoDir\bin;$toolsDir;$pythonDir;$nodeDir"
$currentUserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($currentUserPath -notlike "*$goDir\bin*") {
    [Environment]::SetEnvironmentVariable("PATH", "$addToPath;$currentUserPath", "User")
    Write-Host "PATH updated permanently" -ForegroundColor Green
} else {
    Write-Host "PATH already configured" -ForegroundColor Green
}

# ?? Set for current session ???????????????????????????????????????????????????
$env:CARGO_HOME    = $cargoDir
$env:RUSTUP_HOME   = $rustupDir
$env:GOPATH        = $goDir
$env:MINIKUBE_HOME = $minikubeDir
$env:KUBECONFIG    = "$kubeDir\config"
$env:PATH          = "$addToPath;" + $env:PATH

if ($VerifyOnly) {
    Write-Host "`n--- VERIFY ONLY MODE ---" -ForegroundColor Yellow
} else {

    # ?? Go ????????????????????????????????????????????????????????????????????
    Write-Host "`n[1/5] Checking Go..." -ForegroundColor Yellow
    if (-not (Test-Path "$goDir\bin\go.exe")) {
        Write-Host "  Extracting Go from zip..." -ForegroundColor Gray
        if (Test-Path "$toolsDir\go.zip") {
            Expand-Archive -Path "$toolsDir\go.zip" -DestinationPath "D:\" -Force
            Write-Host "  Go extracted" -ForegroundColor Green
        } else {
            Write-Host "  Downloading Go..." -ForegroundColor Gray
            Invoke-WebRequest -Uri "https://go.dev/dl/go1.23.4.windows-amd64.zip" -OutFile "$toolsDir\go.zip" -UseBasicParsing
            Expand-Archive -Path "$toolsDir\go.zip" -DestinationPath "D:\" -Force
            Write-Host "  Go installed" -ForegroundColor Green
        }
    }

    # ?? Rust/Cargo ????????????????????????????????????????????????????????????
    Write-Host "`n[2/5] Checking Rust..." -ForegroundColor Yellow
    $rustcPath = "$rustupDir\toolchains\stable-x86_64-pc-windows-msvc\bin\rustc.exe"
    if (-not (Test-Path $rustcPath)) {
        Write-Host "  Installing Rust toolchain..." -ForegroundColor Gray
        if (-not (Test-Path "$cargoDir\bin\rustup.exe")) {
            Invoke-WebRequest -Uri "https://win.rustup.rs/x86_64" -OutFile "$toolsDir\rustup-init.exe" -UseBasicParsing
        }
        $rustEnv = @{
            "CARGO_HOME"  = $cargoDir
            "RUSTUP_HOME" = $rustupDir
        }
        Start-Process "$toolsDir\rustup-init.exe" -ArgumentList "-y --no-modify-path --default-toolchain stable-msvc" -Wait -PassThru -Environment $rustEnv
    }

    # ?? Minikube ??????????????????????????????????????????????????????????????
    Write-Host "`n[3/5] Checking Minikube..." -ForegroundColor Yellow
    if (-not (Test-Path "$toolsDir\minikube.exe")) {
        Write-Host "  Downloading minikube..." -ForegroundColor Gray
        Invoke-WebRequest -Uri "https://storage.googleapis.com/minikube/releases/latest/minikube-windows-amd64.exe" -OutFile "$toolsDir\minikube.exe" -UseBasicParsing
    }

    # ?? kubectl ???????????????????????????????????????????????????????????????
    Write-Host "`n[4/5] Checking kubectl..." -ForegroundColor Yellow
    if (-not (Test-Path "$toolsDir\kubectl.exe")) {
        Write-Host "  Downloading kubectl..." -ForegroundColor Gray
        $k8sVer = (Invoke-WebRequest "https://dl.k8s.io/release/stable.txt" -UseBasicParsing).Content.Trim()
        Invoke-WebRequest -Uri "https://dl.k8s.io/release/$k8sVer/bin/windows/amd64/kubectl.exe" -OutFile "$toolsDir\kubectl.exe" -UseBasicParsing
    }

    # ?? Python pip packages ???????????????????????????????????????????????????
    Write-Host "`n[5/5] Installing Python KAI packages..." -ForegroundColor Yellow
    $pipExe = if (Test-Path "$pythonDir\Scripts\pip.exe") { "$pythonDir\Scripts\pip.exe" } else { "pip" }
    & $pipExe install --quiet --upgrade pip
    & $pipExe install --quiet ansible-runner pymisp pandas requests crewai composio-crewai pyfair presidio-analyzer scikit-learn xgboost checkov ssvc
}

# ?? Final Health Report ???????????????????????????????????????????????????????
Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "        KUBRIC ENVIRONMENT HEALTH" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan

function Check-Tool {
    param($name, $cmd, $args)
    try {
        $result = & $cmd $args 2>&1 | Select-Object -First 1
        Write-Host ("  {0,-15} {1}" -f "${name}:", $result) -ForegroundColor Green
    } catch {
        Write-Host ("  {0,-15} MISSING" -f "${name}:") -ForegroundColor Red
    }
}

$goExe      = "$goDir\bin\go.exe"
$rustcExe   = "$rustupDir\toolchains\stable-x86_64-pc-windows-msvc\bin\rustc.exe"
$cargoExe   = "$rustupDir\toolchains\stable-x86_64-pc-windows-msvc\bin\cargo.exe"
$minikubeExe= "$toolsDir\minikube.exe"
$kubectlExe = "$toolsDir\kubectl.exe"
$pythonExe  = "$pythonDir\python.exe"
$nodeExe    = "$nodeDir\node.exe"
$npmCmd     = "$nodeDir\npm.cmd"

# Go
if (Test-Path $goExe) {
    Write-Host ("  {0,-15} {1}" -f "go:", (& $goExe version 2>&1)) -ForegroundColor Green
} else { Write-Host "  go:             MISSING" -ForegroundColor Red }

# Rust
if (Test-Path $rustcExe) {
    Write-Host ("  {0,-15} {1}" -f "rustc:", (& $rustcExe --version 2>&1)) -ForegroundColor Green
} else { Write-Host "  rustc:          MISSING" -ForegroundColor Red }

# Cargo
if (Test-Path $cargoExe) {
    Write-Host ("  {0,-15} {1}" -f "cargo:", (& $cargoExe --version 2>&1)) -ForegroundColor Green
} else { Write-Host "  cargo:          MISSING" -ForegroundColor Red }

# Node
if (Test-Path $nodeExe) {
    Write-Host ("  {0,-15} {1}" -f "node:", (& $nodeExe --version 2>&1)) -ForegroundColor Green
} else { Write-Host "  node:           MISSING" -ForegroundColor Red }

# npm
if (Test-Path $npmCmd) {
    Write-Host ("  {0,-15} {1}" -f "npm:", (cmd /c "`"$npmCmd`" --version" 2>&1)) -ForegroundColor Green
} else { Write-Host "  npm:            MISSING" -ForegroundColor Red }

# Python
if (Test-Path $pythonExe) {
    Write-Host ("  {0,-15} {1}" -f "python:", (& $pythonExe --version 2>&1)) -ForegroundColor Green
} else { Write-Host "  python:         MISSING" -ForegroundColor Red }

# Docker
$dockerResult = docker --version 2>&1
if ($LASTEXITCODE -eq 0 -or $dockerResult -match "Docker") {
    Write-Host ("  {0,-15} {1}" -f "docker:", $dockerResult) -ForegroundColor Green
} else { Write-Host "  docker:         NOT RUNNING (restart required after install)" -ForegroundColor Yellow }

# Minikube
if (Test-Path $minikubeExe) {
    Write-Host ("  {0,-15} {1}" -f "minikube:", (& $minikubeExe version --short 2>&1)) -ForegroundColor Green
} else { Write-Host "  minikube:       MISSING" -ForegroundColor Red }

# kubectl
if (Test-Path $kubectlExe) {
    Write-Host ("  {0,-15} {1}" -f "kubectl:", (& $kubectlExe version --client --short 2>&1 | Select-Object -First 1)) -ForegroundColor Green
} else { Write-Host "  kubectl:        MISSING" -ForegroundColor Red }

# WSL2
Write-Host ""
Write-Host "  WSL2 Distributions:" -ForegroundColor Cyan
wsl --list --verbose 2>&1 | ForEach-Object { Write-Host "    $_" }

Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "  All tools on D:\ — PATH updated" -ForegroundColor Cyan
Write-Host "  RESTART Visual Studio to pick up PATH" -ForegroundColor Yellow
Write-Host "============================================" -ForegroundColor Cyan
