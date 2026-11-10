"""Herdr desktop JSON themes and appearance-specific settings."""

import json
import os
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.catalog import installed_theme, load_catalog, theme_pair
from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme


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


def _read_settings(path: Path) -> tuple[str, dict]:
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
    return existing, data


def setup_herdr_app(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    replace_theme: bool = True,
) -> list[WriteResult]:
    home = app_home(config_dir)
    path = home / "settings.json"
    existing, data = _read_settings(path)
    if replace_theme or "theme" not in data:
        dark, light = theme_pair(theme, themes)
        selection = data.get("theme")
        data["theme"] = {
            **(selection if isinstance(selection, dict) else {}),
            "dark": dark.metadata.selectable_id,
            "light": light.metadata.selectable_id,
        }
        content = json.dumps(data, indent=2) + "\n"
    else:
        content = existing
    # Validate settings and their write policy before installing catalog assets.
    write_file(path, content, dry_run=True, follow_symlinks=follow_symlinks)
    results = [
        write_file(
            home / "themes" / f"{candidate.metadata.selectable_id}.json",
            render_theme(candidate),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
        for candidate in themes
    ]
    results.append(write_file(path, content, dry_run=dry_run, follow_symlinks=follow_symlinks))
    return results


def selection_exists(config_dir: Path | None) -> bool:
    _, data = _read_settings(app_home(config_dir) / "settings.json")
    return "theme" in data


def read_current_id(config_dir: Path | None) -> str:
    path = app_home(config_dir) / "settings.json"
    _, data = _read_settings(path)
    selection = data.get("theme")
    if isinstance(selection, dict):
        dark_id, light_id = selection.get("dark"), selection.get("light")
        if isinstance(dark_id, str) and isinstance(light_id, str):
            catalog = load_catalog()
            dark, light = theme_pair(installed_theme(dark_id, catalog), catalog)
            if (dark_id, light_id) == (dark.metadata.selectable_id, light.metadata.selectable_id):
                return dark_id
    raise ThemeError(f"no matching SF2 light/dark pair in Herdr app settings.json ({path})")
