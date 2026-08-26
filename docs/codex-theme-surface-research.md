# Codex CLI theme surface research

Research date: 2026-08-25.

Scope: official Codex CLI/TUI source, rather than Codex Desktop appearance settings.
The inspected `openai/codex` source revision is [`23cedf48023879d9666e41d8eee765a9482120c0`](https://github.com/openai/codex/tree/23cedf48023879d9666e41d8eee765a9482120c0).

## Supported configuration

The supported persisted setting is TOML at `[tui].theme` in the user Codex configuration, with an optional kebab-case theme name.
The source documents this field as a syntax-highlighting theme override and directs users to `/theme` or `$CODEX_HOME/themes` for custom themes.

**Evidence:** [`config/src/types.rs` lines 751-756](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/config/src/types.rs#L751-L756)

```rust
pub theme: Option<String>,
```

Custom themes use TextMate/Syntect `.tmTheme` files named `<theme-name>.tmTheme` in `$CODEX_HOME/themes/`.
The configured name resolves first against bundled theme names, then against that exact file path.

**Evidence:** [`tui/src/render/highlight.rs` lines 184-191](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/render/highlight.rs#L184-L191) and [215-234](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/render/highlight.rs#L215-L234)

```rust
codex_home.join("themes").join(format!("{name}.tmTheme"))
```

The picker discovers custom files by the `.tmTheme` extension, so an sf2-themes adapter should emit that format and a stable theme name, not JSON, YAML, or a desktop theme object.

**Evidence:** [`tui/src/render/highlight.rs` lines 380-397](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/render/highlight.rs#L380-L397)

## Light/dark behavior

When `[tui].theme` is absent or invalid, Codex chooses `catppuccin-latte` when the terminal default background is light and `catppuccin-mocha` otherwise.
The source obtains the background through its terminal color probe; this is terminal appearance detection, not an independent Codex light/dark config key.

**Evidence:** [`tui/src/render/highlight.rs` lines 194-210](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/render/highlight.rs#L194-L210) and [`tui/src/terminal_palette.rs` lines 130-142](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/terminal_palette.rs#L130-L142)

An explicit `[tui].theme` value overrides that automatic selection when it names a bundled theme or a valid custom `.tmTheme` file.
Therefore, separate sf2 light and dark files should be exposed as separate named themes; the adapter cannot rely on Codex to select between two variants inside one custom file.

The final sentence is an inference from the source: the custom-file resolver accepts one named file, while automatic selection only chooses the two built-in defaults.

## Reload and restart semantics

`/theme` applies a selected theme to the running TUI after persisting `[tui].theme`; the event handler resolves the selected theme, swaps the runtime syntax theme, refreshes the status line, and schedules a frame.

**Evidence:** [`tui/src/app/event_dispatch.rs` lines 2829-2849](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/app/event_dispatch.rs#L2829-L2849)

Editing `config.toml` while Codex is running has a reload path: the TUI sends a `reload_user_config` request, and the session reloads user configuration layers.
The source comments say some derived configuration fields remain session-static, so a full restart is the conservative requirement for startup-resolved theme configuration or any behavior not explicitly refreshed.

**Evidence:** [`tui/src/app_server_session.rs` lines 1519-1533](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/tui/src/app_server_session.rs#L1519-L1533) and [`core/src/session/mod.rs` lines 1883-1889](https://github.com/openai/codex/blob/23cedf48023879d9666e41d8eee765a9482120c0/codex-rs/core/src/session/mod.rs#L1883-L1889)

The source explicitly supports live runtime swaps for the theme picker, but does not establish that an externally replaced `.tmTheme` file is automatically watched.
Treat file replacement plus a fresh Codex process, or an explicit `/theme` re-selection, as the reliable adapter workflow.
The absence of file-watcher evidence is an evidence-bound limitation, not a claim that watching is impossible.

## Recommendation

Implement the sf2-themes Codex adapter as a copy/install operation that writes `sf2-<catalog-id>.tmTheme` to `$CODEX_HOME/themes/` and sets `[tui].theme = "sf2-<catalog-id>"` in `config.toml`.
Provide distinct light and dark theme names, and tell users to restart Codex after file/config changes unless they select the theme again through `/theme`.
Do not target Codex Desktop `appearanceLight*` or `appearanceDark*` settings: those are outside the CLI/TUI surface researched here and are not established by the Codex CLI source.
