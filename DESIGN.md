# SF2 Themes Design System

## 0. Research Log

- Supplied reference packet: `Downloads/sf2-new-design/{Home,Themes,Theme Detail,Install,SiteNav,Design System}.dc.html` and `themes.js` were the visual and interaction source of truth.
- Existing project extraction: `web/src/styles/global.css`, `SiteLayout.astro`, `SiteHeader.astro`, and `theme-data.mjs` established the Astro shell, generated catalog contract, and existing accessibility baseline.
- Skipped image generation: the user supplied precise HTML design references and the product's required visuals are live DOM and canonical palette swatches, so introducing new imagery would not be authoritative.

## 1. Atmosphere & Identity

SF2 Themes is a dark editorial arcade catalog: confident, spacious, and technical without becoming a dashboard.
The signature is a deep navy canvas with warm fighter accents, oversized Archivo Black headlines, and terminal-like mono controls that make every palette feel wearable.

## 2. Color

### Palette

Canonical preview colors come from `web/src/data/generated-theme-data.json` through `theme-data.mjs`.
The site shell maps the selected canonical family/mode to these variables at runtime.

| Role | Token | Source / usage |
|------|-------|----------------|
| Canvas | `--bg` | selected `ui.background` |
| Surface | `--su` | selected `ui.surface0` |
| Overlay | `--ov` | selected `ui.surface1` |
| Muted text | `--mu` | selected `ui.muted` |
| Foreground | `--fg` | selected `ui.foreground` |
| Primary accent | `--ac` | selected `ui.accent` |
| Secondary accent | `--se` | selected `ui.accent_secondary` |
| ANSI slots | `--a0` through `--a15` | selected `ansi.normal` and `ansi.bright` |

### Rules

- Theme values are resolved only by `theme-data.mjs` and the shared site-theme runtime.
- Page CSS consumes variables and does not hand-type canonical theme colors.
- Site chrome selection is separate from game palette selection; the browser game continues to receive its explicit palette payload.

## 3. Typography

| Level | Size | Weight | Usage |
|------|------|--------|-------|
| Display | `clamp(4rem, 9vw, 8.75rem)` | Archivo Black | hero and theme detail titles |
| Section | `clamp(2rem, 5vw, 3rem)` | Archivo Black | major headings |
| Card | `1.625rem` | Archivo 700 | roster and install card headings |
| Lead | `clamp(1.25rem, 2vw, 1.625rem)` | Archivo 500 | route introductions |
| Body | `1.125rem` | Archivo 400 | explanatory copy |
| Meta | `0.8125rem` | IBM Plex Mono 600 | labels, statuses, navigation |
| Code | `1rem` | IBM Plex Mono 400/600 | commands and terminal panels |

Font stacks are Archivo / Archivo Black for editorial text and IBM Plex Mono for controls and code.
The three-family split is intentional because the supplied reference explicitly uses these roles.

## 4. Spacing & Layout

Spacing uses a 4px base with shared tokens from `web/src/styles/global.css`.
The page shell is `--content-width: 1400px` with 40px desktop gutters and `clamp()` gutters on narrow screens.
Major sections use 48-96px vertical rhythm; cards use 24-36px padding; grids use 12-24px gaps.
Responsive breakpoints are 760px for stacked page composition and 640px for compact navigation/control clusters.
All grid tracks use `minmax(0, 1fr)` or `minmax(min(20rem, 100%), 1fr)` so unbroken commands cannot widen the page.

## 5. Components

### Site Header and Theme Picker

- Structure: sticky header, brand link, primary route links, GitHub link, picker button, disclosure panel with dark/light controls and catalog-family buttons.
- Variants: active route, selected family, dark mode, light mode, open, closed.
- States: default, hover, active, focus-visible, disabled unavailable clipboard feedback.
- Accessibility: semantic navigation, `aria-current`, `aria-expanded`, labelled disclosure, keyboard-reachable buttons, Escape-to-close, outside-click close.
- Motion: 120ms color/transform affordance; panel visibility is stateful and respects reduced motion.

### Palette Swatch Strip

