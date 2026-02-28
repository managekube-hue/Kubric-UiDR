#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-SIG-006  —  Sigma Rules Vendor Sync Script
# =============================================================================
# Syncs the SigmaHQ detection rules into vendor/sigma/.
#
# Upstream : https://github.com/SigmaHQ/sigma.git
# License  : Apache 2.0
# Target   : vendor/sigma/
#
# Usage:
#   ./K-VENDOR-SIG-006_sync_script.sh              # uses default VENDOR_DIR
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-SIG-006_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
UPSTREAM_URL="https://github.com/SigmaHQ/sigma.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/sigma"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v git >/dev/null 2>&1 || die "git is required but not found in PATH."

info "K-VENDOR-SIG-006 — Sigma vendor sync starting at $TIMESTAMP"
info "Upstream  : $UPSTREAM_URL"
info "Target    : $DEST_DIR"

# ── Ensure vendor parent directory exists ────────────────────────────────────
mkdir -p "$VENDOR_DIR"

# ── Clone or update (idempotent) ─────────────────────────────────────────────
if [ -d "$DEST_DIR/.git" ]; then
    info "Existing clone detected — pulling latest changes..."
    git -C "$DEST_DIR" fetch --depth=1 origin main 2>/dev/null \
        || git -C "$DEST_DIR" fetch --depth=1 origin master 2>/dev/null \
        || die "Failed to fetch from upstream."
    # Reset to the fetched HEAD to handle any divergence cleanly
    DEFAULT_BRANCH="$(git -C "$DEST_DIR" remote show origin 2>/dev/null \
        | grep 'HEAD branch' | awk '{print $NF}')" || DEFAULT_BRANCH="main"
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
RULE_COUNT=0
if [ -d "$DEST_DIR/rules" ]; then
    RULE_COUNT="$(find "$DEST_DIR/rules" -type f -name '*.yml' 2>/dev/null | wc -l | tr -d ' ')"
fi
COMMIT_SHA="$(git -C "$DEST_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

info "Sigma sync complete."
info "  Commit  : $COMMIT_SHA"
info "  Rules   : $RULE_COUNT YAML files in rules/"
info "  Path    : $DEST_DIR"
info "K-VENDOR-SIG-006 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
