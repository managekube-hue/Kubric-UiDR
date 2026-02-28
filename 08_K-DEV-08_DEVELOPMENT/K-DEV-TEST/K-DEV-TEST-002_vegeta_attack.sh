#!/usr/bin/env bash
###############################################################################
# K-DEV-TEST-002 — Vegeta HTTP Load Testing for Kubric-UiDR
#
# Runs Vegeta load tests against all Kubric API endpoints with three
# pre-defined attack profiles:
#
#   smoke  — 10 rps for 30s   (quick sanity check)
#   load   — 100 rps for 60s  (sustained load)
#   stress — 500 rps for 120s (breaking-point discovery)
#
# Endpoints tested:
#   /healthz, /readyz, /api/v1/alerts, /api/v1/scan,
#   /api/v1/compliance, /api/v1/topology, /api/v1/storage
#
# Usage:
#   K-DEV-TEST-002_vegeta_attack.sh <profile> <base_url>
#
# Examples:
#   K-DEV-TEST-002_vegeta_attack.sh smoke   http://localhost:8080
#   K-DEV-TEST-002_vegeta_attack.sh load    https://api.kubric.internal
#   K-DEV-TEST-002_vegeta_attack.sh stress  http://10.0.1.100:8080
#
# Outputs (written to REPORT_DIR):
#   • results-<profile>.bin   — raw binary results
#   • report-<profile>.html   — HTML report with charts
#   • histogram-<profile>.txt — latency histogram (text)
#   • summary-<profile>.json  — JSON summary (p50/p95/p99/max/success)
#
# Env vars (optional):
#   REPORT_DIR       — output directory (default: .vegeta-reports)
#   AUTH_TOKEN       — Bearer token for authenticated endpoints
#   EXTRA_HEADERS    — additional headers (newline-separated Key: Value)
#   VEGETA_TIMEOUT   — per-request timeout (default: 10s)
#   VEGETA_WORKERS   — max workers (default: 50)
###############################################################################
set -euo pipefail

readonly LOG_PREFIX="[TEST-002-vegeta]"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <profile> <base_url>"
  echo ""
  echo "Profiles: smoke | load | stress"
  echo ""
  echo "Example:"
  echo "  $0 smoke http://localhost:8080"
  exit 1
fi

PROFILE="$1"
BASE_URL="${2%/}"  # strip trailing slash

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
REPORT_DIR="${REPORT_DIR:-.vegeta-reports}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
VEGETA_TIMEOUT="${VEGETA_TIMEOUT:-10s}"
VEGETA_WORKERS="${VEGETA_WORKERS:-50}"
TIMESTAMP="$(date -u '+%Y%m%dT%H%M%SZ')"

# ---------------------------------------------------------------------------
# Profile definitions
# ---------------------------------------------------------------------------
declare -A PROFILE_RATE PROFILE_DURATION PROFILE_DESC

PROFILE_RATE[smoke]=10
PROFILE_DURATION[smoke]=30s
PROFILE_DESC[smoke]="Smoke — 10 rps × 30s"

PROFILE_RATE[load]=100
PROFILE_DURATION[load]=60s
PROFILE_DESC[load]="Load — 100 rps × 60s"

PROFILE_RATE[stress]=500
PROFILE_DURATION[stress]=120s
PROFILE_DESC[stress]="Stress — 500 rps × 120s"

# Validate profile
if [[ -z "${PROFILE_RATE[${PROFILE}]+x}" ]]; then
  echo "ERROR: Unknown profile '${PROFILE}'. Choose: smoke | load | stress" >&2
  exit 1
fi

RATE="${PROFILE_RATE[${PROFILE}]}"
DURATION="${PROFILE_DURATION[${PROFILE}]}"
DESC="${PROFILE_DESC[${PROFILE}]}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()   { echo "${LOG_PREFIX} $(date -u '+%Y-%m-%dT%H:%M:%SZ') $*"; }
err()   { log "ERROR: $*" >&2; }
die()   { err "$@"; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
preflight() {
  log "Pre-flight checks …"
  require_cmd vegeta
  require_cmd jq

  vegeta_ver="$(vegeta -version 2>&1 || echo "unknown")"
  log "vegeta version: ${vegeta_ver}"

  mkdir -p "${REPORT_DIR}"

  # Quick connectivity check
  log "Checking connectivity to ${BASE_URL} …"
  local http_code
  http_code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "${BASE_URL}/healthz" 2>/dev/null || echo "000")"
  if [[ "${http_code}" == "000" ]]; then
    err "Cannot reach ${BASE_URL}/healthz — proceeding anyway (target may not have /healthz)"
  else
    log "Connectivity OK (HTTP ${http_code})."
  fi
}