- Structure: six live swatch spans for accent, secondary, and ANSI preview colors.
- Variants: roster card and terminal preview.
- States: static, hover on containing link, focus on containing link.
- Accessibility: decorative swatches are hidden; card name and mode remain text-visible.

### Theme Roster Card

- Structure: semantic link containing family metadata, display title, world line, six-color strip, detail link, and site-theme action.
- Variants: normal `2 MODES`, selected `WEARING`, dark-mode and light-mode data.
- States: default, hover, active, focus-visible, selected.
- Accessibility: action button stops link navigation and exposes `ACTIVE` when selected.
- Motion: 120ms border/color/transform affordance only.

### Code Panel and Copy Action

- Structure: labelled terminal/code surface with command content and a copy button.
- Variants: install command, theme apply command, terminal preview, TypeScript preview.
- States: default, copied, clipboard failure, focus-visible.
- Accessibility: semantic `pre`/`code`, `aria-live` status, no success state before clipboard resolution.
- Motion: button label changes only after successful write; reduced motion keeps the state change.

### Detail Palette and Mode Controls

- Structure: theme heading, dark/light control cluster, apply-to-site action, adapter command chooser, terminal/code panels, palette grid, adjacent navigation.
- Variants: dark/light preview, selected site theme, adapter selection.
- States: default, active, copied, focus-visible.
- Accessibility: buttons are labelled by mode/adapter, and adjacent links are keyboard reachable.

### Existing Palette Preview

- Structure: canonical palette identity, syntax/code pane, terminal pane, and grouped neutral/accent/ANSI swatches.
- Variants: full preview routes and compact featured home preview.
- States: static fallback, selected persisted site family/mode, copied color/command, clipboard failure, and malformed payload fallback.
- Accessibility: semantic article and headings, native copy buttons, labelled swatches, and complete no-JavaScript fallback.
- Layout: full previews use responsive grids; compact home preview stacks to a readable single column at narrow widths.

### Existing Arcade Cabinet

- Structure: generated terminal cabinet, fighter controls, and deterministic game host.
- Variants: idle, playing, result, and reduced-motion.
- States: selected fighter/theme, credit, pause, combat, result, and renderer failure fallback.
- Accessibility: native controls, pressed selection state, live status, and preserved static poster.
- Layout: game-specific geometry remains owned by `web/src/game/` and is not coupled to site-theme chrome.

### Page Breadcrumb

- Structure: route context link, separator, and current label.
- Variants: existing palette/preview/install routes.
- States: default, hover, and focus-visible.
- Accessibility: readable route context with semantic links and keyboard focus.

## 6. Motion & Interaction

Motion is restrained and functional.
Interactive color/border/transform transitions use `--duration-micro: 120ms` and `ease-out`.
The picker is an explicit disclosure rather than a decorative animation.
Copy labels change on success and automatically restore; no motion is required for comprehension.
`prefers-reduced-motion: reduce` disables transitions and smooth scrolling.

## 7. Depth & Surface

The system uses a mixed borders-and-tonal-shift strategy from the supplied reference.
Surfaces use `--su` and `--ov` with 1px overlay borders, 10-14px radii, and one restrained popover shadow.
The body background remains open and unboxed; nested panels are reserved for terminal/code content that needs a clear boundary.

## 8. Accessibility Constraints & Accepted Debt

- WCAG 2.2 AA target with 4.5:1 body contrast and 3:1 large-text contrast where the selected theme permits it.
- Every interactive element has a visible focus ring, 44px minimum hit target intent, semantic role, and keyboard path.
- Site picker state is announced through labels and `aria-expanded`; copy feedback uses `aria-live`.
- Primary content must reflow to one readable column at 375px with no page-level horizontal overflow.
- Reduced-motion preferences are respected.

| Item | Location | Why accepted | Owner / Exit |
|------|----------|--------------|--------------|
| Remote Google Fonts can be unavailable offline | `SiteLayout.astro` | The supplied design explicitly names Archivo and IBM Plex Mono; local bundling is outside this redesign scope. | Future asset self-hosting pass |
