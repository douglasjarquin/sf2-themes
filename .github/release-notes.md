## Install

```sh
curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/download/v{{VERSION}}/install.sh | SF2_THEMES_VERSION={{VERSION}} sh
```

The installer puts `sf2-themes` in `~/.local/bin`, links `sf2` to it, and checks the download against `checksums.txt` before installing.
Set `SF2_THEMES_INSTALL_DIR` to install somewhere else.
It needs no Python, uv, or other runtime.

To check where a binary was built, run `gh attestation verify sf2-themes_<os>_<arch> --repo douglasjarquin/sf2-themes`.

## Upgrading from 1.x

Version 1.x ran from a checkout or through uv.
If you installed it with uv, remove that install first, because the installer will not write over the links uv created:

```sh
uv tool uninstall sf2-themes
```

Then run the install command above.
Configs that 1.x wrote need no change; `sf2 apply <app>` reports `no changes` on them.

The deprecated `install` command and the hidden `current --theme` flag are gone; use `apply` and `current`.
