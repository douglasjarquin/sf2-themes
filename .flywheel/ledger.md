# Flywheel ledger

| Date | Kind | Target | Verdict | Action | Recheck | Status |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-11-11 | factcheck | README.md, docs/sf2-themes/setup-and-apply.md: Herdr per-mode themes (herdr#2324) "not in a stable release yet" | wrong: shipped in Herdr v0.9.0 (2026-09-07) | corrected to "Herdr v0.9.0 or newer" | 2027-02-11 | proposed |
| 2026-11-11 | factcheck | web/src/pages/index.astro meta "7 apps" | wrong: APP_NAMES lists 9 adapters (8 install targets plus Slack) | corrected to "8 apps plus Slack" | 2026-12-11 | proposed |
| 2026-11-11 | factcheck | web/src/pages/index.astro meta "needs only Python 3.11+" | unsupported: the page's own install step uses uv | softened to "needs Python 3.11+ and uv" | 2027-02-11 | proposed |
| 2026-11-11 | factcheck | 36 themes; roster through Super Street Fighter II Turbo | verified: 17 fighters x 2 + main x 2 in themes/ | none | 2027-02-11 | verified |
| 2026-11-11 | factcheck | Codex: .tmTheme in $CODEX_HOME/themes, [tui].theme, /theme | verified against Codex docs and theme_picker.rs | none | 2027-02-11 | verified |
| 2026-11-11 | factcheck | Claude Code: ~/.claude/themes/*.json, settings.json theme, /theme | verified against Claude Code terminal-config docs | none | 2027-02-11 | verified |
| 2026-11-11 | factcheck | Slack: Preferences, Appearance, Custom theme, Import theme; Darker sidebars | verified against Slack help article 205166337 | none | 2027-02-11 | verified |
| 2026-11-11 | factcheck | seo.mjs operatingSystem "Linux, macOS, Windows" | unverified: CI runs only ubuntu-latest; no Windows evidence either way | none (needs owner input) | 2026-12-11 | open |
| 2026-11-11 | factcheck | pyproject.toml description lists only WezTerm, Herdr, Neovim | incomplete, not false | none | 2026-12-11 | open |
