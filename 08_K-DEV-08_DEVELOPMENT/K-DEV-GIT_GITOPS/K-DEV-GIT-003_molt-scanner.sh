#!/usr/bin/env bash
###############################################################################
# K-DEV-GIT-003 — Molt Database Migration Scanner
#
# Scans PostgreSQL schema for migration-compatibility drift against the
# canonical Atlas HCL definitions stored in the repository.
#
# What it does:
#   1. Connects to the target PostgreSQL database and introspects the live
#      schema using Atlas.
#   2. Compares the live schema against the HCL definitions in migrations/.
#   3. Reports drift (additions, removals, modifications).
#   4. Generates suggested migration SQL when drift is detected.
#   5. Optionally runs a dry-run apply to validate pending migrations.
#
# Usage:
#   K-DEV-GIT-003_molt-scanner.sh [--apply-dry-run] [--schemas public,audit]
#
# Env vars:
#   DATABASE_URL   — (required) PostgreSQL connection string
#                    e.g. postgres://user:pass@host:5432/kubric?sslmode=require
#   ATLAS_HCL_DIR  — directory containing Atlas HCL files (default: migrations/)
#   ATLAS_DEV_URL  — dev database URL for Atlas (default: docker-based ephemeral DB)
#   SCHEMAS        — comma-separated list of schemas to inspect (default: public)
#   REPORT_DIR     — output directory for reports (default: .molt-reports)
###############################################################################
set -euo pipefail

readonly LOG_PREFIX="[GIT-003-molt]"

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
PROJECT_DIR="${PROJECT_DIR:-$(pwd)}"
ATLAS_HCL_DIR="${ATLAS_HCL_DIR:-${PROJECT_DIR}/migrations}"
SCHEMAS="${SCHEMAS:-public}"
REPORT_DIR="${REPORT_DIR:-${PROJECT_DIR}/.molt-reports}"
APPLY_DRY_RUN=false

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --apply-dry-run)  APPLY_DRY_RUN=true; shift ;;
    --schemas)        SCHEMAS="$2"; shift 2 ;;
    --schemas=*)      SCHEMAS="${1#*=}"; shift ;;
    --hcl-dir)        ATLAS_HCL_DIR="$2"; shift 2 ;;
    --hcl-dir=*)      ATLAS_HCL_DIR="${1#*=}"; shift ;;
    --report-dir)     REPORT_DIR="$2"; shift 2 ;;
    --report-dir=*)   REPORT_DIR="${1#*=}"; shift ;;
    -h|--help)
      head -n 25 "$0" | tail -n +3 | sed 's/^# \?//'
      exit 0 ;;
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

sanitise_url() {
  # Strip password from DATABASE_URL for safe logging
  echo "$1" | sed -E 's|://([^:]+):[^@]+@|://\1:***@|'
}

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
preflight() {
  log "Running pre-flight checks …"
  require_cmd atlas
  require_cmd psql
  require_cmd jq

  [[ -n "${DATABASE_URL:-}" ]] || die "DATABASE_URL env var is not set"
  log "Target DB: $(sanitise_url "${DATABASE_URL}")"

  if [[ ! -d "${ATLAS_HCL_DIR}" ]]; then
    die "Atlas HCL directory not found: ${ATLAS_HCL_DIR}"
  fi

  local hcl_count
  hcl_count="$(find "${ATLAS_HCL_DIR}" -name '*.hcl' -o -name '*.sql' | wc -l)"
  log "HCL/SQL files found: ${hcl_count}"
  (( hcl_count > 0 )) || die "No .hcl or .sql files in ${ATLAS_HCL_DIR}"

  mkdir -p "${REPORT_DIR}"

  # Check database connectivity
  log "Testing database connectivity …"
  if ! psql "${DATABASE_URL}" -c "SELECT 1" > /dev/null 2>&1; then
    die "Cannot connect to database — check DATABASE_URL"
  fi
  log "Database connection OK."
}

# ---------------------------------------------------------------------------
# Step 1 — Introspect live schema
# ---------------------------------------------------------------------------
introspect_live() {
  log "Introspecting live database schema (schemas: ${SCHEMAS}) …"

  local schema_flags=""
  IFS=',' read -ra schema_arr <<< "${SCHEMAS}"
  for s in "${schema_arr[@]}"; do
    schema_flags="${schema_flags} --schema ${s}"
  done

  atlas schema inspect \
    --url "${DATABASE_URL}" \
    ${schema_flags} \
    --format '{{ hcl . }}' \
    > "${REPORT_DIR}/live-schema.hcl" 2>&1 \
    || die "atlas schema inspect failed"

  local line_count
  line_count="$(wc -l < "${REPORT_DIR}/live-schema.hcl")"
  log "Live schema captured: ${REPORT_DIR}/live-schema.hcl (${line_count} lines)"
}

