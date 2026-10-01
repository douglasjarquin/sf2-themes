# Install

The CLI installs once with uv and runs as `sf2` (or `sf2-themes`; both entry points are the same program). Moved out of the README.

The command requires uv and Python 3.11 or newer.

## uv tool (persistent)

Install once, then use the bare command:

```sh
uv tool install git+https://github.com/douglasjarquin/sf2-themes.git
sf2 --version
```

## One-shot

Run the CLI straight from GitHub without installing it:

```sh
uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes --version
```

The older `uv run --with git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes …` form still works.

## Checkout

From a checkout, use the committed standalone, a mise task, or the repo wrapper:

```sh
./sf2-themes --version
mise run apply -- wezterm --theme vega
scripts/sf2 --version
```

`./sf2-themes` needs Python 3.11 on `PATH`.
`mise run apply` and `mise run setup` forward arguments to the project CLI.
`scripts/sf2` does the same through `uv run --project .`.
