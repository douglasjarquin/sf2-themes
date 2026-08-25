import os
import re
import tomllib
from collections.abc import Sequence
from pathlib import Path
from xml.sax.saxutils import escape

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme

THEME_DIR_NAME = "themes"
CONFIG_FILE_NAME = "config.toml"
THEME_SUFFIX = ".tmTheme"
THEME_ASSIGNMENT = re.compile(r"^\s*theme\s*=")
SECTION = re.compile(r"^\s*\[([^]]+)\]\s*$")

GLOBAL_COLORS = (
    ("background", "ui", "background"),
    ("foreground", "ui", "foreground"),
    ("caret", "ui", "cursor_bg"),
    ("invisibles", "ui", "overlay0"),
    ("lineHighlight", "ui", "surface0"),
    ("selection", "ui", "selection_bg"),
    ("findHighlight", "semantic", "yellow"),
    ("findHighlightForeground", "ui", "background"),
)

SYNTAX_COLORS = (
    ("comment", "ui", "subtext"),
    ("string", "semantic", "green"),
    ("constant.numeric", "semantic", "orange"),
    ("constant.language", "semantic", "orange"),
    ("entity.name.function", "semantic", "blue"),
    ("entity.name.type", "semantic", "yellow"),
    ("keyword", "semantic", "magenta"),
    ("storage", "semantic", "magenta"),
    ("support", "semantic", "cyan"),
    ("variable", "ui", "foreground"),
    ("invalid", "semantic", "red"),
    ("markup.inserted", "semantic", "green"),
    ("markup.deleted", "semantic", "red"),
)


def codex_home(config_dir: Path | None) -> Path:
    if config_dir is not None:
        return config_dir
    configured = os.environ.get("CODEX_HOME")
    return Path(configured).expanduser() if configured else Path.home() / ".codex"


def config_path(config_dir: Path | None) -> Path:
    return codex_home(config_dir) / CONFIG_FILE_NAME


def themes_dir(config_dir: Path | None) -> Path:
    return codex_home(config_dir) / THEME_DIR_NAME


def theme_path(theme: Theme, config_dir: Path | None) -> Path:
    return themes_dir(config_dir) / f"{theme.metadata.id}{THEME_SUFFIX}"


def _color(theme: Theme, group: str, field: str) -> str:
    source = theme.ui if group == "ui" else theme.semantic
    return str(getattr(source, field))


def _setting(key: str, value: str) -> str:
    return f"<key>{key}</key><string>{escape(value)}</string>"


def render_theme(theme: Theme) -> str:
    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">',
        '<plist version="1.0">',
        "<dict>",
        _setting("name", theme.metadata.display_name),
        "<key>settings</key>",
        "<array>",
        "<dict>",
        "<key>settings</key>",
        "<dict>",
    ]
    lines.extend(_setting(key, _color(theme, group, field)) for key, group, field in GLOBAL_COLORS)
    lines.extend(("</dict>", "</dict>"))
    for scope, group, field in SYNTAX_COLORS:
        lines.extend(
            (
                "<dict>",
                _setting("scope", scope),
                "<key>settings</key>",
                "<dict>",
                _setting("foreground", _color(theme, group, field)),
                "</dict>",
                "</dict>",
            )
        )
    lines.extend(("</array>", "</dict>", "</plist>", ""))
    return "\n".join(lines)


def _section_bounds(lines: list[str], section: str) -> tuple[int, int] | None:
    start = next(
        (
            index
            for index, line in enumerate(lines)
            if (match := SECTION.match(line)) and match.group(1).strip() == section
        ),
        None,
    )
    if start is None:
        return None
    end = next((index for index in range(start + 1, len(lines)) if SECTION.match(lines[index])), len(lines))
    return start, end


def _has_inline_tui(existing: str) -> bool:
    return any(re.match(r"^\s*tui\s*=", line) for line in existing.splitlines())


def _has_tui_theme(existing: str) -> bool:
    lines = existing.splitlines()
    bounds = _section_bounds(lines, "tui")
    if bounds is None:
        return False
    start, end = bounds
    return any(THEME_ASSIGNMENT.match(line) for line in lines[start + 1 : end])


def merge_config(existing: str, theme: Theme) -> str:
    if existing:
        tomllib.loads(existing)
    lines = existing.splitlines()
    bounds = _section_bounds(lines, "tui")
    assignment = f'theme = "{theme.metadata.id}"'
    if bounds is None:
        if _has_inline_tui(existing):
            raise ThemeError("Codex config has an inline tui table; replace it with [tui] before applying")
        prefix = existing.rstrip()
        return f"{prefix}\n\n[tui]\n{assignment}\n" if prefix else f"[tui]\n{assignment}\n"
    start, end = bounds
    for index in range(start + 1, end):
        if THEME_ASSIGNMENT.match(lines[index]):
            lines[index] = assignment
            return "\n".join(lines).rstrip() + "\n"
    lines.insert(end, assignment)
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


def _write_config(
    theme: Theme,
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> WriteResult:
    path = config_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    return write_file(
        path,
        merge_config(existing, theme),
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )


def apply_codex(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    results = _write_themes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    results.append(
        _write_config(
            theme,
            config_dir=config_dir,
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
    )
    return results


def setup_codex(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    replace_theme: bool = True,
) -> list[WriteResult]:
    results = _write_themes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    path = config_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    if replace_theme or not _has_tui_theme(existing):
        results.append(
            _write_config(
                theme,
                config_dir=config_dir,
                dry_run=dry_run,
                follow_symlinks=follow_symlinks,
            )
        )
    return results


def read_current_id(config_dir: Path | None) -> str:
    path = config_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Codex theme applied yet ({path} missing)")
    config = tomllib.loads(path.read_text(encoding="utf-8"))
    tui = config.get("tui")
    if not isinstance(tui, dict) or not isinstance(tui.get("theme"), str):
        raise ThemeError(f"could not read theme id from {path}")
    return tui["theme"]
