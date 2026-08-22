#!/bin/sh
set -eu

# shellcheck disable=SC1007
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_dir=$(mktemp -d -t sf2-theme-test.XXXXXX)
trap 'find "$test_dir" -depth -type f -delete; find "$test_dir" -depth -type d -empty -delete' EXIT

mkdir -p "$test_dir/herdr"
printf '%s\n' '[ui]' 'theme = "follow-system"' > "$test_dir/herdr/config.toml"

"$repo_dir/sf2-theme" install herdr --config-dir "$test_dir/herdr"

python3 - "$test_dir/herdr/config.toml" <<'PY'
from pathlib import Path
import sys
import tomllib

config = tomllib.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert config["ui"]["theme"] == "follow-system"
assert config["theme"]["custom"]["accent"] == "#f2b134"
print("Herdr merge preservation: PASS")
PY
