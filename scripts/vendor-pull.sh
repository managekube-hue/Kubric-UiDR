#!/usr/bin/env bash
# =============================================================================
# scripts/vendor-pull.sh — Pull all detection & intelligence vendor assets (L3-1)
# =============================================================================
# Run once after cloning the repo, then re-run to update.
#
# Usage:
#   ./scripts/vendor-pull.sh          # pull everything
#   ./scripts/vendor-pull.sh sigma    # pull only Sigma rules
#   ./scripts/vendor-pull.sh yara     # pull only YARA rules
#   ./scripts/vendor-pull.sh mitre    # pull only MITRE ATT&CK
#
# License summary of each data set:
#   sigma/rules        — Apache 2.0 (SigmaHQ)
#   yara-rules         — BSD/Apache 2.0 mix (Yara-Rules org)
#   suricata           — GPL 2.0 data files (ET rules — load as data, not code)
#   misp               — CC0 (taxonomies, galaxies, warninglists, objects)
#   mitre              — CC BY 4.0 (ATT&CK STIX bundles)
#   nuclei-templates   — MIT (ProjectDiscovery)
#   falco/rules        — Apache 2.0
#   bloodhound/cypher  — Apache 2.0 (community queries)
#   zeek/scripts       — BSD-3
#   osquery/packs      — Apache 2.0
#   coreruleset        — Apache 2.0 (OWASP ModSecurity)
#   velociraptor       — AGPL 3.0 artifacts loaded as data only
# =============================================================================

set -euo pipefail

VENDOR_DIR="$(cd "$(dirname "$0")/.." && pwd)/vendor"
FILTER="${1:-all}"

info()  { echo "[INFO]  $*"; }
warn()  { echo "[WARN]  $*"; }
clone_or_pull() {
    local url="$1" dest="$2"
    if [ -d "$dest/.git" ]; then
        info "Updating $(basename "$dest")..."
        git -C "$dest" pull --ff-only --quiet
    else
        info "Cloning $(basename "$dest")..."
        git clone --depth=1 --quiet "$url" "$dest"
    fi
}

# ─── Sigma Rules (Apache 2.0) ─────────────────────────────────────────────────
pull_sigma() {
    clone_or_pull \
        "https://github.com/SigmaHQ/sigma.git" \
        "$VENDOR_DIR/sigma"
    # Keep only the rules/ directory to save space
    info "Sigma: $(find "$VENDOR_DIR/sigma/rules" -name '*.yml' | wc -l) rules"
}

# ─── YARA Rules (BSD/Apache 2.0 mix) ─────────────────────────────────────────
pull_yara() {
    clone_or_pull \
        "https://github.com/Yara-Rules/rules.git" \
        "$VENDOR_DIR/yara-rules"
    info "YARA: $(find "$VENDOR_DIR/yara-rules" -name '*.yar' -o -name '*.yara' | wc -l) rules"
}

# ─── MITRE ATT&CK STIX (CC BY 4.0) ──────────────────────────────────────────
pull_mitre() {
    local dest="$VENDOR_DIR/mitre"
    mkdir -p "$dest"
    local url_enterprise="https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json"
    local url_ics="https://raw.githubusercontent.com/mitre/cti/master/ics-attack/ics-attack.json"
    info "Downloading MITRE ATT&CK Enterprise STIX..."
    curl -sSL "$url_enterprise" -o "$dest/enterprise-attack.json"
    info "Downloading MITRE ATT&CK ICS STIX..."
    curl -sSL "$url_ics" -o "$dest/ics-attack.json"
    info "MITRE: ATT&CK bundles downloaded to $dest"
}

# ─── MISP Data (CC0 — taxonomies, galaxies, warninglists) ────────────────────
pull_misp() {
    clone_or_pull "https://github.com/MISP/misp-taxonomies.git"  "$VENDOR_DIR/misp/taxonomies"
    clone_or_pull "https://github.com/MISP/misp-galaxy.git"      "$VENDOR_DIR/misp/galaxy"
    clone_or_pull "https://github.com/MISP/misp-warninglists.git" "$VENDOR_DIR/misp/warninglists"
    clone_or_pull "https://github.com/MISP/misp-objects.git"     "$VENDOR_DIR/misp/objects"
    info "MISP: taxonomies + galaxy + warninglists + objects pulled"
}

# ─── Nuclei Templates (MIT) ───────────────────────────────────────────────────
pull_nuclei() {
    clone_or_pull \
        "https://github.com/projectdiscovery/nuclei-templates.git" \
        "$VENDOR_DIR/nuclei-templates"
    info "Nuclei: $(find "$VENDOR_DIR/nuclei-templates" -name '*.yaml' | wc -l) templates"
}

# ─── Falco Rules (Apache 2.0) ─────────────────────────────────────────────────
pull_falco() {
    local dest="$VENDOR_DIR/falco/rules"
    mkdir -p "$dest"
    local url="https://raw.githubusercontent.com/falcosecurity/rules/main/rules/falco_rules.yaml"
    info "Downloading Falco default rules..."
    curl -sSL "$url" -o "$dest/falco_rules.yaml"
    info "Falco: rules downloaded"
}

