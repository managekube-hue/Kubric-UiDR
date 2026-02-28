#!/usr/bin/env bash
###############################################################################
# K-DEV-CICD-GHA-010 — Snyk Security Scan for Kubric-UiDR CI
#
# Runs Snyk vulnerability scans across all project ecosystems:
#   • Go   — go.mod (backend services + agents)
#   • Rust — Cargo.toml (XRO agents)
#   • Python — requirements.txt / kai/requirements.txt (KAI AI layer)
#   • Node.js — package.json (frontend)
#
# Outputs SARIF files for upload to GitHub Security tab and a unified
# JSON report for archival.
#
# Usage:
#   K-DEV-CICD-GHA-010_snyk_scan.sh [--severity-threshold high|critical]
#
# Env vars:
#   SNYK_TOKEN         — (required) Snyk API token
#   SNYK_ORG           — (optional) Snyk organisation slug
#   SEVERITY_THRESHOLD — severity floor (default: high)
#   PROJECT_DIR        — workspace root (default: current directory)
#   SARIF_DIR          — directory for SARIF output (default: .snyk-results)
###############################################################################
set -euo pipefail

readonly LOG_PREFIX="[GHA-010-snyk]"

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
SEVERITY_THRESHOLD="${SEVERITY_THRESHOLD:-high}"
PROJECT_DIR="${PROJECT_DIR:-$(pwd)}"
SARIF_DIR="${SARIF_DIR:-${PROJECT_DIR}/.snyk-results}"
SNYK_ORG="${SNYK_ORG:-}"

# ---------------------------------------------------------------------------
# Argument parsing — allow --severity-threshold override
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --severity-threshold)
      SEVERITY_THRESHOLD="$2"; shift 2 ;;
    --severity-threshold=*)
      SEVERITY_THRESHOLD="${1#*=}"; shift ;;
    *)
      echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

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
  require_cmd snyk
  require_cmd jq

  [[ -n "${SNYK_TOKEN:-}" ]] || die "SNYK_TOKEN env var is not set"
  export SNYK_TOKEN

  snyk auth --token "${SNYK_TOKEN}" 2>/dev/null || true

  mkdir -p "${SARIF_DIR}"
  log "Severity threshold : ${SEVERITY_THRESHOLD}"
  log "Project root       : ${PROJECT_DIR}"
  log "SARIF output dir   : ${SARIF_DIR}"
}

# ---------------------------------------------------------------------------
# Generic scan helper
# ---------------------------------------------------------------------------
EXIT_CODE=0

run_scan() {
  local label="$1"
  local manifest="$2"
  local sarif_name="$3"
  shift 3
  local extra_args=("$@")

  local manifest_path="${PROJECT_DIR}/${manifest}"

  if [[ ! -f "${manifest_path}" ]]; then
    log "SKIP ${label}: ${manifest_path} not found"
    return 0
  fi

  log "Scanning ${label} (${manifest}) …"

  local org_flag=()
  if [[ -n "${SNYK_ORG}" ]]; then
    org_flag=(--org="${SNYK_ORG}")
  fi

  local sarif_path="${SARIF_DIR}/${sarif_name}.sarif"
  local json_path="${SARIF_DIR}/${sarif_name}.json"

  # Run SARIF output
  set +e
  snyk test \
    --file="${manifest_path}" \
    --severity-threshold="${SEVERITY_THRESHOLD}" \
    "${org_flag[@]}" \
    "${extra_args[@]}" \
    --sarif-file-output="${sarif_path}" \
    --json-file-output="${json_path}" \
    2>&1 | tee "${SARIF_DIR}/${sarif_name}.log"
  local rc=$?
  set -e

  if [[ ${rc} -eq 0 ]]; then
    log "  ✓ ${label}: no vulnerabilities ≥ ${SEVERITY_THRESHOLD}"
  elif [[ ${rc} -eq 1 ]]; then
    local vuln_count
    vuln_count="$(jq '.uniqueCount // .vulnerabilities | length' "${json_path}" 2>/dev/null || echo "?")"
    err "  ✗ ${label}: ${vuln_count} vulnerability/ies found (exit ${rc})"
    EXIT_CODE=1
  else
    err "  ✗ ${label}: snyk exited with code ${rc}"
    EXIT_CODE=1
  fi

  if [[ -f "${sarif_path}" ]]; then
    local sarif_size
    sarif_size="$(wc -c < "${sarif_path}")"
    log "  SARIF: ${sarif_path} (${sarif_size} bytes)"
  fi
}

