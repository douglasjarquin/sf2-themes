import os
import re
import tomllib
from collections.abc import Sequence
from pathlib import Path
from xml.sax.saxutils import escape

from sf2_theme.model import SELECTABLE_PREFIX, AdapterColors, Theme, project_adapter_colors
from sf2_theme.plan import Inspection, Ownership, Remove, Step, Write, recorded

THEME_DIR_NAME = "themes"
CONFIG_FILE_NAME = "config.toml"
THEME_SUFFIX = ".tmTheme"
THEME_ASSIGNMENT = re.compile(r"^\s*theme\s*=")
SECTION = re.compile(r"^\s*\[([^]]+)\]\s*$")

GLOBAL_COLORS = (
    ("background", "ui", "background"),
    ("foreground", "ui", "foreground"),
    ("caret", "adapter", "cursor_bg"),
    ("invisibles", "adapter", "overlay0"),
    ("lineHighlight", "adapter", "surface0"),
    ("selection", "adapter", "selection_bg"),
    ("findHighlight", "semantic", "yellow"),
    ("findHighlightForeground", "ui", "background"),
)

SYNTAX_COLORS = (
    ("comment", "adapter", "subtext"),
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
    return themes_dir(config_dir) / f"{theme.metadata.selectable_id}{THEME_SUFFIX}"


def _color(theme: Theme, adapter: AdapterColors, group: str, field: str) -> str:
    source = adapter if group == "adapter" else theme.ui if group == "ui" else theme.semantic
    return str(getattr(source, field))


def _setting(key: str, value: str) -> str:
    return f"<key>{key}</key><string>{escape(value)}</string>"


def render_theme(theme: Theme) -> str:
    adapter = project_adapter_colors(theme.ui)
    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">',
        '<plist version="1.0">',
        "<dict>",
        _setting("name", theme.metadata.selectable_id),
        "<key>settings</key>",
        "<array>",
        "<dict>",
        "<key>settings</key>",
        "<dict>",
    ]
    lines.extend(_setting(key, _color(theme, adapter, group, field)) for key, group, field in GLOBAL_COLORS)
    lines.extend(("</dict>", "</dict>"))
    for scope, group, field in SYNTAX_COLORS:
        lines.extend(
            (
                "<dict>",
                _setting("scope", scope),
                "<key>settings</key>",
                "<dict>",
                _setting("foreground", _color(theme, adapter, group, field)),
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
    return any(re.match(r"^\s*tui\s*[.=]", line) for line in existing.splitlines())


def merge_config(existing: str, theme: Theme) -> str:
    lines = existing.splitlines()
    bounds = _section_bounds(lines, "tui")
    assignment = f'theme = "{theme.metadata.selectable_id}"'
    if bounds is None:
        prefix = existing.rstrip()
        return f"{prefix}\n\n[tui]\n{assignment}\n" if prefix else f"[tui]\n{assignment}\n"
    start, end = bounds
    for index in range(start + 1, end):
        if THEME_ASSIGNMENT.match(lines[index]):
            lines[index] = assignment
            return "\n".join(lines).rstrip() + "\n"
    lines.insert(end, assignment)
    return "\n".join(lines).rstrip() + "\n"


def inspect_codex(themes: Sequence[Theme], config_dir: Path | None) -> Inspection:
    path = config_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    try:
        config = tomllib.loads(existing)
    except tomllib.TOMLDecodeError as error:
        return Inspection(Ownership.BROKEN, detail=f"{path} is not valid TOML: {error}")
    if _section_bounds(existing.splitlines(), "tui") is None:
        if _has_inline_tui(existing):
            return Inspection(
                Ownership.UNKNOWN_SHAPE,
                detail=(
                    f"Convert the inline tui value in {path} into a [tui] table, for example:\n\n"
                    '[tui]\ntheme = "..."'
                ),
            )
        return Inspection(Ownership.ABSENT)
    configured = config.get("tui", {}).get("theme")
    if configured is None:
        return Inspection(Ownership.ABSENT)
    if isinstance(configured, str) and configured.startswith(SELECTABLE_PREFIX):
        return recorded(Ownership.MANAGED, configured, themes)
    legacy = next((entry for entry in themes if entry.metadata.id == configured), None)
    if legacy is not None:
        return Inspection(Ownership.LEGACY, current=legacy)
    return Inspection(Ownership.FOREIGN)


def plan_codex(theme: Theme, themes: Sequence[Theme], config_dir: Path | None, slot: bool) -> list[Step]:
    steps: list[Step] = [Remove(themes_dir(config_dir) / f"{entry.metadata.id}{THEME_SUFFIX}") for entry in themes]
    steps.extend(Write(theme_path(entry, config_dir), render_theme(entry)) for entry in themes)
    if slot:
        path = config_path(config_dir)
        existing = path.read_text(encoding="utf-8") if path.exists() else ""
        steps.append(Write(path, merge_config(existing, theme)))
    return steps
