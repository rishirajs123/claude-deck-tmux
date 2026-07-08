#!/bin/bash
# Build ClaudeDeck end-to-end: compile the React frontend, embed it, produce the
# single Go binary. Usage: ./build.sh [version]
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="${1:-dev}"

echo "→ building frontend"
cd "$DIR/web/frontend"
[ -d node_modules ] || npm install
npm run build

echo "→ building binary (version $VERSION)"
cd "$DIR"
go build -ldflags "-X main.version=$VERSION" -o claude-deck .

echo "✓ built $DIR/claude-deck"
