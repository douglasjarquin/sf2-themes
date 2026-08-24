# sf2-theme Web Foundations

## 1. Atmosphere and Identity

The site feels like a maintained arcade cabinet: dark, tactile, direct, and legible rather than ornamental.
Its signature is hard-edged gold and red type over layered navy surfaces with a restrained CRT scanline texture.

## 2. Color

All named colors originate in `themes/main.toml` through `src/data/theme-data.mjs`.
The shared layout exposes background, foreground, accent, red, cyan, surface, and character custom properties on the document root.
Shared CSS must consume those properties and must not duplicate palette hex values.

## 3. Typography

- Display and navigation: Press Start 2P.
- Body: Chakra Petch.
- Code and technical metadata: JetBrains Mono.
- Display text uses compact sizes and generous line height so pixel glyphs remain readable.
- Body text never renders below 14px.

## 4. Spacing and Layout

Spacing follows a 4px base unit.
The content frame is 1180px with fluid 16px to 32px inline gutters.
Document scroll owns vertical movement; header and footer stay in normal flow.
Reusable grids use `repeat(auto-fit, minmax(min(..., 100%), 1fr))` so they cannot force narrow viewport overflow.

## 5. Components

### Site header

- Structure: brand link, primary navigation, external repository action.
- States: default, hover, active route through `aria-current`, keyboard focus.
- Layout: wrapping cluster that remains fully visible at 390px.
- Accessibility: labelled navigation and visible focus outlines.

### Site footer

- Structure: project identity and external support links.
- Layout: wrapping cluster within the shared content frame.
- Accessibility: descriptive external link labels.

### Route shell

- Structure: one page heading and a short handoff placeholder inside the shared layout.
- Layout: centered content frame with an intrinsic grid hook for future route content.

## 6. Motion and Interaction

Interactive color and transform feedback uses a 120ms ease-out transition.
Focus is never animated or hidden.
Reduced-motion preferences disable smooth scrolling and transitions.

## 7. Depth and Surface

The mixed depth strategy uses two-pixel borders, tonal surface shifts, and offset hard shadows.
Rounded cards and soft glass effects do not belong in the shared shell.

## 8. Accessibility Constraints and Accepted Debt

Target WCAG 2.2 AA with semantic landmarks, one page heading, full keyboard navigation, visible focus, and no horizontal page overflow at 390px.
There is no accepted shared-shell accessibility debt.