# ---------------------------------------------------------------------------
# Container image scan
# ---------------------------------------------------------------------------
scan_container() {
  local image="$1"
  local sarif_name="$2"

  log "Scanning container image ${image} …"

  local org_flag=()
  if [[ -n "${SNYK_ORG}" ]]; then
    org_flag=(--org="${SNYK_ORG}")
  fi

  local sarif_path="${SARIF_DIR}/${sarif_name}.sarif"
  local json_path="${SARIF_DIR}/${sarif_name}.json"

  set +e
  snyk container test "${image}" \
    --severity-threshold="${SEVERITY_THRESHOLD}" \
    "${org_flag[@]}" \
    --sarif-file-output="${sarif_path}" \
    --json-file-output="${json_path}" \
    2>&1 | tee "${SARIF_DIR}/${sarif_name}.log"
  local rc=$?
  set -e

  if [[ ${rc} -eq 0 ]]; then
    log "  ✓ Container ${image}: clean"
  elif [[ ${rc} -eq 1 ]]; then
    err "  ✗ Container ${image}: vulnerabilities found"
    EXIT_CODE=1
  else
    err "  ✗ Container ${image}: snyk exited with code ${rc}"
    EXIT_CODE=1
  fi
}

# ---------------------------------------------------------------------------
# Scans
# ---------------------------------------------------------------------------
scan_go() {
  run_scan "Go (backend)" "go.mod" "snyk-go" \
    --package-manager=gomodules
}

scan_rust() {
  run_scan "Rust (agents)" "Cargo.toml" "snyk-rust" \
    --package-manager=cargo
}

scan_python() {
  # Scan root requirements and KAI-specific requirements
  run_scan "Python (root)" "requirements.txt" "snyk-python-root" \
    --package-manager=pip

  if [[ -f "${PROJECT_DIR}/kai/requirements.txt" ]]; then
    run_scan "Python (KAI)" "kai/requirements.txt" "snyk-python-kai" \
      --package-manager=pip
  fi
}

scan_node() {
  run_scan "Node.js (frontend)" "package.json" "snyk-node" \
    --package-manager=npm
}

scan_iac() {
  # Scan IaC configurations (Dockerfiles, K8s manifests, Terraform)
  local iac_dirs=("deployments" "infra" "docker")

  for dir in "${iac_dirs[@]}"; do
    local dir_path="${PROJECT_DIR}/${dir}"
    if [[ -d "${dir_path}" ]]; then
      log "Scanning IaC: ${dir} …"
      set +e
      snyk iac test "${dir_path}" \
        --severity-threshold="${SEVERITY_THRESHOLD}" \
        --sarif-file-output="${SARIF_DIR}/snyk-iac-${dir}.sarif" \
        --json-file-output="${SARIF_DIR}/snyk-iac-${dir}.json" \
        2>&1 | tee "${SARIF_DIR}/snyk-iac-${dir}.log"
      local rc=$?
      set -e

      if [[ ${rc} -eq 0 ]]; then
        log "  ✓ IaC ${dir}: clean"
      elif [[ ${rc} -eq 1 ]]; then
        err "  ✗ IaC ${dir}: issues found"
        EXIT_CODE=1
      fi
    fi
  done
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
summary() {
  log ""
  log "════════════════════════════════════════════════════════"
  log "  Snyk Scan Summary"
  log "  Severity threshold : ${SEVERITY_THRESHOLD}"
  log "  SARIF reports      : ${SARIF_DIR}/"
  log "════════════════════════════════════════════════════════"

  local sarif_count
  sarif_count="$(find "${SARIF_DIR}" -name '*.sarif' 2>/dev/null | wc -l)"
  log "  Generated ${sarif_count} SARIF report(s)"

  # List all SARIF files for GHA upload step
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "sarif_dir=${SARIF_DIR}" >> "${GITHUB_OUTPUT}"
    echo "sarif_count=${sarif_count}" >> "${GITHUB_OUTPUT}"
  fi

  if [[ ${EXIT_CODE} -ne 0 ]]; then
    err "  One or more scans found vulnerabilities ≥ ${SEVERITY_THRESHOLD}"
  else
    log "  ✓ All scans passed"
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight

  scan_go
  scan_rust
  scan_python
  scan_node
  scan_iac

  summary
  exit ${EXIT_CODE}
}

main
