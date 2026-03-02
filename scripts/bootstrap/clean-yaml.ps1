# Strip all non-ASCII bytes from docker-compose.yml
$file = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR\docker-compose.yml"
$content = [System.IO.File]::ReadAllText($file, [System.Text.Encoding]::UTF8)
$clean = [System.Text.RegularExpressions.Regex]::Replace($content, '[^\x00-\x7F]', '-')
[System.IO.File]::WriteAllText($file, $clean, (New-Object System.Text.UTF8Encoding $false))
Write-Host "Cleaned. Verifying..."
$bytes = [System.IO.File]::ReadAllBytes($file)
$bad = $bytes | Where-Object { $_ -gt 127 }
if ($bad) {
    Write-Host ("Still has {0} bad bytes" -f $bad.Count) -ForegroundColor Red
} else {
    Write-Host "No non-ASCII bytes remaining - file is clean ASCII" -ForegroundColor Green
}
