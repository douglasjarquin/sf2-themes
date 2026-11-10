# Commands

Inspection commands and boss aliases. Moved out of the README.

```sh
sf2 apps
sf2 themes
sf2 show ryu
sf2 show ryu-light
sf2 validate --all
sf2 current wezterm
sf2 apply herdr --theme boxer --dry-run
sf2 apply herdr-app --theme ryu-light
sf2 current herdr-app
sf2 current nvim
sf2 current codex
sf2 current claude
sf2 current starship
sf2 current lazygit
```

`sf2` and `sf2-themes` are the same entry point, and each command has its own `--help` for supported flags.
From a checkout the same commands work as `scripts/sf2 …` or `./sf2-themes …`.

Boss aliases: `boxer` (Balrog), `claw` (Vega), `dictator` (M. Bison).

`herdr` targets the TUI; `herdr-app` targets the desktop app.
`current herdr-app` reads the configured matching SF2 pair and reports its dark installed ID.
It fails for foreign or independently mixed pairs and cannot observe layout or picker overrides.
