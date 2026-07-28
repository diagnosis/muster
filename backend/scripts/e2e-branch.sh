#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${NEON_PROJECT_ID:-cold-hall-53463147}"
PARENT="${E2E_PARENT:-main}"
BRANCH="e2e-$(date +%m%d-%H%M%S)"
EXPIRES="$(date -u -v+24H +%Y-%m-%dT%H:%M:%SZ)"

neonctl branches create --project-id "$PROJECT_ID" \
  --parent "$PARENT" --name "$BRANCH" \
  --expires-at "$EXPIRES"

DB_URL="$(neonctl connection-string "$BRANCH" --project-id "$PROJECT_ID")"

goose -dir ./migrations postgres "$DB_URL" up

echo ""
echo "  export DATABASE_URL='$DB_URL'"