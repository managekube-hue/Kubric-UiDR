#!/usr/bin/env bash
set -euo pipefail

# Pull upstream security intelligence assets (rules/decoders/policies)
# without deploying vendor wrappers.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UPSTREAM_DIR="${ROOT_DIR}/third_party/intelligence/upstream"
mkdir -p "$UPSTREAM_DIR"

sync_sparse_repo() {
  local name="$1"
  local repo_url="$2"
  local branch="$3"
  shift 3
  local paths=("$@")
  local target="${UPSTREAM_DIR}/${name}"

  if [[ ! -d "$target/.git" ]]; then
    git clone --filter=blob:none --no-checkout "$repo_url" "$target"
    (
      cd "$target"
      git sparse-checkout init --cone
      git sparse-checkout set "${paths[@]}"
      git checkout "$branch"
    )
  else
    (
      cd "$target"
      git fetch origin "$branch"
      git sparse-checkout set "${paths[@]}"
      git checkout "$branch"
      git pull --ff-only origin "$branch"
    )
  fi
}

# Wazuh ruleset (GPL-2.0)
sync_sparse_repo "wazuh" "https://github.com/wazuh/wazuh.git" "master" \
  ruleset/rules \
  ruleset/decoders \
  ruleset/sca \
  LICENSE \
  README.md

# Sigma rules (Detection content licensing varies by source)
sync_sparse_repo "sigma" "https://github.com/SigmaHQ/sigma.git" "master" \
  rules \
  LICENSE \
  README.md

# Suricata signatures and docs (GPL-2.0 for Suricata project)
sync_sparse_repo "suricata" "https://github.com/OISF/suricata.git" "master" \
  rules \
  LICENSE \
  README.md

# Zeek package/scripts examples and license metadata
sync_sparse_repo "zeek" "https://github.com/zeek/zeek.git" "master" \
  scripts \
  LICENSE \
  README

echo "Upstream sync completed under: $UPSTREAM_DIR"
