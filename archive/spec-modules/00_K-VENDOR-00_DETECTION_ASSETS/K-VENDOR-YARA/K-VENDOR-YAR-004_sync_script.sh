#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-YAR-004  —  YARA Rules Vendor Sync Script
# =============================================================================
# Syncs the Yara-Rules community detection rules into vendor/yara-rules/.
#
# Upstream : https://github.com/Yara-Rules/rules.git
# License  : BSD / Apache 2.0 mix
# Target   : vendor/yara-rules/
#
# Usage:
#   ./K-VENDOR-YAR-004_sync_script.sh
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-YAR-004_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
UPSTREAM_URL="https://github.com/Yara-Rules/rules.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/yara-rules"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v git >/dev/null 2>&1 || die "git is required but not found in PATH."

info "K-VENDOR-YAR-004 — YARA Rules vendor sync starting at $TIMESTAMP"
info "Upstream  : $UPSTREAM_URL"
info "Target    : $DEST_DIR"

# ── Ensure vendor parent directory exists ────────────────────────────────────
mkdir -p "$VENDOR_DIR"

# ── Clone or update (idempotent) ─────────────────────────────────────────────
if [ -d "$DEST_DIR/.git" ]; then
    info "Existing clone detected — pulling latest changes..."
    DEFAULT_BRANCH="$(git -C "$DEST_DIR" remote show origin 2>/dev/null \
        | grep 'HEAD branch' | awk '{print $NF}')" || DEFAULT_BRANCH="master"
    git -C "$DEST_DIR" fetch --depth=1 origin "$DEFAULT_BRANCH" 2>/dev/null \
        || die "Failed to fetch from upstream."
    git -C "$DEST_DIR" reset --hard "origin/$DEFAULT_BRANCH" --quiet \
        || die "Failed to reset to origin/$DEFAULT_BRANCH."
    info "Updated to latest upstream commit."
elif [ -d "$DEST_DIR" ]; then
    warn "$DEST_DIR exists but is not a git repo — removing and re-cloning."
    rm -rf "$DEST_DIR"
    git clone --depth=1 --quiet "$UPSTREAM_URL" "$DEST_DIR" \
        || die "git clone failed."
    info "Fresh clone completed."
else
    info "No existing clone — cloning from upstream..."
    git clone --depth=1 --quiet "$UPSTREAM_URL" "$DEST_DIR" \
        || die "git clone failed."
    info "Fresh clone completed."
fi

# ── Post-sync summary ───────────────────────────────────────────────────────
YAR_COUNT="$(find "$DEST_DIR" -type f \( -name '*.yar' -o -name '*.yara' \) 2>/dev/null | wc -l | tr -d ' ')"
COMMIT_SHA="$(git -C "$DEST_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# List the major rule categories found
info "YARA Rules sync complete."
info "  Commit      : $COMMIT_SHA"
info "  Rule files  : $YAR_COUNT (.yar / .yara)"
info "  Path        : $DEST_DIR"

if [ -d "$DEST_DIR/malware" ]; then
    MALWARE_COUNT="$(find "$DEST_DIR/malware" -type f \( -name '*.yar' -o -name '*.yara' \) 2>/dev/null | wc -l | tr -d ' ')"
    info "  malware/    : $MALWARE_COUNT rules"
fi
if [ -d "$DEST_DIR/CVE_Rules" ]; then
    CVE_COUNT="$(find "$DEST_DIR/CVE_Rules" -type f \( -name '*.yar' -o -name '*.yara' \) 2>/dev/null | wc -l | tr -d ' ')"
    info "  CVE_Rules/  : $CVE_COUNT rules"
fi

info "K-VENDOR-YAR-004 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
