"""Herdr managed-block theme integration."""

import os
import re
import tomllib
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme, project_adapter_colors

MANAGED_START = "# >>> sf2-themes managed theme"
MANAGED_END = "# <<< sf2-themes managed theme"
ID_COMMENT = re.compile(r"^# sf2-themes:\s+(\S+)\s*$")
HERDR_TOKENS = (
    ("sidebar_bg", "adapter", "sidebar_bg"),
    ("panel_bg", "adapter", "panel_bg"),
    ("active_row_bg", "adapter", "active_row_bg"),
    ("selection_bg", "adapter", "navigate_row_bg"),
    ("surface0", "adapter", "surface0"),
    ("surface1", "adapter", "surface1"),
    ("surface_dim", "adapter", "surface_dim"),
    ("overlay0", "adapter", "overlay0"),
    ("overlay1", "adapter", "overlay1"),
    ("text", "ui", "foreground"),
    ("subtext0", "adapter", "subtext"),
    ("accent", "ui", "accent"),
    ("mauve", "semantic", "magenta"),
    ("green", "semantic", "green"),
    ("yellow", "semantic", "yellow"),
    ("red", "semantic", "red"),
    ("blue", "semantic", "blue"),
    ("teal", "semantic", "cyan"),
    ("peach", "semantic", "orange"),
)


def herdr_path(config_dir: Path | None) -> Path:
    """Return the Herdr config.toml path."""
    if config_dir is not None:
        return config_dir / "config.toml"
    configured = os.environ.get("HERDR_CONFIG_PATH")
    if configured:
        return Path(configured).expanduser()
    xdg = os.environ.get("XDG_CONFIG_HOME")
    root = Path(xdg).expanduser() if xdg else Path.home() / ".config"
    return root / "herdr" / "config.toml"


def render_block(theme: Theme) -> str:
    """Render a fully resolved, marked Herdr theme block."""
    adapter = project_adapter_colors(theme.ui)
    lines = [
        MANAGED_START,
        f"# sf2-themes: {theme.metadata.selectable_id}",
        "[theme]",
        'name = "terminal"',
        "",
        "[theme.custom]",
    ]
    for token, group, field in HERDR_TOKENS:
        source = adapter if group == "adapter" else theme.ui if group == "ui" else theme.semantic
        lines.append(f'{token} = "{getattr(source, field)}"')
    lines.append(MANAGED_END)
    lines.append("")
    return "\n".join(lines)


def _has_unmarked_theme(content: str) -> bool:
    if MANAGED_START in content:
        return False
    current = ""
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.startswith("[") and stripped.endswith("]"):
            current = stripped.strip("[]").strip()
            if current == "theme" or current.startswith("theme."):
                return True
            continue
        if current == "" and re.match(r"^theme\s*=", stripped):
            return True
    return False


def _strip_theme_sections(content: str) -> str:
    kept: list[str] = []
    skip = False
    current = ""
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.startswith("[") and stripped.endswith("]"):
            current = stripped.strip("[]").strip()
            skip = current == "theme" or current.startswith("theme.")
        top_level = current == "" and re.match(r"^theme\s*=", stripped)
        if skip or top_level:
            continue
        kept.append(line)
    return "\n".join(kept).rstrip()


def merge_theme(existing: str, theme: Theme, *, adopt: bool) -> str:
    """Insert or replace the managed Herdr block."""
    if MANAGED_START in existing or MANAGED_END in existing:
        start = existing.find(MANAGED_START)
        end = existing.find(MANAGED_END)
        if start == -1 or end == -1 or end < start:
            raise ThemeError("Herdr config has incomplete sf2-themes markers")
        after = end + len(MANAGED_END)
        if after < len(existing) and existing[after] == "\n":
            after += 1
        merged = existing[:start] + render_block(theme) + existing[after:]
    elif _has_unmarked_theme(existing):
        if not adopt:
            raise ThemeError("Herdr config already has a [theme] section; pass --adopt to replace it")
        preserved = _strip_theme_sections(existing)
        merged = f"{preserved}\n\n{render_block(theme)}" if preserved else render_block(theme)
    elif existing.strip():
        merged = existing.rstrip() + "\n\n" + render_block(theme)
    else:
        merged = render_block(theme)
    tomllib.loads(merged)
    return merged if merged.endswith("\n") else merged + "\n"


def apply_herdr(
    theme: Theme,
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    adopt: bool,
) -> WriteResult:
    """Write the managed Herdr theme block."""
    path = herdr_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    if existing:
        tomllib.loads(existing)
    merged = merge_theme(existing, theme, adopt=adopt)
    return write_file(path, merged, dry_run=dry_run, follow_symlinks=follow_symlinks)


def read_current_id(config_dir: Path | None) -> str:
    """Read the theme id from the managed Herdr block."""
    path = herdr_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Herdr theme applied yet ({path} missing)")
    for line in path.read_text(encoding="utf-8").splitlines():
        match = ID_COMMENT.match(line.strip())
        if match:
            return match.group(1)
    raise ThemeError(f"could not read theme id from {path}")
