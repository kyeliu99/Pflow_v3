#!/usr/bin/env bash
set -euo pipefail

ACTION=${1:-start}
DATA_DIR="${PFLOW_POSTGRES_DATA_DIR:-.data/postgres}"
PORT="${POSTGRES_PORT:-5432}"
HOST="${POSTGRES_HOST:-127.0.0.1}"
SUPERUSER="${POSTGRES_SUPERUSER:-postgres}"
PASSWORD="${POSTGRES_PASSWORD:-postgres}"
LOG_FILE="${DATA_DIR}/postgres.log"
BIN_DIR="${PFLOW_POSTGRES_BIN_DIR:-}"
SKIP_BOOTSTRAP="${PFLOW_POSTGRES_SKIP_BOOTSTRAP:-}"

if [[ -n "${BIN_DIR}" ]]; then
  PATH="${BIN_DIR}:${PATH}"
fi

require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "missing required command: ${cmd}. please ensure PostgreSQL client tools are installed and available on PATH." >&2
    exit 1
  fi
}

start_cluster() {
  mkdir -p "${DATA_DIR}"

  if [[ ! -f "${DATA_DIR}/PG_VERSION" ]]; then
    require_command initdb
    require_command pg_ctl

    local pwfile
    pwfile="$(mktemp)"
    trap 'rm -f "${pwfile}"' EXIT
    printf '%s\n' "${PASSWORD}" >"${pwfile}"

    echo "initialising postgres data directory at ${DATA_DIR}"
    initdb -D "${DATA_DIR}" -U "${SUPERUSER}" --pwfile="${pwfile}" --auth-host=scram-sha-256 --auth-local=trust >/dev/null

    {
      echo "listen_addresses = '${HOST}'"
      echo "port = ${PORT}"
      echo "max_connections = 100"
      echo "wal_level = replica"
    } >>"${DATA_DIR}/postgresql.conf"

    # Ensure password based access from localhost works out of the box.
    if ! grep -q "127.0.0.1/32" "${DATA_DIR}/pg_hba.conf"; then
      {
        echo "host    all             all             127.0.0.1/32            scram-sha-256"
        echo "host    all             all             ::1/128                 scram-sha-256"
      } >>"${DATA_DIR}/pg_hba.conf"
    fi
  fi

  require_command pg_ctl
  if pg_ctl -D "${DATA_DIR}" status >/dev/null 2>&1; then
    echo "postgres already running from ${DATA_DIR}"
  else
    echo "starting postgres on ${HOST}:${PORT}"
    pg_ctl -D "${DATA_DIR}" -l "${LOG_FILE}" -o "-p ${PORT} -h ${HOST}" start >/dev/null
  fi

  wait_for_ready

  if [[ -z "${SKIP_BOOTSTRAP}" ]]; then
    echo "running database bootstrap"
    POSTGRES_HOST="${HOST}" \
    POSTGRES_PORT="${PORT}" \
    POSTGRES_SUPERUSER="${SUPERUSER}" \
    PGPASSWORD="${PASSWORD}" \
    ./scripts/postgres/bootstrap.sh
  fi

  echo "postgres ready: postgres://${SUPERUSER}:***@${HOST}:${PORT}/postgres"
}

stop_cluster() {
  if [[ ! -f "${DATA_DIR}/PG_VERSION" ]]; then
    echo "no postgres cluster found in ${DATA_DIR}" >&2
    exit 0
  fi

  require_command pg_ctl
  if ! pg_ctl -D "${DATA_DIR}" status >/dev/null 2>&1; then
    echo "postgres is not running"
    exit 0
  fi

  echo "stopping postgres"
  pg_ctl -D "${DATA_DIR}" stop -m fast >/dev/null
  echo "postgres stopped"
}

status_cluster() {
  if [[ ! -f "${DATA_DIR}/PG_VERSION" ]]; then
    echo "postgres is not initialised"
    exit 1
  fi

  if pg_ctl -D "${DATA_DIR}" status; then
    wait_for_ready
  else
    exit 1
  fi
}

wait_for_ready() {
  require_command pg_isready
  local retries=30
  local delay=1
  while ! pg_isready -h "${HOST}" -p "${PORT}" -U "${SUPERUSER}" >/dev/null 2>&1; do
    if [[ ${retries} -le 0 ]]; then
      echo "postgres failed to become ready" >&2
      exit 1
    fi
    sleep "${delay}"
    retries=$((retries - 1))
    if [[ ${delay} -lt 5 ]]; then
      delay=$((delay + 1))
    fi
  done
}

case "${ACTION}" in
  start)
    start_cluster
    ;;
  stop)
    stop_cluster
    ;;
  status)
    status_cluster
    ;;
  *)
    echo "usage: $0 [start|stop|status]" >&2
    exit 1
    ;;
esac
