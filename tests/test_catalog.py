"""Catalog discovery, aliases, and default theme."""

import pytest

from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError


def test_main_is_default_and_has_legacy_alias() -> None:
    catalog = parse_catalog()
    main = get_theme("main", catalog)
    assert get_theme("street-fighter-2", catalog) is main
    assert get_theme("street-fighter-ii-main", catalog) is main
    assert main.metadata.display_name == "Street Fighter II - Main"


def test_unknown_theme_errors() -> None:
    with pytest.raises(ThemeError, match="unknown theme"):
        get_theme("bison", parse_catalog())


def test_catalog_has_thirty_six_themes_and_role_aliases() -> None:
    catalog = parse_catalog()
    assert len(catalog) == 36
    assert get_theme("boxer", catalog).metadata.id == "balrog"
    assert get_theme("claw", catalog).metadata.id == "vega"
    assert get_theme("dictator", catalog).metadata.id == "m-bison"


def test_catalog_has_light_variant_for_every_dark_theme() -> None:
    catalog = parse_catalog()
    by_id = {theme.metadata.id: theme for theme in catalog}
    dark_themes = tuple(theme for theme in catalog if not theme.metadata.id.endswith("-light"))

    assert len(catalog) == 36
    assert len(dark_themes) == 18
    for dark in dark_themes:
        light = by_id[f"{dark.metadata.id}-light"]
        assert light.metadata.display_name == f"{dark.metadata.display_name} Light"
        assert light.ui.background != dark.ui.background
        assert light.ui.foreground != dark.ui.foreground
