#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

IMAGE_PREFIX="ghcr.io/douglasjarquin/sf2-themes-toolchain"
FINGERPRINT="$(scripts/ci/dev-image-fingerprint.sh --target toolchain)"
TAG="${IMAGE_PREFIX}:sha-${FINGERPRINT}"

if docker image inspect "$TAG" >/dev/null 2>&1; then
  printf '%s\n' "$TAG"
  exit 0
fi

if docker pull "$TAG" >&2 2>/dev/null; then
  printf '%s\n' "$TAG"
  exit 0
fi

printf '[ensure-toolchain-image] building %s locally (not found local or remote)\n' "$TAG" >&2
docker build -f docker/toolchain/Dockerfile -t "$TAG" . >&2
printf '%s\n' "$TAG"
