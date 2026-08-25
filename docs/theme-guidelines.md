# Street Fighter II theme design rules

These rules apply to every committed theme in this repository.

The catalog contains a dark and light variant for every theme.

Dark variants keep their original IDs and light variants append `-light`.

## 0. Light-mode variants

Light variants must remain recognizably related to their dark counterparts through the same character accent, semantic hue meanings, and warm arcade-era family.

Light variants must use an explicitly reviewed light background and dark foreground rather than runtime color inversion or inherited tokens.

Light variants must satisfy the same contrast, ANSI distinctness, complete-resolution, adapter, and automated validation requirements as dark variants.

Light variants must keep red, green, yellow, blue, magenta, cyan, and orange semantic meanings unchanged.

Light variants must be authored as complete TOML files with reviewed hexadecimal values.

## 1. Theme identity

Every theme must clearly belong to the same Street Fighter II family.

The common family traits are:

- dark arcade-style base
- warm, slightly aged foreground rather than clinical white
- high-saturation character accents
- strong active-state treatment
- clear separation between panels and terminal background
- no pastel, low-contrast, or modern washed-out interpretation

## 2. Source hierarchy

Character colors must be selected in this order:

1. The character's canonical default Street Fighter II costume
2. The character's home stage
3. Signature attacks, effects, or equipment
4. General arcade interface colors
5. Readability corrections

Readability always overrides exact pixel sampling.

Alternate costumes must not be used as the primary reference for version 1.

## 3. Character identity distribution

Each character theme should roughly use:

- 60-75% neutral/background family
- 15-25% character primary color
- 5-10% signature secondary color
- remaining colors for stable semantic states

A theme must not fill the entire UI with the character's costume color.

## 4. Semantic colors remain semantic

Character identity must not destroy terminal meaning.

Across every theme:

- red means error, failure, blocked, or destructive
- green means success, completed, or healthy
- yellow means warning, pending, or active
- blue means information or neutral activity
- magenta means branch, special, or alternate state
- cyan means secondary information or notification
- orange means interrupted, warning, or attention

Character colors may tint those hue families, but must not exchange their meaning.

## 5. Contrast requirements

Every theme must satisfy:

- Primary foreground/background: 7:1 minimum
- Normal text on surfaces: 4.5:1 minimum
- Selection foreground/background: 4.5:1 minimum
- Cursor glyph/cursor background: 4.5:1 minimum
- Focus boundaries and essential non-text states: 3:1 minimum
- Muted text may fall below 4.5:1 only when it is genuinely secondary

## 6. ANSI requirements

Every theme must provide:

- exactly eight named normal colors
- exactly eight named bright colors
- the standard ANSI meanings
- visibly distinguishable normal and bright variants
- no accidental duplicate entries within a row
- a readable black/bright-black pair

The manifest uses named keys rather than positional arrays.

## 7. Complete theme resolution

Every committed theme must be fully resolved.

Do not rely on runtime inheritance such as `extends = "main"` for required colors.

A scaffold tool may copy `main` as an authoring starting point, but each committed character theme contains every required token explicitly.

## 8. No hidden production color generation

Mixing and lightening functions may generate an initial candidate.

Final committed theme files must contain reviewed hexadecimal values.

## 9. Adapter completeness

Each adapter must map the complete semantic theme contract.

Herdr receives every current theme token, including overlays, mauve, and peach.

WezTerm receives foreground, background, cursor, selection, split, scrollbar, ANSI, and bright colors.

Neovim receives every current theme token, including overlays and subtext, mapped onto both `colors` values and editor highlight groups.

Codex receives the current UI and semantic colors through its supported Syntect/TextMate `.tmTheme` surface under `$CODEX_HOME/themes/`.

## 10. Automated validation

Every theme must pass schema validation, exact `#RRGGBB` validation, unique id and alias checks, contrast tests, ANSI completeness tests, renderer parsing, golden-output snapshots, and idempotent apply tests.

No theme is accepted based only on appearance in one screenshot.
