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
assert config["theme"]["auto_switch"] is True
assert config["theme"]["name"] == "catppuccin"
assert config["theme"]["light_name"] == "catppuccin-latte"
assert config["theme"]["dark_name"] == "catppuccin"
custom = config["theme"]["custom"]
dark = custom["dark"]
light = custom["light"]
assert dark["accent"] == "#ad8705"
assert dark["text"] == "#cad1de"
assert dark["active_row_bg"] == "#1d2433"
assert dark["selection_bg"] == "#2a3243"
assert dark["overlay0"] == "#7c8597"
assert dark["mauve"]
assert dark["peach"]
assert light["accent"] != dark["accent"]
assert light["text"] != dark["text"]
print("Herdr merge preservation: PASS")
PY

install -m 755 "$repo_dir/sf2-themes" "$test_dir/bin/sf2-themes"
test ! -e "$test_dir/bin/palette.toml"
test ! -e "$test_dir/bin/themes"

(cd "$test_dir" && "$test_dir/bin/sf2-themes" apply wezterm --config-dir "$test_dir/wezterm")
(cd "$test_dir" && "$test_dir/bin/sf2-themes" apply herdr --config-dir "$test_dir/installed-herdr")
test -f "$test_dir/wezterm/colors/sf2-main.toml"
test -f "$test_dir/installed-herdr/config.toml"
"$repo_dir/sf2-themes" setup nvim --config-dir "$test_dir/installed-nvim"
"$repo_dir/sf2-themes" apply nvim --theme ryu-light --config-dir "$test_dir/installed-nvim"
test -f "$test_dir/installed-nvim/colors/sf2-main.lua"
test -f "$test_dir/installed-nvim/colors/sf2-ryu-light.lua"
test -f "$test_dir/installed-nvim/plugin/sf2-theme.lua"
grep -q 'colorscheme sf2-ryu-light' "$test_dir/installed-nvim/sf2-theme/current.lua"
grep -q 'colorscheme sf2-ryu' "$test_dir/installed-nvim/sf2-theme/current.lua"
grep -q 'TERM_THEME' "$test_dir/installed-nvim/sf2-theme/current.lua"
printf '%s\n' 'Copy-only installed Neovim: PASS'
"$repo_dir/sf2-themes" setup codex --config-dir "$test_dir/installed-codex"
"$repo_dir/sf2-themes" apply codex --theme ryu-light --config-dir "$test_dir/installed-codex"
test -f "$test_dir/installed-codex/themes/sf2-main.tmTheme"
test -f "$test_dir/installed-codex/themes/sf2-main-light.tmTheme"
test -f "$test_dir/installed-codex/themes/sf2-ryu-light.tmTheme"
grep -q 'theme = "sf2-ryu-light"' "$test_dir/installed-codex/config.toml"
grep -q '#131927' "$test_dir/installed-codex/themes/sf2-main.tmTheme"
test "$("$repo_dir/sf2-themes" current codex --config-dir "$test_dir/installed-codex")" = "sf2-ryu-light"
printf '%s\n' 'Copy-only installed Codex: PASS'
"$repo_dir/sf2-themes" setup claude --config-dir "$test_dir/installed-claude"
"$repo_dir/sf2-themes" apply claude --theme ryu-light --config-dir "$test_dir/installed-claude"
test -f "$test_dir/installed-claude/themes/sf2-main.json"
test -f "$test_dir/installed-claude/themes/sf2-main-light.json"
test -f "$test_dir/installed-claude/themes/sf2-ryu-light.json"
grep -q '"theme": "custom:sf2-ryu-light"' "$test_dir/installed-claude/settings.json"
grep -q '#ad8705' "$test_dir/installed-claude/themes/sf2-main.json"
test "$("$repo_dir/sf2-themes" current claude --config-dir "$test_dir/installed-claude")" = "sf2-ryu-light"
printf '%s\n' 'Copy-only installed Claude: PASS'
"$repo_dir/sf2-themes" setup lazygit --config-dir "$test_dir/installed-lazygit"
"$repo_dir/sf2-themes" apply lazygit --theme ryu-light --config-dir "$test_dir/installed-lazygit"
test "$(find "$test_dir/installed-lazygit/themes" -type f -name 'sf2-*.yml' | wc -l | tr -d ' ')" -eq 36
test -f "$test_dir/installed-lazygit/themes/sf2-main.yml"
test -f "$test_dir/installed-lazygit/themes/sf2-ryu-light.yml"
grep -q 'activeBorderColor:' "$test_dir/installed-lazygit/themes/sf2-main.yml"
grep -q 'authorColors:' "$test_dir/installed-lazygit/themes/sf2-main.yml"
grep -q '# sf2-themes: sf2-ryu-light' "$test_dir/installed-lazygit/config.yml"
test "$("$repo_dir/sf2-themes" current lazygit --config-dir "$test_dir/installed-lazygit")" = "sf2-ryu-light"
printf '%s\n' 'Copy-only installed Lazygit: PASS'
grep -q 'background = "#131927"' "$test_dir/wezterm/colors/sf2-main.toml"
grep -q 'name = "sf2-main"' "$test_dir/wezterm/colors/sf2-main.toml"
grep -q 'sf2-street-fighter-2' "$test_dir/wezterm/colors/sf2-main.toml"
grep -q 'ansi = \["#2b323f", "#c86e67", "#4e975b", "#a58324", "#5b89cc"' "$test_dir/wezterm/colors/sf2-main.toml"
grep -q 'accent = "#ad8705"' "$test_dir/installed-herdr/config.toml"
grep -q 'auto_switch = true' "$test_dir/installed-herdr/config.toml"
grep -q '^\[theme.custom.dark\]$' "$test_dir/installed-herdr/config.toml"
grep -q '^\[theme.custom.light\]$' "$test_dir/installed-herdr/config.toml"
test -f "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
grep -q 'sf2-themes: sf2-main' "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
grep -q 'get_appearance' "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
grep -q 'return "sf2-main-light"' "$XDG_CONFIG_HOME/sf2-theme/wezterm-current.lua"
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
test -f "$test_dir/wezterm-existing/colors/sf2-main.toml"
printf '%s\n' 'WezTerm unknown lua left unchanged: PASS'

mkdir -p "$test_dir/wezterm-safe"
cat >"$test_dir/wezterm-safe/wezterm.lua" <<'LUA'
local wezterm = require("wezterm")
local config = wezterm.config_builder()
return config
LUA

"$repo_dir/sf2-themes" setup wezterm --config-dir "$test_dir/wezterm-safe"
grep -q 'local sf2_scheme = dofile(sf2_current)' "$test_dir/wezterm-safe/wezterm.lua"
grep -q 'config.color_scheme = sf2_scheme' "$test_dir/wezterm-safe/wezterm.lua"
grep -q 'TERM_THEME' "$test_dir/wezterm-safe/wezterm.lua"
grep -q 'return config' "$test_dir/wezterm-safe/wezterm.lua"
printf '%s\n' 'WezTerm safe builder integration: PASS'
