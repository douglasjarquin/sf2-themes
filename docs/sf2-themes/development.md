# Development

Local toolchain and the Astro site. Moved out of the README.

The repository's `mise.toml` pins the local toolchain to Go 1.27, golangci-lint 2.15, Node 24, aube 2.2.4, and shellcheck 0.11.0.

```sh
mise install
mise run deps
mise run build
mise run test
mise run lint
mise run apply -- wezterm --theme vega
mise run verify
```

`mise run test` runs the Go unit, parity, and repository contract tests.
`mise run verify` runs every pre-PR check, Go and web, in order; [VERIFY.md](../../VERIFY.md) lists them.

## Generated files

```sh
mise run generate-previews
mise run generate-web-theme-data
mise run parity:record
```

Regenerate `docs/previews/` and the site's theme data after a catalog change.
Re-record the parity goldens only for an intended change to CLI output or written files; a golden diff in review is a behavior change.

## Releases

The owner publishes a release by pushing a `v<semver>` tag that is on `main`.
`.github/workflows/release.yml` builds the four binaries with `scripts/ci/build-release.sh`, attests their provenance, and attaches them with `checksums.txt` and `install.sh` to the GitHub Release.

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
