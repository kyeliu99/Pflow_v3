#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
SQL_FILE="$SCRIPT_DIR/init.sql"

if ! command -v psql >/dev/null 2>&1; then
        echo "psql command not found. Please install PostgreSQL client tools." >&2
        exit 1
fi

PGHOST=${POSTGRES_HOST:-${PGHOST:-localhost}}
PGPORT=${POSTGRES_PORT:-${PGPORT:-5432}}
PGUSER=${POSTGRES_SUPERUSER:-${PGUSER:-postgres}}
PGDATABASE=${POSTGRES_DB:-${PGDATABASE:-postgres}}

PSQL_CONN="host=$PGHOST port=$PGPORT user=$PGUSER dbname=$PGDATABASE"

psql "$PSQL_CONN" -f "$SQL_FILE"
