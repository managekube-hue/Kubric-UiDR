#!/usr/bin/env bash
set -euo pipefail

# Safe, idempotent provisioning for Ubuntu/Codespaces
# Installs only missing baseline tools required by Kubric developers.

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

SUDO=""
if need_cmd sudo; then
  SUDO="sudo"
fi

if ! need_cmd apt-get; then
  echo "apt-get not found. This script currently supports Ubuntu/Debian." >&2
  exit 1
fi

# Avoid apt failure from broken third-party yarn apt repo key on some environments.
YARN_LIST="/etc/apt/sources.list.d/yarn.list"
if [[ -f "$YARN_LIST" ]]; then
  $SUDO mv "$YARN_LIST" "${YARN_LIST}.disabled"
fi

$SUDO apt-get update -y

APT_PKGS=()
for pkg in ansible rustc cargo protobuf-compiler unzip; do
  case "$pkg" in
    ansible) need_cmd ansible || APT_PKGS+=("$pkg") ;;
    rustc) need_cmd rustc || APT_PKGS+=("$pkg") ;;
    cargo) need_cmd cargo || APT_PKGS+=("$pkg") ;;
    protobuf-compiler) need_cmd protoc || APT_PKGS+=("$pkg") ;;
    unzip) need_cmd unzip || APT_PKGS+=("$pkg") ;;
  esac
done

if (( ${#APT_PKGS[@]} > 0 )); then
  $SUDO DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "${APT_PKGS[@]}"
fi

if ! need_cmd terraform; then
  TF_VER="1.9.8"
  ARCH="linux_amd64"
  TMP_ZIP="/tmp/terraform_${TF_VER}_${ARCH}.zip"
  curl -fsSL -o "$TMP_ZIP" "https://releases.hashicorp.com/terraform/${TF_VER}/terraform_${TF_VER}_${ARCH}.zip"
  unzip -o "$TMP_ZIP" -d /tmp >/dev/null
  $SUDO install -m 0755 /tmp/terraform /usr/local/bin/terraform
fi

echo "Provisioning complete."
for c in git make curl jq node yarn python3 pip3 docker kubectl helm terraform ansible rustc cargo protoc; do
  if need_cmd "$c"; then
    echo "- $c: ok"
  else
    echo "- $c: missing"
  fi
done
