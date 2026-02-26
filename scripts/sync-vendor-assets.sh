#!/usr/bin/env bash
# =============================================================================
# Kubric — Master vendor detection asset sync script
# Pulls all external detection rules, threat intel, and security data files.
# Safe to re-run: uses git sparse-checkout or targeted wget/curl downloads.
#
# Usage:
#   scripts/sync-vendor-assets.sh [--all] [--sigma] [--mitre] [--nuclei] \
#                                  [--misp] [--yara] [--suricata] \
#                                  [--bloodhound] [--velociraptor] [--wazuh] \
#                                  [--falco] [--osquery] [--openscap]
#
# Requirements: git, curl, jq (optional for MISP), python3 (optional for STIX)
# =============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VENDOR_ROOT="${REPO_ROOT}/00_K-VENDOR-00_DETECTION_ASSETS"

# ─── Color output ─────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ─── helpers ──────────────────────────────────────────────────────────────────
require() { command -v "$1" &>/dev/null || error "Required tool not found: $1"; }
require git
require curl

clone_or_pull() {
    local url="$1" dest="$2"
    if [[ -d "$dest/.git" ]]; then
        info "Updating: $dest"
        git -C "$dest" pull --ff-only --quiet
    else
        info "Cloning:  $url"
        git clone --depth 1 --quiet "$url" "$dest"
    fi
}

sparse_clone() {
    local url="$1" dest="$2"; shift 2; local paths=("$@")
    if [[ -d "$dest/.git" ]]; then
        info "Updating sparse checkout: $dest"
        git -C "$dest" pull --ff-only --quiet
    else
        info "Sparse-cloning: $url (${paths[*]})"
        git clone --no-checkout --depth 1 --filter=blob:none --quiet "$url" "$dest"
        git -C "$dest" sparse-checkout init --cone
        git -C "$dest" sparse-checkout set "${paths[@]}"
        git -C "$dest" checkout --quiet
    fi
}

# =============================================================================
# SIGMA — SigmaHQ detection rules (Apache 2.0)
# =============================================================================
sync_sigma() {
    info "=== Sigma rules (SigmaHQ) ==="
    local dest="${VENDOR_ROOT}/K-VENDOR-SIGMA/rules"
    sparse_clone \
        "https://github.com/SigmaHQ/sigma.git" \
        "$dest" \
        "rules" "rules-emerging-threats" "rules-placeholder" "rules-unsupported"
    local count
    count=$(find "$dest" -name "*.yml" 2>/dev/null | wc -l)
    info "Sigma: ${count} YAML rule files"
}

# =============================================================================
# YARA — Multiple BSD/MIT licensed repositories
# =============================================================================
sync_yara() {
    info "=== YARA signatures ==="
    local base="${VENDOR_ROOT}/K-VENDOR-YARA"

    # Elastic/detection-rules YARA (Apache 2.0)
    sparse_clone \
        "https://github.com/elastic/detection-rules.git" \
        "${base}/elastic-detection-rules" \
        "rules" "hunting"

    # CAPE sandbox YARA rules (many licenses — check before use)
    clone_or_pull \
        "https://github.com/kevthehermit/YaraRules.git" \
        "${base}/yara-misc"

    # Neo23x0 signatures (Florian Roth — BSD-like)
    clone_or_pull \
        "https://github.com/Neo23x0/signature-base.git" \
        "${base}/signature-base"

    local count
    count=$(find "$base" -name "*.yar" -o -name "*.yara" 2>/dev/null | wc -l)
    info "YARA: ${count} rule files"
}

# =============================================================================
# MITRE — ATT&CK STIX bundles (CC BY 4.0)
# =============================================================================
sync_mitre() {
    info "=== MITRE ATT&CK + CWE + CAPEC ==="
    local base="${VENDOR_ROOT}/K-VENDOR-MITRE"
    mkdir -p "$base/stix2"

    # Enterprise ATT&CK
    info "  Downloading enterprise-attack.json..."
    curl -sSfL \
        "https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json" \
        -o "${base}/stix2/enterprise-attack.json"

    # Mobile ATT&CK
    info "  Downloading mobile-attack.json..."
    curl -sSfL \
        "https://raw.githubusercontent.com/mitre/cti/master/mobile-attack/mobile-attack.json" \
        -o "${base}/stix2/mobile-attack.json"

    # ICS ATT&CK
    info "  Downloading ics-attack.json..."
    curl -sSfL \
        "https://raw.githubusercontent.com/mitre/cti/master/ics-attack/ics-attack.json" \
        -o "${base}/stix2/ics-attack.json"

    # CAPEC (Attack Patterns)
    info "  Downloading capec.json..."
    curl -sSfL \
        "https://raw.githubusercontent.com/mitre/cti/master/capec/2.1/stix-capec.json" \
        -o "${base}/stix2/capec.json"

    # CWE (weakness enumeration)
    info "  Downloading CWE XML..."
    mkdir -p "${base}/cwe"
    curl -sSfL \
        "https://cwe.mitre.org/data/xml/cwec_latest.xml.zip" \
        -o "${base}/cwe/cwec_latest.xml.zip"
    command -v unzip &>/dev/null && \
        unzip -q -o "${base}/cwe/cwec_latest.xml.zip" -d "${base}/cwe/" || \
        warn "unzip not available — CWE zip left compressed"

    info "MITRE: enterprise + mobile + ICS ATT&CK, CAPEC, CWE downloaded"
}

