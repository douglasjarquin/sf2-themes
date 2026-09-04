#!/usr/bin/env bash
set -euo pipefail

if [ -z "${GITHUB_RUN_ID:-}" ] || [ -z "${GITHUB_JOB:-}" ]; then
  printf 'gc-job-overlay-volumes: GITHUB_RUN_ID and GITHUB_JOB required; skipping\n' >&2
  exit 0
fi

job_scope="$(printf 'ci-%s-%s' "$GITHUB_RUN_ID" "$GITHUB_JOB" | tr -c '[:alnum:]._-' '-')"
prefix="sf2-themes-${job_scope}-"

if ! command -v docker >/dev/null 2>&1; then
  printf 'gc-job-overlay-volumes: docker not on PATH; skipping\n' >&2
  exit 0
fi

if ! output="$(docker volume ls -q 2>/dev/null)"; then
  printf 'gc-job-overlay-volumes: docker volume ls failed; skipping\n' >&2
  exit 0
fi

while IFS= read -r vol; do
  [ -n "$vol" ] || continue
  case "$vol" in
    "$prefix"*)
      if docker volume rm "$vol" >/dev/null 2>&1; then
        printf 'gc-job-overlay-volumes: removed %s\n' "$vol" >&2
      else
        printf 'gc-job-overlay-volumes: failed to remove %s\n' "$vol" >&2
      fi
      ;;
  esac
done <<<"$output"
exit 0
