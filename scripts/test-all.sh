#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "Run Go tests"
cd "$ROOT_DIR/services/api"
go test ./...

echo "Run Vue unit tests"
cd "$ROOT_DIR/apps/desktop"
npm run test

echo "Run desktop Playwright tests"
npx playwright test