# ---------------------------------------------------------------------------
# Step 2 — Diff: desired vs live
# ---------------------------------------------------------------------------
compute_diff() {
  log "Computing schema diff (desired → live) …"

  local dev_url="${ATLAS_DEV_URL:-docker://postgres/16/dev?search_path=public}"

  # Build --to flags from HCL files
  local to_flags=""
  while IFS= read -r hcl_file; do
    to_flags="${to_flags} --to file://${hcl_file}"
  done < <(find "${ATLAS_HCL_DIR}" -name '*.hcl' | sort)

  # If no HCL but SQL migration dir, use migration directory mode
  if [[ -z "${to_flags}" ]]; then
    to_flags="--to file://${ATLAS_HCL_DIR}"
  fi

  set +e
  atlas schema diff \
    --from "${DATABASE_URL}" \
    ${to_flags} \
    --dev-url "${dev_url}" \
    --format '{{ sql . "  " }}' \
    > "${REPORT_DIR}/drift.sql" 2>&1
  local rc=$?
  set -e

  if [[ ${rc} -ne 0 && ! -s "${REPORT_DIR}/drift.sql" ]]; then
    err "atlas schema diff exited with code ${rc}"
    cat "${REPORT_DIR}/drift.sql" >&2 2>/dev/null || true
    return 1
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Step 3 — Analyse drift
# ---------------------------------------------------------------------------
analyse_drift() {
  local drift_file="${REPORT_DIR}/drift.sql"

  if [[ ! -s "${drift_file}" ]]; then
    log "✓ No schema drift detected — live database matches desired state."
    echo '{"drift":false,"statements":0,"additions":0,"removals":0,"modifications":0}' \
      > "${REPORT_DIR}/drift-summary.json"
    return 0
  fi

  log "Schema drift detected — analysing …"

  local total additions removals modifications
  total="$(grep -cE '^\s*(CREATE|ALTER|DROP|ADD|MODIFY|RENAME)' "${drift_file}" || echo 0)"
  additions="$(grep -cE '^\s*(CREATE|ADD)' "${drift_file}" || echo 0)"
  removals="$(grep -cE '^\s*DROP' "${drift_file}" || echo 0)"
  modifications="$(grep -cE '^\s*(ALTER|MODIFY|RENAME)' "${drift_file}" || echo 0)"

  cat > "${REPORT_DIR}/drift-summary.json" <<EOF
{
  "drift": true,
  "statements": ${total},
  "additions": ${additions},
  "removals": ${removals},
  "modifications": ${modifications},
  "generated_at": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "database": "$(sanitise_url "${DATABASE_URL}")",
  "schemas": "${SCHEMAS}"
}
EOF

  log "Drift summary:"
  log "  Statements    : ${total}"
  log "  Additions     : ${additions}"
  log "  Removals      : ${removals}"
  log "  Modifications : ${modifications}"
  log ""
  log "Suggested migration SQL → ${drift_file}"

  # Print the migration SQL snippets
  echo "────────────────────────────────────────────"
  cat "${drift_file}"
  echo "────────────────────────────────────────────"

  return 0
}

# ---------------------------------------------------------------------------
# Step 4 — Pending migration lint (optional)
# ---------------------------------------------------------------------------
lint_pending() {
  log "Linting pending migrations …"

  local dev_url="${ATLAS_DEV_URL:-docker://postgres/16/dev?search_path=public}"

  set +e
  atlas migrate lint \
    --dir "file://${ATLAS_HCL_DIR}" \
    --dev-url "${dev_url}" \
    --format '{{ json . }}' \
    > "${REPORT_DIR}/lint.json" 2>&1
  local rc=$?
  set -e

  if [[ -s "${REPORT_DIR}/lint.json" ]]; then
    local diag_count
    diag_count="$(jq '[.Steps[]?.Diagnostics // [] | .[] ] | length' "${REPORT_DIR}/lint.json" 2>/dev/null || echo "?")"
    log "Migration lint diagnostics: ${diag_count}"
    if (( diag_count > 0 )) 2>/dev/null; then
      jq '.Steps[]?.Diagnostics[]?' "${REPORT_DIR}/lint.json" >&2
    fi
  fi

  return ${rc}
}

# ---------------------------------------------------------------------------
# Step 5 — Dry-run apply (optional)
# ---------------------------------------------------------------------------
dry_run_apply() {
  if [[ "${APPLY_DRY_RUN}" != "true" ]]; then
    return 0
  fi

  log "Running dry-run migration apply …"

  local dev_url="${ATLAS_DEV_URL:-docker://postgres/16/dev?search_path=public}"

  atlas migrate apply \
    --dir "file://${ATLAS_HCL_DIR}" \
    --url "${DATABASE_URL}" \
    --dev-url "${dev_url}" \
    --dry-run \
    > "${REPORT_DIR}/dry-run-apply.sql" 2>&1 \
    || { err "Dry-run apply failed"; return 1; }

  if [[ -s "${REPORT_DIR}/dry-run-apply.sql" ]]; then
    log "Dry-run SQL → ${REPORT_DIR}/dry-run-apply.sql"
  else
    log "No pending migrations to apply."
  fi
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
summary() {
  log ""
  log "════════════════════════════════════════════════════════"
  log "  Molt Scanner Report"
  log "  Reports    : ${REPORT_DIR}/"
  log "  Schemas    : ${SCHEMAS}"

  if [[ -f "${REPORT_DIR}/drift-summary.json" ]]; then
    local has_drift
    has_drift="$(jq -r '.drift' "${REPORT_DIR}/drift-summary.json")"
    if [[ "${has_drift}" == "true" ]]; then
      log "  Drift      : YES — review ${REPORT_DIR}/drift.sql"
    else
      log "  Drift      : NONE"
    fi
  fi

  log "════════════════════════════════════════════════════════"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight
  introspect_live
  compute_diff
  analyse_drift
  lint_pending || true
  dry_run_apply
  summary
}

main "$@"
