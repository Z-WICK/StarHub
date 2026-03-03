#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT_DIR/apps/desktop"
npm install
npm run electron:build

echo "Build artifacts are in apps/desktop/dist-electron-packages"