# =============================================================================
# NUCLEI — ProjectDiscovery templates (MIT)
# =============================================================================
sync_nuclei() {
    info "=== Nuclei templates ==="
    local dest="${VENDOR_ROOT}/K-VENDOR-NUCLEI/nuclei-templates"
    clone_or_pull \
        "https://github.com/projectdiscovery/nuclei-templates.git" \
        "$dest"
    local count
    count=$(find "$dest" -name "*.yaml" 2>/dev/null | wc -l)
    info "Nuclei: ${count} template files"
}

# =============================================================================
# BLOODHOUND — BloodHound Cypher queries (Apache 2.0)
# =============================================================================
sync_bloodhound() {
    info "=== BloodHound Cypher queries ==="
    local base="${VENDOR_ROOT}/K-VENDOR-BLOODHOUND"

    # SpecterOps BloodHound Community Edition queries
    sparse_clone \
        "https://github.com/SpecterOps/BloodHound.git" \
        "${base}/bloodhound-ce" \
        "packages/go/analysis" "cmd/api/src/queries"

    # Hausec BloodHound query collection
    clone_or_pull \
        "https://github.com/hausec/Bloodhound-Custom-Queries.git" \
        "${base}/custom-queries"

    info "BloodHound: Cypher query libraries downloaded"
}

# =============================================================================
# VELOCIRAPTOR — Artifact exchange (AGPL 3.0 — data files only, not code)
# =============================================================================
sync_velociraptor() {
    info "=== Velociraptor artifacts ==="
    local base="${VENDOR_ROOT}/K-VENDOR-VELOCIRAPTOR"

    # Official artifact exchange
    sparse_clone \
        "https://github.com/Velocidex/velociraptor-docs.git" \
        "${base}/velociraptor-docs" \
        "content/artifact_references/pages"

    # Community artifact exchange
    clone_or_pull \
        "https://github.com/Velocidex/velociraptor-artifact-exchange.git" \
        "${base}/artifact-exchange"

    local count
    count=$(find "$base" -name "*.yaml" 2>/dev/null | wc -l)
    info "Velociraptor: ${count} artifact YAML files"
}

# =============================================================================
# MISP — Threat intelligence data assets (CC0 / CC BY 4.0)
# =============================================================================
sync_misp() {
    info "=== MISP data assets ==="
    local base="${VENDOR_ROOT}/K-VENDOR-MISP"

    clone_or_pull "https://github.com/MISP/misp-taxonomies.git"  "${base}/taxonomies"
    clone_or_pull "https://github.com/MISP/misp-galaxy.git"      "${base}/galaxies"
    clone_or_pull "https://github.com/MISP/misp-warninglists.git" "${base}/warninglists"
    clone_or_pull "https://github.com/MISP/misp-objects.git"     "${base}/objects"

    # CISA KEV (Known Exploited Vulnerabilities) — public domain
    info "  Downloading CISA KEV JSON..."
    mkdir -p "${base}/cisa-kev"
    curl -sSfL \
        "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json" \
        -o "${base}/cisa-kev/known_exploited_vulnerabilities.json"

    info "MISP: taxonomies, galaxies, warninglists, objects, CISA KEV downloaded"
}

