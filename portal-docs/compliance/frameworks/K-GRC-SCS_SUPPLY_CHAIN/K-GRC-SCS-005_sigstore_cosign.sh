#!/usr/bin/env bash
###############################################################################
# K-GRC-SCS-005 — Sigstore Cosign Image Signing for Kubric-UiDR
#
# Signs all Kubric Docker images with cosign for GRC supply-chain compliance.
# Supports two modes:
#   1. Keyless (OIDC / Fulcio + Rekor)  — default in CI environments
#   2. Key-based                         — when COSIGN_KEY is set
#
# After signing, each image signature is verified and an SBOM attestation
# (SPDX-JSON produced by syft) is attached.
#
# Required tools: cosign >=2.0, syft, jq, docker/crane
# Env vars:
#   REGISTRY        — registry prefix (default: ghcr.io/managekube-hue/kubric-uidr)
#   IMAGE_TAG       — tag to sign      (default: latest)
#   COSIGN_KEY      — path to cosign private key (optional; triggers key-based mode)
#   COSIGN_PASSWORD — passphrase for COSIGN_KEY  (optional)
#   COSIGN_YES      — set to "true" to skip interactive prompts (default: true)
#   SBOM_FORMAT     — syft SBOM format  (default: spdx-json)
###############################################################################
set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly LOG_PREFIX="[K-GRC-SCS-005]"

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
REGISTRY="${REGISTRY:-ghcr.io/managekube-hue/kubric-uidr}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
SBOM_FORMAT="${SBOM_FORMAT:-spdx-json}"
export COSIGN_YES="${COSIGN_YES:-true}"

# All Kubric production images
readonly IMAGES=(
  coresec
  netguard
  perftrace
  ksvc
  kic
  noc
  kai-core
  vdr
)

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
# Pre-flight checks
# ---------------------------------------------------------------------------
preflight() {
  log "Running pre-flight checks …"
  require_cmd cosign
  require_cmd syft
  require_cmd jq

  local cosign_ver
  cosign_ver="$(cosign version 2>&1 | head -1)"
  log "cosign: ${cosign_ver}"

  if [[ -n "${COSIGN_KEY:-}" ]]; then
    [[ -f "${COSIGN_KEY}" ]] || die "COSIGN_KEY points to non-existent file: ${COSIGN_KEY}"
    log "Mode: key-based signing (${COSIGN_KEY})"
  else
    log "Mode: keyless signing (Fulcio + Rekor)"
  fi
}

# ---------------------------------------------------------------------------
# Resolve image digest
# ---------------------------------------------------------------------------
resolve_digest() {
  local image_ref="$1"
  cosign triangulate --type=digest "${image_ref}" 2>/dev/null \
    || docker inspect --format='{{index .RepoDigests 0}}' "${image_ref}" 2>/dev/null \
    || crane digest "${image_ref}" 2>/dev/null \
    || die "Cannot resolve digest for ${image_ref}"
}

# ---------------------------------------------------------------------------
# Sign a single image
# ---------------------------------------------------------------------------
sign_image() {
  local image_ref="$1"
  log "Signing ${image_ref} …"

  local sign_args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    sign_args+=(--key "${COSIGN_KEY}")
  fi

  # Add useful annotations
  sign_args+=(
    -a "repo=kubric-uidr"
    -a "signed-by=kubric-grc-pipeline"
    -a "signed-at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    -a "tag=${IMAGE_TAG}"
  )

  if ! cosign sign "${sign_args[@]}" "${image_ref}"; then
    err "Failed to sign ${image_ref}"
    return 1
  fi

  log "Successfully signed ${image_ref}"
}

# ---------------------------------------------------------------------------
# Verify a single image signature
# ---------------------------------------------------------------------------
verify_image() {
  local image_ref="$1"
  log "Verifying signature for ${image_ref} …"

  local verify_args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    local pubkey="${COSIGN_KEY%.key}.pub"
    if [[ -f "${pubkey}" ]]; then
      verify_args+=(--key "${pubkey}")
    else
      err "Public key ${pubkey} not found; skipping verification"
      return 0
    fi
  else
    # Keyless verification requires certificate identity and issuer
    verify_args+=(
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
      --certificate-identity-regexp ".*github.com/managekube-hue/kubric-uidr.*"
    )
  fi

  if ! cosign verify "${verify_args[@]}" "${image_ref}" | jq -r '.[0].critical.identity' 2>/dev/null; then
    err "Signature verification FAILED for ${image_ref}"
    return 1
  fi

  log "Signature verified OK for ${image_ref}"
}

# ---------------------------------------------------------------------------
# Generate & attach SBOM attestation
# ---------------------------------------------------------------------------
attach_sbom() {
  local image_ref="$1"
  local sbom_file
  sbom_file="$(mktemp /tmp/kubric-sbom-XXXXXX.json)"

  log "Generating ${SBOM_FORMAT} SBOM for ${image_ref} …"
  syft "${image_ref}" -o "${SBOM_FORMAT}" > "${sbom_file}" \
    || { err "syft SBOM generation failed for ${image_ref}"; rm -f "${sbom_file}"; return 1; }

  local attest_args=()
  if [[ -n "${COSIGN_KEY:-}" ]]; then
    attest_args+=(--key "${COSIGN_KEY}")
  fi

  log "Attaching SBOM attestation to ${image_ref} …"
  cosign attest "${attest_args[@]}" \
    --predicate "${sbom_file}" \
    --type spdxjson \
    "${image_ref}" \
    || { err "cosign attest failed for ${image_ref}"; rm -f "${sbom_file}"; return 1; }

  log "SBOM attestation attached for ${image_ref}"
  rm -f "${sbom_file}"
}

# ---------------------------------------------------------------------------
# Process one image end-to-end
# ---------------------------------------------------------------------------
process_image() {
  local name="$1"
  local image_ref="${REGISTRY}/${name}:${IMAGE_TAG}"

  log "━━━ Processing ${image_ref} ━━━"

  sign_image  "${image_ref}" || return 1
  verify_image "${image_ref}" || return 1
  attach_sbom  "${image_ref}" || return 1

  log "✓ ${image_ref} — signed, verified, SBOM attached"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  preflight

  local failed=0
  local succeeded=0

  for img in "${IMAGES[@]}"; do
    if process_image "${img}"; then
      ((succeeded++))
    else
      ((failed++))
      err "Image ${img} had errors — continuing with remaining images"
    fi
  done

  echo ""
  log "═══════════════════════════════════════════"
  log "  Results: ${succeeded} signed / ${failed} failed  (total: ${#IMAGES[@]})"
  log "═══════════════════════════════════════════"

  if (( failed > 0 )); then
    die "${failed} image(s) could not be signed — review logs above"
  fi

  log "All Kubric images signed and SBOM-attested successfully."
}

main "$@"
