"""Immutable theme value objects."""

import re
from dataclasses import dataclass
from enum import StrEnum, unique
from typing import Final, NewType

from sf2_theme.errors import ThemeError

HexColor = NewType("HexColor", str)

ANSI_ORDER: Final[tuple[str, ...]] = (
    "black",
    "red",
    "green",
    "yellow",
    "blue",
    "magenta",
    "cyan",
    "white",
)
HEX_PATTERN: Final = r"^#[0-9A-Fa-f]{6}$"
SELECTABLE_PREFIX: Final = "sf2-"

UI_COLOR_FIELDS: Final[tuple[str, ...]] = (
    "background",
    "surface",
    "overlay",
    "border",
    "foreground",
    "muted",
    "subtle",
    "accent",
    "accent_secondary",
    "cursor",
    "cursor_text",
    "selection_background",
    "selection_foreground",
)
SEMANTIC_FIELDS: Final[tuple[str, ...]] = (
    "red",
    "green",
    "yellow",
    "blue",
    "magenta",
    "cyan",
    "orange",
)


@unique
class ThemeKind(StrEnum):
    MAIN = "main"
    CHARACTER = "character"


@unique
class IntroducedIn(StrEnum):
    MAIN = "main"
    WORLD_WARRIOR = "world-warrior"
    CHAMPION_EDITION = "champion-edition"
    SUPER = "super"
    SUPER_TURBO = "super-turbo"


@unique
class ThemeVariant(StrEnum):
    DARK = "dark"
    LIGHT = "light"


@dataclass(frozen=True, slots=True)
class ThemeMetadata:
    """Identity and catalog lookup keys for one theme."""

    id: str
    display_name: str
    kind: ThemeKind
    introduced_in: IntroducedIn
    character: str | None
    aliases: tuple[str, ...]
    name: str
    variant: ThemeVariant
    family: str
    stage: str

    @property
    def selectable_id(self) -> str:
        """Return the prefixed identity used by installed adapters."""
        return selectable_id(self.id)


@dataclass(frozen=True, slots=True)
class UiColors:
    """Canonical designer palette colors."""

    background: HexColor
    surface: HexColor
    overlay: HexColor
    border: HexColor
    foreground: HexColor
    muted: HexColor
    subtle: HexColor
    accent: HexColor
    accent_secondary: HexColor
    cursor: HexColor
    cursor_text: HexColor
    selection_background: HexColor
    selection_foreground: HexColor

    @property
    def cursor_bg(self) -> HexColor:
        return project_adapter_colors(self).cursor_bg

    @property
    def cursor_fg(self) -> HexColor:
        return project_adapter_colors(self).cursor_fg

    @property
    def selection_bg(self) -> HexColor:
        return project_adapter_colors(self).selection_bg

    @property
    def selection_fg(self) -> HexColor:
        return project_adapter_colors(self).selection_fg

    @property
    def panel_bg(self) -> HexColor:
        return project_adapter_colors(self).panel_bg

    @property
    def sidebar_bg(self) -> HexColor:
        return project_adapter_colors(self).sidebar_bg

    @property
    def active_row_bg(self) -> HexColor:
        return project_adapter_colors(self).active_row_bg

    @property
    def navigate_row_bg(self) -> HexColor:
        return project_adapter_colors(self).navigate_row_bg

    @property
    def surface_dim(self) -> HexColor:
        return project_adapter_colors(self).surface_dim

    @property
    def surface0(self) -> HexColor:
        return project_adapter_colors(self).surface0

    @property
    def surface1(self) -> HexColor:
        return project_adapter_colors(self).surface1

    @property
    def overlay0(self) -> HexColor:
        return project_adapter_colors(self).overlay0

    @property
    def overlay1(self) -> HexColor:
        return project_adapter_colors(self).overlay1

    @property
    def subtext(self) -> HexColor:
        return project_adapter_colors(self).subtext


@dataclass(frozen=True, slots=True)
class AdapterColors:
    """Current operational roles derived from the canonical designer palette."""

    cursor_bg: HexColor
    cursor_fg: HexColor
    selection_bg: HexColor
    selection_fg: HexColor
    panel_bg: HexColor
    sidebar_bg: HexColor
    active_row_bg: HexColor
    navigate_row_bg: HexColor
    surface_dim: HexColor
    surface0: HexColor
    surface1: HexColor
    overlay0: HexColor
    overlay1: HexColor
    subtext: HexColor


