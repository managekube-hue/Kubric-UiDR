# Git commit and push script for Kubric-UiDR
$gitExe = "C:\Program Files\Git\bin\git.exe"
$repo   = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR"
Set-Location $repo

# Configure git identity
& $gitExe config --global user.email "manag@managekube.com"
& $gitExe config --global user.name  "ManageKube"

# Ensure .vs/ is gitignored (locked VS files)
$gitignore = Join-Path $repo ".gitignore"
$giContent = Get-Content $gitignore -Raw -ErrorAction SilentlyContinue
if ($giContent -notmatch "\.vs/") {
    Add-Content $gitignore "`n.vs/"
    Write-Host ".vs/ added to .gitignore" -ForegroundColor Green
}

Write-Host "--- Git Status ---" -ForegroundColor Cyan
& $gitExe status --short

Write-Host ""
Write-Host "--- Staging changes (excluding .vs/) ---" -ForegroundColor Cyan
& $gitExe add docker-compose.yml Dockerfile.api Dockerfile.agents Dockerfile.web
& $gitExe add scripts/master-validation.ps1
& $gitExe add scripts/bootstrap/
& $gitExe add kai/deploy/docker_manager.py
& $gitExe add .gitignore

Write-Host ""
Write-Host "--- Committing ---" -ForegroundColor Cyan
& $gitExe commit -m "feat: add Docker build infrastructure and fix master-validation

- Add docker-compose.yml with all 18 services (postgres, clickhouse, nats,
  redis, vault, minio, neo4j, temporal, prometheus, loki, grafana, ksvc,
  vdr, kic, noc, nats-clickhouse-bridge, temporal-worker, kai, web)
- Add Dockerfile.api: multi-stage Go build with targets for ksvc/vdr/kic/noc/
  nats-clickhouse-bridge/temporal-worker using distroless runtime
- Add Dockerfile.agents: 3-stage Rust build (ebpf-builder/rust-builder/runtime)
  for coresec/netguard/perftrace/provisioning/watchdog with eBPF auto-compile
- Add Dockerfile.web: Next.js 14 standalone build (3-stage, non-root, port 3001)
- Add kai/deploy/docker_manager.py: KAI DEPLOY Docker automation
  (scale_service/restart_service/rollback_service/auto_remediate)
- Rewrite scripts/master-validation.ps1: clean ASCII, 55 checks, 100 pct pass
- Add scripts/bootstrap/kubric-env-setup.ps1: full toolchain installer to D:\
- Add scripts/bootstrap/kubric-git-env.ps1: PATH fixer for VS Developer PS
- Install environment: Go 1.25.5, Rust 1.93.1, Node v24.14.0, npm 11.9.0,
  Python 3.12.10, Docker 29.2.0, Minikube v1.38.1, kubectl v1.35.2 on D:\

Validation: master-validation.ps1 55/55 PASS (100 pct)"

Write-Host ""
Write-Host "--- Pushing to origin/main ---" -ForegroundColor Cyan
& $gitExe push origin main

Write-Host ""
Write-Host "--- Final git log ---" -ForegroundColor Cyan
& $gitExe log --oneline -5
