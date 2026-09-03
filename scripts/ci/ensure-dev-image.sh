#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

IMAGE_PREFIX="ghcr.io/douglasjarquin/sf2-themes-dev"
FINGERPRINT="$(scripts/ci/dev-image-fingerprint.sh --target dev)"
TAG="${IMAGE_PREFIX}:sha-${FINGERPRINT}"

if docker image inspect "$TAG" >/dev/null 2>&1; then
  printf '%s\n' "$TAG"
  exit 0
fi

if docker pull "$TAG" >&2 2>/dev/null; then
  printf '%s\n' "$TAG"
  exit 0
fi

TOOLCHAIN_TAG="$(scripts/ci/ensure-toolchain-image.sh)"
printf '[ensure-dev-image] building %s locally on top of %s (not found local or remote)\n' "$TAG" "$TOOLCHAIN_TAG" >&2
docker buildx build --load \
  --cache-from "type=registry,ref=${IMAGE_PREFIX}:buildcache" \
  -f docker/dev/Dockerfile --build-arg "TOOLCHAIN_IMAGE=${TOOLCHAIN_TAG}" -t "$TAG" . >&2
printf '%s\n' "$TAG"
