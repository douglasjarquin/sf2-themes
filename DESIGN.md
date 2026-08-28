# sf2-themes Design System

## 1. Atmosphere & Identity

sf2-themes feels like an arcade cabinet wired directly into a developer's terminal.
The signature is pixel-era display type, dark navy surfaces, and fighter colors used as functional visual signals.

## 2. Color

### Palette

| Role | Token | Value | Usage |
|------|------|-------|-------|
| Surface/primary | `--color-background` | `#101a3a` | Page background |
| Surface/panel | `--color-panel` | `#101a3a` | Cards and footer |
| Surface/0 | `--color-surface-0` | `#28304a` | Recessed surfaces |
| Surface/1 | `--color-surface-1` | `#363d53` | Raised surfaces |
| Text/primary | `--color-foreground` | `#fff4d6` | Headings and body text |
| Accent/primary | `--color-accent` | `#f2b134` | Primary actions and emphasis |
| Status/error | `--color-red` | `#e8565f` | Error and combat signal |
| Status/info | `--color-cyan` | `#35c4c2` | Labels, focus, and informational signal |
| Theme-specific | `--character-<id>-<role>` | Theme TOML values | Palette swatches and fighter previews |

### Rules

Theme-specific colors come from the TOML catalog and are never retyped in page markup.
Accent colors are used for interactive emphasis and meaningful visual signals.

## 3. Typography

### Scale

| Level | Size | Weight | Line Height | Usage |
|------|------|--------|-------------|-------|
| Display | `26px` | 700 | 1.5 | Page titles |
| Body/lg | `17px` | 400 | 1.7 | Lead paragraphs |
| Body | `14px` | 400 | 1.7 | Supporting copy |
| Caption | `12px` | 400 | 1.5 | Hex values and metadata |
| Label | `10px` | 700 | 1.8 | Arcade labels and card names |
| Overline | `9px` | 700 | 1.9 | Section eyebrows |

### Font Stack

- Display: `Press Start 2P`, monospace.
- Body: `Chakra Petch`, sans-serif.
- Mono: `JetBrains Mono`, monospace.

## 4. Spacing & Layout

### Base Unit

All spacing derives from a base unit of 4px.

The shared spacing tokens are defined in `web/src/styles/global.css` from `--space-1` through `--space-20`.

The shared radius tokens are `--radius-panel` for substantial surfaces and `--radius-control` for compact controls.

Preview-specific geometry tokens such as `--swatch-height` and `--code-line-number-width` are defined alongside the shared tokens so the audit surface does not introduce one-off spacing values.

### Grid

The content width is `1180px` plus responsive horizontal padding.
Palette source cards use a five-column grid on wide screens and one column below 621px.
Palette variant swatches use five columns on wide screens and intrinsic responsive columns below 621px.
The primary mobile breakpoint is 620px and the shared navigation breakpoint is 720px.

## 5. Components

### Site Shell

- **Structure**: header, main content, footer.
- **Variants**: route-aware header navigation.
- **Spacing**: shared frame and spacing tokens.
- **States**: active route, focus-visible links.
- **Accessibility**: semantic landmarks and keyboard-visible focus.
- **Motion**: micro transitions on links and controls.
- **Layout**: full-height grid shell.

### Palette Card

- **Structure**: interactive button, color block, name, hex value, and token label.
- **Variants**: main source color and catalog theme palette.
- **Spacing**: shared frame, grid, and spacing tokens.
- **States**: default, hover, active, focus-visible, and copy feedback.
- **Accessibility**: native button semantics and an explicit copy label containing the color value.
- **Motion**: 120ms transform on hover, disabled for reduced motion.
- **Layout**: responsive grid.

### Palette Variant Row

- **Structure**: theme identity and a responsive group of five interactive swatches.
- **Variants**: dark and light catalog themes.
- **Spacing**: shared card, grid, and spacing tokens.
- **States**: default, hover, active, focus-visible, and copy feedback.
- **Accessibility**: native buttons, visible focus, and named swatches.
- **Motion**: no decorative motion.
- **Layout**: two-column row on wide screens and a single-column stack on mobile.

### Palette Preview Family

- **Structure**: a family identity row followed by paired dark and light palette mode panels.
- **Variants**: one family section for Main and each TOML-backed fighter, with every canonical mode rendered in source order.
- **Spacing**: wide family sections with breathable separators and compact mode-panel internals.
- **States**: default, focused code/terminal surfaces, and copy feedback for commands and color values.
- **Accessibility**: semantic headings, native copy buttons, labeled swatch groups, and readable static content without JavaScript.
- **Motion**: no decorative motion; copy controls use the shared micro transition only for feedback.
- **Layout**: mode panels sit side by side on wide screens and stack below 720px; swatch grids reflow below 620px.

### Featured Palette Preview

