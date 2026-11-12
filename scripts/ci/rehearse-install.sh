#!/bin/sh
# Install a locally built release with install.sh into a fresh HOME and apply a theme,
# the way a user would after `curl ... | sh`. Run by the release rehearsal job.
#
# Usage: scripts/ci/rehearse-install.sh <release dir> <expected version>
set -eu

release="${1:?usage: rehearse-install.sh <release dir> <expected version>}"
expected="${2:?usage: rehearse-install.sh <release dir> <expected version>}"

HOME="$(mktemp -d)"
export HOME
SF2_THEMES_BASE_URL="$release" sh "$release/install.sh"

version="$("$HOME/.local/bin/sf2" --version)"
[ "$version" = "$expected" ] || { echo "sf2 --version printed $version, want $expected" >&2; exit 1; }

"$HOME/.local/bin/sf2" apply wezterm --theme vega
test -f "$HOME/.config/wezterm/colors/sf2-vega.toml"
test -f "$HOME/.config/wezterm/wezterm.lua"
test -f "$HOME/.config/sf2-theme/wezterm-current.lua"
echo "install rehearsal: PASS"
