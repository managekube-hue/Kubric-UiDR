#!/usr/bin/env bash
# =============================================================================
# scripts/nats-stream-init.sh
# Kubric Platform — NATS JetStream Bootstrap
#
# Creates all required JetStream streams and durable push consumers for the
# Kubric platform.  Safe to run multiple times; existing streams/consumers are
# detected and skipped (idempotent).
#
# Requirements:
#   - nats CLI (https://github.com/nats-io/natscli/releases >= 0.1.x)
#
# Environment:
#   NATS_URL   NATS server URL (default: nats://localhost:4222)
#
# Exit codes:
#   0  — all streams and consumers created (or already existed) successfully
#   1  — one or more create operations failed
# =============================================================================
set -uo pipefail

# --------------------------------------------------------------------------- #
# Terminal colours                                                             #
# --------------------------------------------------------------------------- #
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

ok()     { printf "${GREEN}  ✓ %s${NC}\n" "$*"; }
err()    { printf "${RED}  ✗ %s${NC}\n" "$*" >&2; }
info()   { printf "${YELLOW}  ► %s${NC}\n" "$*"; }
banner() { printf "\n${BOLD}${CYAN}════════════════════════════════════════${NC}\n  ${BOLD}%s${NC}\n${BOLD}${CYAN}════════════════════════════════════════${NC}\n" "$*"; }

# --------------------------------------------------------------------------- #
# Configuration                                                                #
# --------------------------------------------------------------------------- #
NATS_URL="${NATS_URL:-nats://localhost:4222}"
NATS_CLI="${NATS_CLI:-nats}"
FAILURES=0

# Byte constants (NATS CLI --max-bytes takes integers)
readonly BYTES_5GB=5368709120
readonly BYTES_2GB=2147483648

# --------------------------------------------------------------------------- #
# Pre-flight checks                                                            #
# --------------------------------------------------------------------------- #
banner "Kubric NATS JetStream Initialisation"
printf "  Server : %s\n\n" "${NATS_URL}"

if ! command -v "${NATS_CLI}" &>/dev/null; then
  err "nats CLI not found in PATH."
  err "Install from: https://github.com/nats-io/natscli/releases"
  exit 1
fi

# Verify server is reachable before doing anything else
if ! "${NATS_CLI}" --server "${NATS_URL}" server ping --count 1 >/dev/null 2>&1; then
  err "Cannot reach NATS server at ${NATS_URL}"
  err "Ensure NATS is running and NATS_URL is correct."
  exit 1
fi

ok "Connected to NATS at ${NATS_URL}"

# --------------------------------------------------------------------------- #
# Helper: ensure_stream                                                        #
#                                                                              #
# Usage: ensure_stream <STREAM_NAME> [nats-stream-add flags...]               #
#                                                                              #
# Attempts to create the stream.  If the stream already exists the add        #
# command will fail with "already exists" — that is treated as success.       #
# Any other failure increments FAILURES and returns 1.                        #
# --------------------------------------------------------------------------- #
ensure_stream() {
  local name="$1"; shift
  local output rc

  output=$(
    "${NATS_CLI}" --server "${NATS_URL}" stream add "${name}" \
      --defaults \
      "$@" 2>&1
  )
  rc=$?

  if [[ ${rc} -eq 0 ]]; then
    ok "Stream ${name} created"
    return 0
  fi

  # nats CLI prints "stream ${name} already exists" on duplicate add
  if echo "${output}" | grep -qiE "already exist|stream name already in use"; then
    ok "Stream ${name} already exists — skipped"
    return 0
  fi

  err "Stream ${name} FAILED: ${output}"
  (( FAILURES++ )) || true
  return 1
}

# --------------------------------------------------------------------------- #
# Helper: ensure_consumer                                                      #
#                                                                              #
# Usage: ensure_consumer <STREAM> <CONSUMER_NAME> [nats-consumer-add flags]   #
# --------------------------------------------------------------------------- #
ensure_consumer() {
  local stream="$1" consumer="$2"; shift 2
  local output rc

  output=$(
    "${NATS_CLI}" --server "${NATS_URL}" consumer add "${stream}" "${consumer}" \
      --defaults \
      "$@" 2>&1
  )
  rc=$?

  if [[ ${rc} -eq 0 ]]; then
    ok "Consumer ${stream}/${consumer} created"
    return 0
  fi

  if echo "${output}" | grep -qiE "already exist|consumer name already in use"; then
    ok "Consumer ${stream}/${consumer} already exists — skipped"
    return 0
  fi

  err "Consumer ${stream}/${consumer} FAILED: ${output}"
  (( FAILURES++ )) || true
  return 1
}

# =========================================================================== #
# STREAMS                                                                      #
# =========================================================================== #
banner "Creating Streams"

