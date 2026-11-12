import json
import os
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.catalog import installed_theme, theme_pair
from sf2_theme.errors import ThemeError
from sf2_theme.model import Theme
from sf2_theme.plan import Inspection, Ownership, Step, Write


def app_home(config_dir: Path | None) -> Path:
    if config_dir is not None:
        return config_dir
    return Path(os.environ.get("XDG_CONFIG_HOME", str(Path.home() / ".config"))).expanduser() / "herdr-app"


def render_theme(theme: Theme) -> str:
    ui = theme.ui
    return (
        json.dumps(
            {
                "background": ui.background,
                "foreground": ui.foreground,
                "selection": ui.selection_background,
                "cursor": ui.cursor,
                "ansi": theme.ansi_normal.as_tuple() + theme.ansi_bright.as_tuple(),
                "colors": {
                    "canvas": ui.background,
                    "chrome": ui.surface,
                    "surface": ui.overlay,
                    "border": ui.border,
                    "accent": ui.accent,
                },
            },
            indent=2,
        )
        + "\n"
    )


def _read_settings(path: Path) -> dict:
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    try:
        data = json.loads(existing) if existing.strip() else {}
    except json.JSONDecodeError as error:
        raise ThemeError(f"Herdr app settings.json is not valid JSON: {error}") from error
    if not isinstance(data, dict):
        raise ThemeError(f"Herdr app settings.json must contain an object ({path})")
    selection = data.get("theme")
    if "theme" in data and not isinstance(selection, str | dict):
        raise ThemeError(f"Herdr app settings.json theme must be a name or object ({path})")
    return data


def inspect_herdr_app(themes: Sequence[Theme], config_dir: Path | None) -> Inspection:
    try:
        data = _read_settings(app_home(config_dir) / "settings.json")
    except ThemeError as error:
        return Inspection(Ownership.BROKEN, detail=error.message)
    if "theme" not in data:
        return Inspection(Ownership.ABSENT)
    selection = data["theme"]
    if isinstance(selection, dict):
        dark_id, light_id = selection.get("dark"), selection.get("light")
        if isinstance(dark_id, str) and isinstance(light_id, str):
            try:
                dark, light = theme_pair(installed_theme(dark_id, themes), themes)
            except ThemeError:
                return Inspection(Ownership.FOREIGN)
            if (dark_id, light_id) == (dark.metadata.selectable_id, light.metadata.selectable_id):
                return Inspection(Ownership.MANAGED, current=dark)
    return Inspection(Ownership.FOREIGN)


def plan_herdr_app(theme: Theme, themes: Sequence[Theme], config_dir: Path | None, slot: bool) -> list[Step]:
    home = app_home(config_dir)
    steps: list[Step] = [
        Write(home / "themes" / f"{candidate.metadata.selectable_id}.json", render_theme(candidate))
        for candidate in themes
    ]
    if slot:
        path = home / "settings.json"
        data = _read_settings(path)
        dark, light = theme_pair(theme, themes)
        selection = data.get("theme")
        data["theme"] = {
            **(selection if isinstance(selection, dict) else {}),
            "dark": dark.metadata.selectable_id,
            "light": light.metadata.selectable_id,
        }
        steps.append(Write(path, json.dumps(data, indent=2) + "\n"))
    return steps