# ---------------------------------------------------------------------------
# Build targets file
# ---------------------------------------------------------------------------
build_targets() {
  local targets_file="${REPORT_DIR}/targets-${PROFILE}.txt"

  # Common headers
  local headers=""
  headers+="Content-Type: application/json\n"
  if [[ -n "${AUTH_TOKEN}" ]]; then
    headers+="Authorization: Bearer ${AUTH_TOKEN}\n"
  fi
  if [[ -n "${EXTRA_HEADERS:-}" ]]; then
    headers+="${EXTRA_HEADERS}\n"
  fi

  # Build target list in Vegeta HTTP format
  cat > "${targets_file}" <<EOF
GET ${BASE_URL}/healthz
$(echo -e "${headers}")
GET ${BASE_URL}/readyz
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/alerts
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/alerts?severity=critical&limit=10
$(echo -e "${headers}")
POST ${BASE_URL}/api/v1/scan
$(echo -e "${headers}")
@${REPORT_DIR}/scan-body.json

GET ${BASE_URL}/api/v1/compliance
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/compliance?framework=nist-800-53
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/topology
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/topology?depth=2
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/storage
$(echo -e "${headers}")
GET ${BASE_URL}/api/v1/storage/buckets
$(echo -e "${headers}")
EOF

  # Create request bodies for POST endpoints
  cat > "${REPORT_DIR}/scan-body.json" <<'BODY'
{
  "target": "cluster-default",
  "scan_type": "vulnerability",
  "severity_threshold": "high",
  "namespaces": ["default", "kubric-system"]
}
BODY

  log "Targets file: ${targets_file} ($(grep -c '^GET\|^POST\|^PUT\|^DELETE' "${targets_file}" || echo 0) endpoints)"
  echo "${targets_file}"
}

# ---------------------------------------------------------------------------
# Run attack
# ---------------------------------------------------------------------------
run_attack() {
  local targets_file="$1"
  local results_bin="${REPORT_DIR}/results-${PROFILE}.bin"

  log ""
  log "════════════════════════════════════════════════════════"
  log "  ${DESC}"
  log "  Target  : ${BASE_URL}"
  log "  Rate    : ${RATE} rps"
  log "  Duration: ${DURATION}"
  log "  Workers : ${VEGETA_WORKERS}"
  log "  Timeout : ${VEGETA_TIMEOUT}"
  log "════════════════════════════════════════════════════════"
  log ""

  vegeta attack \
    -targets="${targets_file}" \
    -rate="${RATE}" \
    -duration="${DURATION}" \
    -timeout="${VEGETA_TIMEOUT}" \
    -workers="${VEGETA_WORKERS}" \
    -http2 \
    -keepalive \
    > "${results_bin}" \
    || die "vegeta attack failed"

  log "Attack complete. Raw results: ${results_bin} ($(wc -c < "${results_bin}") bytes)"
  echo "${results_bin}"
}