def project_adapter_colors(ui: UiColors) -> AdapterColors:
    """Map canonical tokens to every current adapter role deterministically."""
    return AdapterColors(
        cursor_bg=ui.cursor,
        cursor_fg=ui.cursor_text,
        selection_bg=ui.selection_background,
        selection_fg=ui.selection_foreground,
        panel_bg=ui.surface,
        sidebar_bg=ui.background,
        active_row_bg=ui.selection_background,
        navigate_row_bg=ui.overlay,
        surface_dim=ui.background,
        surface0=ui.surface,
        surface1=ui.overlay,
        overlay0=ui.border,
        overlay1=ui.muted,
        subtext=ui.subtle,
    )


@dataclass(frozen=True, slots=True)
class SemanticColors:
    """Hue families that must keep their terminal meanings."""

    red: HexColor
    green: HexColor
    yellow: HexColor
    blue: HexColor
    magenta: HexColor
    cyan: HexColor
    orange: HexColor


@dataclass(frozen=True, slots=True)
class AnsiColors:
    """Named ANSI slots in WezTerm order."""

    black: HexColor
    red: HexColor
    green: HexColor
    yellow: HexColor
    blue: HexColor
    magenta: HexColor
    cyan: HexColor
    white: HexColor

    def as_tuple(self) -> tuple[HexColor, ...]:
        """Return colors in black, red, green, yellow, blue, magenta, cyan, white order."""
        return (
            self.black,
            self.red,
            self.green,
            self.yellow,
            self.blue,
            self.magenta,
            self.cyan,
            self.white,
        )


@dataclass(frozen=True, slots=True)
class Theme:
    """A fully resolved Street Fighter II theme."""

    metadata: ThemeMetadata
    ui: UiColors
    semantic: SemanticColors
    ansi_normal: AnsiColors
    ansi_bright: AnsiColors

    def lookup_keys(self) -> tuple[str, ...]:
        """Return ids that should resolve to this theme."""
        keys = (self.metadata.id, *self.metadata.aliases)
        return keys


def selectable_id(theme_id: str) -> str:
    """Prefix an adapter-facing identity while leaving catalog ids unchanged."""
    return theme_id if theme_id.startswith(SELECTABLE_PREFIX) else f"{SELECTABLE_PREFIX}{theme_id}"


def parse_hex(raw: str) -> HexColor:
    """Parse an exact #RRGGBB color into a lowercase HexColor."""
    if re.fullmatch(HEX_PATTERN, raw) is None:
        raise ThemeError(f"color must be #RRGGBB, got {raw!r}")
    return HexColor(raw.lower())


def theme_to_toml(theme: Theme) -> str:
    """Render a theme back to canonical TOML for `show`."""
    meta = theme.metadata
    lines = [
        "schema_version = 1",
        "",
        "[meta]",
        f'id = "{meta.id}"',
        f'display_name = "{meta.display_name}"',
        f'kind = "{meta.kind.value}"',
        f'introduced_in = "{meta.introduced_in.value}"',
    ]
    if meta.character is not None:
        lines.append(f'character = "{meta.character}"')
    alias_items = ", ".join(f'"{alias}"' for alias in meta.aliases)
    lines.append(f"aliases = [{alias_items}]")
    lines.extend(
        (
            f'name = "{meta.name}"',
            f'variant = "{meta.variant.value}"',
            f'family = "{meta.family}"',
            f'stage = "{meta.stage}"',
        )
    )
    lines.extend(["", "[ui]"])
    for name in UI_COLOR_FIELDS:
        lines.append(f'{name} = "{getattr(theme.ui, name)}"')
    lines.extend(["", "[semantic]"])
    for name in SEMANTIC_FIELDS:
        lines.append(f'{name} = "{getattr(theme.semantic, name)}"')
    lines.extend(["", "[ansi.normal]"])
    for name in ANSI_ORDER:
        lines.append(f'{name} = "{getattr(theme.ansi_normal, name)}"')
    lines.extend(["", "[ansi.bright]"])
    for name in ANSI_ORDER:
        lines.append(f'{name} = "{getattr(theme.ansi_bright, name)}"')
    lines.append("")
    return "\n".join(lines)
