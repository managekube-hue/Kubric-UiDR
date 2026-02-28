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
        throw "[batch-07] FAIL: $Message"
    }

    Write-Host "[batch-07] OK: $Message" -ForegroundColor Green
}

if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
}

Write-Host "[batch-07] Verifying Go dependency wiring — 7 libraries..." -ForegroundColor Cyan

# ── 1. Neo4j graph store ─────────────────────────────────────────────────────

$neo4jFile = Join-Path $RepoRoot "internal\neo4j\graph_store.go"
Assert-True -Condition (Test-Path $neo4jFile) -Message "Neo4j graph store package exists (internal/neo4j/graph_store.go)"

$neo4jContent = Get-Content $neo4jFile -Raw
Assert-True -Condition ($neo4jContent -match 'github\.com/neo4j/neo4j-go-driver/v5/neo4j') -Message "Neo4j package imports neo4j-go-driver/v5"
Assert-True -Condition ($neo4jContent -match 'neo4j\.NewDriverWithContext') -Message "Neo4j package calls NewDriverWithContext"
Assert-True -Condition ($neo4jContent -match 'UpsertAsset') -Message "Neo4j package has UpsertAsset method"
Assert-True -Condition ($neo4jContent -match 'BlastRadius') -Message "Neo4j package has BlastRadius method"
Assert-True -Condition ($neo4jContent -match 'Topology') -Message "Neo4j package has Topology method"

# Verify NOC server wires Neo4j
$nocServer = Get-Content (Join-Path $RepoRoot "internal\noc\server.go") -Raw
Assert-True -Condition ($nocServer -match 'internal/neo4j') -Message "NOC server.go imports internal/neo4j"
Assert-True -Condition ($nocServer -match 'graphStore') -Message "NOC server struct has graphStore field"

# ── 2. OpenSSF Scorecard ─────────────────────────────────────────────────────

$scorecardFile = Join-Path $RepoRoot "internal\scorecard\runner.go"
Assert-True -Condition (Test-Path $scorecardFile) -Message "Scorecard runner package exists (internal/scorecard/runner.go)"

$scorecardContent = Get-Content $scorecardFile -Raw
Assert-True -Condition ($scorecardContent -match 'github\.com/ossf/scorecard/v4') -Message "Scorecard package imports ossf/scorecard/v4"
Assert-True -Condition ($scorecardContent -match 'func.*Score\(') -Message "Scorecard package has Score method"
Assert-True -Condition ($scorecardContent -match 'ScoreMultiple') -Message "Scorecard package has ScoreMultiple method"

# Verify KIC server wires scorecard
$kicServer = Get-Content (Join-Path $RepoRoot "internal\kic\server.go") -Raw
Assert-True -Condition ($kicServer -match 'internal/scorecard') -Message "KIC server.go imports internal/scorecard"
Assert-True -Condition ($kicServer -match 'scorecard\.NewRunner') -Message "KIC server creates Scorecard runner"

# Verify supply-chain handler
$scHandler = Join-Path $RepoRoot "internal\kic\handler_supply_chain.go"
Assert-True -Condition (Test-Path $scHandler) -Message "Supply-chain handler exists (internal/kic/handler_supply_chain.go)"
$scContent = Get-Content $scHandler -Raw
Assert-True -Condition ($scContent -match 'runScorecard') -Message "Supply-chain handler has runScorecard method"
Assert-True -Condition ($scContent -match '/supply-chain/scorecard') -Message "Supply-chain scorecard route documented"

# ── 3. MinIO object store ────────────────────────────────────────────────────

$minioFile = Join-Path $RepoRoot "internal\storage\minio.go"
Assert-True -Condition (Test-Path $minioFile) -Message "MinIO storage package exists (internal/storage/minio.go)"

