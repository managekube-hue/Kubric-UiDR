#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Kubric: Fetching detection rule repositories ==="

fetch_repo() {
    local url="$1"
    local dest="$2"
    if [ -d "$dest/.git" ]; then
        echo "  Updating $dest ..."
        git -C "$dest" pull --ff-only 2>/dev/null || git -C "$dest" fetch --depth 1 origin main && git -C "$dest" reset --hard origin/main
    else
        echo "  Cloning $url → $dest ..."
        rm -rf "$dest"
        git clone --depth 1 "$url" "$dest"
    fi
}

fetch_repo "https://github.com/SigmaHQ/sigma.git" "$REPO_ROOT/vendor/sigma/rules"
fetch_repo "https://github.com/Yara-Rules/rules.git" "$REPO_ROOT/vendor/yara-rules"
fetch_repo "https://github.com/projectdiscovery/nuclei-templates.git" "$REPO_ROOT/vendor/nuclei-templates"

echo ""
echo "=== Detection rules fetched ==="
echo "  Sigma rules:      $(find "$REPO_ROOT/vendor/sigma/rules" -name '*.yml' 2>/dev/null | wc -l) files"
echo "  YARA rules:       $(find "$REPO_ROOT/vendor/yara-rules" -name '*.yar' -o -name '*.yara' 2>/dev/null | wc -l) files"
echo "  Nuclei templates: $(find "$REPO_ROOT/vendor/nuclei-templates" -name '*.yaml' 2>/dev/null | wc -l) files"