# =============================================================================
# SURICATA — Emerging Threats rules (GPL 2.0 — data file use boundary)
# =============================================================================
sync_suricata() {
    info "=== Suricata Emerging Threats rules ==="
    local base="${VENDOR_ROOT}/K-VENDOR-SURICATA/emerging-threats"
    mkdir -p "$base"

    local ET_BASE="https://rules.emergingthreats.net/open/suricata-7.0/rules"
    local categories=(
        "emerging-malware.rules"
        "emerging-c2.rules"
        "emerging-web_specific_apps.rules"
        "emerging-exploit.rules"
        "emerging-trojan.rules"
        "emerging-phishing.rules"
        "emerging-info.rules"
    )

    for cat in "${categories[@]}"; do
        info "  Downloading ${cat}..."
        curl -sSfL "${ET_BASE}/${cat}" -o "${base}/${cat}" || \
            warn "Failed to download ${cat} (may require ET Pro subscription)"
    done

    local count
    count=$(grep -rl 'alert\|drop\|pass' "$base" 2>/dev/null | wc -l)
    info "Suricata: ${count} rule files downloaded"
}

# =============================================================================
# WAZUH — SIEM detection rules (GPL 2.0 — data file use)
# =============================================================================
sync_wazuh() {
    info "=== Wazuh rules ==="
    sparse_clone \
        "https://github.com/wazuh/wazuh.git" \
        "${VENDOR_ROOT}/K-VENDOR-WAZUH/wazuh-rules" \
        "ruleset/rules" "ruleset/decoders"
    info "Wazuh: rules and decoders synced"
}

# =============================================================================
# FALCO — Runtime security rules (Apache 2.0)
# =============================================================================
sync_falco() {
    info "=== Falco rules ==="
    sparse_clone \
        "https://github.com/falcosecurity/rules.git" \
        "${VENDOR_ROOT}/K-VENDOR-FALCO/falco-rules" \
        "rules"
    info "Falco: runtime security rules synced"
}

# =============================================================================
# OSQUERY — Query packs (Apache 2.0)
# =============================================================================
sync_osquery() {
    info "=== osquery packs ==="
    sparse_clone \
        "https://github.com/osquery/osquery.git" \
        "${VENDOR_ROOT}/K-VENDOR-OSQUERY/osquery-packs" \
        "packs"
    info "osquery: query packs synced"
}

# =============================================================================
# OPENSCAP — SCAP security content (mixed Apache/LGPL)
# =============================================================================
sync_openscap() {
    info "=== OpenSCAP / ComplianceAsCode content ==="
    sparse_clone \
        "https://github.com/ComplianceAsCode/content.git" \
        "${VENDOR_ROOT}/K-VENDOR-OPENSCAP/scap-content" \
        "linux_os/guide" "products/rhel9" "products/ubuntu2204"
    info "OpenSCAP: ComplianceAsCode security content synced"
}

# =============================================================================
# MAIN — argument parsing
# =============================================================================
run_all=false
declare -A flags=(
    [sigma]=false [yara]=false [mitre]=false [nuclei]=false
    [misp]=false [suricata]=false [bloodhound]=false [velociraptor]=false
    [wazuh]=false [falco]=false [osquery]=false [openscap]=false
)

for arg in "$@"; do
    case "$arg" in
        --all)         run_all=true ;;
        --sigma)       flags[sigma]=true ;;
        --yara)        flags[yara]=true ;;
        --mitre)       flags[mitre]=true ;;
        --nuclei)      flags[nuclei]=true ;;
        --misp)        flags[misp]=true ;;
        --suricata)    flags[suricata]=true ;;
        --bloodhound)  flags[bloodhound]=true ;;
        --velociraptor) flags[velociraptor]=true ;;
        --wazuh)       flags[wazuh]=true ;;
        --falco)       flags[falco]=true ;;
        --osquery)     flags[osquery]=true ;;
        --openscap)    flags[openscap]=true ;;
        *) warn "Unknown flag: $arg" ;;
    esac
done

# Default to --all if no flags provided
no_flags=true
for v in "${flags[@]}"; do [[ "$v" == "true" ]] && no_flags=false && break; done
[[ "$run_all" == "true" || "$no_flags" == "true" ]] && run_all=true

run_if() { local key="$1"; shift; [[ "$run_all" == "true" || "${flags[$key]}" == "true" ]] && "$@"; }

run_if sigma       sync_sigma
run_if yara        sync_yara
run_if mitre       sync_mitre
run_if nuclei      sync_nuclei
run_if bloodhound  sync_bloodhound
run_if velociraptor sync_velociraptor
run_if misp        sync_misp
run_if suricata    sync_suricata
run_if wazuh       sync_wazuh
run_if falco       sync_falco
run_if osquery     sync_osquery
run_if openscap    sync_openscap

echo ""
info "Vendor asset sync complete. See ${VENDOR_ROOT}/"
