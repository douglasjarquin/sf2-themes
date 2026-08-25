#!/bin/sh
set -eu

# shellcheck disable=SC1007
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_dir=$(mktemp -d -t sf2-themes-test.XXXXXX)
trap 'find "$test_dir" -depth -type f -delete; find "$test_dir" -depth -type d -empty -delete' EXIT

export XDG_CONFIG_HOME="$test_dir/xdg"
mkdir -p "$test_dir/herdr" "$test_dir/bin"
printf '%s\n' '[ui]' 'theme = "follow-system"' >"$test_dir/herdr/config.toml"

"$repo_dir/sf2-themes" setup herdr --config-dir "$test_dir/herdr"

python3 - "$test_dir/herdr/config.toml" <<'PY'
from pathlib import Path
import sys
import tomllib

config = tomllib.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert config["ui"]["theme"] == "follow-system"
custom = config["theme"]["custom"]
assert custom["accent"] == "#f2b134"
assert custom["text"] == "#fff4d6"
assert custom["active_row_bg"] == "#124158"
assert custom["selection_bg"] == "#423b39"
assert custom["overlay0"] == "#7f89aa"
assert custom["mauve"]
assert custom["peach"]
assert config["theme"]["name"] == "terminal"
print("Herdr merge preservation: PASS")
PY

install -m 755 "$repo_dir/sf2-themes" "$test_dir/bin/sf2-themes"
test ! -e "$test_dir/bin/palette.toml"
test ! -e "$test_dir/bin/themes"

(cd "$test_dir" && "$test_dir/bin/sf2-themes" apply wezterm --config-dir "$test_dir/wezterm")
(cd "$test_dir" && "$test_dir/bin/sf2-themes" apply herdr --config-dir "$test_dir/installed-herdr")
test -f "$test_dir/wezterm/colors/street-fighter-ii-main.toml"
test -f "$test_dir/installed-herdr/config.toml"
"$repo_dir/sf2-themes" setup nvim --config-dir "$test_dir/installed-nvim"
"$repo_dir/sf2-themes" apply nvim --theme ryu-light --config-dir "$test_dir/installed-nvim"
test -f "$test_dir/installed-nvim/colors/street-fighter-ii-main.lua"
test -f "$test_dir/installed-nvim/colors/street-fighter-ii-ryu-light.lua"
test -f "$test_dir/installed-nvim/plugin/sf2-theme.lua"
grep -q 'colorscheme street-fighter-ii-ryu-light' "$test_dir/installed-nvim/sf2-theme/current.lua"
printf '%s\n' 'Copy-only installed Neovim: PASS'
grep -q 'background = "#101a3a"' "$test_dir/wezterm/colors/street-fighter-ii-main.toml"
grep -q 'name = "Street Fighter II - Main"' "$test_dir/wezterm/colors/street-fighter-ii-main.toml"
grep -q 'street-fighter-2' "$test_dir/wezterm/colors/street-fighter-ii-main.toml"
grep -q 'ansi = \["#101a3a", "#e8565f", "#6ecb78", "#f2b134", "#4aa5ff"' "$test_dir/wezterm/colors/street-fighter-ii-main.toml"
grep -q 'accent = "#f2b134"' "$test_dir/installed-herdr/config.toml"
test -f "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
grep -q 'sf2-themes: main' "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
printf '%s\n' 'Copy-only installed CLI: PASS'

mkdir -p "$test_dir/wezterm-existing"
cat >"$test_dir/wezterm-existing/wezterm.lua" <<'LUA'
local wezterm = require("wezterm")
local config = wezterm.config_builder()
config.color_scheme = scheme_for_appearance(wezterm.gui.get_appearance())
config.font_size = 16
return config
LUA

"$repo_dir/sf2-themes" setup wezterm --config-dir "$test_dir/wezterm-existing" >"$test_dir/setup-out.txt" 2>"$test_dir/setup-err.txt"
grep -q 'scheme_for_appearance' "$test_dir/wezterm-existing/wezterm.lua"
grep -q 'config.font_size = 16' "$test_dir/wezterm-existing/wezterm.lua"
grep -q 'WezTerm config was left unchanged' "$test_dir/setup-err.txt"
test -f "$test_dir/wezterm-existing/colors/street-fighter-ii-main.toml"
printf '%s\n' 'WezTerm unknown lua left unchanged: PASS'

mkdir -p "$test_dir/wezterm-safe"
cat >"$test_dir/wezterm-safe/wezterm.lua" <<'LUA'
local wezterm = require("wezterm")
local config = wezterm.config_builder()
return config
LUA

"$repo_dir/sf2-themes" setup wezterm --config-dir "$test_dir/wezterm-safe"
grep -q 'config.color_scheme = dofile(sf2_current)' "$test_dir/wezterm-safe/wezterm.lua"
grep -q 'return config' "$test_dir/wezterm-safe/wezterm.lua"
printf '%s\n' 'WezTerm safe builder integration: PASS'
