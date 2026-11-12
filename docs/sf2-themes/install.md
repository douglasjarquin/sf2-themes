# Install

One command installs the `sf2-themes` binary and links `sf2` to it; both names run the same program.
The binary carries the whole theme catalog and needs no Python, uv, or other runtime.
Releases cover macOS and Linux on arm64 and amd64.

## Latest release

```sh
curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh
sf2 --version
```

The installer puts `sf2-themes` in `~/.local/bin` and links `sf2` to it.
If that directory is not on your `PATH`, it prints the line to add.
It uses `curl`, or `wget` when `curl` is missing.
Run the same command again to upgrade.

## Pin a version

```sh
curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | SF2_THEMES_VERSION=2.0.0 sh
```

The installer from a specific release works the same way:

```sh
curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/download/v2.0.0/install.sh | SF2_THEMES_VERSION=2.0.0 sh
```

## Installer settings

| Variable | Default | Effect |
| --- | --- | --- |
| `SF2_THEMES_VERSION` | the latest release | Release to install, such as `2.0.0`. |
| `SF2_THEMES_INSTALL_DIR` | `~/.local/bin` | Directory for `sf2-themes` and the `sf2` link. |
| `SF2_THEMES_BASE_URL` | the GitHub Release for the version | Where the release assets live, as a URL or a local directory. |

## Verify the download

The installer checks the binary's SHA-256 against the release's `checksums.txt` before it installs anything.
That proves the file was not corrupted or truncated in transit.
It does not prove who built the file, because the checksums come from the same release as the binary.

Each release binary also carries a GitHub build provenance attestation.
To confirm a binary was built by this repository's release workflow, download the asset and check it with the GitHub CLI:

```sh
gh attestation verify sf2-themes_<os>_<arch> --repo douglasjarquin/sf2-themes
```

Use `darwin` or `linux` for `<os>` and `arm64` or `amd64` for `<arch>`.

## Upgrade from 1.x

Version 1.x ran from a checkout or through uv.
Remove a uv install first, because the installer will not write through the links uv created:

```sh
uv tool uninstall sf2-themes
```

Then run the install command above.
Configs that 1.x wrote need no change; `sf2 apply <app>` reports `no changes` on them.
The deprecated `install` command and the hidden `current --theme` flag are gone; use `apply` and `current`.

## Checkout

From a checkout, mise runs the CLI from source:

```sh
mise run apply -- wezterm --theme vega
mise run setup -- wezterm
go run ./cmd/sf2-themes --version
```

`mise run apply` and `mise run setup` forward their arguments to the project CLI.
The toolchain is in [development.md](development.md).
