# SF2 Themes Design System

## 0. Research Log

- Supplied reference packet: `Downloads/design-sf2/{SF2 Themes v2,SampleBlock}.dc.html` and `data/sf2-palettes.json` were the visual and interaction source of truth.
- The earlier v1 export (sidebar layout, IBM Plex Sans + Source Serif 4) is superseded by the all-mono single-page v2 export.
- Skipped image generation: the product's required visuals are live DOM and canonical palette swatches.

## 1. Atmosphere & Identity

SF2 Themes is a single-page mono catalog: quiet, dense, and terminal-native.
The signature is a 760px column of IBM Plex Mono over the selected fighter's palette, with `#`/`##` markdown-style heading prefixes and a `❯` prompt motif.

## 2. Color

### Palette

Canonical colors come from `themes/` through `generated-theme-data.json` and `site-theme-data.mjs`.
`SiteLayout.astro` embeds the full 18-family payload as `#site-theme-data`; the inline boot script applies the saved or deep-linked variant before paint, and `src/scripts/site-theme.mjs` owns runtime switching.

| Role | Token | Source / usage |
|------|-------|----------------|
| Canvas | `--bg` | selected `ui.background` |
| Surface | `--surface` | selected `ui.surface` |
| Border | `--border`, `--line` | selected `ui.border` |
| Muted text | `--muted` | selected `ui.muted` |
| Subtle text | `--subtle` | selected `ui.subtle` |
| Foreground | `--fg` | selected `ui.foreground` |
| Primary accent | `--accent` | selected `ui.accent` |
| Secondary accent | `--accent2` | selected `ui.accent_secondary` |
| Selection | `--sel` | selected `ui.selection_background` |
| Literal | `--orange` | selected `semantic.orange` |
| ANSI slots | `--c0` through `--c15` | selected `ansi.normal` and `ansi.bright` |

### Rules

- Theme values are resolved only by the embedded payload and the shared site-theme module.
- Page CSS consumes variables and does not hand-type canonical theme colors.
- `color-scheme` tracks the active mode so form controls and scrollbars match.

## 3. Typography

The site is entirely IBM Plex Mono (400/600/italic).

| Level | Size | Weight | Usage |
|------|------|--------|-------|
| Page title | 20px | 600 | `#` hero heading |
| Section | 16px | 600 | `##` section headings |
| Body | 14px | 400 | all prose and controls |
| Meta | 12–13px | 400 | hints, file ids, footer |

## 4. Spacing & Layout

The home column is `max-width: 760px` with `24px` inline padding and `64px` between major sections.
The header is a full-width flex bar that wraps its controls below the brand/nav row on narrow screens.
The command palette overlay is `min(580px, 92vw)` and the SampleBlock tab bar scrolls horizontally rather than wrapping.

## 5. Components

### Site Header

- Structure: brand link, `install`/`ports`/`github` anchors, family `<select>`, segmented `dark`/`light` toggle, `⌘K` trigger.
- States: current family/mode reflected in control state; `aria-pressed` on the mode buttons; `aria-expanded` on the palette trigger.

### SampleBlock

- Structure: bordered panel with a scrollable tab bar (`shell · neovim · python · toml · diff · prose · ansi`), seven hand-authored panes, and a footer with `{fileId} · {mode}` plus key hints.
- Dynamic values render through `data-t` spans rewritten by the shared module; colors come from `--c0`–`--c15` and the UI tokens so panes recolor without a DOM rewrite.

### Command Palette

- Structure: fixed backdrop, `❯` filter input, listbox of 36 dark/light entries plus a mode-toggle meta entry, empty-state line, and a hint footer.
- Behavior: `⌘K`/`Ctrl+K` toggles, multi-word substring filter including boss aliases, ArrowUp/Down selection, Enter applies, Escape or backdrop click closes and restores focus.

### Install Steps

- Structure: port picker (7 apps as bare underlined buttons), three labelled rounds, bordered command rows with copy buttons, and a per-port note.
- Clipboard payloads carry the `uvx --from …` prefix while displayed commands stay short; buttons are hidden when `navigator.clipboard` is unavailable.

## 6. Motion & Interaction

Motion is restrained to 120ms link color/background transitions.
`prefers-reduced-motion: reduce` disables transitions and smooth scrolling.

## 7. Depth & Surface

One level of elevation: surfaces use `--surface` and `--border` with 5–8px radii.
The command palette adds a single `0 24px 64px` shadow over a `--bg`-tinted backdrop.

## 8. Accessibility Constraints & Accepted Debt

- WCAG 2.2 AA target with 4.5:1 body contrast and 3:1 large-text contrast where the selected theme permits it.
- Every interactive element has a visible `:focus-visible` ring, semantic role, and keyboard path.
- Primary content must reflow to one readable column at 375px with no page-level horizontal overflow.
- Reduced-motion preferences are respected.

| Item | Location | Why accepted | Owner / Exit |
|------|----------|--------------|--------------|
| Remote Google Fonts can be unavailable offline | `SiteLayout.astro` | The supplied design names IBM Plex Mono; local bundling is outside this redesign scope. | Future asset self-hosting pass |
