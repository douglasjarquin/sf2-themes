"""Starship managed palette so prompt styles track the active SF2 theme."""

import os
import re
import tomllib
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.adapters.zsh_syntax import plan_zsh_syntax
from sf2_theme.model import Theme
from sf2_theme.plan import Inspection, Ownership, Step, Write, read_existing, recorded, recorded_id

MANAGED_START = "# >>> sf2-themes managed theme"
MANAGED_END = "# <<< sf2-themes managed theme"
ID_COMMENT = re.compile(r"^# sf2-themes:\s+(\S+)\s*$")
TABLE_HEADER = re.compile(r"^\s*\[")
TOP_LEVEL_PALETTE = re.compile(r"^\s*palette\s*=")

# Starship `style = "blue"` resolves through `palette`, not the terminal ANSI
# table. Remap the named colors the stock prompt uses onto SF2 tokens so dirname
# and related chrome follow accent/identity instead of cool ANSI blue/green.
PALETTE_KEYS = (
    ("black", "ui", "background"),
    ("red", "semantic", "red"),
    ("green", "semantic", "green"),
    ("yellow", "semantic", "yellow"),
    ("blue", "ui", "accent"),
    ("purple", "ui", "accent_secondary"),
    ("cyan", "semantic", "cyan"),
    ("white", "ui", "foreground"),
    ("bright-black", "ui", "muted"),
    ("bright-red", "semantic", "red"),
    ("bright-green", "semantic", "green"),
    ("bright-yellow", "semantic", "yellow"),
    ("bright-blue", "ui", "accent"),
    ("bright-purple", "ui", "accent_secondary"),
    ("bright-cyan", "semantic", "cyan"),
    ("bright-white", "ui", "foreground"),
)


def starship_path(config_dir: Path | None) -> Path:
    """Return the Starship config.toml path."""
    if config_dir is not None:
        return config_dir / "starship.toml"
    configured = os.environ.get("STARSHIP_CONFIG")
    if configured:
        return Path(configured).expanduser()
    xdg = os.environ.get("XDG_CONFIG_HOME")
    root = Path(xdg).expanduser() if xdg else Path.home() / ".config"
    return root / "starship.toml"


def render_block(theme: Theme) -> str:
    """Render a managed Starship palette block for the theme."""
    lines = [
        MANAGED_START,
        f"# sf2-themes: {theme.metadata.selectable_id}",
        'palette = "sf2"',
        "",
        "[palettes.sf2]",
    ]
    for key, group, field in PALETTE_KEYS:
        source = theme.ui if group == "ui" else theme.semantic
        lines.append(f'{key} = "{getattr(source, field)}"')
    lines.append(MANAGED_END)
    lines.append("")
    return "\n".join(lines)


def _without_block(existing: str) -> str | None:
    """Return the config with the managed block removed, or None when its markers are incomplete."""
    if MANAGED_START not in existing and MANAGED_END not in existing:
        return existing
    start = existing.find(MANAGED_START)
    end = existing.find(MANAGED_END)
    if start == -1 or end == -1 or end < start:
        return None
    after = end + len(MANAGED_END)
    if after < len(existing) and existing[after] == "\n":
        after += 1
    return existing[:start] + existing[after:]


def _top_level_end(lines: list[str]) -> int:
    """Return where the top-level key region ends, keeping a comment that heads the first table with it."""
    end = next((index for index, line in enumerate(lines) if TABLE_HEADER.match(line)), len(lines))
    while end > 0 and lines[end - 1].lstrip().startswith("#"):
        end -= 1
    return end


def merge_theme(existing: str, theme: Theme) -> str:
    """Place the managed palette block in the top-level key region.

    Starship reads `palette` only as a top-level key, so the block goes before
    the first table. Any top-level `palette` outside the block is replaced.
    """
    remaining = _without_block(existing)
    lines = (existing if remaining is None else remaining).splitlines()
    split = _top_level_end(lines)
    head = [line for line in lines[:split] if not TOP_LEVEL_PALETTE.match(line)]
    parts = ["\n".join(head).rstrip("\n"), render_block(theme).rstrip("\n"), "\n".join(lines[split:]).strip("\n")]
    merged = "\n\n".join(part for part in parts if part) + "\n"
    tomllib.loads(merged)
    return merged


def inspect_starship(themes: Sequence[Theme], config_dir: Path | None) -> Inspection:
    path = starship_path(config_dir)
    existing = read_existing(path)
    outside = _without_block(existing)
    if outside is None:
        return Inspection(Ownership.BROKEN, detail=f"{path} has incomplete sf2-themes markers")
    try:
        tomllib.loads(existing)
        user_palette = "palette" in tomllib.loads(outside)
    except tomllib.TOMLDecodeError as error:
        return Inspection(Ownership.BROKEN, detail=f"{path} is not valid TOML: {error}")
    if user_palette:
        return Inspection(Ownership.FOREIGN)
    if outside != existing:
        return recorded(Ownership.MANAGED, recorded_id(existing, ID_COMMENT), themes)
    return Inspection(Ownership.ABSENT)


def plan_starship(theme: Theme, themes: Sequence[Theme], config_dir: Path | None, slot: bool) -> list[Step]:
    """Plan the managed palette block and the matching zsh highlight snippet."""
    if not slot:
        return []
    path = starship_path(config_dir)
    existing = read_existing(path)
    return [Write(path, merge_theme(existing, theme)), plan_zsh_syntax(theme)]
