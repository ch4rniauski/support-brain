#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# From the host machine Postgres is on localhost even if services use host "postgres".
DB_HOST="${POSTGRES_HOST:-localhost}"
if [[ "$DB_HOST" == "postgres" ]]; then
  DB_HOST="localhost"
fi

DB_PORT="${POSTGRES_PORT:-5432}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
DB_NAME="${POSTGRES_DB:-support-brain}"

DBSTRING="${GOOSE_DBSTRING:-postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable}"
COMMAND="${1:-up}"

services=(
  "auth"
  "intake"
  "template"
  "knowledge"
  "generator"
  "embedding"
)

if ! command -v goose >/dev/null 2>&1; then
  echo "goose is not installed. Install with: go install github.com/pressly/goose/v3/cmd/goose@latest" >&2
  exit 1
fi

for service in "${services[@]}"; do
  dir="internal/${service}/migrations"
  table="goose_${service}_version"
  echo "==> ${service}: goose ${COMMAND} (table=${table})"
  goose -dir "$dir" -table "$table" postgres "$DBSTRING" "$COMMAND"
done

echo "Done."
