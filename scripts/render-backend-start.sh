#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${DB_PATH:-/var/data/pcc.db}"
export DB_PATH

if [ -s "$DB_PATH" ]; then
  echo "Found existing SQLite database at $DB_PATH; starting API server."
  exec ./pipeline serve
fi

echo "No SQLite database found at $DB_PATH; starting API server with background initial ingest."
exec ./pipeline run
