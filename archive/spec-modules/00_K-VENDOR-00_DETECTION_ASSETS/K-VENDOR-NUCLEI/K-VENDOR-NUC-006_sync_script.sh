#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-NUC-006  —  Nuclei Templates Vendor Sync Script
# =============================================================================
# Syncs ProjectDiscovery Nuclei templates into vendor/nuclei-templates/.
#
# Upstream : https://github.com/projectdiscovery/nuclei-templates.git
# License  : MIT
# Target   : vendor/nuclei-templates/
#
# Usage:
#   ./K-VENDOR-NUC-006_sync_script.sh
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-NUC-006_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
UPSTREAM_URL="https://github.com/projectdiscovery/nuclei-templates.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/nuclei-templates"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v git >/dev/null 2>&1 || die "git is required but not found in PATH."

info "K-VENDOR-NUC-006 — Nuclei Templates vendor sync starting at $TIMESTAMP"
info "Upstream  : $UPSTREAM_URL"
info "Target    : $DEST_DIR"

# ── Ensure vendor parent directory exists ────────────────────────────────────
mkdir -p "$VENDOR_DIR"

# ── Clone or update (idempotent) ─────────────────────────────────────────────
if [ -d "$DEST_DIR/.git" ]; then
    info "Existing clone detected — pulling latest changes..."
    DEFAULT_BRANCH="$(git -C "$DEST_DIR" remote show origin 2>/dev/null \
        | grep 'HEAD branch' | awk '{print $NF}')" || DEFAULT_BRANCH="main"
    git -C "$DEST_DIR" fetch --depth=1 origin "$DEFAULT_BRANCH" 2>/dev/null \
        || die "Failed to fetch from upstream."
    git -C "$DEST_DIR" reset --hard "origin/$DEFAULT_BRANCH" --quiet \
        || die "Failed to reset to origin/$DEFAULT_BRANCH."
    info "Updated to latest upstream commit."
elif [ -d "$DEST_DIR" ]; then
    # Directory exists but is not a git clone (e.g., leftover partial sync)
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

# ── Post-sync: count templates by category ───────────────────────────────────
COMMIT_SHA="$(git -C "$DEST_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
TOTAL_TEMPLATES="$(find "$DEST_DIR" -type f -name '*.yaml' 2>/dev/null | wc -l | tr -d ' ')"

info "Nuclei Templates sync complete."
info "  Commit     : $COMMIT_SHA"
info "  Templates  : $TOTAL_TEMPLATES YAML files"

# Break down by major template categories
TEMPLATE_CATEGORIES=(
    "http"
    "dns"
    "file"
    "network"
    "headless"
    "ssl"
    "websocket"
    "cloud"
    "code"
    "workflows"
)

for category in "${TEMPLATE_CATEGORIES[@]}"; do
    cat_dir="$DEST_DIR/$category"
    if [ -d "$cat_dir" ]; then
        cat_count="$(find "$cat_dir" -type f -name '*.yaml' 2>/dev/null | wc -l | tr -d ' ')"
        info "  $category/ : $cat_count templates"
    fi
done

# CVE templates are an important sub-category within http/
CVE_DIR="$DEST_DIR/http/cves"
if [ -d "$CVE_DIR" ]; then
    CVE_COUNT="$(find "$CVE_DIR" -type f -name '*.yaml' 2>/dev/null | wc -l | tr -d ' ')"
    info "  http/cves/ : $CVE_COUNT CVE templates"
fi

info "  Path       : $DEST_DIR"
info "K-VENDOR-NUC-006 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
