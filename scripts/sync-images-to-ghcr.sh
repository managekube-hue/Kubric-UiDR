#!/usr/bin/env bash
set -euo pipefail

OWNER="${GHCR_OWNER:-managekube-hue}"
REPO="${GHCR_REPO:-kubric-uidr}"
TAG="${KUBRIC_TAG:-latest}"
PAT="${GHCR_KUBRIC_PAT:-}"

if [[ -z "$PAT" ]]; then
  echo "GHCR_KUBRIC_PAT is required"
  exit 1
fi

echo "$PAT" | docker login ghcr.io -u "$OWNER" --password-stdin

images=(
  "kubric/coresec:${TAG}:coresec"
  "kubric/netguard:${TAG}:netguard"
  "kubric/ndr-rita:${TAG}:ndr-rita"
)

for triple in "${images[@]}"; do
  IFS=":" read -r src tag name <<< "$triple"
  source_image="${src}:${tag}"
  target_image="ghcr.io/${OWNER}/${REPO}/${name}:${tag}"
  docker tag "$source_image" "$target_image"
  docker push "$target_image"
  echo "? $source_image -> $target_image"
done
