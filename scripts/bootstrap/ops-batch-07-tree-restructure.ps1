<#
.SYNOPSIS
    ops-batch-07 — Enterprise tree restructure verification
.DESCRIPTION
    Validates all changes from the enterprise tree restructure:
    1. K8s infra archived to archive/k8s-legacy/
    2. Vendor/ITIL/MAP specs collapsed into docs/ and archived
    3. .kubric/ config files created
    4. docker-compose.prod.yml created with production hardening
    5. Makefile deploy-prod updated for Docker Compose
    6. No build system references broken
#>
param(
    [string]$Root = (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'
$pass = 0; $fail = 0; $total = 0

function Assert-True {
    param([string]$Label, [bool]$Condition)
    $script:total++
    if ($Condition) {
        $script:pass++
        Write-Host "  PASS  $Label" -ForegroundColor Green
    } else {
        $script:fail++
        Write-Host "  FAIL  $Label" -ForegroundColor Red
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  ops-batch-07: Tree Restructure Gate"   -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

Push-Location $Root

# ── 1. K8s archive ─────────────────────────────────────────────────────────
Write-Host "--- K8s Archive ---" -ForegroundColor Yellow
Assert-True "archive/k8s-legacy/k8s exists"                  (Test-Path "archive/k8s-legacy/k8s")
Assert-True "archive/k8s-legacy/authentik exists"             (Test-Path "archive/k8s-legacy/authentik")
Assert-True "archive/k8s-legacy/deployments-k8s exists"       (Test-Path "archive/k8s-legacy/deployments-k8s")
Assert-True "infra/k8s removed from active tree"              (-not (Test-Path "infra/k8s"))
Assert-True "infra/authentik removed from active tree"        (-not (Test-Path "infra/authentik"))
Assert-True "deployments/k8s removed from active tree"        (-not (Test-Path "deployments/k8s"))

# ── 2. Spec module collapse ───────────────────────────────────────────────
Write-Host "`n--- Spec Collapse ---" -ForegroundColor Yellow
Assert-True "docs/VENDOR-INTEGRATIONS.md exists"              (Test-Path "docs/VENDOR-INTEGRATIONS.md")
Assert-True "docs/COMPLIANCE.md exists"                       (Test-Path "docs/COMPLIANCE.md")
Assert-True "docs/DR-COVERAGE.md exists"                      (Test-Path "docs/DR-COVERAGE.md")

Assert-True "VENDOR-INTEGRATIONS.md > 5KB"                    ((Get-Item "docs/VENDOR-INTEGRATIONS.md").Length -gt 5000)
Assert-True "COMPLIANCE.md > 3KB"                              ((Get-Item "docs/COMPLIANCE.md").Length -gt 3000)
Assert-True "DR-COVERAGE.md > 5KB"                             ((Get-Item "docs/DR-COVERAGE.md").Length -gt 5000)

# Originals archived
Assert-True "00_K-VENDOR archived"                            (Test-Path "archive/spec-modules/00_K-VENDOR-00_DETECTION_ASSETS")
Assert-True "10_K-ITIL archived"                              (Test-Path "archive/spec-modules/10_K-ITIL-10_ITIL_MATRIX")
Assert-True "11_K-MAP archived"                               (Test-Path "archive/spec-modules/11_K-MAP-11_DR_MODULE_MAPPING")

# Originals gone from root
Assert-True "00_K-VENDOR removed from root"                   (-not (Test-Path "00_K-VENDOR-00_DETECTION_ASSETS"))
Assert-True "10_K-ITIL removed from root"                     (-not (Test-Path "10_K-ITIL-10_ITIL_MATRIX"))
Assert-True "11_K-MAP removed from root"                      (-not (Test-Path "11_K-MAP-11_DR_MODULE_MAPPING"))

# ── 3. .kubric/ configs ───────────────────────────────────────────────────
Write-Host "`n--- .kubric/ Configs ---" -ForegroundColor Yellow
Assert-True ".kubric/stack.yaml exists"                       (Test-Path ".kubric/stack.yaml")
Assert-True ".kubric/deployment-rules.yaml exists"            (Test-Path ".kubric/deployment-rules.yaml")
Assert-True ".kubric/health-checks.yaml exists"               (Test-Path ".kubric/health-checks.yaml")
Assert-True "stack.yaml > 1KB"                                ((Get-Item ".kubric/stack.yaml").Length -gt 1000)
Assert-True "deployment-rules.yaml > 1KB"                     ((Get-Item ".kubric/deployment-rules.yaml").Length -gt 1000)
Assert-True "health-checks.yaml > 1KB"                        ((Get-Item ".kubric/health-checks.yaml").Length -gt 1000)

# ── 4. docker-compose.prod.yml ─────────────────────────────────────────────
Write-Host "`n--- Production Compose ---" -ForegroundColor Yellow
Assert-True "docker-compose.prod.yml exists"                  (Test-Path "docker-compose.prod.yml")
Assert-True "prod compose > 10KB"                              ((Get-Item "docker-compose.prod.yml").Length -gt 10000)

$prodContent = Get-Content "docker-compose.prod.yml" -Raw
Assert-True "prod compose has watchtower"                     ($prodContent -match "watchtower")
Assert-True "prod compose has resource limits"                ($prodContent -match "limits:")
Assert-True "prod compose has replicas: 3"                    ($prodContent -match "replicas: 3")
Assert-True "prod compose has deploy-api anchor"              ($prodContent -match "deploy-api")
Assert-True "prod compose has json-file logging"              ($prodContent -match "json-file")
Assert-True "prod compose has max-size"                       ($prodContent -match "max-size")
Assert-True "prod compose has health checks"                  ($prodContent -match "healthcheck:")
Assert-True "prod compose has no 'profiles:' gate"            (-not ($prodContent -match "^\s+profiles:"))

# ── 5. Makefile ────────────────────────────────────────────────────────────
Write-Host "`n--- Makefile ---" -ForegroundColor Yellow
$makeContent = Get-Content "Makefile" -Raw
Assert-True "deploy-prod uses docker compose"                 ($makeContent -match "docker compose -f docker-compose.prod.yml")
Assert-True "deploy-prod no kustomize reference"              (-not ($makeContent -match "deploy-prod:[\s\S]*?kustomize build"))
Assert-True "ops-batch-07 target exists"                      ($makeContent -match "ops-batch-07:")

# ── 6. Build system integrity ─────────────────────────────────────────────
Write-Host "`n--- Build Integrity ---" -ForegroundColor Yellow
Assert-True "docker-compose.yml still exists"                 (Test-Path "docker-compose.yml")
Assert-True "Dockerfile.api exists"                           (Test-Path "Dockerfile.api")
Assert-True "Dockerfile.web exists"                           (Test-Path "Dockerfile.web")
Assert-True "Dockerfile.agents exists"                        (Test-Path "Dockerfile.agents")
Assert-True "build/ directory exists"                         (Test-Path "build")
Assert-True "go.mod exists"                                   (Test-Path "go.mod")
Assert-True "Cargo.toml exists"                               (Test-Path "Cargo.toml")

# Verify no Go imports reference archived paths
$goImportIssues = Select-String -Path "internal/**/*.go","cmd/**/*.go" -Pattern "00_K-VENDOR|10_K-ITIL|11_K-MAP|infra/k8s|deployments/k8s" -ErrorAction SilentlyContinue
Assert-True "No Go imports reference archived paths"          ($null -eq $goImportIssues -or $goImportIssues.Count -eq 0)

# Verify no Dockerfile references archived paths
$dockerIssues = Select-String -Path "Dockerfile*","build/*/Dockerfile" -Pattern "00_K-VENDOR|10_K-ITIL|11_K-MAP" -ErrorAction SilentlyContinue
Assert-True "No Dockerfiles reference archived paths"         ($null -eq $dockerIssues -or $dockerIssues.Count -eq 0)

# ── 7. Vendor content validation ──────────────────────────────────────────
Write-Host "`n--- Vendor Doc Content ---" -ForegroundColor Yellow
$vendorDoc = Get-Content "docs/VENDOR-INTEGRATIONS.md" -Raw
Assert-True "VENDOR doc covers Sigma"                         ($vendorDoc -match "Sigma")
Assert-True "VENDOR doc covers YARA"                          ($vendorDoc -match "YARA")
Assert-True "VENDOR doc covers Suricata"                      ($vendorDoc -match "Suricata")
Assert-True "VENDOR doc covers Wazuh"                         ($vendorDoc -match "Wazuh")
Assert-True "VENDOR doc covers BloodHound"                    ($vendorDoc -match "BloodHound")
Assert-True "VENDOR doc covers Cortex"                        ($vendorDoc -match "Cortex")
Assert-True "VENDOR doc covers TheHive"                       ($vendorDoc -match "TheHive")
Assert-True "VENDOR doc covers Velociraptor"                  ($vendorDoc -match "Velociraptor")
Assert-True "VENDOR doc covers MITRE"                         ($vendorDoc -match "MITRE")
Assert-True "VENDOR doc covers Nuclei"                        ($vendorDoc -match "Nuclei")
Assert-True "VENDOR doc covers OSCAL"                         ($vendorDoc -match "OSCAL")
Assert-True "VENDOR doc covers Zeek"                          ($vendorDoc -match "Zeek")
Assert-True "VENDOR doc covers Falco"                         ($vendorDoc -match "Falco")

# ── 8. GRC Framework validation ──────────────────────────────────────────
Write-Host "`n--- GRC Framework Coverage ---" -ForegroundColor Yellow
Assert-True "07_K-GRC-07_COMPLIANCE exists"                   (Test-Path "07_K-GRC-07_COMPLIANCE")
Assert-True "GRC framework index exists"                      (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-000_framework_index.md")
Assert-True "NIST 800-53 OSCAL exists"                        (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal.md")
Assert-True "PCI-DSS OSCAL exists"                            (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal.md")
Assert-True "ISO 27001 OSCAL exists"                          (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal.md")
Assert-True "SOC 2 OSCAL exists"                              (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal.md")

$grcIndex = Get-Content "07_K-GRC-07_COMPLIANCE/K-GRC-FW-000_framework_index.md" -Raw
Assert-True "GRC index mentions 200 frameworks"               ($grcIndex -match "200")
Assert-True "GRC index covers NIST 800-53"                    ($grcIndex -match "NIST 800-53")
Assert-True "GRC index covers PCI DSS"                        ($grcIndex -match "PCI DSS")

# ── Summary ────────────────────────────────────────────────────────────────
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  PASS: $pass / $total" -ForegroundColor Green
Write-Host "  FAIL: $fail / $total" -ForegroundColor Red
Write-Host "========================================`n" -ForegroundColor Cyan

Pop-Location

if ($fail -gt 0) {
    exit 1
}
exit 0─────────────────────────────────────────
Write-Host "`n--- GRC Framework Coverage ---" -ForegroundColor Yellow
Assert-True "07_K-GRC-07_COMPLIANCE exists"                   (Test-Path "07_K-GRC-07_COMPLIANCE")
Assert-True "GRC framework index exists"                      (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-000_framework_index.md")
Assert-True "NIST 800-53 OSCAL exists"                        (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal.md")
Assert-True "PCI-DSS OSCAL exists"                            (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal.md")
Assert-True "ISO 27001 OSCAL exists"                          (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal.md")
Assert-True "SOC 2 OSCAL exists"                              (Test-Path "07_K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal.md")

$grcIndex = Get-Content "07_K-GRC-07_COMPLIANCE/K-GRC-FW-000_framework_index.md" -Raw
Assert-True "GRC index mentions 200 frameworks"               ($grcIndex -match "200")
Assert-True "GRC index covers NIST 800-53"                    ($grcIndex -match "NIST 800-53")
Assert-True "GRC index covers PCI DSS"                        ($grcIndex -match "PCI DSS")e boundaries"          ($vendorDoc -match "[Ll]icense")

# ── 8. ITIL/Compliance content ────────────────────────────────────────────
Write-Host "`n--- ITIL/Compliance Doc Content ---" -ForegroundColor Yellow
$complianceDoc = Get-Content "docs/COMPLIANCE.md" -Raw
Assert-True "COMPLIANCE doc covers ITIL 4"                   ($complianceDoc -match "ITIL")
Assert-True "COMPLIANCE doc covers NIST 800-53"              ($complianceDoc -match "NIST")
Assert-True "COMPLIANCE doc covers ISO 27001"                ($complianceDoc -match "ISO")
Assert-True "COMPLIANCE doc covers SOC 2"                    ($complianceDoc -match "SOC")
Assert-True "COMPLIANCE doc covers PCI-DSS"                  ($complianceDoc -match "PCI")

# ── 9. DR Coverage content ────────────────────────────────────────────────
Write-Host "`n--- DR Coverage Doc Content ---" -ForegroundColor Yellow
$drDoc = Get-Content "docs/DR-COVERAGE.md" -Raw
Assert-True "DR doc covers EDR"                              ($drDoc -match "EDR")
Assert-True "DR doc covers NDR"                              ($drDoc -match "NDR")
Assert-True "DR doc covers ITDR"                             ($drDoc -match "ITDR")
Assert-True "DR doc covers CDR"                              ($drDoc -match "CDR")
Assert-True "DR doc covers VDR"                              ($drDoc -match "VDR")
Assert-True "DR doc covers MDR"                              ($drDoc -match "MDR")

# ── 10. Archive integrity ─────────────────────────────────────────────────
Write-Host "`n--- Archive Integrity ---" -ForegroundColor Yellow
Assert-True "archive/spec-modules has 107+ files"            ((Get-ChildItem -Path "archive/spec-modules" -Recurse -File).Count -ge 107)
Assert-True "archive/k8s-legacy has content"                 ((Get-ChildItem -Path "archive/k8s-legacy" -Recurse -File).Count -gt 10)
Assert-True "docs/archive has historical docs"               ((Get-ChildItem -Path "docs/archive" -File).Count -ge 10)

Pop-Location

# ── Summary ────────────────────────────────────────────────────────────────
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  ops-batch-07 Results: $pass/$total passed" -ForegroundColor $(if ($fail -eq 0) { 'Green' } else { 'Yellow' })
Write-Host "========================================`n" -ForegroundColor Cyan

if ($fail -gt 0) {
    Write-Host "[batch-07] Tree restructure verification FAILED ($fail failures)" -ForegroundColor Red
    exit 1
} else {
    Write-Host "[batch-07] Tree restructure verification PASSED" -ForegroundColor Green
    exit 0
}
