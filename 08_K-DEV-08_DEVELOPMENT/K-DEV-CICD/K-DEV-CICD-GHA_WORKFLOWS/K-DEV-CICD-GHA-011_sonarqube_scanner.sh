#!/usr/bin/env bash
###############################################################################
# K-DEV-CICD-GHA-011 — SonarQube Scanner for Kubric-UiDR CI
#
# Runs sonar-scanner across the Kubric polyglot codebase:
#   • Go        — go test coverage + go vet
#   • Python    — pytest coverage (xml)
#   • TypeScript — frontend sources (ESLint + coverage)
#
# After analysis, the script polls the SonarQube quality gate and fails
# the CI job if the gate status is not OK.
#
# Env vars (required):
#   SONAR_HOST_URL  — SonarQube server URL
#   SONAR_TOKEN     — authentication token
#
# Env vars (optional):
#   SONAR_PROJECT_KEY     — project key (default: kubric-uidr)
#   SONAR_PROJECT_NAME    — display name (default: Kubric-UiDR)
#   SONAR_PROJECT_VERSION — version string (default: git describe)
#   PROJECT_DIR           — workspace root (default: cwd)
#   GATE_TIMEOUT          — quality gate poll timeout in seconds (default: 300)
#   SKIP_TESTS            — set "true" to skip test/coverage generation
###############################################################################
set -euo pipefail

readonly LOG_PREFIX="[GHA-011-sonar]"

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
PROJECT_DIR="${PROJECT_DIR:-$(pwd)}"
SONAR_PROJECT_KEY="${SONAR_PROJECT_KEY:-kubric-uidr}"
SONAR_PROJECT_NAME="${SONAR_PROJECT_NAME:-Kubric-UiDR}"
SONAR_PROJECT_VERSION="${SONAR_PROJECT_VERSION:-$(git -C "${PROJECT_DIR}" describe --tags --always 2>/dev/null || echo "0.0.0-dev")}"
GATE_TIMEOUT="${GATE_TIMEOUT:-300}"
SKIP_TESTS="${SKIP_TESTS:-false}"

# Coverage output paths
readonly GO_COVER_DIR="${PROJECT_DIR}/.sonar/go-coverage"
readonly PY_COVER_DIR="${PROJECT_DIR}/.sonar/py-coverage"
readonly TS_COVER_DIR="${PROJECT_DIR}/.sonar/ts-coverage"

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
  log "Running pre-flight checks …"
  require_cmd sonar-scanner
  require_cmd jq
  require_cmd curl

  [[ -n "${SONAR_HOST_URL:-}" ]] || die "SONAR_HOST_URL is not set"
  [[ -n "${SONAR_TOKEN:-}" ]]    || die "SONAR_TOKEN is not set"

  export SONAR_HOST_URL SONAR_TOKEN

  log "SonarQube host   : ${SONAR_HOST_URL}"
  log "Project key      : ${SONAR_PROJECT_KEY}"
  log "Project version  : ${SONAR_PROJECT_VERSION}"

  mkdir -p "${GO_COVER_DIR}" "${PY_COVER_DIR}" "${TS_COVER_DIR}"
}

# ---------------------------------------------------------------------------
# Step 1 — Generate coverage reports
# ---------------------------------------------------------------------------
generate_go_coverage() {
  log "Generating Go test coverage …"
  local cover_out="${GO_COVER_DIR}/coverage.out"
  local report_json="${GO_COVER_DIR}/test-report.json"

  if command -v go &>/dev/null && [[ -f "${PROJECT_DIR}/go.mod" ]]; then
    pushd "${PROJECT_DIR}" > /dev/null

    # Run tests with coverage; allow failures (sonar captures them)
    set +e
    go test ./cmd/... ./internal/... ./services/... \
      -coverprofile="${cover_out}" \
      -covermode=atomic \
      -json > "${report_json}" 2>&1
    set -e

    # Convert to Sonar generic coverage format if gocov is available
    if command -v gocov &>/dev/null && command -v gocov-xml &>/dev/null; then
      gocov convert "${cover_out}" | gocov-xml > "${GO_COVER_DIR}/coverage.xml"
      log "Go coverage XML generated."
    fi

    popd > /dev/null
    log "Go coverage profile: ${cover_out}"
  else
    log "SKIP: Go toolchain or go.mod not found"
  fi
}

