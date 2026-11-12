"""A mutation described in full before anything is written."""

from dataclasses import dataclass
from pathlib import Path


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
