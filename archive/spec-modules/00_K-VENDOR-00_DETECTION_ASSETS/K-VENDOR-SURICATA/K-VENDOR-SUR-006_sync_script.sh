#!/usr/bin/env bash
# =============================================================================
# K-VENDOR-SUR-006  —  Suricata ET Open Rules Vendor Sync Script
# =============================================================================
# Syncs Emerging Threats Open Suricata rules into vendor/suricata/.
#
# Upstream : https://rules.emergingthreats.net/open/suricata-6.0/rules/
#            (tarball: emerging.rules.tar.gz)
# License  : GPL 2.0 (loaded as data files, not linked as code)
# Target   : vendor/suricata/
#
# Usage:
#   ./K-VENDOR-SUR-006_sync_script.sh
#   VENDOR_DIR=/opt/kubric/vendor ./K-VENDOR-SUR-006_sync_script.sh
# =============================================================================

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
ET_BASE_URL="https://rules.emergingthreats.net/open/suricata-6.0"
TARBALL_URL="${ET_BASE_URL}/emerging.rules.tar.gz"
MD5_URL="${ET_BASE_URL}/emerging.rules.tar.gz.md5"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENDOR_DIR="${VENDOR_DIR:-$REPO_ROOT/vendor}"
DEST_DIR="$VENDOR_DIR/suricata"
LOG_FILE="${LOG_FILE:-/dev/null}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# ── Logging helpers ──────────────────────────────────────────────────────────
info()  { echo "[INFO]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE"; }
warn()  { echo "[WARN]  $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
error() { echo "[ERROR] $(date -u +%H:%M:%S) $*" | tee -a "$LOG_FILE" >&2; }
die()   { error "$@"; exit 1; }

# ── Preflight checks ────────────────────────────────────────────────────────
command -v curl >/dev/null 2>&1 || die "curl is required but not found in PATH."
command -v tar  >/dev/null 2>&1 || die "tar is required but not found in PATH."

info "K-VENDOR-SUR-006 — Suricata ET Open vendor sync starting at $TIMESTAMP"
info "Upstream  : $TARBALL_URL"
info "Target    : $DEST_DIR"

# ── Ensure target directory exists ───────────────────────────────────────────
mkdir -p "$DEST_DIR"

# ── Download tarball ─────────────────────────────────────────────────────────
TARBALL_PATH="$DEST_DIR/emerging.rules.tar.gz"

info "Downloading ET Open rules tarball..."
curl -sSL --retry 3 --retry-delay 5 \
    -o "$TARBALL_PATH" "$TARBALL_URL" \
    || die "Failed to download $TARBALL_URL"

# ── Optional: verify MD5 checksum ────────────────────────────────────────────
if command -v md5sum >/dev/null 2>&1; then
    info "Verifying MD5 checksum..."
    EXPECTED_MD5="$(curl -sSL "$MD5_URL" 2>/dev/null | awk '{print $1}')" || true
    if [ -n "${EXPECTED_MD5:-}" ]; then
        ACTUAL_MD5="$(md5sum "$TARBALL_PATH" | awk '{print $1}')"
        if [ "$EXPECTED_MD5" = "$ACTUAL_MD5" ]; then
            info "MD5 checksum verified: $ACTUAL_MD5"
        else
            warn "MD5 mismatch — expected $EXPECTED_MD5, got $ACTUAL_MD5 (continuing anyway)"
        fi
    else
        warn "Could not fetch upstream MD5 — skipping verification."
    fi
else
    warn "md5sum not available — skipping checksum verification."
fi

# ── Extract rules (idempotent — overwrites previous extraction) ──────────────
RULES_DIR="$DEST_DIR/rules"
info "Extracting rules to $RULES_DIR ..."

# The tarball typically extracts into a rules/ subdirectory
tar -xzf "$TARBALL_PATH" -C "$DEST_DIR" 2>/dev/null \
    || die "Failed to extract $TARBALL_PATH"

# Clean up tarball after successful extraction
rm -f "$TARBALL_PATH"

# ── Download individual rule category files for granular access ──────────────
INDIVIDUAL_RULES_URL="${ET_BASE_URL}/rules/"
RULE_CATEGORIES=(
    "emerging-malware.rules"
    "emerging-trojan.rules"
    "emerging-exploit.rules"
    "emerging-web_server.rules"
    "emerging-web_client.rules"
    "emerging-scan.rules"
    "emerging-dos.rules"
    "emerging-dns.rules"
    "emerging-info.rules"
    "emerging-policy.rules"
    "emerging-hunting.rules"
)

info "Downloading individual rule category files..."
mkdir -p "$RULES_DIR"
DOWNLOAD_FAILURES=0
for rulefile in "${RULE_CATEGORIES[@]}"; do
    if curl -sSL --retry 2 -o "$RULES_DIR/$rulefile" \
        "${INDIVIDUAL_RULES_URL}${rulefile}" 2>/dev/null; then
        : # success
    else
        warn "Could not fetch $rulefile (non-fatal)"
        DOWNLOAD_FAILURES=$((DOWNLOAD_FAILURES + 1))
    fi
done

# ── Write sync metadata ─────────────────────────────────────────────────────
cat > "$DEST_DIR/.sync-metadata.json" <<METAEOF
{
    "vendor_id": "K-VENDOR-SUR-006",
    "upstream": "$TARBALL_URL",
    "synced_at": "$TIMESTAMP",
    "license": "GPL-2.0 (data files)"
}
METAEOF

# ── Post-sync summary ───────────────────────────────────────────────────────
RULE_FILE_COUNT="$(find "$DEST_DIR" -type f -name '*.rules' 2>/dev/null | wc -l | tr -d ' ')"
info "Suricata ET Open sync complete."
info "  Rule files   : $RULE_FILE_COUNT"
info "  Dl failures  : $DOWNLOAD_FAILURES"
info "  Path         : $DEST_DIR"
info "K-VENDOR-SUR-006 finished at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