$minioContent = Get-Content $minioFile -Raw
Assert-True -Condition ($minioContent -match 'github\.com/minio/minio-go/v7') -Message "Storage package imports minio-go/v7"
Assert-True -Condition ($minioContent -match 'minio\.New\(') -Message "Storage package calls minio.New"
Assert-True -Condition ($minioContent -match 'PutObject') -Message "Storage package has PutObject"
Assert-True -Condition ($minioContent -match 'GetObject') -Message "Storage package has GetObject"
Assert-True -Condition ($minioContent -match 'ListObjects') -Message "Storage package has ListObjects"
Assert-True -Condition ($minioContent -match 'PresignedGet') -Message "Storage package has PresignedGet"
Assert-True -Condition ($minioContent -match 'BucketEvidence') -Message "Storage package defines BucketEvidence constant"
Assert-True -Condition ($minioContent -match 'BucketSBOM') -Message "Storage package defines BucketSBOM constant"
Assert-True -Condition ($minioContent -match 'EnsureDefaultBuckets') -Message "Storage package has EnsureDefaultBuckets"

# Verify NOC server wires MinIO
Assert-True -Condition ($nocServer -match 'internal/storage') -Message "NOC server.go imports internal/storage"
Assert-True -Condition ($nocServer -match 'objectStore') -Message "NOC server struct has objectStore field"

# ── 4. DuckDB analytics ─────────────────────────────────────────────────────

$duckdbFile = Join-Path $RepoRoot "internal\analytics\duckdb.go"
Assert-True -Condition (Test-Path $duckdbFile) -Message "DuckDB analytics package exists (internal/analytics/duckdb.go)"

$duckdbContent = Get-Content $duckdbFile -Raw
Assert-True -Condition ($duckdbContent -match 'github\.com/marcboeker/go-duckdb') -Message "Analytics package imports go-duckdb"
Assert-True -Condition ($duckdbContent -match 'sql\.Open\("duckdb"') -Message "Analytics package opens DuckDB via database/sql"
Assert-True -Condition ($duckdbContent -match 'IngestEvent') -Message "Analytics package has IngestEvent"
Assert-True -Condition ($duckdbContent -match 'EventSummaryByHour') -Message "Analytics package has EventSummaryByHour"
Assert-True -Condition ($duckdbContent -match 'ComplianceTrendDaily') -Message "Analytics package has ComplianceTrendDaily"
Assert-True -Condition ($duckdbContent -match 'IngestMetric') -Message "Analytics package has IngestMetric"

# Verify KIC server wires DuckDB
Assert-True -Condition ($kicServer -match 'internal/analytics') -Message "KIC server.go imports internal/analytics"
Assert-True -Condition ($kicServer -match 'analytics\.New') -Message "KIC server creates DuckDB analytics engine"

# Verify analytics handler
$anHandler = Join-Path $RepoRoot "internal\kic\handler_analytics.go"
Assert-True -Condition (Test-Path $anHandler) -Message "Analytics handler exists (internal/kic/handler_analytics.go)"
$anContent = Get-Content $anHandler -Raw
Assert-True -Condition ($anContent -match 'eventSummary') -Message "Analytics handler has eventSummary"
Assert-True -Condition ($anContent -match 'complianceTrend') -Message "Analytics handler has complianceTrend"

# ── 5. Sigstore verifier ────────────────────────────────────────────────────

$sigstoreFile = Join-Path $RepoRoot "internal\sigstore\verifier.go"
Assert-True -Condition (Test-Path $sigstoreFile) -Message "Sigstore verifier package exists (internal/sigstore/verifier.go)"

$sigstoreContent = Get-Content $sigstoreFile -Raw
Assert-True -Condition ($sigstoreContent -match 'github\.com/sigstore/sigstore') -Message "Sigstore package imports sigstore/sigstore"
Assert-True -Condition ($sigstoreContent -match 'signature\.LoadECDSAVerifier') -Message "Sigstore package loads ECDSA verifier"
Assert-True -Condition ($sigstoreContent -match 'VerifyImageSignature') -Message "Sigstore package has VerifyImageSignature"

# Verify KIC server wires Sigstore
Assert-True -Condition ($kicServer -match 'internal/sigstore') -Message "KIC server.go imports internal/sigstore"
Assert-True -Condition ($kicServer -match 'NewVerifier') -Message "KIC server creates Sigstore verifier"

# Verify supply-chain handler
Assert-True -Condition ($scContent -match 'verifyImage') -Message "Supply-chain handler has verifyImage method"

# ── 6. ZMQ4 — already imported, now wired into NOC ──────────────────────────

$zmqFile = Join-Path $RepoRoot "internal\messaging\zmq.go"
Assert-True -Condition (Test-Path $zmqFile) -Message "ZMQ messaging package exists (internal/messaging/zmq.go)"

