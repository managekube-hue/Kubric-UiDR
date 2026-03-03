$root = "C:\Users\manag\source\repos\managekube-hue\Kubric-UiDR\vendor"
$skip = @("github.com","golang.org","google.golang.org","go.opentelemetry","go.temporal","go.yaml","cel.dev","cloud.google","dario.cat","mvdan.cc","gocloud.dev","gopkg.in","modules.txt")
Get-ChildItem $root -Recurse -File | ForEach-Object {
    $rel = $_.FullName.Replace($root + "\","")
    $skip_item = $false
    foreach ($s in $skip) { if ($rel -match [regex]::Escape($s)) { $skip_item = $true; break } }
    if (-not $skip_item) {
        Write-Host ("{0,8} {1}" -f $_.Length, $rel)
    }
}
