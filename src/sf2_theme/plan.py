"""A mutation described in full before anything is written, and the inspection that decides it."""

import re
from collections.abc import Sequence
from dataclasses import dataclass
from enum import StrEnum, unique
from pathlib import Path

from sf2_theme.catalog import installed_theme
from sf2_theme.errors import ThemeError
from sf2_theme.model import Theme


@dataclass(frozen=True, slots=True)
class Write:
    """Replace `path` with `content`."""

    path: Path
    content: str


@dataclass(frozen=True, slots=True)
class Remove:
    """Delete `path` when it exists."""

    path: Path


Step = Write | Remove


@unique
class Ownership(StrEnum):
    """Who owns the theme setting in an app's configuration."""

    ABSENT = "absent"
    MANAGED = "managed"
    LEGACY = "legacy"
    FOREIGN = "foreign"
    UNKNOWN_SHAPE = "unknown shape"
    BROKEN = "broken"


@dataclass(frozen=True, slots=True)
class Inspection:
    """What one read of an app's configuration found.

    `current` is the sf2 theme the app selects when it can be read; `unreadable`
    marks an sf2 selection that exists but does not resolve. `detail` explains a
    broken configuration or carries the manual fix for an unknown shape.
    """

    ownership: Ownership
    current: Theme | None = None
    unreadable: bool = False
    detail: str = ""


def read_existing(path: Path) -> str:
    """Return a config file's text, or an empty string when it does not exist yet."""
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""


def recorded_id(text: str, pattern: re.Pattern[str]) -> str | None:
    """Return the theme id the first managed id comment in `text` records."""
    for line in text.splitlines():
        if match := pattern.match(line.strip()):
            return match.group(1)
    return None


def recorded(ownership: Ownership, theme_id: str | None, themes: Sequence[Theme]) -> Inspection:
    """Inspect an sf2-owned setting whose recorded theme id may not resolve."""
    if theme_id is not None:
        try:
            return Inspection(ownership, current=installed_theme(theme_id, themes))
        except ThemeError:
            pass
    return Inspection(ownership, unreadable=True)


class Refusal(ThemeError):
    """A mutation the ownership of an app's configuration does not allow."""

    def __init__(self, message: str, ownership: Ownership) -> None:
        super().__init__(message)
        self.ownership = ownership
