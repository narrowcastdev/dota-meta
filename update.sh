#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "Building dota-meta..."
go build -o dota-meta ./cmd/dota-meta

echo "Fetching latest hero stats..."
./dota-meta --html

echo "Generating Reddit post..."
./dota-meta --output reddit.md

echo ""
echo "=== Done ==="
echo "Site data:   docs/data.json (updated)"
echo "Reddit post: reddit.md (ready to copy)"
echo ""
echo "To publish:"
echo "  git add docs/data.json && git commit -m 'data: update hero stats' && git push"
