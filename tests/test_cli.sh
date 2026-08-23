#!/bin/sh
set -eu

# shellcheck disable=SC1007
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_dir=$(mktemp -d -t sf2-theme-test.XXXXXX)
trap 'find "$test_dir" -depth -type f -delete; find "$test_dir" -depth -type d -empty -delete' EXIT

mkdir -p "$test_dir/herdr"
printf '%s\n' '[ui]' 'theme = "follow-system"' >"$test_dir/herdr/config.toml"

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

mkdir -p "$test_dir/bin"
install -m 755 "$repo_dir/sf2-theme" "$test_dir/bin/sf2-theme"
test ! -e "$test_dir/bin/palette.toml"

(cd "$test_dir" && "$test_dir/bin/sf2-theme" install wezterm --config-dir "$test_dir/wezterm")
(cd "$test_dir" && "$test_dir/bin/sf2-theme" install herdr --config-dir "$test_dir/installed-herdr")
test -f "$test_dir/wezterm/colors/street-fighter-2.toml"
test -f "$test_dir/installed-herdr/config.toml"
grep -q 'background = "#101a3a"' "$test_dir/wezterm/colors/street-fighter-2.toml"
grep -q 'accent = "#f2b134"' "$test_dir/installed-herdr/config.toml"
printf '%s\n' 'Copy-only installed CLI: PASS'
