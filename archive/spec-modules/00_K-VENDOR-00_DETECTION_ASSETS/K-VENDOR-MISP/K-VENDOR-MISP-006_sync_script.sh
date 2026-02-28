#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-MISP-006  —  MISP Data Feeds Vendor Sync Script
# =============================================================================
# Syncs MISP community data (taxonomies, galaxies, warninglists, objects, and
# default feeds) into vendor/misp/.
#
# Upstream repos:
#   - https://github.com/MISP/misp-taxonomies.git   → vendor/misp/taxonomies/
#   - https://github.com/MISP/misp-galaxy.git       → vendor/misp/galaxy/
#   - https://github.com/MISP/misp-warninglists.git → vendor/misp/warninglists/
#   - https://github.com/MISP/misp-objects.git      → vendor/misp/objects/
#   - https://github.com/MISP/misp-feeds.git        → vendor/misp/feeds/
#
# License  : CC0 / AGPL (data only — not linked as code)
# Target   : vendor/misp/
#
# Usage:
#   ./K-VENDOR-MISP-006_sync_script.sh                  # sync all MISP data
#   ./K-VENDOR-MISP-006_sync_script.sh taxonomies       # sync only taxonomies
#   ./K-VENDOR-MISP-006_sync_script.sh galaxy            # sync only galaxy
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-MISP-006_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/misp"
LOG_FILE="${LOG_FILE:-/dev/null}"
FILTER="${1:-all}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# MISP upstream repositories
declare -A MISP_REPOS=(
    [taxonomies]="https://github.com/MISP/misp-taxonomies.git"
    [galaxy]="https://github.com/MISP/misp-galaxy.git"
    [warninglists]="https://github.com/MISP/misp-warninglists.git"
    [objects]="https://github.com/MISP/misp-objects.git"
    [feeds]="https://github.com/MISP/misp-feeds.git"
)

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v git >/dev/null 2>&1 || die "git is required but not found in PATH."

info "K-VENDOR-MISP-006 — MISP data vendor sync starting at $TIMESTAMP"
info "Target    : $DEST_DIR"
info "Filter    : $FILTER"

# ── Clone-or-update helper ───────────────────────────────────────────────────
clone_or_update() {
    local url="$1"
    local dest="$2"
    local name
    name="$(basename "$dest")"

    if [ -d "$dest/.git" ]; then
        info "  $name — updating existing clone..."
        local branch
        branch="$(git -C "$dest" remote show origin 2>/dev/null \
            | grep 'HEAD branch' | awk '{print $NF}')" || branch="main"
        git -C "$dest" fetch --depth=1 origin "$branch" 2>/dev/null \
            || { warn "  $name — fetch failed, attempting re-clone."; rm -rf "$dest"; git clone --depth=1 --quiet "$url" "$dest" || die "Re-clone of $name failed."; return; }
        git -C "$dest" reset --hard "origin/$branch" --quiet \
            || die "  $name — reset failed."
        info "  $name — updated to latest."
    elif [ -d "$dest" ]; then
        warn "  $name — directory exists but is not a git repo. Removing and cloning."
        rm -rf "$dest"
        git clone --depth=1 --quiet "$url" "$dest" \
            || die "  $name — clone failed."
        info "  $name — fresh clone completed."
    else
        info "  $name — cloning from upstream..."
        git clone --depth=1 --quiet "$url" "$dest" \
            || die "  $name — clone failed."
        info "  $name — fresh clone completed."
    fi
}

# ── Sync individual MISP component ───────────────────────────────────────────
sync_component() {
    local component="$1"
    local url="${MISP_REPOS[$component]:-}"
    if [ -z "$url" ]; then
        die "Unknown MISP component: $component"
    fi
    local dest="$DEST_DIR/$component"
    clone_or_update "$url" "$dest"
}

# ── Main dispatch ────────────────────────────────────────────────────────────
mkdir -p "$DEST_DIR"

case "$FILTER" in
    all)
        for component in taxonomies galaxy warninglists objects feeds; do
            sync_component "$component"
        done
        ;;
    taxonomies|galaxy|warninglists|objects|feeds)
        sync_component "$FILTER"
        ;;
    *)
        die "Unknown filter: $FILTER. Valid options: all, taxonomies, galaxy, warninglists, objects, feeds"
        ;;
esac

# ── Post-sync summary ───────────────────────────────────────────────────────
info "MISP data sync complete."
for component in taxonomies galaxy warninglists objects feeds; do
    comp_dir="$DEST_DIR/$component"
    if [ -d "$comp_dir" ]; then
        json_count="$(find "$comp_dir" -type f -name '*.json' 2>/dev/null | wc -l | tr -d ' ')"
        sha="$(git -C "$comp_dir" rev-parse --short HEAD 2>/dev/null || echo 'n/a')"
        info "  $component : $json_count JSON files (commit $sha)"
    fi
done
info "  Path : $DEST_DIR"
info "K-VENDOR-MISP-006 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
