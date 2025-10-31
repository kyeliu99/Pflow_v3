#!/usr/bin/env bash
set -euo pipefail

ACTION=${1:-start}
CAMUNDA_VERSION="${CAMUNDA_VERSION:-8.3.0}"
DATA_DIR="${PFLOW_CAMUNDA_DATA_DIR:-.data/camunda}"
DOWNLOADS_DIR="${DATA_DIR}/downloads"
DIST_DIR="${DATA_DIR}/camunda-zeebe-${CAMUNDA_VERSION}"
CONFIG_DIR="${DATA_DIR}/config"
RUNTIME_DIR="${DATA_DIR}/runtime"
LOG_DIR="${DATA_DIR}/logs"
CONFIG_FILE="${CONFIG_DIR}/zeebe.cfg.toml"
LOG_FILE="${LOG_DIR}/zeebe.log"
PID_FILE="${DATA_DIR}/zeebe.pid"
GATEWAY_PORT="${CAMUNDA_GATEWAY_PORT:-26500}"
MONITORING_PORT="${CAMUNDA_MONITORING_PORT:-9600}"

require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "missing required command: ${cmd}. please install it before proceeding" >&2
    exit 1
  fi
}

download_distribution() {
  mkdir -p "${DOWNLOADS_DIR}"
  local archive="${DOWNLOADS_DIR}/camunda-zeebe-${CAMUNDA_VERSION}.tar.gz"
  if [[ -f "${archive}" ]]; then
    echo "zeebe archive already present"
    echo "${archive}"
    return
  fi

  require_command curl
  local url="https://github.com/camunda/camunda-platform/releases/download/${CAMUNDA_VERSION}/camunda-zeebe-${CAMUNDA_VERSION}.tar.gz"
  echo "downloading zeebe from ${url}"
  curl -L --fail -o "${archive}" "${url}"
  echo "${archive}"
}

extract_distribution() {
  if [[ -d "${DIST_DIR}" ]]; then
    echo "zeebe distribution already extracted"
    return
  fi

  mkdir -p "${DATA_DIR}"
  local archive
  archive=$(download_distribution)
  echo "extracting zeebe archive"
  tar -xzf "${archive}" -C "${DATA_DIR}"
}

prepare_config() {
  mkdir -p "${CONFIG_DIR}" "${RUNTIME_DIR}" "${LOG_DIR}"
  if [[ -f "${CONFIG_FILE}" ]]; then
    return
  fi

  cat >"${CONFIG_FILE}" <<EOF_CONF
[network]
host = "0.0.0.0"
advertisedHost = "127.0.0.1"
commandApiPort = 26501
internalApiPort = 26502
monitoringApiPort = ${MONITORING_PORT}

[gateway]
network.host = "0.0.0.0"
network.port = ${GATEWAY_PORT}
security.enabled = false

[data]
directory = "${RUNTIME_DIR}/data"
EOF_CONF
}

start_zeebe() {
  require_command java
  extract_distribution
  prepare_config

  if [[ -f "${PID_FILE}" ]]; then
    local existing
    existing=$(cat "${PID_FILE}")
    if [[ -n "${existing}" ]] && ps -p "${existing}" >/dev/null 2>&1; then
      echo "zeebe already running with pid ${existing}"
      return
    fi
  fi

  mkdir -p "${LOG_DIR}"
  echo "starting zeebe broker with embedded gateway on port ${GATEWAY_PORT}"
  nohup "${DIST_DIR}/bin/broker" --config "${CONFIG_FILE}" >"${LOG_FILE}" 2>&1 &
  local pid=$!
  echo "${pid}" >"${PID_FILE}"

  wait_for_ready
  echo "zeebe ready on localhost:${GATEWAY_PORT}"
}

stop_zeebe() {
  if [[ ! -f "${PID_FILE}" ]]; then
    echo "zeebe is not running"
    return
  fi

  local pid
  pid=$(cat "${PID_FILE}")
  if [[ -z "${pid}" ]]; then
    echo "invalid pid file; removing"
    rm -f "${PID_FILE}"
    return
  fi

  if ! ps -p "${pid}" >/dev/null 2>&1; then
    echo "zeebe process ${pid} not found; cleaning up pid file"
    rm -f "${PID_FILE}"
    return
  fi

  echo "stopping zeebe"
  kill "${pid}" >/dev/null 2>&1 || true
  local retries=60
  while ps -p "${pid}" >/dev/null 2>&1; do
    if [[ ${retries} -le 0 ]]; then
      echo "force killing zeebe"
      kill -9 "${pid}" >/dev/null 2>&1 || true
      break
    fi
    sleep 1
    retries=$((retries - 1))
  done
  rm -f "${PID_FILE}"
  echo "zeebe stopped"
}

status_zeebe() {
  if [[ -f "${PID_FILE}" ]]; then
    local pid
    pid=$(cat "${PID_FILE}")
    if [[ -n "${pid}" ]] && ps -p "${pid}" >/dev/null 2>&1; then
      echo "zeebe running with pid ${pid}"
      wait_for_ready
      return
    fi
  fi
  echo "zeebe is not running"
  exit 1
}

wait_for_ready() {
  require_command curl
  local retries=60
  local delay=2
  while true; do
    if curl -fs "http://localhost:${MONITORING_PORT}/actuator/health" >/dev/null 2>&1; then
      break
    fi
    if [[ ${retries} -le 0 ]]; then
      echo "zeebe failed to become ready" >&2
      echo "see ${LOG_FILE} for details" >&2
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
    start_zeebe
    ;;
  stop)
    stop_zeebe
    ;;
  status)
    status_zeebe
    ;;
  *)
    echo "usage: $0 [start|stop|status]" >&2
    exit 1
    ;;
esac
