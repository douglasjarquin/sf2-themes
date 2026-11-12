import json
import os
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.model import Theme
from sf2_theme.plan import Step, Write

THEME_DIR_NAME = "themes"
SETTINGS_FILE_NAME = "settings.json"
CUSTOM_THEME_PREFIX = "custom:"

OVERRIDE_TOKENS = (
    ("claude", "ui", "accent"),
    ("error", "semantic", "red"),
    ("success", "semantic", "green"),
    ("warning", "semantic", "yellow"),
    ("text", "ui", "foreground"),
    ("permission", "semantic", "orange"),
    ("bashBorder", "ui", "border"),
    ("diffAdded", "semantic", "green"),
    ("diffRemoved", "semantic", "red"),
    ("suggestion", "ui", "accent_secondary"),
)


def claude_home(config_dir: Path | None) -> Path:
    if config_dir is not None:
        return config_dir
    configured = os.environ.get("CLAUDE_CONFIG_DIR")
    return Path(configured).expanduser() if configured else Path.home() / ".claude"


def settings_path(config_dir: Path | None) -> Path:
    return claude_home(config_dir) / SETTINGS_FILE_NAME


def themes_dir(config_dir: Path | None) -> Path:
    return claude_home(config_dir) / THEME_DIR_NAME


def theme_path(theme: Theme, config_dir: Path | None) -> Path:
    return themes_dir(config_dir) / f"{theme.metadata.selectable_id}.json"


def theme_key(theme: Theme) -> str:
    return f"{CUSTOM_THEME_PREFIX}{theme.metadata.selectable_id}"


def _token_value(theme: Theme, group: str, field: str) -> str:
    source = theme.ui if group == "ui" else theme.semantic
    return str(getattr(source, field))


def render_theme(theme: Theme) -> str:
    document = {
        "name": f"SF2 {theme.metadata.display_name}",
        "base": theme.metadata.variant.value,
        "overrides": {key: _token_value(theme, group, field) for key, group, field in OVERRIDE_TOKENS},
    }
    return json.dumps(document, indent=2) + "\n"


def merge_settings(existing: str, theme: Theme) -> str:
    try:
        data = json.loads(existing) if existing.strip() else {}
    except json.JSONDecodeError as error:
        raise ThemeError(f"Claude settings.json is not valid JSON: {error}") from error
    data["theme"] = theme_key(theme)
    return json.dumps(data, indent=2) + "\n"


def plan_claude(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    replace_theme: bool = True,
) -> list[Step]:
    steps: list[Step] = [Write(theme_path(entry, config_dir), render_theme(entry)) for entry in themes]
    path = settings_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    selected: Theme | None = theme
    if not replace_theme and _has_sf2_theme(existing):
        current_id = json.loads(existing).get("theme", "").removeprefix(CUSTOM_THEME_PREFIX)
        selected = next((candidate for candidate in themes if candidate.metadata.selectable_id == current_id), None)
    if selected is not None:
        steps.append(Write(path, merge_settings(existing, selected)))
    return steps


def _has_sf2_theme(existing: str) -> bool:
    if not existing.strip():
        return False
    configured = json.loads(existing).get("theme")
    return isinstance(configured, str) and configured.startswith(CUSTOM_THEME_PREFIX)


def read_current_id(config_dir: Path | None) -> str:
    path = settings_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Claude theme applied yet ({path} missing)")
    configured = json.loads(path.read_text(encoding="utf-8")).get("theme")
    if not isinstance(configured, str) or not configured.startswith(CUSTOM_THEME_PREFIX):
        raise ThemeError(f"could not read theme id from {path}")
    return configured.removeprefix(CUSTOM_THEME_PREFIX)