$zmqContent = Get-Content $zmqFile -Raw
Assert-True -Condition ($zmqContent -match 'github\.com/go-zeromq/zmq4') -Message "ZMQ package imports go-zeromq/zmq4"
Assert-True -Condition ($zmqContent -match 'NewPublisher') -Message "ZMQ package has NewPublisher"
Assert-True -Condition ($zmqContent -match 'NewSubscriber') -Message "ZMQ package has NewSubscriber"

# Verify NOC server wires ZMQ
Assert-True -Condition ($nocServer -match 'internal/messaging') -Message "NOC server.go imports internal/messaging"
Assert-True -Condition ($nocServer -match 'zmqPub') -Message "NOC server struct has zmqPub field"
Assert-True -Condition ($nocServer -match 'messaging\.NewPublisher') -Message "NOC server creates ZMQ publisher"

# Verify NOC config has ZMQ fields
$nocConfig = Get-Content (Join-Path $RepoRoot "internal\noc\config.go") -Raw
Assert-True -Condition ($nocConfig -match 'ZMQPublishAddr') -Message "NOC config has ZMQPublishAddr field"

# ── 7. Twilio — already imported, now wired into NOC ────────────────────────

$twilioFile = Join-Path $RepoRoot "internal\alerting\twilio.go"
Assert-True -Condition (Test-Path $twilioFile) -Message "Twilio alerting package exists (internal/alerting/twilio.go)"

$twilioContent = Get-Content $twilioFile -Raw
Assert-True -Condition ($twilioContent -match 'github\.com/twilio/twilio-go') -Message "Twilio package imports twilio-go"
Assert-True -Condition ($twilioContent -match 'NewTwilioAlerter') -Message "Twilio package has NewTwilioAlerter"
Assert-True -Condition ($twilioContent -match 'SendSMS') -Message "Twilio package has SendSMS"

# Verify NOC server wires Twilio
Assert-True -Condition ($nocServer -match 'internal/alerting') -Message "NOC server.go imports internal/alerting"
Assert-True -Condition ($nocServer -match 'twilioAlerter') -Message "NOC server struct has twilioAlerter field"
Assert-True -Condition ($nocServer -match 'alerting\.NewTwilioAlerter') -Message "NOC server creates TwilioAlerter"

# Verify NOC config has Twilio fields
Assert-True -Condition ($nocConfig -match 'TwilioAccountSID') -Message "NOC config has TwilioAccountSID field"
Assert-True -Condition ($nocConfig -match 'TwilioAuthToken') -Message "NOC config has TwilioAuthToken field"
Assert-True -Condition ($nocConfig -match 'TwilioFromNumber') -Message "NOC config has TwilioFromNumber field"

# ── Cross-check: all 7 libs declared in go.mod ──────────────────────────────

$gomod = Get-Content (Join-Path $RepoRoot "go.mod") -Raw
Assert-True -Condition ($gomod -match 'neo4j/neo4j-go-driver/v5') -Message "go.mod declares neo4j-go-driver/v5"
Assert-True -Condition ($gomod -match 'ossf/scorecard/v4') -Message "go.mod declares ossf/scorecard/v4"
Assert-True -Condition ($gomod -match 'minio/minio-go/v7') -Message "go.mod declares minio-go/v7"
Assert-True -Condition ($gomod -match 'marcboeker/go-duckdb') -Message "go.mod declares go-duckdb"
Assert-True -Condition ($gomod -match 'sigstore/sigstore') -Message "go.mod declares sigstore/sigstore"
Assert-True -Condition ($gomod -match 'go-zeromq/zmq4') -Message "go.mod declares go-zeromq/zmq4"
Assert-True -Condition ($gomod -match 'twilio/twilio-go') -Message "go.mod declares twilio-go"

# ── Route wiring checks ─────────────────────────────────────────────────────

# KIC supply-chain + analytics routes
Assert-True -Condition ($kicServer -match '/supply-chain') -Message "KIC server wires /supply-chain routes"
Assert-True -Condition ($kicServer -match '/analytics') -Message "KIC server wires /analytics routes"

# NOC graph + storage routes
Assert-True -Condition ($nocServer -match '/graph') -Message "NOC server wires /graph routes"
Assert-True -Condition ($nocServer -match '/storage') -Message "NOC server wires /storage routes"

