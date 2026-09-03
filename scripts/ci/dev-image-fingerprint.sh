#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

TARGET=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --target)
      [ "$#" -ge 2 ] || { printf 'error: --target requires a value\n' >&2; exit 1; }
      TARGET="$2"
      shift 2
      ;;
    *)
      printf 'error: unknown argument: %s\n' "$1" >&2
      exit 1
      ;;
  esac
done

case "$TARGET" in
  toolchain)
    FILES=(mise.toml mise.lock docker/toolchain/Dockerfile docker/toolchain/apt-packages.txt)
    ;;
  dev)
    FILES=(
      mise.toml mise.lock docker/toolchain/Dockerfile docker/toolchain/apt-packages.txt
      pyproject.toml uv.lock web/package.json web/package-lock.json
      docker/dev/Dockerfile docker/dev/apt-packages.txt docker/dev/entrypoint.sh
    )
    ;;
  *)
    printf 'error: --target must be "toolchain" or "dev"\n' >&2
    exit 1
    ;;
esac

cat "${FILES[@]}" | sha256sum | cut -c1-12
