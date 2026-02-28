#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-MIT-005  —  MITRE ATT&CK / CWE / CAPEC STIX Vendor Sync Script
# =============================================================================
# Syncs MITRE ATT&CK STIX data, CWE, and CAPEC bundles into vendor/mitre/.
#
# Upstream :
#   - https://github.com/mitre-attack/attack-stix-data.git (ATT&CK STIX 2.1)
#   - CAPEC STIX: https://raw.githubusercontent.com/mitre/cti/master/capec/2.1/stix-capec.json
#   - CWE  XML : https://cwe.mitre.org/data/xml/cwec_latest.xml.zip
#
# License  : CC BY 4.0 (ATT&CK), public domain (CWE/CAPEC)
# Target   : vendor/mitre/
#
# Usage:
#   ./K-VENDOR-MIT-005_sync_script.sh
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-MIT-005_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
ATTACK_STIX_URL="https://github.com/mitre-attack/attack-stix-data.git"
CAPEC_STIX_URL="https://raw.githubusercontent.com/mitre/cti/master/capec/2.1/stix-capec.json"
CWE_XML_URL="https://cwe.mitre.org/data/xml/cwec_latest.xml.zip"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/mitre"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v git  >/dev/null 2>&1 || die "git is required but not found in PATH."
command -v curl >/dev/null 2>&1 || die "curl is required but not found in PATH."

info "K-VENDOR-MIT-005 — MITRE vendor sync starting at $TIMESTAMP"
info "Target    : $DEST_DIR"

# ── Ensure target directory exists ───────────────────────────────────────────
mkdir -p "$DEST_DIR"

# ── 1. ATT&CK STIX 2.1 data (git clone) ─────────────────────────────────────
ATTACK_DIR="$DEST_DIR/attack-stix-data"
info "Syncing MITRE ATT&CK STIX 2.1 data..."

if [ -d "$ATTACK_DIR/.git" ]; then
    info "  Existing clone detected — pulling latest..."
    DEFAULT_BRANCH="$(git -C "$ATTACK_DIR" remote show origin 2>/dev/null \
        | grep 'HEAD branch' | awk '{print $NF}')" || DEFAULT_BRANCH="master"
    git -C "$ATTACK_DIR" fetch --depth=1 origin "$DEFAULT_BRANCH" 2>/dev/null \
        || die "Failed to fetch attack-stix-data."
    git -C "$ATTACK_DIR" reset --hard "origin/$DEFAULT_BRANCH" --quiet \
        || die "Failed to reset attack-stix-data."
    info "  ATT&CK STIX updated."
elif [ -d "$ATTACK_DIR" ]; then
    warn "  $ATTACK_DIR exists but is not a git repo — re-cloning."
    rm -rf "$ATTACK_DIR"
    git clone --depth=1 --quiet "$ATTACK_STIX_URL" "$ATTACK_DIR" \
        || die "git clone of attack-stix-data failed."
    info "  ATT&CK STIX cloned."
else
    info "  Cloning attack-stix-data..."
    git clone --depth=1 --quiet "$ATTACK_STIX_URL" "$ATTACK_DIR" \
        || die "git clone of attack-stix-data failed."
    info "  ATT&CK STIX cloned."
fi

# ── 2. CAPEC STIX bundle (single JSON file) ─────────────────────────────────
CAPEC_DIR="$DEST_DIR/capec"
mkdir -p "$CAPEC_DIR"
info "Downloading CAPEC STIX bundle..."
if curl -sSL --retry 3 --retry-delay 5 \
    -o "$CAPEC_DIR/stix-capec.json" "$CAPEC_STIX_URL" 2>/dev/null; then
    CAPEC_SIZE="$(wc -c < "$CAPEC_DIR/stix-capec.json" | tr -d ' ')"
    info "  CAPEC downloaded ($CAPEC_SIZE bytes)"
else
    warn "  CAPEC download failed (non-fatal) — previous version may still exist."
fi

# ── 3. CWE XML data (zip archive) ───────────────────────────────────────────
CWE_DIR="$DEST_DIR/cwe"
mkdir -p "$CWE_DIR"
info "Downloading CWE XML data..."
CWE_ZIP="$CWE_DIR/cwec_latest.xml.zip"
if curl -sSL --retry 3 --retry-delay 5 \
    -o "$CWE_ZIP" "$CWE_XML_URL" 2>/dev/null; then
    if command -v unzip >/dev/null 2>&1; then
        unzip -oq "$CWE_ZIP" -d "$CWE_DIR" 2>/dev/null \
            || warn "  CWE unzip failed — zip file may be corrupt."
        rm -f "$CWE_ZIP"
        info "  CWE XML extracted."
    else
        warn "  unzip not available — keeping $CWE_ZIP as-is."
    fi
else
    warn "  CWE download failed (non-fatal) — previous version may still exist."
fi

# ── 4. Legacy enterprise/ICS single-file downloads (convenience copies) ─────
info "Downloading legacy single-file ATT&CK bundles..."
LEGACY_DIR="$DEST_DIR/legacy"
mkdir -p "$LEGACY_DIR"

ENTERPRISE_URL="https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json"
ICS_URL="https://raw.githubusercontent.com/mitre/cti/master/ics-attack/ics-attack.json"
MOBILE_URL="https://raw.githubusercontent.com/mitre/cti/master/mobile-attack/mobile-attack.json"

for entry in "enterprise-attack.json|$ENTERPRISE_URL" "ics-attack.json|$ICS_URL" "mobile-attack.json|$MOBILE_URL"; do
    fname="${entry%%|*}"
    url="${entry##*|}"
    if curl -sSL --retry 2 -o "$LEGACY_DIR/$fname" "$url" 2>/dev/null; then
        info "  Downloaded $fname"
    else
        warn "  Failed to download $fname (non-fatal)"
    fi
done

# ── Write sync metadata ─────────────────────────────────────────────────────
ATTACK_SHA="$(git -C "$ATTACK_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
cat > "$DEST_DIR/.sync-metadata.json" <<METAEOF
{
    "vendor_id": "K-VENDOR-MIT-005",
    "synced_at": "$TIMESTAMP",
    "attack_stix_commit": "$ATTACK_SHA",
    "sources": {
        "attack_stix": "$ATTACK_STIX_URL",
        "capec_stix": "$CAPEC_STIX_URL",
        "cwe_xml": "$CWE_XML_URL"
    },
    "license": "CC-BY-4.0 (ATT&CK), Public Domain (CWE/CAPEC)"
}
METAEOF

# ── Post-sync summary ───────────────────────────────────────────────────────
STIX_COUNT="$(find "$ATTACK_DIR" -type f -name '*.json' 2>/dev/null | wc -l | tr -d ' ')"
info "MITRE vendor sync complete."
info "  ATT&CK STIX : $STIX_COUNT JSON files (commit $ATTACK_SHA)"
info "  CAPEC       : $CAPEC_DIR/stix-capec.json"
info "  CWE         : $CWE_DIR/"
info "  Legacy      : $LEGACY_DIR/"
info "  Path        : $DEST_DIR"
info "K-VENDOR-MIT-005 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
