#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "[1/3] Setup desktop dependencies"
cd "$ROOT_DIR/apps/desktop"
npm install

echo "[2/3] Setup Go dependencies"
cd "$ROOT_DIR/services/api"
go mod tidy

echo "[3/3] Bootstrap done"
echo "Desktop: cd apps/desktop && npm run dev"
echo "API: cd services/api && PORT=8080 DATABASE_URL=... GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=... APP_SECRET=... ENCRYPTION_KEY=... go run ./cmd/server"
