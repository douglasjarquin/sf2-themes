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

### Palette Mode Filter

- **Structure**: a grouped set of native toggle buttons above the catalog.
- **Variants**: all, dark, and light.
- **Spacing**: shared spacing tokens and compact control gaps.
- **States**: default, hover, active, focus-visible, and pressed.
- **Accessibility**: `role="group"`, an accessible group label, and `aria-pressed` on each control.
- **Motion**: color and border feedback only.
- **Layout**: inline cluster that stacks with the catalog heading on mobile.

## 6. Motion & Interaction

The micro duration is 120ms with ease-out timing.
Only transform, color, background, and border presentation transitions are used for interactive feedback.
Reduced-motion users receive no non-essential transform animation.
Copy actions announce success or failure in a live status region.

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
