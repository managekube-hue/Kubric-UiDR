#!/bin/bash

# Molt scanner for vulnerability scanning
# Part of pre-commit hooks

echo "🔍 Running Molt vulnerability scanner..."

SCANNERS=(
  "grype"
  "syft"
)

for scanner in "${SCANNERS[@]}"; do
  if command -v "$scanner" >/dev/null 2>&1; then
    echo "Running $scanner..."
    case "$scanner" in
      grype)
        grype dir:. --fail-on high
        ;;
      syft)
        syft dir:. -o json > sbom.json
        echo "SBOM generated: sbom.json"
        ;;
    esac
  fi
done

echo "✅ Security scanning complete!"
