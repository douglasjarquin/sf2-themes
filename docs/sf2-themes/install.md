# Install

Run the CLI from GitHub with uv, or from a checkout. Moved out of the README.

Run the CLI from GitHub with uv.
You do not need a checkout or a globally installed binary.

## One-shot

```sh
uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes --version
```

## uv tool

Install once as a uv tool, then use the bare command:

```sh
uv tool install git+https://github.com/douglasjarquin/sf2-themes.git
sf2-themes --version
```

## Checkout

From a checkout, use the committed CLI, a mise task, or the repo wrapper:

```sh
./sf2-themes --version
mise run apply -- wezterm --theme vega
scripts/sf2 --version
```

`./sf2-themes` needs Python 3.11 on `PATH`.
`mise run apply` and `mise run setup` forward arguments to the project CLI.
`scripts/sf2` does the same through `uv run --project .`.

The older `uv run --with git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes …` form still works.

The command requires uv and Python 3.11 or newer.
