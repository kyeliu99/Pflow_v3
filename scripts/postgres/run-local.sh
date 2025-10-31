#!/usr/bin/env bash
set -euo pipefail

DATA_DIR="${PFLOW_POSTGRES_DATA_DIR:-.data/postgres}"
CONTAINER_NAME="${PFLOW_POSTGRES_CONTAINER:-pflow-postgres}"
IMAGE="${POSTGRES_IMAGE:-postgres:16}"
HOST_PORT="${POSTGRES_PORT:-5432}"
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
    docker start "${CONTAINER_NAME}" >/dev/null
  else
    echo "creating postgres container ${CONTAINER_NAME} (image ${IMAGE})"
    docker run -d \
      --name "${CONTAINER_NAME}" \
      -e POSTGRES_USER="${SUPERUSER}" \
      -e POSTGRES_PASSWORD="${PASSWORD}" \
      -e POSTGRES_DB="${DB}" \
      -p "${HOST_PORT}:5432" \
      -v "$(pwd)/${DATA_DIR}:/var/lib/postgresql/data" \
      "${IMAGE}" >/dev/null
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
