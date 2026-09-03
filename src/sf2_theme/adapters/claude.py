import json
import os
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme

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


def _write_themes(
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    return [
        write_file(
            theme_path(candidate, config_dir),
            render_theme(candidate),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
        for candidate in themes
    ]


def _write_settings(
    theme: Theme,
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> WriteResult:
    path = settings_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    return write_file(
        path,
        merge_settings(existing, theme),
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )


def apply_claude(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    results = _write_themes(themes, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks)
    results.append(_write_settings(theme, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks))
    return results


def setup_claude(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    replace_theme: bool = True,
) -> list[WriteResult]:
    results = _write_themes(themes, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks)
    path = settings_path(config_dir)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    if replace_theme or not _has_sf2_theme(existing):
        results.append(_write_settings(theme, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks))
    else:
        configured = json.loads(existing).get("theme", "")
        current_id = configured.removeprefix(CUSTOM_THEME_PREFIX)
        legacy_theme = next((candidate for candidate in themes if candidate.metadata.selectable_id == current_id), None)
        if legacy_theme is not None:
            results.append(
                _write_settings(legacy_theme, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks)
            )
    return results


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
