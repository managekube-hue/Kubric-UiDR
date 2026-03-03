param(
    [string]$TenantId = "dev-tenant-01",
    [string]$ComposeNetwork = "kubric_kubric",
    [string]$NatsUrl = "nats://nats:4222"
)

$ErrorActionPreference = "Stop"

Write-Host "Testing first-3 module message contracts on network '$ComposeNetwork'" -ForegroundColor Cyan
Write-Host "NATS endpoint: $NatsUrl"

$subjects = @(
    "kubric.$TenantId.endpoint.process.v1",
    "kubric.$TenantId.itdr.auth.v1",
    "kubric.$TenantId.network.activity.v1"
)

foreach ($subj in $subjects) {
    Write-Host "- smoke subject: $subj"

    $subContainer = "nats-sub-" + ([Guid]::NewGuid().ToString("N").Substring(0, 8))
    & docker run -d --name $subContainer --network $ComposeNetwork `
        natsio/nats-box nats --server $NatsUrl sub $subj --count=1 --raw | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start subscriber for subject $subj"
    }

    Start-Sleep -Seconds 2

    $payload = '{"contract":"ok","subject":"' + $subj + '"}'
    $prevErrPref = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $pubOut = (& docker run --rm --network $ComposeNetwork natsio/nats-box `
        nats --server $NatsUrl pub $subj $payload 2>&1 | Out-String).Trim()
    $pubCode = $LASTEXITCODE
    $ErrorActionPreference = $prevErrPref

    if ($pubCode -ne 0) {
        docker rm -f $subContainer | Out-Null
        throw "Publish failed for ${subj}: $pubOut"
    }

    Start-Sleep -Seconds 2

    $status = docker ps -a --filter "name=$subContainer" --format "{{.Status}}"
    $log = (docker logs $subContainer 2>&1 | Out-String).Trim()
    docker rm -f $subContainer | Out-Null

    if ([string]::IsNullOrWhiteSpace($log)) {
        throw "Subscriber timeout for subject $subj (status: $status)"
    }

    Write-Host $pubOut
    Write-Host $log
}

Write-Host "Message contract smoke complete." -ForegroundColor Green