# NOC graph handler
$graphHandler = Join-Path $RepoRoot "internal\noc\handler_graph.go"
Assert-True -Condition (Test-Path $graphHandler) -Message "NOC graph handler exists (internal/noc/handler_graph.go)"

# NOC storage handler
$storageHandler = Join-Path $RepoRoot "internal\noc\handler_storage.go"
Assert-True -Condition (Test-Path $storageHandler) -Message "NOC storage handler exists (internal/noc/handler_storage.go)"

# ── Docker image checks ─────────────────────────────────────────────────────

$dockerApi = Get-Content (Join-Path $RepoRoot "Dockerfile.api") -Raw
$dockerCompose = Get-Content (Join-Path $RepoRoot "docker-compose.yml") -Raw
$buildKicDockerfile = Get-Content (Join-Path $RepoRoot "build\kic\Dockerfile") -Raw

# KIC Dockerfile enables CGO for go-duckdb
Assert-True -Condition ($buildKicDockerfile -match 'CGO_ENABLED=1') -Message "build/kic/Dockerfile enables CGO (required for go-duckdb)"
Assert-True -Condition ($buildKicDockerfile -match 'gcc') -Message "build/kic/Dockerfile installs gcc for CGO"

# Dockerfile.api also enables CGO for KIC target only
Assert-True -Condition ($dockerApi -match 'CGO_ENABLED=1.*kic') -Message "Dockerfile.api uses CGO_ENABLED=1 for KIC binary"

# docker-compose: KIC env vars for new integrations
Assert-True -Condition ($dockerCompose -match 'GITHUB_AUTH_TOKEN') -Message "docker-compose passes GITHUB_AUTH_TOKEN to KIC"
Assert-True -Condition ($dockerCompose -match 'DUCKDB_PATH') -Message "docker-compose passes DUCKDB_PATH to KIC"
Assert-True -Condition ($dockerCompose -match 'COSIGN_PUB_KEY') -Message "docker-compose passes COSIGN_PUB_KEY to KIC"

# docker-compose: NOC env vars for new integrations
Assert-True -Condition ($dockerCompose -match 'NEO4J_URI') -Message "docker-compose passes NEO4J_URI to NOC"
Assert-True -Condition ($dockerCompose -match 'NEO4J_USER') -Message "docker-compose passes NEO4J_USER to NOC"
Assert-True -Condition ($dockerCompose -match 'NEO4J_PASSWORD') -Message "docker-compose passes NEO4J_PASSWORD to NOC"
Assert-True -Condition ($dockerCompose -match 'MINIO_ENDPOINT') -Message "docker-compose passes MINIO_ENDPOINT to NOC"
Assert-True -Condition ($dockerCompose -match 'MINIO_ACCESS_KEY') -Message "docker-compose passes MINIO_ACCESS_KEY to NOC"
Assert-True -Condition ($dockerCompose -match 'MINIO_SECRET_KEY') -Message "docker-compose passes MINIO_SECRET_KEY to NOC"
Assert-True -Condition ($dockerCompose -match 'ZMQ_PUBLISH_ADDR') -Message "docker-compose passes ZMQ_PUBLISH_ADDR to NOC"
Assert-True -Condition ($dockerCompose -match 'TWILIO_ACCOUNT_SID') -Message "docker-compose passes TWILIO_ACCOUNT_SID to NOC"
Assert-True -Condition ($dockerCompose -match 'TWILIO_AUTH_TOKEN') -Message "docker-compose passes TWILIO_AUTH_TOKEN to NOC"
Assert-True -Condition ($dockerCompose -match 'TWILIO_FROM_NUMBER') -Message "docker-compose passes TWILIO_FROM_NUMBER to NOC"

# docker-compose: NOC depends_on neo4j + minio
Assert-True -Condition ($dockerCompose -match 'neo4j:\s*\n\s*condition:') -Message "docker-compose NOC depends_on neo4j"
Assert-True -Condition ($dockerCompose -match 'minio:\s*\n\s*condition:') -Message "docker-compose NOC depends_on minio"

Write-Host ""
Write-Host "[batch-07] Go dependency wiring verification PASSED - all 7 libraries wired" -ForegroundColor Green
