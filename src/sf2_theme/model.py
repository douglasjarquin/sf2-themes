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

UI_COLOR_FIELDS: Final[tuple[str, ...]] = (
    "background",
    "foreground",
    "cursor_bg",
    "cursor_fg",
    "selection_bg",
    "selection_fg",
    "panel_bg",
    "sidebar_bg",
    "active_row_bg",
    "navigate_row_bg",
    "surface_dim",
    "surface0",
    "surface1",
    "overlay0",
    "overlay1",
    "subtext",
    "accent",
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


@dataclass(frozen=True, slots=True)
class ThemeMetadata:
    """Identity and catalog lookup keys for one theme."""

    id: str
    display_name: str
    kind: ThemeKind
    introduced_in: IntroducedIn
    character: str | None
    aliases: tuple[str, ...]


@dataclass(frozen=True, slots=True)
class UiColors:
    """Application chrome, selection, and surface colors."""

    background: HexColor
    foreground: HexColor
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
    accent: HexColor


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
