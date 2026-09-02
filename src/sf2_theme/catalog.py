"""Discover, load, and resolve packaged theme files."""

import os
import tomllib
from collections.abc import Mapping, Sequence
from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.model import SELECTABLE_PREFIX, Theme, theme_to_toml
from sf2_theme.parse import parse_theme
from sf2_theme.validation import Severity, require_valid, validate_catalog, validate_theme


def themes_directory() -> Path:
    """Return the authoritative themes directory."""
    configured = os.environ.get("SF2_THEME_DIR")
    if configured:
        return Path(configured).expanduser()
    repo = Path(__file__).resolve().parents[2] / "themes"
    if (repo / "main.toml").is_file():
        return repo
    raise ThemeError("theme catalog not found; set SF2_THEME_DIR or run from the repository")


def _embedded_files() -> Mapping[str, str] | None:
    try:
        from sf2_theme._embedded import THEME_FILES
    except ImportError:
        return None
    return THEME_FILES


def parse_catalog() -> tuple[Theme, ...]:
    """Load every packaged theme without contrast/uniqueness enforcement."""
    embedded = _embedded_files()
    if embedded is not None:
        return tuple(_parse_embedded(embedded))
    return tuple(_parse_directory(themes_directory()))


def load_catalog() -> tuple[Theme, ...]:
    """Load the catalog and reject themes that fail error-severity checks."""
    themes = parse_catalog()
    errors = [issue.message for issue in validate_catalog(themes) if issue.severity is Severity.ERROR]
    if errors:
        raise ThemeError("\n".join(errors))
    for theme in themes:
        require_valid(theme)
    return themes


def _parse_directory(root: Path) -> list[Theme]:
    paths = [root / "main.toml"]
    main_light = root / "main-light.toml"
    if main_light.is_file():
        paths.append(main_light)
    paths.extend(sorted((root / "characters").glob("*.toml")))
    themes: list[Theme] = []
    for path in paths:
        if not path.is_file():
            raise ThemeError(f"missing theme file: {path}")
        raw = tomllib.loads(path.read_text(encoding="utf-8"))
        themes.append(parse_theme(raw, source=str(path)))
    return themes


def _parse_embedded(files: Mapping[str, str]) -> list[Theme]:
    names = ["main.toml"]
    if "main-light.toml" in files:
        names.append("main-light.toml")
    names.extend(sorted(name for name in files if name.startswith("characters/")))
    themes: list[Theme] = []
    for name in names:
        source = files.get(name)
        if source is None:
            raise ThemeError(f"embedded catalog is missing {name}")
        themes.append(parse_theme(tomllib.loads(source), source=name))
    return themes


def get_theme(name: str, themes: tuple[Theme, ...] | None = None) -> Theme:
    """Resolve a theme by id, alias, or display name."""
    catalog = themes if themes is not None else load_catalog()
    lowered = name.lower()
    for theme in catalog:
        keys = {theme.metadata.id.lower(), theme.metadata.display_name.lower()}
        keys.update(alias.lower() for alias in theme.metadata.aliases)
        if lowered in keys:
            return theme
    raise ThemeError(f"unknown theme: {name}")


def theme_pair(theme: Theme, themes: Sequence[Theme]) -> tuple[Theme, Theme]:
    """Return the dark and light siblings for a selected catalog theme."""
    catalog = tuple(themes)
    dark_id = theme.metadata.id.removesuffix("-light")
    return get_theme(dark_id, catalog), get_theme(f"{dark_id}-light", catalog)


def installed_theme(name: str, themes: Sequence[Theme]) -> Theme:
    """Resolve a catalog id or an adapter-installed sf2- identity."""
    catalog = tuple(themes)
    if name.startswith(SELECTABLE_PREFIX):
        return get_theme(name.removeprefix(SELECTABLE_PREFIX), catalog)
    return get_theme(name, catalog)


def default_theme(themes: tuple[Theme, ...] | None = None) -> Theme:
    """Return the main theme."""
    return get_theme("main", themes)


def show_theme(theme: Theme) -> str:
    """Render canonical TOML for a resolved theme."""
    return theme_to_toml(theme)


def catalog_issues(themes: tuple[Theme, ...]) -> list[str]:
    """Return printable error and warning lines for the whole catalog."""
    lines: list[str] = []
    for issue in validate_catalog(themes):
        lines.append(f"{issue.severity.value}: {issue.message}")
    for theme in themes:
        for issue in validate_theme(theme):
            lines.append(f"{issue.severity.value}: {issue.message}")
    return lines
