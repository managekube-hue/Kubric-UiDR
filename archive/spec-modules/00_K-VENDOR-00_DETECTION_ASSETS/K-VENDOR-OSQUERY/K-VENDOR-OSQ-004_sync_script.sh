#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-OSQ-004  —  osquery Packs Vendor Sync Script
# =============================================================================
# Syncs osquery detection/compliance packs from the upstream repository
# into vendor/osquery/. Only the packs/ directory is fetched (not the full
# osquery source tree) to minimize disk usage.
#
# Upstream : https://github.com/osquery/osquery.git  (packs/ directory)
# License  : Apache 2.0
# Target   : vendor/osquery/
#
# Usage:
#   ./K-VENDOR-OSQ-004_sync_script.sh
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-OSQ-004_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
UPSTREAM_URL="https://github.com/osquery/osquery.git"
PACKS_RAW_BASE="https://raw.githubusercontent.com/osquery/osquery/master/packs"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/osquery"
PACKS_DIR="$DEST_DIR/packs"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# Known upstream pack files
PACK_FILES=(
    "it-compliance.conf"
    "incident-response.conf"
    "ossec-rootkit.conf"
    "vuln-management.conf"
    "hardware-monitoring.conf"
    "windows-hardening.conf"
    "unwanted-chrome-extensions.conf"
    "osx-attacks.conf"
)

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
HAS_GIT=false
HAS_CURL=false
command -v git  >/dev/null 2>&1 && HAS_GIT=true
command -v curl >/dev/null 2>&1 && HAS_CURL=true

if ! $HAS_GIT && ! $HAS_CURL; then
    die "Either git or curl is required but neither was found in PATH."
fi

info "K-VENDOR-OSQ-004 — osquery packs vendor sync starting at $TIMESTAMP"
info "Target    : $DEST_DIR"

# ── Ensure target directory exists ───────────────────────────────────────────
mkdir -p "$PACKS_DIR"

# ── Strategy: sparse checkout (git) or direct download (curl) ────────────────
# Prefer git sparse-checkout to get the packs/ directory only, avoiding the
# full osquery source tree (~200 MB). Fall back to curl if git is unavailable
# or sparse-checkout is not supported.

sync_with_git() {
    local clone_dir="$DEST_DIR/.osquery-sparse"

    if [ -d "$clone_dir/.git" ]; then
        info "Updating existing sparse clone..."
        git -C "$clone_dir" fetch --depth=1 origin master 2>/dev/null \
            || git -C "$clone_dir" fetch --depth=1 origin main 2>/dev/null \
            || { warn "Sparse fetch failed — falling back to curl method."; sync_with_curl; return; }
        local branch
        branch="$(git -C "$clone_dir" remote show origin 2>/dev/null \
            | grep 'HEAD branch' | awk '{print $NF}')" || branch="master"
        git -C "$clone_dir" reset --hard "origin/$branch" --quiet || true
    else
        info "Creating sparse clone for packs/ only..."
        rm -rf "$clone_dir"
        git clone --depth=1 --filter=blob:none --sparse --quiet \
            "$UPSTREAM_URL" "$clone_dir" 2>/dev/null || {
            warn "Sparse clone not supported — falling back to curl method."
            sync_with_curl
            return
        }
        git -C "$clone_dir" sparse-checkout set packs 2>/dev/null || {
            warn "sparse-checkout set failed — falling back to curl method."
            rm -rf "$clone_dir"
            sync_with_curl
            return
        }
    fi

    # Copy packs into the clean output directory
    if [ -d "$clone_dir/packs" ]; then
        cp -a "$clone_dir/packs/"* "$PACKS_DIR/" 2>/dev/null || true
        info "Packs synced from sparse checkout."
    else
        warn "packs/ not found in sparse checkout — falling back to curl."
        sync_with_curl
    fi
}

sync_with_curl() {
    info "Downloading pack files individually via curl..."
    local success=0
    local fail=0
    for pack in "${PACK_FILES[@]}"; do
        if curl -sSL --retry 3 --retry-delay 5 \
            -o "$PACKS_DIR/$pack" "${PACKS_RAW_BASE}/$pack" 2>/dev/null; then
            info "  Downloaded $pack"
            success=$((success + 1))
        else
            warn "  Failed to download $pack"
            # Clean up empty files from failed downloads
            [ -f "$PACKS_DIR/$pack" ] && [ ! -s "$PACKS_DIR/$pack" ] && rm -f "$PACKS_DIR/$pack"
            fail=$((fail + 1))
        fi
    done
    info "curl downloads: $success succeeded, $fail failed."
    if [ "$success" -eq 0 ]; then
        die "All pack downloads failed — check network connectivity."
    fi
}

# ── Execute sync ─────────────────────────────────────────────────────────────
if $HAS_GIT; then
    sync_with_git
else
    sync_with_curl
fi

# ── Write sync metadata ─────────────────────────────────────────────────────
cat > "$DEST_DIR/.sync-metadata.json" <<METAEOF
{
    "vendor_id": "K-VENDOR-OSQ-004",
    "upstream": "$UPSTREAM_URL",
    "scope": "packs/ directory only",
    "synced_at": "$TIMESTAMP",
    "license": "Apache-2.0"
}
METAEOF

# ── Post-sync summary ───────────────────────────────────────────────────────
PACK_COUNT="$(find "$PACKS_DIR" -type f -name '*.conf' 2>/dev/null | wc -l | tr -d ' ')"

info "osquery packs sync complete."
info "  Pack files : $PACK_COUNT .conf files"

# List each pack with its size
for packfile in "$PACKS_DIR"/*.conf; do
    if [ -f "$packfile" ]; then
        size="$(wc -c < "$packfile" | tr -d ' ')"
        info "  $(basename "$packfile") ($size bytes)"
    fi
done

info "  Path       : $PACKS_DIR"
info "K-VENDOR-OSQ-004 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
