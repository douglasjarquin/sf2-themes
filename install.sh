#!/bin/sh
# Install sf2-themes from a GitHub Release.
#
#   curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh
#
# Environment:
#   SF2_THEMES_VERSION      release to install, such as 2.0.0 (default: the latest release)
#   SF2_THEMES_INSTALL_DIR  directory for sf2-themes and the sf2 link (default: ~/.local/bin)
#   SF2_THEMES_BASE_URL     where the release assets live: a URL or a local directory
#                           (default: the GitHub Release for SF2_THEMES_VERSION)
#
# The whole body runs from main on the last line, so a truncated download runs nothing.

set -eu

repo="douglasjarquin/sf2-themes"

fail() {
	printf 'sf2-themes install: %s\n' "$*" >&2
	exit 1
}

detect_os() {
	case "$(uname -s)" in
	Darwin) echo darwin ;;
	Linux) echo linux ;;
	*) fail "unsupported operating system: $(uname -s) (supported: macOS and Linux)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
	x86_64 | amd64) echo amd64 ;;
	arm64 | aarch64) echo arm64 ;;
	*) fail "unsupported architecture: $(uname -m) (supported: x86_64 and arm64)" ;;
	esac
}

# fetch <name> <destination> copies one release asset from the base URL or directory.
fetch() {
	case "$base" in
	http://* | https://*)
		if command -v curl >/dev/null 2>&1; then
			curl -fsSL --retry 3 --connect-timeout 15 "$base/$1" -o "$2" || fail "could not download $base/$1"
		elif command -v wget >/dev/null 2>&1; then
			wget -q --tries=3 --timeout=30 "$base/$1" -O "$2" || fail "could not download $base/$1"
		else
			fail "curl or wget is required to download $base/$1"
		fi
		;;
	*)
		cp "$base/$1" "$2" || fail "could not copy $base/$1"
		;;
	esac
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

main() {
	os="$(detect_os)"
	arch="$(detect_arch)"
	asset="sf2-themes_${os}_${arch}"

	version="${SF2_THEMES_VERSION:-}"
	version="${version#v}"
	if [ -n "${SF2_THEMES_BASE_URL:-}" ]; then
		base="${SF2_THEMES_BASE_URL%/}"
	elif [ -n "$version" ]; then
		base="https://github.com/$repo/releases/download/v$version"
	else
		base="https://github.com/$repo/releases/latest/download"
	fi

	dir="${SF2_THEMES_INSTALL_DIR:-$HOME/.local/bin}"
	target="$dir/sf2-themes"
	link="$dir/sf2"

	# Never write through a symlink. A symlink here is usually a 1.x install made by
	# uv, which owns the file it points at.
	if [ -L "$target" ]; then
		fail "$target is a symlink, probably from a 1.x install. Remove it first with: uv tool uninstall sf2-themes"
	fi
	if [ -L "$link" ] && [ "$(readlink "$link")" != "sf2-themes" ]; then
		fail "$link is a symlink to $(readlink "$link"), probably from a 1.x install. Remove it first with: uv tool uninstall sf2-themes"
	fi
	if [ -e "$link" ] && [ ! -L "$link" ]; then
		fail "$link exists and is not a link to sf2-themes; remove it and run the installer again"
	fi

	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT INT TERM

	fetch checksums.txt "$tmp/checksums.txt"
	fetch "$asset" "$tmp/$asset"

	expected="$(awk -v name="$asset" '$2 == name || $2 == "*" name { print $1 }' "$tmp/checksums.txt")"
	[ -n "$expected" ] || fail "checksums.txt has no entry for $asset"
	actual="$(sha256_of "$tmp/$asset")"
	[ "$actual" = "$expected" ] || fail "checksum mismatch for $asset: expected $expected, got $actual"

	mkdir -p "$dir"
	chmod 755 "$tmp/$asset"
	mv "$tmp/$asset" "$dir/.sf2-themes.new"
	mv -f "$dir/.sf2-themes.new" "$target" || {
		rm -f "$dir/.sf2-themes.new"
		fail "could not move the new binary into $target"
	}
	[ -L "$link" ] || ln -s sf2-themes "$link"

	printf 'installed sf2-themes %s to %s\n' "$("$target" --version)" "$target"
	# shellcheck disable=SC2016 # $PATH is printed literally for the user to paste.
	case ":$PATH:" in
	*":$dir:"*) ;;
	*) printf '%s is not on PATH; add it with:\n  export PATH="%s:$PATH"\n' "$dir" "$dir" ;;
	esac
}

main "$@"
