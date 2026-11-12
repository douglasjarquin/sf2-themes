# Apply, and the setup fallback

One command per app, then nothing. Moved out of the README.

`apply` prepares the theme assets, installs or repairs the app's integration, and selects a theme (default: `main`) in one step.
It is idempotent: the first run and the hundredth do the same work, and a repeat with nothing to do reports `no changes`.

`setup` runs the same integration but keeps an existing selection unless `--theme` is given.
Use it to repair integration without changing the active theme.

## Who owns the theme setting

Before writing anything, every command reads the app's config and decides who owns its theme setting.
The whole run is planned first, so a run that refuses writes nothing, removes nothing, and creates no backup.
`--dry-run` prints exactly the writes and removals the real run would make.

| The app's theme setting | `apply`, or `setup --theme` | `setup` |
|---|---|---|
| Not set yet | selects the theme | selects the theme |
| Managed by sf2-themes | replaces it in place | keeps the selection |
| Written by an older sf2-themes | upgrades it | upgrades it and keeps the selection |
| Your own theme | refuses; `--adopt` replaces it | installs the theme files and keeps your theme; `--adopt` replaces it |
| A shape sf2-themes cannot edit | refuses and prints the manual fix | same |
| Broken (invalid file or incomplete sf2-themes markers) | refuses | refuses |

Every refusal exits nonzero with one message on stderr.
For a shape it cannot edit, the message is the manual fix, then an instruction to run the command again.
WezTerm is the one exception: it still writes its scheme files and pointer, prints the snippet, and exits nonzero, because a config that loads the pointer from your own Lua module keeps switching themes.

`--adopt` works for every app except Slack, and does nothing where no theme of your own is in the way.

`setup` reports what it did with the selection:

- `kept sf2-ryu` when it found a selection it can read, which is what `current` prints.
- `no selection found, selected sf2-main` when there was none.
- `selection unreadable, reset to sf2-main` when an sf2-themes selection exists but does not name a catalog theme.
- `kept your own theme` when the setting belongs to you.

`install` still works as a deprecated alias for `apply`.

The CLI keeps short catalog ids such as `ken` and `ryu-light` for input.
File-writing adapters install and select the corresponding prefixed ids such as `sf2-ken` and `sf2-ryu-light`.
Slack prints a paste string and does not install an `sf2-` identity.

Every mutation command takes `--dry-run` to print the full plan without writing, `--verbose` to list every file operation, `--config-dir` to write under a different directory, `--follow-symlinks` to write through destination symlinks, and `--adopt` to replace a theme setting sf2-themes does not manage.

Examples below assume the installed binary; `sf2-themes` works in place of `sf2`.
From a checkout, use `mise run apply -- …` or `mise run setup -- …` in place of `sf2 apply …` or `sf2 setup …`.

```sh
sf2 apply wezterm
sf2 apply wezterm --theme ryu

sf2 apply herdr --theme chun-li
herdr server reload-config

sf2 apply nvim --theme ryu-light

sf2 apply codex --theme ryu-light

sf2 apply claude --theme ryu-light

sf2 apply starship --theme vega

sf2 apply lazygit --theme vega

sf2 apply slack --theme ken
```

