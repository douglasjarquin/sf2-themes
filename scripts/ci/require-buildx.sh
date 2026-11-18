#!/usr/bin/env bash
# Print the Docker Buildx version to stderr and fail unless it is at least MIN_BUILDX_VERSION.
# Every image build path runs this before `docker buildx build`.
set -euo pipefail

# Buildx 0.38.1 is built with Go 1.26.10, which rejects the compromised eMudhra intermediates.
MIN_BUILDX_VERSION="0.38.1"

if ! output="$(docker buildx version 2>&1)"; then
  printf '[require-buildx] docker buildx version failed:\n%s\n' "$output" >&2
  exit 1
fi
printf '[require-buildx] %s\n' "$output" >&2

# `github.com/docker/buildx v0.39.0 <commit>`; distro and Desktop builds may drop the `v` or add a suffix.
version="$(printf '%s\n' "$output" | awk 'NR == 1 { print $2 }')"
version="${version#v}"
version="${version%%[-+]*}"
if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  printf '[require-buildx] cannot parse a Buildx version from: %s\n' "$output" >&2
  exit 1
fi

# Compare major, minor, then patch; 10# keeps a leading zero from reading as octal.
ordinal() {
  local major minor patch
  IFS=. read -r major minor patch <<<"$1"
  printf '%s\n' "$((10#$major * 1000000 + 10#$minor * 1000 + 10#$patch))"
}
if (($(ordinal "$version") >= $(ordinal "$MIN_BUILDX_VERSION"))); then
  exit 0
fi

printf '[require-buildx] Buildx %s is older than the required %s; upgrade Docker Buildx before building images\n' \
  "$version" "$MIN_BUILDX_VERSION" >&2
exit 1