- **Structure**: one featured `article` contains an identity header, paired syntax-highlighted code and terminal panes, and 25 individually labeled swatches grouped as six neutral, three accent, eight normal ANSI, and eight bright ANSI colors.
- **Variants**: the server-rendered fallback is the canonical Main palette, and the browser selects one canonical `paletteVariants` entry at runtime after its serialized payload has parsed successfully.
- **Static fallback**: JavaScript-disabled, malformed-payload, and missing-payload sessions retain the complete server-rendered Main preview instead of replacing it.
- **Syntax**: code tokens expose semantic kinds for comments, keywords, properties, strings, numbers, punctuation, and plain text, with colors derived from the selected canonical palette.
- **Spacing**: the primitive uses shared typography, spacing, border, radius, and shadow tokens, while canonical palette CSS custom properties supply all preview colors.
- **Accessibility**: the preview has a descriptive article label, semantic section headings for code and terminal content, readable text labels and values for every swatch, and complete useful content before client JavaScript runs.
- **Motion**: the preview has no decorative animation, so reduced-motion users see the same static fallback and no runtime transition is required.
- **Layout**: code and terminal panes plus swatch ramps use two columns above 980px, one column from 761px through 980px, two compact columns from 521px through 760px, and one readable column at 520px and below without horizontal overflow.

### Palette Mode Panel

- **Structure**: mode header, live code preview, truthful `sf2-themes show` terminal preview, six neutral swatches, three accent swatches, and two eight-color ANSI rows.
- **Variants**: dark and light, each styled from its canonical TOML-backed UI and semantic tokens.
- **Spacing**: 12px panel padding with 4px-derived internal gaps and a 12px radius from the revised reference direction.
- **States**: default, hover/focus copy controls, and polite per-panel copy status.
- **Accessibility**: mode labels, explicit color names and values, visible focus, and no image-only information.
- **Motion**: color and border feedback only, with reduced-motion support inherited from the shared shell.
- **Layout**: code and terminal previews stack inside each mode panel; palette ramps remain readable at 375px.

### Palette Mode Filter

- **Structure**: a grouped set of native toggle buttons above the catalog.
- **Variants**: all, dark, and light.
- **Spacing**: shared spacing tokens and compact control gaps.
- **States**: default, hover, active, focus-visible, and pressed.
- **Accessibility**: `role="group"`, an accessible group label, and `aria-pressed` on each control.
- **Motion**: color and border feedback only.
- **Layout**: inline cluster that stacks with the catalog heading on mobile.

### Arcade Cabinet

- **Structure**: marquee, live terminal transcript, fighter selector, and coin action.
- **Variants**: idle, playing, and complete sequence states with a selected TOML-backed fighter theme.
- **States**: selected fighter, pressed fighter controls, credit count, active fight frame, and completed K.O. frame.
- **Accessibility**: native buttons, exclusive `aria-pressed` selection, and a polite live transcript.
- **Motion**: 120ms color and border transition for theme changes; transcript frames advance every 480ms and use no transform animation when reduced motion is preferred.
- **Layout**: cabinet remains a readable single-column terminal surface at mobile widths.

### Screenshot Card

- **Structure**: a generated cabinet image with a fighter label and captured game moment.
- **Variants**: one card for each playable character theme.
- **Spacing**: two-column archive grid on wide screens with shared frame and card spacing.
- **States**: default, loading image, and keyboard focus when the image is inspected by assistive technology.
- **Accessibility**: semantic figure captions, descriptive alternative text, intrinsic image dimensions, and a single labeled screenshot landmark.
- **Motion**: no decorative card motion; images remain stable records of the cabinet.
- **Layout**: cards collapse to one column below 621px without cropping the source image's 16:9 frame.

## 6. Motion & Interaction

The micro duration is 120ms with ease-out timing.
Only transform, color, background, and border presentation transitions are used for interactive feedback.
Reduced-motion users receive no non-essential transform animation.
Copy actions announce success or failure in a live status region.
The arcade cabinet advances meaningful text fight frames at a 480ms cadence after a credit is inserted.

## 7. Depth & Surface

The system uses a mixed strategy of tonal surfaces and offset shadows.
Cards use panel surfaces with a 4px offset shadow, while color blocks use a 5px offset shadow.
Dividers use low-contrast foreground mixes to preserve the arcade material without adding visual noise.

## 8. Accessibility Constraints & Accepted Debt

### Constraints

The target is WCAG 2.2 AA with keyboard reachability, visible focus, semantic landmarks, readable contrast, and reduced-motion support.
Interactive palette elements must remain native buttons with accessible names.
The page must reflow to one readable column at 375px without horizontal overflow.

### Accepted Debt

| Item | Location | Why accepted | Owner / Exit |
|------|----------|--------------|--------------|
| None recorded | - | No new accessibility debt is introduced by the palette catalog expansion. | Revisit during the next visual QA pass. |
