# Find and report non-ASCII bytes in docker-compose.yml
$file = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR\docker-compose.yml"
$bytes = [System.IO.File]::ReadAllBytes($file)
$lineNum = 1
$col = 0
for ($i = 0; $i -lt $bytes.Length; $i++) {
    if ($bytes[$i] -eq 10) { $lineNum++; $col = 0 }
    $col++
    if ($bytes[$i] -gt 127) {
        Write-Host ("Non-ASCII byte 0x{0:X2} at offset {1} line {2} col {3}" -f $bytes[$i], $i, $lineNum, $col)
    }
}
Write-Host "Scan complete. Total bytes: $($bytes.Length)"
