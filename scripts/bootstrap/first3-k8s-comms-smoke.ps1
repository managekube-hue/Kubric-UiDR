param(
    [string]$TenantId = "dev-tenant-01",
    [string]$Namespace = "kubric"
)

$ErrorActionPreference = "Stop"
$server = "nats://nats.$Namespace.svc.cluster.local:4222"
$subjects = @(
    "kubric.$TenantId.endpoint.process.v1",
    "kubric.$TenantId.itdr.auth.v1",
    "kubric.$TenantId.network.activity.v1"
)

Write-Host "K8s first-3 module comms smoke in namespace '$Namespace'" -ForegroundColor Cyan

foreach ($subj in $subjects) {
    Write-Host "- testing: $subj"
    $payload = '{"contract":"ok","subject":"' + $subj + '"}'
    $pod = "nats-smoke-" + ([Guid]::NewGuid().ToString("N").Substring(0, 8))
    $cmd = "nats --server $server sub $subj --count=1 --raw >/tmp/msg.out & pid=\$!; sleep 2; nats --server $server pub $subj '$payload'; wait \$pid; cat /tmp/msg.out"

    $prevErr = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $out = (kubectl run $pod -n $Namespace --rm -i --restart=Never --image=natsio/nats-box -- sh -c $cmd 2>&1 | Out-String)
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prevErr

    if ($code -ne 0) {
        throw "k8s comm smoke failed for $subj : $out"
    }

    if ([string]::IsNullOrWhiteSpace($out) -or -not ($out -match "contract")) {
        throw "k8s comm smoke missing payload for $subj"
    }

    Write-Host $out
}

Write-Host "K8s communication smoke complete." -ForegroundColor Green
