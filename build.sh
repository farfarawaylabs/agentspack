#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==> Syncing system folder to internal/content/system..."
rm -rf internal/content/system
cp -r system internal/content/system

echo "==> Building agentspack..."
go build -o agentspack .

echo "==> Done! Binary created at: $SCRIPT_DIR/agentspack"
