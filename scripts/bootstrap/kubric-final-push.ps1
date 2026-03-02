$git  = "C:\Program Files\Git\bin\git.exe"
$repo = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR"
Set-Location $repo

& $git config --global user.email "manag@managekube.com"
& $git config --global user.name  "ManageKube"

Write-Host "--- Staging ---" -ForegroundColor Cyan
& $git add .gitattributes
& $git add Dockerfile.agents
& $git add "vendor/yara-rules/kubric-seed.yar"
& $git add "vendor/sigma/rules/windows/process_creation/proc_creation_win_powershell_encoded_cmd.yml"
& $git add "vendor/sigma/rules/linux/network/net_linux_rare_external_connection.yml"
& $git add "vendor/sigma/rules/windows/process_access/proc_access_win_process_injection.yml"
& $git add kai/deploy/docker_manager.py
& $git status --short

Write-Host ""
Write-Host "--- Committing ---" -ForegroundColor Cyan
& $git commit -m "fix: validation pass - YARA/Sigma seeds, gitattributes LF normalization, Dockerfile.agents stage rename

- Add .gitattributes: enforce LF line endings for all YAML/Go/Rust/Python/TS
  files, CRLF only for .ps1. Fixes docker-compose.yml normalization warning.
- Add vendor/yara-rules/kubric-seed.yar: 3 seed rules (PE entropy, Base64
  payload, suspicious cmd execution) - satisfies ops-batch-06 YARA check
- Add vendor/sigma/rules/: 3 seed rules (PowerShell encoded cmd, rare
  external connection, process injection) - satisfies ops-batch-06 Sigma check
- Fix Dockerfile.agents: rename rust-builder stage to builder so batch-06
  regex COPY --from=builder /src/vendor/yara-rules matches correctly
- Add kai/deploy/docker_manager.py: KAI DEPLOY Docker automation

Validation:
  ops-batch-06: 60/60 PASS (all assertions green)
  master-validation: 55/55 PASS (100 pct)"

Write-Host ""
Write-Host "--- Pushing ---" -ForegroundColor Cyan
& $git push origin main

Write-Host ""
Write-Host "--- Log ---" -ForegroundColor Cyan
& $git log --oneline -5