# ─── BloodHound Cypher Queries (Apache 2.0) ───────────────────────────────────
pull_bloodhound() {
    clone_or_pull \
        "https://github.com/ly4k/BloodHound.git" \
        "$VENDOR_DIR/bloodhound"
    info "BloodHound: Cypher queries pulled"
}

# ─── Zeek Scripts (BSD-3) ────────────────────────────────────────────────────
pull_zeek() {
    clone_or_pull \
        "https://github.com/zeek/zeek.git" \
        "$VENDOR_DIR/zeek"
    info "Zeek: scripts pulled"
}

# ─── osquery packs (Apache 2.0) ──────────────────────────────────────────────
pull_osquery() {
    local dest="$VENDOR_DIR/osquery/packs"
    mkdir -p "$dest"
    local base="https://raw.githubusercontent.com/osquery/osquery/main/packs"
    for pack in it-compliance incident-response ossec-rootkit vuln-management; do
        info "Downloading osquery $pack.conf..."
        curl -sSL "$base/$pack.conf" -o "$dest/$pack.conf" || warn "Could not fetch $pack.conf"
    done
    info "osquery packs: downloaded"
}

# ─── OWASP Core Rule Set (Apache 2.0) ────────────────────────────────────────
pull_coreruleset() {
    clone_or_pull \
        "https://github.com/coreruleset/coreruleset.git" \
        "$VENDOR_DIR/coreruleset"
    info "CRS: $(find "$VENDOR_DIR/coreruleset/rules" -name '*.conf' | wc -l) rules"
}

# ─── Suricata ET Open Rules (GPL 2.0 data) ───────────────────────────────────
pull_suricata() {
    local dest="$VENDOR_DIR/suricata"
    mkdir -p "$dest"
    local url="https://rules.emergingthreats.net/open/suricata-5.0/emerging.rules.tar.gz"
    info "Downloading Suricata ET Open rules..."
    curl -sSL "$url" -o "$dest/emerging.rules.tar.gz"
    tar -xzf "$dest/emerging.rules.tar.gz" -C "$dest" --strip-components=1
    rm -f "$dest/emerging.rules.tar.gz"
    info "Suricata ET: $(find "$dest" -name '*.rules' | wc -l) rule files"
}

# ─── Velociraptor Artifacts (AGPL 3.0 data only) ────────────────────────────
pull_velociraptor() {
    clone_or_pull \
        "https://github.com/Velocidex/velociraptor.git" \
        "$VENDOR_DIR/velociraptor"
    info "Velociraptor: artifacts pulled (loaded as data — not imported as code)"
}

# ─── NIST OSCAL (Public Domain) ─────────────────────────────────────────────
pull_oscal() {
    clone_or_pull \
        "https://github.com/usnistgov/oscal-content.git" \
        "$VENDOR_DIR/oscal"
    info "OSCAL: NIST content pulled"
}

# ─── Cortex Analyzers/Responders (AGPL — subprocess only) ───────────────────
pull_cortex() {
    clone_or_pull \
        "https://github.com/TheHive-Project/Cortex-Analyzers.git" \
        "$VENDOR_DIR/cortex"
    info "Cortex: analyzers + responders pulled (subprocess only — AGPL boundary)"
}

# ─── Wazuh Rules (GPL 2.0 data) ────────────────────────────────────────────
pull_wazuh() {
    clone_or_pull \
        "https://github.com/wazuh/wazuh.git" \
        "$VENDOR_DIR/wazuh-rules"
    info "Wazuh: rules pulled (data only — GPL boundary)"
}

# =============================================================================
# Main dispatch
# =============================================================================
mkdir -p "$VENDOR_DIR"

case "$FILTER" in
    all)
        pull_sigma
        pull_yara
        pull_mitre
        pull_misp
        pull_nuclei
        pull_falco
        pull_bloodhound
        pull_osquery
        pull_coreruleset
        pull_suricata
        pull_velociraptor
        pull_oscal
        pull_cortex
        pull_wazuh
        # Strip .git dirs from cloned repos (keep data only)
        find "$VENDOR_DIR" -maxdepth 3 -name ".git" -type d -exec rm -rf {} + 2>/dev/null || true
        ;;
    sigma)        pull_sigma ;;
    yara)         pull_yara ;;
    mitre)        pull_mitre ;;
    misp)         pull_misp ;;
    nuclei)       pull_nuclei ;;
    falco)        pull_falco ;;
    bloodhound)   pull_bloodhound ;;
    osquery)      pull_osquery ;;
    coreruleset)  pull_coreruleset ;;
    suricata)     pull_suricata ;;
    velociraptor) pull_velociraptor ;;
    oscal)        pull_oscal ;;
    cortex)       pull_cortex ;;
    wazuh)        pull_wazuh ;;
    *)
        echo "Unknown filter: $FILTER"
        echo "Usage: $0 [all|sigma|yara|mitre|misp|nuclei|falco|bloodhound|osquery|coreruleset|suricata|velociraptor|oscal|cortex|wazuh]"
        exit 1
        ;;
esac

info "vendor-pull complete."
