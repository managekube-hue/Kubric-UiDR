$root = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR"
$files = @(
    "Dockerfile.agents","Dockerfile.api","Dockerfile.web","docker-compose.yml",
    "agents\coresec\src\hooks\ebpf.rs","agents\coresec\src\hooks\etw.rs",
    "agents\perftrace\src\agent.rs","agents\netguard\src\capture.rs",
    "agents\netguard\src\dpi.rs","agents\netguard\src\ids.rs",
    "agents\netguard\src\tls.rs","agents\netguard\src\ipsum_lookup.rs",
    "agents\netguard\src\rita_client.rs","vendor\ebpf\execve_hook.o",
    "vendor\ipsum\ipsum.txt","internal\kic\handler_ciso.go",
    "internal\kic\server.go","internal\kic\store_assessment.go",
    "internal\kic\framework_registry.go","services\grc\ciso_bridge.go",
    "frontend\lib\api-client.ts"
)
Write-Host "`n=== FILE AUDIT ===" -ForegroundColor Cyan
foreach ($f in $files) {
    $exists = Test-Path (Join-Path $root $f)
    $status = if ($exists) { "EXISTS " } else { "MISSING" }
    $color  = if ($exists) { "Green"  } else { "Red"    }
    Write-Host "  $status  $f" -ForegroundColor $color
}
