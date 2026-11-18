#!/usr/bin/env bash
# Build the release assets: one static sf2-themes binary per supported platform,
# checksums.txt over them, and install.sh. The release workflow and the pull
# request rehearsal both run this script, so the rehearsal proves what ships.
#
# Usage: scripts/ci/build-release.sh <version> [<output dir>]
set -euo pipefail

version="${1:?usage: build-release.sh <version> [<output dir>]}"
out="${2:-dist}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

# Released binaries embed the toolchain that built them, so build only with the
# exact Go that the committed mise.toml pins. Read the pin from the file, not
# from `mise current`, which MISE_GO_VERSION or `mise shell` can lower; never let
# GOTOOLCHAIN switch toolchains; and refuse any other go found first on PATH.
export GOTOOLCHAIN=local
pinned_go="$(awk '/^\[/ { tools = ($0 == "[tools]") } tools && $1 == "go" && $2 == "=" { gsub(/"/, "", $3); print $3 }' mise.toml)"
if [[ ! "$pinned_go" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "mise.toml must pin an exact Go version (x.y.z) under [tools], not '$pinned_go'" >&2
  exit 1
fi
want_go="go$pinned_go"
build_go="$(go env GOVERSION)"
if [[ "$build_go" != "$want_go" ]]; then
  echo "build Go $build_go is not the mise-pinned $want_go" >&2
  exit 1
fi

rm -rf "$out"
mkdir -p "$out"

# Never strip or pack the binaries after linking: the Go linker ad-hoc signs
# darwin/arm64 output, and a post-link rewrite breaks that signature, which
# makes macOS kill the binary on launch.
for target in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64; do
  os="${target%/*}"
  arch="${target#*/}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-X github.com/douglasjarquin/sf2-themes/v2/internal/version.Version=$version" \
    -o "$out/sf2-themes_${os}_${arch}" ./cmd/sf2-themes
  # Read build info by basename so an output path with spaces cannot split the field.
  embedded_go="$(cd "$out" && go version "sf2-themes_${os}_${arch}" | awk '{print $2}')"
  if [[ "$embedded_go" != "$want_go" ]]; then
    echo "sf2-themes_${os}_${arch} embeds $embedded_go, want $want_go" >&2
    exit 1
  fi
done
(cd "$out" && go version sf2-themes_*)

cp install.sh "$out/install.sh"

cd "$out"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum sf2-themes_* > checksums.txt
else
  shasum -a 256 sf2-themes_* > checksums.txt
fi
cat checksums.txt