generate_python_coverage() {
  log "Generating Python (pytest) coverage …"

  if command -v python3 &>/dev/null || command -v python &>/dev/null; then
    local py_cmd
    py_cmd="$(command -v python3 || command -v python)"

    # Determine test directories
    local test_dirs=()
    for d in kai agents; do
      if [[ -d "${PROJECT_DIR}/${d}" ]]; then
        test_dirs+=("${PROJECT_DIR}/${d}")
      fi
    done

    if [[ ${#test_dirs[@]} -gt 0 ]]; then
      pushd "${PROJECT_DIR}" > /dev/null
      set +e
      "${py_cmd}" -m pytest \
        "${test_dirs[@]}" \
        --co -q 2>/dev/null && \
      "${py_cmd}" -m pytest \
        "${test_dirs[@]}" \
        --cov=kai --cov=agents \
        --cov-report=xml:"${PY_COVER_DIR}/coverage.xml" \
        --junitxml="${PY_COVER_DIR}/test-report.xml" \
        -q 2>&1 || true
      set -e
      popd > /dev/null
      log "Python coverage XML: ${PY_COVER_DIR}/coverage.xml"
    else
      log "SKIP: No Python test directories found"
    fi
  else
    log "SKIP: Python not available"
  fi
}

generate_ts_coverage() {
  log "Generating TypeScript coverage …"

  if [[ -f "${PROJECT_DIR}/frontend/package.json" ]]; then
    pushd "${PROJECT_DIR}/frontend" > /dev/null

    if command -v npx &>/dev/null; then
      set +e
      npx vitest run --coverage --reporter=json \
        --outputFile="${TS_COVER_DIR}/test-report.json" \
        2>&1 || \
      npx jest --coverage --coverageDirectory="${TS_COVER_DIR}" \
        --coverageReporters=lcov \
        2>&1 || true
      set -e

      # Copy lcov if generated
      if [[ -f "${TS_COVER_DIR}/lcov.info" ]]; then
        log "TS coverage LCOV: ${TS_COVER_DIR}/lcov.info"
      elif [[ -f "coverage/lcov.info" ]]; then
        cp "coverage/lcov.info" "${TS_COVER_DIR}/lcov.info"
        log "TS coverage LCOV: ${TS_COVER_DIR}/lcov.info"
      fi
    fi

    popd > /dev/null
  else
    log "SKIP: frontend/package.json not found"
  fi
}

generate_coverage() {
  if [[ "${SKIP_TESTS}" == "true" ]]; then
    log "SKIP_TESTS=true — using pre-existing coverage reports"
    return 0
  fi

  generate_go_coverage
  generate_python_coverage
  generate_ts_coverage
}

# ---------------------------------------------------------------------------
# Step 2 — Run sonar-scanner
# ---------------------------------------------------------------------------
run_scanner() {
  log "Launching sonar-scanner …"

  local props=()
  props+=("-Dsonar.projectKey=${SONAR_PROJECT_KEY}")
  props+=("-Dsonar.projectName=${SONAR_PROJECT_NAME}")
  props+=("-Dsonar.projectVersion=${SONAR_PROJECT_VERSION}")
  props+=("-Dsonar.projectBaseDir=${PROJECT_DIR}")

  # Source directories per language
  props+=("-Dsonar.sources=cmd,internal,services,kai,agents,frontend/src")
  props+=("-Dsonar.tests=internal,kai,agents,frontend/src")
  props+=("-Dsonar.test.inclusions=**/*_test.go,**/test_*.py,**/*.test.ts,**/*.test.tsx,**/*.spec.ts")
  props+=("-Dsonar.exclusions=vendor/**,node_modules/**,target/**,third_party/**,.sonar/**,**/testdata/**")

  # Go
  if [[ -f "${GO_COVER_DIR}/coverage.out" ]]; then
    props+=("-Dsonar.go.coverage.reportPaths=${GO_COVER_DIR}/coverage.out")
  fi
  if [[ -f "${GO_COVER_DIR}/test-report.json" ]]; then
    props+=("-Dsonar.go.tests.reportPaths=${GO_COVER_DIR}/test-report.json")
  fi

  # Python
  if [[ -f "${PY_COVER_DIR}/coverage.xml" ]]; then
    props+=("-Dsonar.python.coverage.reportPaths=${PY_COVER_DIR}/coverage.xml")
  fi
  if [[ -f "${PY_COVER_DIR}/test-report.xml" ]]; then
    props+=("-Dsonar.python.xunit.reportPath=${PY_COVER_DIR}/test-report.xml")
  fi

  # TypeScript / JavaScript
  if [[ -f "${TS_COVER_DIR}/lcov.info" ]]; then
    props+=("-Dsonar.javascript.lcov.reportPaths=${TS_COVER_DIR}/lcov.info")
  fi

  # SCM
  props+=("-Dsonar.scm.provider=git")
  props+=("-Dsonar.scm.revision=$(git -C "${PROJECT_DIR}" rev-parse HEAD 2>/dev/null || echo unknown)")

  # Branch / PR awareness
  if [[ -n "${GITHUB_HEAD_REF:-}" ]]; then
    props+=("-Dsonar.pullrequest.key=${GITHUB_EVENT_NUMBER:-0}")
    props+=("-Dsonar.pullrequest.branch=${GITHUB_HEAD_REF}")
    props+=("-Dsonar.pullrequest.base=${GITHUB_BASE_REF:-main}")
  elif [[ -n "${GITHUB_REF_NAME:-}" && "${GITHUB_REF_NAME}" != "main" ]]; then
    props+=("-Dsonar.branch.name=${GITHUB_REF_NAME}")
  fi

  sonar-scanner "${props[@]}" \
    -Dsonar.host.url="${SONAR_HOST_URL}" \
    -Dsonar.token="${SONAR_TOKEN}" \
    -Dsonar.qualitygate.wait=false \
    || die "sonar-scanner failed"

  log "sonar-scanner completed."
}

# ---------------------------------------------------------------------------
# Step 3 — Quality gate check
# ---------------------------------------------------------------------------
check_quality_gate() {
  log "Checking quality gate (timeout: ${GATE_TIMEOUT}s) …"

  # Retrieve the analysis task ID from the scanner report
  local report_task="${PROJECT_DIR}/.scannerwork/report-task.txt"
  if [[ ! -f "${report_task}" ]]; then
    err "report-task.txt not found — cannot check quality gate"
    return 1
  fi

  local ce_task_url
  ce_task_url="$(grep '^ceTaskUrl=' "${report_task}" | cut -d= -f2-)"
  [[ -n "${ce_task_url}" ]] || die "ceTaskUrl not found in report-task.txt"

  # Poll until the compute engine task finishes
  local status="PENDING"
  local elapsed=0
  local interval=5

  while [[ "${status}" == "PENDING" || "${status}" == "IN_PROGRESS" ]]; do
    if (( elapsed >= GATE_TIMEOUT )); then
      die "Quality gate timed out after ${GATE_TIMEOUT}s (task status: ${status})"
    fi

    sleep "${interval}"
    elapsed=$(( elapsed + interval ))

    status="$(curl -sS -u "${SONAR_TOKEN}:" "${ce_task_url}" \
      | jq -r '.task.status')"
    log "  CE task status: ${status} (${elapsed}s elapsed)"
  done

  if [[ "${status}" != "SUCCESS" ]]; then
    die "SonarQube analysis task failed with status: ${status}"
  fi

  # Fetch analysis ID and quality gate
  local analysis_id
  analysis_id="$(curl -sS -u "${SONAR_TOKEN}:" "${ce_task_url}" \
    | jq -r '.task.analysisId')"

  local gate_status
  gate_status="$(curl -sS -u "${SONAR_TOKEN}:" \
    "${SONAR_HOST_URL}/api/qualitygates/project_status?analysisId=${analysis_id}" \
    | jq -r '.projectStatus.status')"

  log "Quality gate status: ${gate_status}"

  if [[ "${gate_status}" != "OK" ]]; then
    err "Quality gate FAILED (status=${gate_status})"

    # Print conditions for diagnostics
    curl -sS -u "${SONAR_TOKEN}:" \
      "${SONAR_HOST_URL}/api/qualitygates/project_status?analysisId=${analysis_id}" \
      | jq '.projectStatus.conditions[] | select(.status != "OK")' >&2

    return 1
  fi

  log "✓ Quality gate passed."
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight
  generate_coverage
  run_scanner
  check_quality_gate

  log "════════════════════════════════════════════════════════"
  log "  SonarQube analysis complete"
  log "  Dashboard: ${SONAR_HOST_URL}/dashboard?id=${SONAR_PROJECT_KEY}"
  log "════════════════════════════════════════════════════════"
}

main "$@"
