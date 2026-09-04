#!/usr/bin/env bash
set -euo pipefail

seed_if_empty() {
  local target="$1" seed="$2"
  if [ -d "$target" ] && [ -z "$(ls -A "$target" 2>/dev/null)" ] && [ -d "$seed" ]; then
    cp -a "$seed"/. "$target"/
  fi
}

seed_if_empty /workspace/sf2-themes/web/node_modules /opt/sf2-themes/manifests/web/node_modules
seed_if_empty /workspace/sf2-themes/.venv /opt/sf2-themes/manifests/.venv

cd /workspace/sf2-themes
exec "$@"
