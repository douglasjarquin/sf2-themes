"""Lazygit theme fragments and managed configuration."""

import os
import re
import sys
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme, project_adapter_colors

THEME_DIR_NAME = "themes"
CONFIG_FILE_NAME = "config.yml"
THEME_SUFFIX = ".yml"
MANAGED_THEME_START = "# >>> sf2-themes managed theme"
MANAGED_THEME_END = "# <<< sf2-themes managed theme"
MANAGED_AUTHOR_START = "# >>> sf2-themes managed author colors"
MANAGED_AUTHOR_END = "# <<< sf2-themes managed author colors"
ID_COMMENT = re.compile(r"^# sf2-themes:\s+(\S+)\s*$")


def lazygit_dir(config_dir: Path | None) -> Path:
    """Return the Lazygit configuration directory."""
    if config_dir is not None:
        return config_dir
    configured = os.environ.get("LAZYGIT_CONFIG_DIR")
    if configured:
        return Path(configured).expanduser()
    if sys.platform == "darwin":
        return Path.home() / "Library" / "Application Support" / "lazygit"
    xdg = os.environ.get("XDG_CONFIG_HOME")
    root = Path(xdg).expanduser() if xdg else Path.home() / ".config"
    return root / "lazygit"


def config_path(config_dir: Path | None) -> Path:
    """Return the Lazygit config.yml path."""
    return lazygit_dir(config_dir) / CONFIG_FILE_NAME


def themes_dir(config_dir: Path | None) -> Path:
    """Return the directory containing installed SF2 theme fragments."""
    return lazygit_dir(config_dir) / THEME_DIR_NAME


def theme_path(theme: Theme, config_dir: Path | None) -> Path:
    """Return the installed path for one catalog theme fragment."""
    return themes_dir(config_dir) / f"{theme.metadata.selectable_id}{THEME_SUFFIX}"


def _theme_colors(theme: Theme) -> tuple[tuple[str, str, bool], ...]:
    adapter = project_adapter_colors(theme.ui)
    return (
        ("activeBorderColor", str(theme.ui.accent), True),
        ("inactiveBorderColor", str(adapter.overlay0), False),
        ("searchingActiveBorderColor", str(theme.semantic.yellow), False),
        ("optionsTextColor", str(theme.ui.accent), False),
        ("selectedLineBgColor", str(adapter.selection_bg), False),
        ("inactiveViewSelectedLineBgColor", str(adapter.surface0), False),
        ("cherryPickedCommitFgColor", str(theme.semantic.green), False),
        ("cherryPickedCommitBgColor", str(adapter.surface1), False),
        ("markedBaseCommitFgColor", str(theme.semantic.orange), False),
        ("markedBaseCommitBgColor", str(adapter.surface1), False),
        ("unstagedChangesColor", str(theme.semantic.red), False),
        ("defaultFgColor", str(theme.ui.foreground), False),
    )


def _theme_lines(theme: Theme, *, indent: str = "") -> list[str]:
    lines = [f"{indent}gui:", f"{indent}  theme:"]
    for key, color, bold in _theme_colors(theme):
        lines.append(f"{indent}    {key}:")
        lines.append(f"{indent}      - '{color}'")
        if bold:
            lines.append(f"{indent}      - bold")
    lines.extend(
        (
            f"{indent}  authorColors:",
            f"{indent}    '*': '{theme.ui.accent_secondary}'",
        )
    )
    return lines


def _theme_section_lines(theme: Theme) -> list[str]:
    lines = ["  theme:"]
    for key, color, bold in _theme_colors(theme):
        lines.append(f"    {key}:")
        lines.append(f"      - '{color}'")
        if bold:
            lines.append("      - bold")
    return lines


def render_theme(theme: Theme) -> str:
    """Render a complete Lazygit theme fragment."""
    return "\n".join(_theme_lines(theme)) + "\n"


def _replace_marked(lines: list[str], start_marker: str, end_marker: str, replacement: list[str]) -> list[str]:
    starts = [index for index, line in enumerate(lines) if line.strip() == start_marker]
    ends = [index for index, line in enumerate(lines) if line.strip() == end_marker]
    if not starts and not ends:
        return lines
    if len(starts) != 1 or len(ends) != 1 or ends[0] < starts[0]:
        raise ThemeError("Lazygit config has incomplete sf2-themes markers")
    return lines[: starts[0]] + replacement + lines[ends[0] + 1 :]


def _normalized_key(line: str) -> str:
    return line.split(" #", 1)[0].rstrip()


def _top_level_section(lines: list[str], name: str) -> tuple[int, int] | None:
    start = next((index for index, line in enumerate(lines) if _normalized_key(line) == f"{name}:"), None)
    if start is None:
        return None
    end = next((index for index in range(start + 1, len(lines)) if line_is_top_level(lines[index])), len(lines))
    return start, end


def line_is_top_level(line: str) -> bool:
    """Return whether a non-empty YAML line starts at document level."""
    return bool(line.strip()) and not line.startswith((" ", "\t", "#"))


