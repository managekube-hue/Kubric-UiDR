#!/usr/bin/env bash
###############################################################################
# K-DEV-CICD-GHA-009 — CI Pipeline Cosign Signing
#
# Called from GitHub Actions or Woodpecker CI to sign a single container image
# after it has been built and pushed.
#
# Usage:
#   K-DEV-CICD-GHA-009_cosign_signing.sh <image_ref> <digest>
#
# Example:
#   K-DEV-CICD-GHA-009_cosign_signing.sh \
#     ghcr.io/managekube-hue/kubric-uidr/coresec \
#     sha256:abc123...
#
# The script:
#   1. Signs the image@digest with cosign (keyless or key-based).
#   2. Generates an SBOM with syft and attaches it as a cosign attestation.
#   3. Verifies the resulting signature.
#
# Env vars:
#   COSIGN_KEY            — (optional) path to a cosign private key
#   COSIGN_PASSWORD       — (optional) passphrase for the private key
#   COSIGN_YES            — set "true" to skip prompts (default: true)
#   COSIGN_EXPERIMENTAL   — set "1" for keyless transparency log (default: 1)
#   SBOM_FORMAT           — syft output format (default: spdx-json)
#   OIDC_ISSUER           — OIDC issuer URL for keyless (default: GitHub Actions)
#   OIDC_IDENTITY_REGEXP  — identity pattern for verification
#   GITHUB_ACTIONS        — automatically set in GHA runners
###############################################################################
set -euo pipefail

readonly LOG_PREFIX="[GHA-009-cosign]"

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
export COSIGN_YES="${COSIGN_YES:-true}"
export COSIGN_EXPERIMENTAL="${COSIGN_EXPERIMENTAL:-1}"
SBOM_FORMAT="${SBOM_FORMAT:-spdx-json}"
OIDC_ISSUER="${OIDC_ISSUER:-https://token.actions.githubusercontent.com}"
OIDC_IDENTITY_REGEXP="${OIDC_IDENTITY_REGEXP:-.*github.com/managekube-hue/kubric-uidr.*}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()   { echo "${LOG_PREFIX} $(date -u '+%Y-%m-%dT%H:%M:%SZ') $*"; }
err()   { log "ERROR: $*" >&2; }
die()   { err "$@"; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

usage() {
  echo "Usage: $0 <image_ref> <digest>"
  echo ""
  echo "  image_ref   Full image reference (e.g. ghcr.io/managekube-hue/kubric-uidr/coresec:v1.2.3)"
  echo "  digest      Image digest        (e.g. sha256:abcdef123456...)"
  exit 1
}

# ---------------------------------------------------------------------------
# Argument validation
# ---------------------------------------------------------------------------
if [[ $# -lt 2 ]]; then
  usage
fi

IMAGE_REF="$1"
DIGEST="$2"

# Normalise: strip tag if present and use digest form
IMAGE_NO_TAG="${IMAGE_REF%%:*}"
IMAGE_DIGEST_REF="${IMAGE_NO_TAG}@${DIGEST}"

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
preflight() {
  log "Pre-flight checks …"
  require_cmd cosign
  require_cmd syft
  require_cmd jq

  log "Image ref : ${IMAGE_REF}"
  log "Digest    : ${DIGEST}"
  log "Digest ref: ${IMAGE_DIGEST_REF}"

  if [[ -n "${COSIGN_KEY:-}" ]]; then
    [[ -f "${COSIGN_KEY}" ]] || die "COSIGN_KEY file not found: ${COSIGN_KEY}"
    log "Signing mode: key-based (${COSIGN_KEY})"
  else
    log "Signing mode: keyless (OIDC issuer=${OIDC_ISSUER})"
  fi
}

# ---------------------------------------------------------------------------
# Step 1 — Sign
# ---------------------------------------------------------------------------
sign() {
  log "Step 1/3 — Signing image …"

  local args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    args+=(--key "${COSIGN_KEY}")
  fi

  args+=(
    -a "ci=true"
    -a "pipeline=${CI_PIPELINE_ID:-${GITHUB_RUN_ID:-unknown}}"
    -a "commit=${CI_COMMIT_SHA:-${GITHUB_SHA:-unknown}}"
    -a "ref=${CI_COMMIT_REF_NAME:-${GITHUB_REF_NAME:-unknown}}"
    -a "signed-at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  )

  cosign sign "${args[@]}" "${IMAGE_DIGEST_REF}" \
    || die "cosign sign failed for ${IMAGE_DIGEST_REF}"

  log "Image signed successfully."
}

# ---------------------------------------------------------------------------
# Step 2 — Generate & attach SBOM attestation
# ---------------------------------------------------------------------------
attach_sbom() {
  log "Step 2/3 — Generating SBOM (${SBOM_FORMAT}) and attaching attestation …"

  local sbom_file
  sbom_file="$(mktemp /tmp/cosign-sbom-XXXXXX.json)"
  trap 'rm -f "${sbom_file}"' EXIT

  syft "${IMAGE_DIGEST_REF}" -o "${SBOM_FORMAT}" > "${sbom_file}" \
    || die "syft failed to generate SBOM for ${IMAGE_DIGEST_REF}"

  local sbom_size
  sbom_size="$(wc -c < "${sbom_file}")"
  log "SBOM generated (${sbom_size} bytes)."

  local attest_args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    attest_args+=(--key "${COSIGN_KEY}")
  fi

  cosign attest "${attest_args[@]}" \
    --predicate "${sbom_file}" \
    --type spdxjson \
    "${IMAGE_DIGEST_REF}" \
    || die "cosign attest failed for ${IMAGE_DIGEST_REF}"

  log "SBOM attestation attached."
  rm -f "${sbom_file}"
  trap - EXIT
}

# ---------------------------------------------------------------------------
# Step 3 — Verify
# ---------------------------------------------------------------------------
verify() {
  log "Step 3/3 — Verifying signature …"

  local args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    local pubkey="${COSIGN_KEY%.key}.pub"
    if [[ -f "${pubkey}" ]]; then
      args+=(--key "${pubkey}")
    else
      log "WARN: Public key ${pubkey} not found; falling back to keyless verification"
      args+=(
        --certificate-oidc-issuer "${OIDC_ISSUER}"
        --certificate-identity-regexp "${OIDC_IDENTITY_REGEXP}"
      )
    fi
  else
    args+=(
      --certificate-oidc-issuer "${OIDC_ISSUER}"
      --certificate-identity-regexp "${OIDC_IDENTITY_REGEXP}"
    )
  fi

  cosign verify "${args[@]}" "${IMAGE_DIGEST_REF}" \
    | jq -r '.[0] | {critical, optional}' \
    || die "Signature verification FAILED for ${IMAGE_DIGEST_REF}"

  log "Signature verification OK."
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight
  sign
  attach_sbom
  verify

  log "════════════════════════════════════════════"
  log "  ✓ ${IMAGE_DIGEST_REF}"
  log "    signed · attested · verified"
  log "════════════════════════════════════════════"
}

main
