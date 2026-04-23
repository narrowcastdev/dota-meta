#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Hand-bumped patch string. Update on patch day.
PATCH="${DOTA_PATCH:-7.40b}"

echo "Building dota-meta..."
go build -o dota-meta ./cmd/dota-meta

echo "Fetching latest hero stats..."
# Archiving of the prior docs/data.json into docs/history/ happens inside
# dota-meta --html itself, keyed by the prior file's snapshot_date.
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
