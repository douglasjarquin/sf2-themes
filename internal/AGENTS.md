# GO PACKAGE BOUNDARY

## OVERVIEW

`themes/` is the canonical data source; the Go packages turn its TOML into typed, validated `catalog.Theme` values and adapter output.
`themes/embed.go` embeds the catalog, so a copied binary needs no sibling files.
The runtime path is `catalog.DecodeTOML -> catalog.ParseTheme -> Theme -> catalog.Load -> cli/adapters`.
A mutation command runs `inspect -> decide -> plan -> commit`: adapters inspect and plan, the CLI decides, and the commit step alone writes.
`catalog.Parse`/`ParseFS` is the intentionally unvalidated discovery path used by validation and reporting; `catalog.Load`/`LoadFS` adds catalog uniqueness and all error-severity theme checks.
The Python CLI at the parity reference is the behavior contract: the Go binary prints the same stdout, stderr, and exit codes and writes the same bytes, except where the plan lists an exception.

## WHERE TO LOOK

| Concern | Owner | Boundary |
|---|---|---|
| Theme values and names | `catalog/model.go` | Palette types, exact hex parsing, canonical IDs, adapter-facing IDs, and `show` rendering. |
| TOML reading | `catalog/toml.go` | The one TOML reader, held to Python 3.11 `tomllib` acceptance for the catalog and user configs. |
| TOML shape | `catalog/parse.go` | Schema, required fields, enum values, unknown keys, and typed construction. |
| Semantic validity | `catalog/validate.go` | Contrast, ANSI distinctions, lookup-key uniqueness, and error versus warning severity. |
| Catalog lifecycle | `catalog/catalog.go` | `fs.FS` discovery, deterministic order, validated loading, lookup, pairing, and defaults. |
| Argument parsing | `cli/args.go` | The command set, flags in any position, `--name=value`, `--`, and argparse-compatible error lines. |
| Help text | `cli/help.go` | Hand-written help and usage lines that never depend on terminal width. |
| Public dispatch | `cli/cli.go`, `cmd/sf2-themes` | `cli.Main`, the command handlers, the app table, the ownership gate, selection, refusal text, and summaries. |
| Python behaviors | `pycompat/` | Text reads, line splitting, whitespace, regex classes, paths, `repr`, JSON, and unified diffs as Python produces them. |
| Parity proof | `parity/` | Goldens recorded from the Python reference and the TOML/JSON parser corpus. |

## CONVENTIONS

- Keep structural rejection in `parse.go`, semantic checks in `validate.go`, and their composition in `catalog.go`.
- Route normal runtime consumers through `catalog.Load`; reserve `catalog.Parse` for callers that must inspect or report invalid catalog data.
- Load the catalog once per command and pass that slice into lookups and adapters so one operation uses one deterministic snapshot.
- Resolve a selected catalog id to its dark and light siblings through `catalog.ThemePair` instead of reconstructing `-light` names in adapters.
- Resolve a managed `sf2-` pointer identity through `catalog.InstalledTheme` so setup can refresh an existing pair without `--theme`.
- Keep catalog and CLI selection IDs short, such as `ryu-light`.
- Derive installed names through `ThemeMetadata.SelectableID` or `catalog.SelectableID` so adapter-facing identities are consistently `sf2-<catalog-id>`.
- Preserve the intentional spelling split: public commands `sf2`/`sf2-themes`, Go module `sf2-themes`, and managed state directory `sf2-theme`.
- The binary always reads the embedded catalog; `SF2_THEME_DIR` is not honored, and maintainer tools pass `os.DirFS(dir)` to `ParseFS`/`LoadFS`.
- Decode every TOML document through `catalog.DecodeTOML` and every JSON document through `pycompat.Loads`, after `pycompat.ReadText` or `pycompat.DecodeText`.
- Reproduce any Python behavior that reaches output bytes through `pycompat`, never with an ad hoc helper.
- Command handlers write only to the `cli.Context` streams and return an error for an `error:` line; nothing below `cmd/` calls `os.Exit`.
- Let adapters own syntax-aware reads and merges, then return final content as plan steps for the commit step.
- Keep `github.com/BurntSushi/toml` the only runtime dependency.

## ANTI-PATTERNS

- Do not use `catalog.Parse` as a shortcut for runtime behavior that promises a valid catalog.
- Do not duplicate TOML reading, contrast rules, catalog ordering, or lookup semantics inside adapters or commands.
- Do not hand-build `sf2-` names from aliases, display names, or storage paths.
- Do not write live adapter configuration from an adapter; return plan steps so the commit step keeps its safety contract.
- Do not refuse inside an adapter; report the ownership class and let the CLI gate decide.
- Do not move application parsing into the commit step; it is format-agnostic and receives final content.
- Do not upgrade BurntSushi/toml past a version that reads TOML 1.0 by default without the parser corpus passing.

## VERIFICATION

- Pair catalog, model, parser, or validator changes with `internal/catalog` tests whose expected values are recorded from the Python reference.
- Pair argument or help changes with `internal/cli` tests; usage errors keep the reference's error line and exit status 2.
- Pair TOML or JSON reader changes with the parser corpus in `internal/parity`.
- Pair output or written-file changes with the parity goldens, and re-record them only from the Python reference.
- Finish with `go test`, `go vet`, `gofmt`, and `golangci-lint` on the changed packages.
