#!/usr/bin/env bash
set -euo pipefail

ACTION=${1:-start}
KAFKA_VERSION="${KAFKA_VERSION:-3.7.0}"
SCALA_VERSION="${KAFKA_SCALA_VERSION:-2.13}"
DATA_DIR="${PFLOW_KAFKA_DATA_DIR:-.data/kafka}"
DOWNLOADS_DIR="${DATA_DIR}/downloads"
DIST_DIR="${DATA_DIR}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}"
CONFIG_DIR="${DATA_DIR}/config"
RUNTIME_DIR="${DATA_DIR}/runtime"
LOG_DIR="${DATA_DIR}/logs"
PORT="${KAFKA_PORT:-9092}"
CONTROLLER_PORT="${KAFKA_CONTROLLER_PORT:-9093}"
PROCESS_LOG="${LOG_DIR}/kafka.log"
PID_FILE="${DATA_DIR}/kafka.pid"
CONFIG_FILE="${CONFIG_DIR}/server.properties"

require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "missing required command: ${cmd}. please install it before proceeding" >&2
    exit 1
  fi
}

download_distribution() {
  mkdir -p "${DOWNLOADS_DIR}"
  local archive="${DOWNLOADS_DIR}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz"
  if [[ -f "${archive}" ]]; then
    echo "kafka distribution archive already present"
    echo "${archive}"
    return
  fi

  require_command curl
  local url="https://downloads.apache.org/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz"
  echo "downloading kafka from ${url}"
  curl -L --fail -o "${archive}" "${url}"
  echo "${archive}"
}

extract_distribution() {
  if [[ -d "${DIST_DIR}" ]]; then
    echo "kafka distribution already extracted"
    return
  fi

  mkdir -p "${DATA_DIR}"
  local archive
  archive=$(download_distribution)
  echo "extracting kafka archive"
  tar -xzf "${archive}" -C "${DATA_DIR}"
}

prepare_config() {
  mkdir -p "${CONFIG_DIR}" "${RUNTIME_DIR}" "${LOG_DIR}"
  if [[ -f "${CONFIG_FILE}" ]]; then
    return
  fi

  cp "${DIST_DIR}/config/kraft/server.properties" "${CONFIG_FILE}"

  cat >"${CONFIG_FILE}" <<EOF_CONF
process.roles=broker,controller
node.id=1
controller.listener.names=CONTROLLER
listener.security.protocol.map=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
listeners=PLAINTEXT://:${PORT},CONTROLLER://:${CONTROLLER_PORT}
advertised.listeners=PLAINTEXT://localhost:${PORT}
num.network.threads=3
num.io.threads=8
log.dirs=${RUNTIME_DIR}/logs
metadata.log.dir=${RUNTIME_DIR}/metadata
controller.quorum.voters=1@localhost:${CONTROLLER_PORT}
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000
group.initial.rebalance.delay.ms=0
EOF_CONF
}

format_storage() {
  mkdir -p "${RUNTIME_DIR}/logs"
  if [[ -f "${RUNTIME_DIR}/metadata/meta.properties" ]]; then
    return
  fi

  require_command uuidgen
  local cluster_id
  cluster_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
  echo "initialising kafka storage with cluster id ${cluster_id}"
  "${DIST_DIR}/bin/kafka-storage.sh" format \
    --ignore-formatted \
    -t "${cluster_id}" \
    -c "${CONFIG_FILE}"
}

start_kafka() {
  require_command java
  extract_distribution
  prepare_config
  format_storage

  if [[ -f "${PID_FILE}" ]]; then
    local existing
    existing=$(cat "${PID_FILE}")
    if [[ -n "${existing}" ]] && ps -p "${existing}" >/dev/null 2>&1; then
      echo "kafka already running with pid ${existing}"
      return
    fi
  fi

  mkdir -p "${LOG_DIR}"
  echo "starting kafka on localhost:${PORT}"
  KAFKA_HEAP_OPTS="${KAFKA_HEAP_OPTS:--Xmx512M -Xms512M}" \
    KAFKA_LOG4J_OPTS="-Dlog4j.configuration=file:${DIST_DIR}/config/log4j.properties" \
    nohup "${DIST_DIR}/bin/kafka-server-start.sh" "${CONFIG_FILE}" >"${PROCESS_LOG}" 2>&1 &
  local pid=$!
  echo "${pid}" >"${PID_FILE}"

  wait_for_ready
  echo "kafka ready on localhost:${PORT}"
}

stop_kafka() {
  if [[ ! -f "${PID_FILE}" ]]; then
    echo "kafka is not running"
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
    echo "kafka process ${pid} not found; cleaning up pid file"
    rm -f "${PID_FILE}"
    return
  fi

  echo "stopping kafka"
  kill "${pid}" >/dev/null 2>&1 || true
  local retries=30
  while ps -p "${pid}" >/dev/null 2>&1; do
    if [[ ${retries} -le 0 ]]; then
      echo "force killing kafka"
      kill -9 "${pid}" >/dev/null 2>&1 || true
      break
    fi
    sleep 1
    retries=$((retries - 1))
  done
  rm -f "${PID_FILE}"
  echo "kafka stopped"
}

status_kafka() {
  if [[ -f "${PID_FILE}" ]]; then
    local pid
    pid=$(cat "${PID_FILE}")
    if [[ -n "${pid}" ]] && ps -p "${pid}" >/dev/null 2>&1; then
      echo "kafka running with pid ${pid}"
      wait_for_ready
      return
    fi
  fi
  echo "kafka is not running"
  exit 1
}

wait_for_ready() {
  local retries=30
  local delay=1
  while ! "${DIST_DIR}/bin/kafka-topics.sh" --bootstrap-server "localhost:${PORT}" --list >/dev/null 2>&1; do
    if [[ ${retries} -le 0 ]]; then
      echo "kafka failed to become ready" >&2
      echo "see ${PROCESS_LOG} for details" >&2
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
    start_kafka
    ;;
  stop)
    stop_kafka
    ;;
  status)
    status_kafka
    ;;
  *)
    echo "usage: $0 [start|stop|status]" >&2
    exit 1
    ;;
esac
