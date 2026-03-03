#!/usr/bin/env bash
set -euo pipefail

echo "?? Verifying Kubric deployment"
echo "==============================="

FILE="deploy/docker-compose.yml"

running=$(docker compose -f "$FILE" ps --services --filter status=running | wc -l | tr -d ' ')
total=$(docker compose -f "$FILE" config --services | wc -l | tr -d ' ')

echo "Running services: $running / $total"

docker compose -f "$FILE" ps

if [[ "$running" -lt 1 ]]; then
  echo "? No services running"
  exit 1
fi

echo "? Basic deployment verification complete"
