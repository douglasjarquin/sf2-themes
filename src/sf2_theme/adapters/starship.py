"""Starship managed palette so prompt styles track the active SF2 theme."""

import os
import re
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme

MANAGED_START = "# >>> sf2-themes managed theme"
MANAGED_END = "# <<< sf2-themes managed theme"
ID_COMMENT = re.compile(r"^# sf2-themes:\s+(\S+)\s*$")

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


def merge_theme(existing: str, theme: Theme) -> str:
    """Insert or replace the managed Starship palette block."""
    block = render_block(theme)
    if MANAGED_START in existing or MANAGED_END in existing:
        start = existing.find(MANAGED_START)
        end = existing.find(MANAGED_END)
        if start == -1 or end == -1 or end < start:
            raise ThemeError("Starship config has incomplete sf2-themes markers")
        after = end + len(MANAGED_END)
        if after < len(existing) and existing[after] == "\n":
            after += 1
        # Drop a stale top-level palette assignment outside the managed block so
        # we do not leave two competing palette = lines.
        head = existing[:start]
        tail = existing[after:]
        merged = head + block + tail
    elif existing.strip():
        merged = existing.rstrip() + "\n\n" + block
    else:
        merged = block
    return merged if merged.endswith("\n") else merged + "\n"


def apply_starship(
    theme: Theme,
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> WriteResult:
    """Write the managed Starship palette block."""
    path = starship_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    merged = merge_theme(existing, theme)
    return write_file(path, merged, dry_run=dry_run, follow_symlinks=follow_symlinks)


def read_current_id(config_dir: Path | None) -> str:
    """Read the theme id from the managed Starship block."""
    path = starship_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Starship theme applied yet ({path} missing)")
    for line in path.read_text(encoding="utf-8").splitlines():
        match = ID_COMMENT.match(line.strip())
        if match:
            return match.group(1)
    raise ThemeError(f"could not read theme id from {path}")