def _gui_subsection_bounds(lines: list[str], name: str) -> tuple[int, int] | None:
    gui = _top_level_section(lines, "gui")
    if gui is None:
        return None
    start, end = gui
    section = next((index for index in range(start + 1, end) if _normalized_key(lines[index]) == f"  {name}:"), None)
    if section is None:
        return None
    section_end = next(
        (
            index
            for index in range(section + 1, end)
            if lines[index].startswith("  ") and not lines[index].startswith("    ")
        ),
        end,
    )
    return section, section_end


def _managed_theme_lines(theme: Theme) -> list[str]:
    return [
        f"  {MANAGED_THEME_START}",
        f"  # sf2-themes: {theme.metadata.selectable_id}",
        *_theme_section_lines(theme),
        f"  {MANAGED_THEME_END}",
    ]


def _managed_author_lines(theme: Theme) -> list[str]:
    return [
        f"  {MANAGED_AUTHOR_START}",
        f"  # sf2-themes: {theme.metadata.selectable_id}",
        "  authorColors:",
        f"    '*': '{theme.ui.accent_secondary}'",
        f"  {MANAGED_AUTHOR_END}",
    ]


def _managed_author_entry_lines(theme: Theme) -> list[str]:
    indent = "    "
    return [
        f"{indent}{MANAGED_AUTHOR_START}",
        f"{indent}# sf2-themes: {theme.metadata.selectable_id}",
        f"{indent}'*': '{theme.ui.accent_secondary}'",
        f"{indent}{MANAGED_AUTHOR_END}",
    ]


def merge_config(existing: str, theme: Theme, *, adopt: bool) -> str:
    """Merge the selected theme into config.yml while preserving user settings."""
    lines = existing.splitlines()
    lines = _replace_marked(lines, MANAGED_THEME_START, MANAGED_THEME_END, _managed_theme_lines(theme))
    author_marker = next((line for line in lines if line.strip() == MANAGED_AUTHOR_START), None)
    had_managed_author = author_marker is not None
    if author_marker is not None:
        indent = author_marker[: len(author_marker) - len(author_marker.lstrip())]
        author_replacement = _managed_author_entry_lines(theme) if len(indent) >= 4 else _managed_author_lines(theme)
        lines = _replace_marked(lines, MANAGED_AUTHOR_START, MANAGED_AUTHOR_END, author_replacement)

    theme_bounds = _gui_subsection_bounds(lines, "theme")
    if theme_bounds is not None:
        start, end = theme_bounds
        gui = _top_level_section(lines, "gui")
        has_managed_theme = gui is not None and any(
            line.strip() == MANAGED_THEME_START for line in lines[gui[0] : gui[1]]
        )
        if has_managed_theme:
            pass
        elif not adopt:
            raise ThemeError("Lazygit config already has a gui.theme section; pass --adopt to replace it")
        else:
            lines[start:end] = _managed_theme_lines(theme)
    else:
        gui = _top_level_section(lines, "gui")
        if gui is None:
            lines.extend(["", "gui:", *_managed_theme_lines(theme)])
        else:
            _, end = gui
            lines[end:end] = _managed_theme_lines(theme)

    if not had_managed_author:
        author = _gui_subsection_bounds(lines, "authorColors")
        if author is None:
            gui = _top_level_section(lines, "gui")
            _, end = gui
            lines[end:end] = _managed_author_lines(theme)
        elif not any(line.strip() == MANAGED_AUTHOR_START for line in lines[author[0] : author[1]]):
            start, end = author
            wildcard = next(
                (index for index in range(start + 1, end) if lines[index].strip().startswith("'*':")),
                None,
            )
            if wildcard is None:
                lines[end:end] = _managed_author_entry_lines(theme)
            else:
                lines[wildcard:wildcard + 1] = _managed_author_entry_lines(theme)
    return "\n".join(lines).rstrip() + "\n"


def _write_themes(
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    return [
        write_file(
            theme_path(theme, config_dir),
            render_theme(theme),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
        for theme in themes
    ]


def apply_lazygit(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    adopt: bool,
) -> list[WriteResult]:
    """Install every catalog fragment and select one in config.yml."""
    results = _write_themes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    path = config_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    results.append(
        write_file(
            path,
            merge_config(existing, theme, adopt=adopt),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
    )
    return results


def setup_lazygit(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    adopt: bool,
) -> list[WriteResult]:
    """Set up Lazygit theme fragments and select the requested theme."""
    return apply_lazygit(
        theme,
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
        adopt=adopt,
    )


def read_current_id(config_dir: Path | None) -> str:
    """Read the selected SF2 theme id from managed Lazygit configuration."""
    path = config_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Lazygit theme applied yet ({path} missing)")
    for line in path.read_text(encoding="utf-8").splitlines():
        match = ID_COMMENT.match(line.strip())
        if match:
            return match.group(1)
    raise ThemeError(f"could not read theme id from {path}")
