#!/usr/bin/env bash
set -euo pipefail

# Helper to pull an image and gracefully fall back to a tag without patch version
# when the more specific tag has been retired from the registry (common for Bitnami images).
pull_with_fallback() {
  local image="$1"
  if docker pull "$image"; then
    return 0
  fi

  # Only attempt to fall back if the tag looks like x.y.z
  if [[ "$image" =~ ^([^:]+):([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    local repo="${BASH_REMATCH[1]}"
    local version="${BASH_REMATCH[2]}"
    local minor_tag="${version%.*}"
    local fallback="${repo}:${minor_tag}"
    echo "[INFO] Pulling ${image} failed, retrying with ${fallback}" >&2
    docker pull "$fallback"
  else
    return 1
  fi
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

IMAGES=(
  "${POSTGRES_IMAGE:-postgres:16}"
  "${ZOOKEEPER_IMAGE:-bitnami/zookeeper:3.9}"
  "${KAFKA_IMAGE:-bitnami/kafka:3.7}"
  "${CAMUNDA_IMAGE:-camunda/zeebe:8.3.0}"
)

for image in "${IMAGES[@]}"; do
  echo "[INFO] Ensuring ${image} is available"
  if ! pull_with_fallback "$image"; then
    echo "[ERROR] Failed to pull ${image}. Please verify the image name or provide an alternative via environment variables." >&2
    exit 1
  fi
  echo "[INFO] ${image} ready"
done

echo "[INFO] All dependency images are available locally. You can now run 'docker compose up -d postgres zookeeper kafka camunda'."