# ---------------------------------------------------------------------------
# Generate reports
# ---------------------------------------------------------------------------
generate_reports() {
  local results_bin="$1"

  log "Generating reports …"

  # Text summary
  local summary_txt="${REPORT_DIR}/summary-${PROFILE}.txt"
  vegeta report -type=text "${results_bin}" > "${summary_txt}"
  log "Text summary → ${summary_txt}"
  echo ""
  cat "${summary_txt}"
  echo ""

  # JSON summary with extra percentiles
  local summary_json="${REPORT_DIR}/summary-${PROFILE}.json"
  vegeta report -type=json "${results_bin}" > "${summary_json}"
  log "JSON summary → ${summary_json}"

  # Extract key metrics
  local p50 p95 p99 max_lat success_ratio status_codes
  p50="$(jq -r '.latencies."50th" // 0 | . / 1000000 | tostring + "ms"' "${summary_json}")"
  p95="$(jq -r '.latencies."95th" // 0 | . / 1000000 | tostring + "ms"' "${summary_json}")"
  p99="$(jq -r '.latencies."99th" // 0 | . / 1000000 | tostring + "ms"' "${summary_json}")"
  max_lat="$(jq -r '.latencies.max // 0 | . / 1000000 | tostring + "ms"' "${summary_json}")"
  success_ratio="$(jq -r '.success // 0 | . * 100 | tostring + "%"' "${summary_json}")"
  status_codes="$(jq -r '.status_codes // {} | to_entries | map("\(.key)=\(.value)") | join(", ")' "${summary_json}")"

  log "┌──────────────────────────────────────────┐"
  log "│  Latency  P50: ${p50}"
  log "│           P95: ${p95}"
  log "│           P99: ${p99}"
  log "│           Max: ${max_lat}"
  log "│  Success     : ${success_ratio}"
  log "│  Status codes: ${status_codes}"
  log "└──────────────────────────────────────────┘"

  # HTML report
  local report_html="${REPORT_DIR}/report-${PROFILE}.html"
  vegeta report -type=hdrplot "${results_bin}" > "${REPORT_DIR}/hdrplot-${PROFILE}.txt" 2>/dev/null || true

  # Vegeta plot for HTML chart
  if vegeta plot "${results_bin}" > "${report_html}" 2>/dev/null; then
    log "HTML report  → ${report_html}"
  else
    log "WARN: vegeta plot not available; skipping HTML report"
  fi

  # Latency histogram
  local histogram="${REPORT_DIR}/histogram-${PROFILE}.txt"
  vegeta report -type=hist["0,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s,5s"] "${results_bin}" \
    > "${histogram}" 2>/dev/null || \
  vegeta report -type=text "${results_bin}" > "${histogram}"
  log "Histogram    → ${histogram}"
  echo ""
  cat "${histogram}"
  echo ""
}

# ---------------------------------------------------------------------------
# Evaluate pass/fail thresholds
# ---------------------------------------------------------------------------
evaluate_thresholds() {
  local summary_json="${REPORT_DIR}/summary-${PROFILE}.json"

  # Define per-profile SLO thresholds
  local max_p99_ms max_error_pct
  case "${PROFILE}" in
    smoke)
      max_p99_ms=500
      max_error_pct=1
      ;;
    load)
      max_p99_ms=1000
      max_error_pct=2
      ;;
    stress)
      max_p99_ms=5000
      max_error_pct=10
      ;;
  esac

  local p99_ns p99_ms success_rate error_pct
  p99_ns="$(jq '.latencies."99th" // 0' "${summary_json}")"
  p99_ms=$(( p99_ns / 1000000 ))
  success_rate="$(jq '.success // 0' "${summary_json}")"
  error_pct="$(echo "scale=2; (1 - ${success_rate}) * 100" | bc 2>/dev/null || echo "0")"

  local passed=true

  if (( p99_ms > max_p99_ms )); then
    err "THRESHOLD BREACH: P99 latency ${p99_ms}ms exceeds limit ${max_p99_ms}ms"
    passed=false
  fi

  # Use awk for floating-point comparison
  if echo "${error_pct} ${max_error_pct}" | awk '{exit ($1 > $2) ? 0 : 1}' 2>/dev/null; then
    err "THRESHOLD BREACH: Error rate ${error_pct}% exceeds limit ${max_error_pct}%"
    passed=false
  fi

  if [[ "${passed}" == "true" ]]; then
    log "✓ All SLO thresholds passed for profile '${PROFILE}'"
    return 0
  else
    err "✗ SLO thresholds breached for profile '${PROFILE}'"
    return 1
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight

  local targets_file
  targets_file="$(build_targets)"

  local results_bin
  results_bin="$(run_attack "${targets_file}")"

  generate_reports "${results_bin}"

  local exit_code=0
  evaluate_thresholds || exit_code=$?

  log ""
  log "════════════════════════════════════════════════════════"
  log "  Vegeta Load Test Complete"
  log "  Profile : ${DESC}"
  log "  Target  : ${BASE_URL}"
  log "  Reports : ${REPORT_DIR}/"
  log "════════════════════════════════════════════════════════"

  exit ${exit_code}
}

main