Starship apply writes a marked block with `palette = "sf2"` and the `[palettes.sf2]` table before the first table in `starship.toml`, because Starship reads `palette` only as a top-level key.
A marked block an older version appended at the end of the file moves there on the next run.
A top-level `palette` of your own is your own theme, so `apply` refuses it until you pass `--adopt`.
Starship apply also refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`; source it after `zsh-syntax-highlighting.zsh` to tint zsh command tokens.
When plain `setup` keeps your own palette, it writes neither file.

Lazygit installs all 36 complete theme fragments under its `themes/` directory and selects the requested theme in `config.yml`.
An unmarked `gui.theme` section is your own theme.

Slack setup and apply print one four-color import line: `ui.background`, `ui.accent`, `semantic.green`, and `semantic.red`.
Paste that line into Preferences, Appearance, Custom theme, Import theme.
Set Color Mode to match the theme and turn Darker sidebars off.
Slack writes no file, ignores `--config-dir`, `--follow-symlinks`, and `--adopt`, and `current slack` fails because there is no theme file to read.

WezTerm apply writes every catalog scheme, a pointer that returns the selected character's dark or light sibling from `wezterm.gui.get_appearance()`, and a Lua integration in `wezterm.lua` that loads the pointer.
Applying `ryu` or `ryu-light` selects the same pair.
An empty or missing `wezterm.lua` gets a starter config, and a recognized `wezterm.config_builder()` shape gets the integration inserted before its return.
If `wezterm.lua` already selects `street-fighter-2` from an older install, the assignment is upgraded to the managed pointer.
A different `color_scheme` on a recognized builder shape is your own theme; `--adopt` replaces it and keeps every other line.
Any other shape is left byte-for-byte unchanged while the scheme files and pointer are still written, and the run prints a pasteable snippet and exits nonzero.
An integration that already loads the pointer counts as managed in any shape, so pasting the snippet makes the next run succeed.
Commented-out Lua never counts as integration.
The pointer is the selection: plain `setup` keeps a readable pointer even when `wezterm.lua` keeps your own color scheme.

Neovim apply installs every catalog colorscheme as `sf2-<catalog-id>.lua` under `~/.config/nvim/colors/`, a managed current-theme pointer under `~/.config/nvim/sf2-theme/current.lua`, and a plugin loader under `~/.config/nvim/plugin/sf2-theme.lua` so the pointer is read on every startup.
The pointer selects the matching dark or light colorscheme from `TERM_THEME` (set by the WezTerm integration) or `'background'`. Applying `ryu` or `ryu-light` selects the same pair.

Codex apply writes every catalog theme as `sf2-<catalog-id>.tmTheme` under `$CODEX_HOME/themes/` and selects the active prefixed theme with `[tui].theme` in `$CODEX_HOME/config.toml`.
A `[tui].theme` that is neither an `sf2-` id nor a short catalog id from an older version is your own theme.
An inline `tui = { ... }` value is a shape sf2-themes cannot edit; convert it to a `[tui]` table and run the command again.
Restart Codex after applying a theme, or reselect it with `/theme` in an existing session.

Claude Code apply writes every catalog theme as `sf2-<catalog-id>.json` under `~/.claude/themes/` and selects it with `theme` in `~/.claude/settings.json`. Reselect it with `/theme` in an existing session. Claude Code has no host-appearance auto-switch for custom themes, so applying `ryu` or `ryu-light` just pins that one sibling, like Codex.

Applying or setting up a theme replaces the managed unprefixed files from older versions so the old and `sf2-` identities are not left side by side.

Herdr apply enables `auto_switch` for the selected character (or `main`) and writes per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays, so the character palette itself follows host appearance (needs Herdr v0.9.0 or newer, which ships herdr#2324). Applying `chun-li` or `chun-li-light` selects the same pair. Reload with `herdr server reload-config` after applying.

An unmarked `[theme]`, `[theme.*]`, or top-level `theme =` in the Herdr config is your own theme.

## Herdr desktop app

```sh
sf2 apply herdr-app --theme chun-li
sf2 setup herdr-app
sf2 current herdr-app
```

`herdr-app` installs all catalog entries as `sf2-<catalog-id>.json` in `~/.config/herdr-app/themes/`.
It merges `theme.light` and `theme.dark` into `settings.json` in that directory's parent, preserving unrelated settings and theme keys.
Applying either sibling selects the same character pair.
A selection that is not a matching SF2 pair, including two different SF2 characters, is your own theme.
Use `--config-dir` for a different app directory, or `XDG_CONFIG_HOME` to change the config root.

Each theme contains its terminal palette and mode-specific canvas, chrome, surface, border, and accent colors.
Settings apply live.
Appearance is preserved: omit it in Herdr settings to follow the system, or use Herdr's `appearance` setting to force light or dark.
Layout themes take precedence over settings, and the theme picker overrides both.
Top-level settings and layout color overrides apply to both modes; SF2 keeps its mode-specific colors inside theme files.
The `herdr` target continues to configure the TUI separately.
