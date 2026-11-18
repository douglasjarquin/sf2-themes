#!/usr/bin/env bash
# Log the Docker Buildx version to stderr and fail unless it is at least MIN_BUILDX_VERSION.
# Every image build path runs this before `docker buildx build`.
set -euo pipefail

# Buildx 0.38.1 is built with Go 1.26.10, which rejects the compromised eMudhra intermediates.
MIN_BUILDX_VERSION="0.38.1"

fail() {
  printf '[require-buildx] %s\n' "$1" >&2
  exit 1
}

# Docker diagnostics stay on stderr; the version record is on stdout.
output="$(docker buildx version)" || fail "docker buildx version failed"
record="$(printf '%s\n' "$output" | awk '$1 == "github.com/docker/buildx" { print; exit }')"
[ -n "$record" ] || fail "docker buildx version printed no github.com/docker/buildx record: ${output}"
printf '[require-buildx] %s\n' "$record" >&2

# SemVer with an optional leading `v`: core, optional -prerelease, optional +build metadata.
semver='^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?(\+[0-9A-Za-z.-]+)?$'
version="$(printf '%s\n' "$record" | awk '{ print $2 }')"
[[ "$version" =~ $semver ]] || fail "cannot parse a Buildx version from: ${record}"
have=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}")
prerelease="${BASH_REMATCH[5]}"
# Docker Desktop ships released Buildx as `-desktop.N`; that is packaging, not a prerelease.
[[ "$prerelease" =~ ^desktop\.[0-9]+$ ]] && prerelease=""
IFS=. read -r -a want <<<"$MIN_BUILDX_VERSION"

# Compare decimal strings of any length: drop leading zeros, then the longer is larger.
compare() {
  local a="${1#"${1%%[!0]*}"}" b="${2#"${2%%[!0]*}"}"
  if ((${#a} != ${#b})); then
    ((${#a} > ${#b})) && echo 1 || echo -1
  elif [[ "$a" > "$b" ]]; then
    echo 1
  elif [[ "$a" < "$b" ]]; then
    echo -1
  else
    echo 0
  fi
}

for i in 0 1 2; do
  case "$(compare "${have[i]}" "${want[i]}")" in
    1) exit 0 ;;
    -1) fail "Buildx ${version} is older than the required ${MIN_BUILDX_VERSION}; upgrade Docker Buildx before building images" ;;
  esac
done
# Same core: a prerelease precedes its release.
[ -z "$prerelease" ] || fail "Buildx ${version} is a prerelease of ${MIN_BUILDX_VERSION}; upgrade Docker Buildx before building images"
