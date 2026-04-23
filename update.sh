#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Hand-bumped patch string. Update on patch day.
PATCH="${DOTA_PATCH:-7.40b}"

echo "Building dota-meta..."
go build -o dota-meta ./cmd/dota-meta

if [[ -f docs/data.json ]]; then
  DATE=$(date +%Y-%m-%d)
  cp docs/data.json "docs/history/data-${DATE}.json"
  echo "Archived previous snapshot → docs/history/data-${DATE}.json"
fi

echo "Fetching latest hero stats..."
./dota-meta --html --patch "$PATCH"

echo "Generating Reddit post..."
./dota-meta --output reddit.md --patch "$PATCH"

echo ""
echo "=== Done ==="
echo "Site data:   docs/data.json (updated, patch=$PATCH)"
echo "Reddit post: reddit.md (ready to copy)"
echo ""
echo "To publish:"
echo "  git add docs/data.json docs/history reddit.md && git commit -m 'data: update hero stats' && git push"