# --------------------------------------------------------------------------- #
# 1. KUBRIC_EVENTS                                                             #
#    EDR / NDR / ITDR detection events from all tenants                       #
#    retention=Limits  maxAge=72h  maxBytes=5GB  dedup=2m                     #
# --------------------------------------------------------------------------- #
info "KUBRIC_EVENTS"
ensure_stream "KUBRIC_EVENTS" \
  --subjects  "kubric.*.edr.>,kubric.*.ndr.>,kubric.*.itdr.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "72h" \
  --max-bytes "${BYTES_5GB}" \
  --dupe-window "2m" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 2. KUBRIC_AGENTS                                                             #
#    Agent trigger messages for AI persona dispatch                            #
#    retention=WorkQueue  maxAge=24h                                           #
# --------------------------------------------------------------------------- #
info "KUBRIC_AGENTS"
ensure_stream "KUBRIC_AGENTS" \
  --subjects  "kubric.*.agent.>" \
  --retention "workqueue" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "24h" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 3. KUBRIC_ALERTS                                                             #
#    Security alert fan-out to NOC / SIEM pipelines                           #
#    retention=Limits  maxAge=168h (7 days)  maxBytes=2GB                     #
# --------------------------------------------------------------------------- #
info "KUBRIC_ALERTS"
ensure_stream "KUBRIC_ALERTS" \
  --subjects  "kubric.*.security.alert.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "168h" \
  --max-bytes "${BYTES_2GB}" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 4. KUBRIC_INTEL                                                              #
#    Threat intelligence IOC and feed updates                                  #
#    retention=Limits  maxAge=720h (30 days)                                  #
# --------------------------------------------------------------------------- #
info "KUBRIC_INTEL"
ensure_stream "KUBRIC_INTEL" \
  --subjects  "kubric.*.ti.ioc.>,kubric.*.ti.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "720h" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 5. KUBRIC_BILLING                                                            #
#    Usage metering events — long retention for reconciliation and audit      #
#    retention=Limits  maxAge=8760h (1 year)                                  #
# --------------------------------------------------------------------------- #
info "KUBRIC_BILLING"
ensure_stream "KUBRIC_BILLING" \
  --subjects  "kubric.*.billing.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "8760h" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 6. KUBRIC_HEALTH                                                             #
#    Platform and tenant health scores measured by the Keeper agent           #
#    retention=Limits  maxAge=720h (30 days)                                  #
# --------------------------------------------------------------------------- #
info "KUBRIC_HEALTH"
ensure_stream "KUBRIC_HEALTH" \
  --subjects  "kubric.health.score.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "720h" \
  --discard   "old"

# --------------------------------------------------------------------------- #
# 7. KUBRIC_GRC                                                                #
#    Governance, Risk and Compliance events (policies, audits, posture)       #
#    retention=Limits  maxAge=8760h (1 year)                                  #
# --------------------------------------------------------------------------- #
info "KUBRIC_GRC"
ensure_stream "KUBRIC_GRC" \
  --subjects  "kubric.*.grc.>" \
  --retention "limits" \
  --storage   "file" \
  --replicas  1 \
  --max-age   "8760h" \
  --discard   "old"

# =========================================================================== #
# PUSH CONSUMERS — KUBRIC_AGENTS                                               #
#                                                                              #
# One durable push consumer per AI persona.                                    #
# Each consumer:                                                               #
#   - filters on kubric.*.agent.{persona}.trigger                             #
#   - delivers to the queue group kubric-{persona}-workers                    #
#   - ackPolicy=Explicit  maxDeliver=5  ackWait=30s                           #
# =========================================================================== #
banner "Creating Push Consumers on KUBRIC_AGENTS"

readonly PERSONAS=(
  triage
  analyst
  hunter
  keeper
  risk
  invest
  sentinel
  foresight
  house
  bill
  comm
  deploy
  simulate
)

for persona in "${PERSONAS[@]}"; do
  consumer_name="kubric-${persona}-worker"
  deliver_subject="kubric-${persona}-workers"
  filter_subject="kubric.*.agent.${persona}.trigger"

  info "Consumer: ${consumer_name}"
  ensure_consumer "KUBRIC_AGENTS" "${consumer_name}" \
    --filter        "${filter_subject}" \
    --durable       "${consumer_name}" \
    --deliver       "new" \
    --ack           "explicit" \
    --max-deliver   5 \
    --wait          "30s" \
    --target        "${deliver_subject}" \
    --deliver-group "${deliver_subject}"
done

# =========================================================================== #
# Summary                                                                      #
# =========================================================================== #
banner "Summary"

printf "\nStreams present on %s:\n" "${NATS_URL}"
"${NATS_CLI}" --server "${NATS_URL}" stream ls 2>/dev/null || true

printf "\nConsumers present on KUBRIC_AGENTS:\n"
"${NATS_CLI}" --server "${NATS_URL}" consumer ls KUBRIC_AGENTS 2>/dev/null || true

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  err "Initialization completed with ${FAILURES} failure(s). Review output above."
  exit 1
fi

ok "All streams and consumers initialised successfully (FAILURES=0)"
exit 0
