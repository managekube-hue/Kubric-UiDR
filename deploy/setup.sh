#!/usr/bin/env bash
set -euo pipefail

echo "?? Kubric Deployment"
echo "===================="

: "${GHCR_OWNER:=managekube-hue}"
: "${GHCR_KUBRIC_PAT:=}"

repo_status=$(curl -s -o /dev/null -w "%{http_code}" "https://github.com/${GHCR_OWNER}/Kubric-UiDR" || true)
if [[ "$repo_status" == "404" || "$repo_status" == "401" ]]; then
  echo "?? Private repository/package access path detected"
  if [[ -z "$GHCR_KUBRIC_PAT" ]]; then
    read -rsp "Enter GHCR_KUBRIC_PAT: " GHCR_KUBRIC_PAT
    echo
  fi
  echo "$GHCR_KUBRIC_PAT" | docker login ghcr.io -u "$GHCR_OWNER" --password-stdin
else
  echo "?? Public repository path detected"
fi

if [[ ! -f .env && -f deploy/.env.example ]]; then
  cp deploy/.env.example .env
  echo "Created .env from deploy/.env.example"
fi

docker compose -f deploy/docker-compose.yml pull
docker compose -f deploy/docker-compose.yml up -d

echo "? Deployment started"
