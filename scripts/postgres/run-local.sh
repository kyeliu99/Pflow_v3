#!/usr/bin/env bash
set -euo pipefail

DATA_DIR="${PFLOW_POSTGRES_DATA_DIR:-.data/postgres}"
CONTAINER_NAME="${PFLOW_POSTGRES_CONTAINER:-pflow-postgres}"
IMAGE="${POSTGRES_IMAGE:-postgres:16}"
HOST_PORT="${POSTGRES_PORT:-5432}"
REQUESTED_HOST_PORT="${POSTGRES_PORT+x}"
SUPERUSER="${POSTGRES_SUPERUSER:-postgres}"
PASSWORD="${POSTGRES_PASSWORD:-postgres}"
DB="${POSTGRES_DB:-postgres}"

mkdir -p "${DATA_DIR}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found. please install docker before running this helper." >&2
  exit 1
fi

if docker ps --format '{{.Names}}' | grep -Fxq "${CONTAINER_NAME}"; then
  echo "postgres container ${CONTAINER_NAME} already running"
else
  if docker ps -a --format '{{.Names}}' | grep -Fxq "${CONTAINER_NAME}"; then
    echo "starting existing postgres container ${CONTAINER_NAME}"
    if ! docker start "${CONTAINER_NAME}" >/dev/null 2>&1; then
      echo "failed to start container ${CONTAINER_NAME}. ensure the port mapping is free or recreate the container." >&2
      exit 1
    fi
  else
    echo "creating postgres container ${CONTAINER_NAME} (image ${IMAGE})"
    ATTEMPT_PORT="${HOST_PORT}"
    MAX_OFFSET=${PFLOW_POSTGRES_PORT_SEARCH_RANGE:-10}
    if ! [[ "${MAX_OFFSET}" =~ ^[0-9]+$ ]]; then
      echo "PFLOW_POSTGRES_PORT_SEARCH_RANGE must be a non-negative integer" >&2
      exit 1
    fi
    while true; do
      TMP_LOG="$(mktemp)"
      set +e
      docker run -d \
        --name "${CONTAINER_NAME}" \
        -e POSTGRES_USER="${SUPERUSER}" \
        -e POSTGRES_PASSWORD="${PASSWORD}" \
        -e POSTGRES_DB="${DB}" \
        -p "${ATTEMPT_PORT}:5432" \
        -v "$(pwd)/${DATA_DIR}:/var/lib/postgresql/data" \
        "${IMAGE}" >/dev/null 2>"${TMP_LOG}"
      STATUS=$?
      set -e
      if [ ${STATUS} -eq 0 ]; then
        HOST_PORT="${ATTEMPT_PORT}"
        rm -f "${TMP_LOG}"
        break
      fi

      if grep -qi "port is already allocated" "${TMP_LOG}"; then
        rm -f "${TMP_LOG}"
        if [ -n "${REQUESTED_HOST_PORT}" ]; then
          echo "host port ${ATTEMPT_PORT} is already in use. set POSTGRES_PORT to an available port or stop the conflicting service." >&2
          exit 1
        fi

        if [ ${MAX_OFFSET} -le 0 ]; then
          echo "unable to find a free port for postgres within the search range. set POSTGRES_PORT to a specific free port." >&2
          exit 1
        fi

        NEXT_PORT=$((ATTEMPT_PORT + 1))
        echo "port ${ATTEMPT_PORT} is occupied, trying ${NEXT_PORT} instead..."
        ATTEMPT_PORT=${NEXT_PORT}
        MAX_OFFSET=$((MAX_OFFSET - 1))
        continue
      fi

      cat "${TMP_LOG}" >&2
      rm -f "${TMP_LOG}"
      exit ${STATUS}
    done
  fi
fi

RETRIES=30
SLEEP=2
while ! docker exec "${CONTAINER_NAME}" pg_isready -U "${SUPERUSER}" >/dev/null 2>&1; do
  if [ ${RETRIES} -le 0 ]; then
    echo "postgres container failed to become ready" >&2
    exit 1
  fi
  echo "waiting for postgres to become available..."
  sleep ${SLEEP}
  RETRIES=$((RETRIES - 1))
  SLEEP=$((SLEEP + 1))
  if [ ${SLEEP} -gt 5 ]; then
    SLEEP=5
  fi
done

echo "postgres container ${CONTAINER_NAME} is ready on port ${HOST_PORT}"
