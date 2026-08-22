# Street Fighter II Theme Pack

A small CLI that installs a Street Fighter II color palette into WezTerm and Herdr.

The palette lives in [`palette.toml`](palette.toml), so both adapters use the same deep navy, arcade red, gold, teal, and cream source data.

## Install

Clone this repository and put `sf2-theme` somewhere on your `PATH`.

```sh
git clone https://github.com/douglasjarquin/street-fighter-2-theme.git
cd street-fighter-2-theme
install -m 755 sf2-theme "$HOME/.local/bin/sf2-theme"
```

The CLI has no package dependencies.

## Usage

List the supported applications:

```sh
sf2-theme apps
```

Install or apply the WezTerm theme.

```sh
sf2-theme install wezterm
sf2-theme apply wezterm
```

This writes `~/.config/wezterm/colors/street-fighter-2.toml`.

Select it from `wezterm.lua` with `config.color_scheme = "street-fighter-2"`.

Set `WEZTERM_CONFIG_DIR` or pass `--config-dir PATH` to target another WezTerm config directory.

Install or apply the Herdr theme.

```sh
sf2-theme install herdr
sf2-theme apply herdr
```

This writes the documented Herdr theme tables to `~/.config/herdr/config.toml` while preserving unrelated configuration.

Set `HERDR_CONFIG_PATH` to target a specific config file, or pass `--config-dir PATH` to target a different Herdr config directory.

After changing a running Herdr instance, reload the config:

```sh
herdr server reload-config
```

Herdr's current configuration contract is documented at [herdr.dev/docs/configuration](https://herdr.dev/docs/configuration/).
