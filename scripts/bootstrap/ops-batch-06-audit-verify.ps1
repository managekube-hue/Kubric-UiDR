param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

function Assert-True {
    param(
        [Parameter(Mandatory = $true)][bool]$Condition,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if (-not $Condition) {
        throw "[batch-06] FAIL: $Message"
    }

    Write-Host "[batch-06] OK: $Message" -ForegroundColor Green
}

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

Write-Host "[batch-06] Verifying remediation of audit gaps..." -ForegroundColor Cyan

$ebpfExecObj = Join-Path $RepoRoot "vendor\ebpf\execve_hook.o"
$ebpfOpenObj = Join-Path $RepoRoot "vendor\ebpf\openat2_hook.o"
$agentsDocker = Get-Content (Join-Path $RepoRoot "Dockerfile.agents") -Raw
$ebpfRs = Get-Content (Join-Path $RepoRoot "agents\coresec\src\hooks\ebpf.rs") -Raw
$etwRs = Get-Content (Join-Path $RepoRoot "agents\coresec\src\hooks\etw.rs") -Raw
$perfAgent = Get-Content (Join-Path $RepoRoot "agents\perftrace\src\agent.rs") -Raw

$hasEbpfObjects = (Test-Path $ebpfExecObj) -and (Test-Path $ebpfOpenObj)
$dockerBuildsEbpf = $agentsDocker -match [regex]::Escape("make -C vendor/ebpf")
$ebpfAutoBuildCode = $ebpfRs -match [regex]::Escape("ensure_ebpf_object")
Assert-True -Condition ($hasEbpfObjects -or ($dockerBuildsEbpf -and $ebpfAutoBuildCode)) -Message "eBPF object availability is handled (objects present or auto-build path exists)"

Assert-True -Condition ($etwRs -notmatch [regex]::Escape("WMI fallback")) -Message "Windows hook file no longer uses legacy WMI fallback placeholder"
Assert-True -Condition ($etwRs -match [regex]::Escape("blocking_send")) -Message "Windows hook emits process events"

Assert-True -Condition ($perfAgent -notmatch "disk_read_bytes:\s*0") -Message "PerfTrace no longer hardcodes disk_read_bytes to zero"
Assert-True -Condition ($perfAgent -notmatch "disk_write_bytes:\s*0") -Message "PerfTrace no longer hardcodes disk_write_bytes to zero"
Assert-True -Condition ($perfAgent -match [regex]::Escape("collect_disk_io_bytes")) -Message "PerfTrace has disk I/O collector function"

$yaraCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\yara-rules") -Recurse -File -Filter *.yar -ErrorAction SilentlyContinue).Count
$sigmaYmlCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\sigma\rules") -Recurse -File -Filter *.yml -ErrorAction SilentlyContinue).Count
$sigmaYamlCount = (Get-ChildItem -Path (Join-Path $RepoRoot "vendor\sigma\rules") -Recurse -File -Filter *.yaml -ErrorAction SilentlyContinue).Count

Assert-True -Condition ($yaraCount -gt 0) -Message "YARA rules are populated ($yaraCount)"
Assert-True -Condition (($sigmaYmlCount + $sigmaYamlCount) -gt 0) -Message "Sigma rules are populated ($($sigmaYmlCount + $sigmaYamlCount))"

Write-Host "[batch-06] Audit remediation verification PASSED" -ForegroundColor Green