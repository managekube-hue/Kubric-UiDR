# Kubric Git + Tool environment bootstrapper
# Fixes PATH for VS Developer PowerShell sessions
# Usage: . .\scripts\bootstrap\kubric-git-env.ps1

$gitPaths = @(
    "C:\Program Files\Git\cmd",
    "C:\Program Files\Git\bin"
)
$toolPaths = @(
    "D:\go\bin",
    "D:\.cargo\bin",
    "D:\kubric-tools",
    "D:\Python312",
    "D:\Python312\Scripts",
    "D:\"
)

$all = ($gitPaths + $toolPaths) -join ";"
$env:PATH = $all + ";" + [System.Environment]::GetEnvironmentVariable("PATH","Machine")

$env:CARGO_HOME    = "D:\.cargo"
$env:RUSTUP_HOME   = "D:\.rustup"
$env:GOPATH        = "D:\go"
$env:MINIKUBE_HOME = "D:\.minikube"
$env:KUBECONFIG    = "D:\.kube\config"
$env:GIT_SSH_COMMAND = "ssh"

Set-Location "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR"
Write-Host "Environment ready. git: $(git --version 2>&1)" -ForegroundColor Green
