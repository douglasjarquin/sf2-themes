# Development

Local toolchain and the Astro site. Moved out of the README.

The repository's `mise.toml` pins the local toolchain to Python 3.11, Node 24, uv 0.11, and aube 2.2.4.

```sh
mise install
mise run test
mise run apply -- wezterm --theme vega
```

## Astro site

```sh
mise run web:install
mise run web:check
mise run web:build
mise run web:test
mise run web:dev
```

`mise run web:dev` starts the Astro site at `http://127.0.0.1:4321`.
For a stable HTTPS domain instead of a raw port, run `mise run web:install` once, then `mise run web:dev:local` for `https://sf2-themes.test` via [portless](https://github.com/vercel-labs/portless) - see [web/AGENTS.md](web/AGENTS.md) for setup notes.
