#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

PUBLISH=""
BIND_DIST=0
while true; do
  case "${1:-}" in
    --publish)
      PUBLISH="$2"
      shift 2
      ;;
    --bind-dist)
      BIND_DIST=1
      shift
      ;;
    *)
      break
      ;;
  esac
done
if [ "$#" -eq 0 ]; then
  printf 'error: command required\n' >&2
  exit 1
fi

IMAGE="$(scripts/ci/ensure-dev-image.sh)"
NAME="sf2-themes-dev-$$"

if [ -n "${GITHUB_RUN_ID:-}" ] && [ -n "${GITHUB_JOB:-}" ]; then
  job_scope="$(printf 'ci-%s-%s' "$GITHUB_RUN_ID" "$GITHUB_JOB" | tr -c '[:alnum:]._-' '-')"
else
  job_scope="dev"
fi
VENV_VOL="sf2-themes-${job_scope}-venv"
FE_NM_VOL="sf2-themes-${job_scope}-web-node-modules"
FE_ASTRO_VOL="sf2-themes-${job_scope}-web-astro"
FE_DIST_VOL="sf2-themes-${job_scope}-web-dist"

cleanup() {
  docker rm -f "$NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

docker_env=(
  -e HOST_UID="$(id -u)"
  -e HOST_GID="$(id -g)"
  -e HOME=/home/dev
  -e BIND_DIST="$BIND_DIST"
)
for var in CI PLAYWRIGHT_PORT; do
  if [ -n "${!var:-}" ]; then
    docker_env+=(-e "$var=${!var}")
  fi
done

publish_args=()
if [ -n "$PUBLISH" ]; then
  publish_args=(-p "$PUBLISH")
fi

dist_mount_args=(--mount "type=volume,src=$FE_DIST_VOL,dst=/workspace/sf2-themes/web/dist")
if [ "$BIND_DIST" -eq 1 ]; then
  # dist must land on the real host filesystem (e.g. for actions/upload-pages-artifact),
  # not a named volume the CI runner's own filesystem can never see.
  dist_mount_args=()
fi

docker run --name "$NAME" --rm --init \
  --user root \
  --entrypoint /bin/bash \
  --mount type=bind,src="$PWD",dst=/workspace/sf2-themes \
  --mount type=volume,src="$VENV_VOL",dst=/workspace/sf2-themes/.venv \
  --mount type=volume,src="$FE_NM_VOL",dst=/workspace/sf2-themes/web/node_modules \
  --mount type=volume,src="$FE_ASTRO_VOL",dst=/workspace/sf2-themes/web/.astro \
  "${dist_mount_args[@]}" \
  "${publish_args[@]}" \
  --workdir /workspace/sf2-themes \
  "${docker_env[@]}" \
  "$IMAGE" \
  -c '
    set -euo pipefail
    if ! getent group "$HOST_GID" >/dev/null 2>&1; then
      groupadd -g "$HOST_GID" -o hostgroup
    fi
    if ! getent passwd "$HOST_UID" >/dev/null 2>&1; then
      useradd -r -u "$HOST_UID" -g "$HOST_GID" -o -d "$HOME" -M -s /bin/bash hostuser
    fi
    mkdir -p /tmp/fslock
    chmod 1777 /tmp/fslock
    chown -R "$HOST_UID:$HOST_GID" "$HOME"
    chown "$HOST_UID:$HOST_GID" \
      /workspace/sf2-themes/.venv \
      /workspace/sf2-themes/web/node_modules \
      /workspace/sf2-themes/web/.astro
    if [ "$BIND_DIST" -eq 1 ]; then
      # Unlike the volume mounts above, a bind-mounted dist has no
      # auto-created mount point - it may not exist at all yet on a fresh
      # checkout, so it must be created before it can be chowned.
      mkdir -p /workspace/sf2-themes/web/dist
    fi
    chown "$HOST_UID:$HOST_GID" /workspace/sf2-themes/web/dist
    exec setpriv --reuid="$HOST_UID" --regid="$HOST_GID" --clear-groups -- \
      /usr/local/bin/sf2-themes-dev-entrypoint "$@"
  ' _ "$@"
